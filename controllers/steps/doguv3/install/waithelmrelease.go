package install

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pending"
	fluxstate "github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
	flux "github.com/fluxcd/helm-controller/api/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// longWaitRequeueAfter is the safety net for a lost change of the Healthy condition.
// Normally, the change itself triggers the next reconciliation earlier.
const longWaitRequeueAfter = 5 * time.Minute

// HelmReleaseStatusStep translates the state of the dogu's HelmRelease and its workloads into the conditions of the dogu resource.
type HelmReleaseStatusStep struct {
	k8sClient     K8sClient
	releaseReader *fluxstate.HelmReleaseReader
	eventRecorder EventRecorder
}

func NewHelmReleaseStatusStep(k8sClient K8sClient, releaseReader *fluxstate.HelmReleaseReader, recorder EventRecorder) *HelmReleaseStatusStep {
	return &HelmReleaseStatusStep{
		k8sClient:     k8sClient,
		releaseReader: releaseReader,
		eventRecorder: recorder,
	}
}

// PendingChange compares the desired release with Flux and Helm data without changing resources.
func (hrs *HelmReleaseStatusStep) PendingChange(ctx context.Context, dogu *v3beta1.Dogu, desired *flux.HelmRelease) (pending.PendingChange, error) {
	release := &flux.HelmRelease{}
	err := hrs.k8sClient.Get(ctx, client.ObjectKey{Namespace: dogu.Namespace, Name: dogu.Spec.Name}, release)
	if errors.IsNotFound(err) {
		return pending.PendingReleaseChange(nil, desired, nil, dogu.Spec.Version), nil
	}
	if err != nil {
		return pending.PendingChange{Status: metav1.ConditionUnknown, Message: "The HelmRelease could not be read"}, fmt.Errorf("failed to read HelmRelease for pending changes: %w", err)
	}
	// Detect known differences before reading Helm's release secret.
	change := pending.PendingReleaseChange(release, desired, nil, dogu.Spec.Version)
	if change.Status != metav1.ConditionUnknown || desired == nil {
		return change, nil
	}
	// Complete the comparison with the values and manifest stored by Helm.
	stored, err := hrs.releaseReader.StoredRelease(release)
	if err != nil {
		return change, err
	}
	return pending.PendingReleaseChange(release, desired, stored, dogu.Spec.Version), nil
}

func (hrs *HelmReleaseStatusStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	if doguResource.Spec.PauseReconciliation {
		return stepsv3.Abort(v3beta1.ReasonPauseReconciliation, "Dogu reconciliation is paused")
	}
	// Get the HelmRelease
	release := &flux.HelmRelease{}
	err := hrs.k8sClient.Get(ctx, client.ObjectKey{Namespace: doguResource.Namespace, Name: doguResource.Spec.Name}, release)
	// cache lag
	if errors.IsNotFound(err) {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, err.Error())
	}
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to get HelmRelease: %w", err), v3beta1.ReasonInstalling)
	}
	if release.Spec.Suspend {
		// A confirmed stop does not need a rollout; the release stays suspended.
		if doguResource.Spec.Stopped && doguResource.Status.Stopped {
			return stepsv3.Continue()
		}
		// Starting requires the separate suspension step to let Flux apply the prepared release.
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonStarting, "Waiting for the suspension step to release the prepared HelmRelease")
	}
	// Do not confirm a stop while Flux can still apply the release.
	if doguResource.Spec.Stopped {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonStopping, "Waiting for HelmRelease suspension")
	}

	desired := fluxstate.NewDesiredRelease(doguResource.Spec.Version, release)

	state := fluxstate.EvaluateRelease(release, desired)

	return hrs.stepResultForRelease(state, doguResource)
}

func (hrs *HelmReleaseStatusStep) stepResultForRelease(state fluxstate.ReleaseState, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	reason := v3beta1.ReasonInstalling
	if state.EverDeployed {
		reason = v3beta1.ReasonUpgrading
	}

	switch state.Phase {
	case fluxstate.PhaseChartUnavailable:
		return stepsv3.Abort(v3beta1.ReasonDownloadFailed, state.Message)
	case fluxstate.PhaseInstalling:
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, state.Message)
	case fluxstate.PhaseUpgrading:
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonUpgrading, state.Message)
	case fluxstate.PhaseInstallFailed:
		return stepsv3.Abort(v3beta1.ReasonInstallFailed, state.Message)
	case fluxstate.PhaseUpgradeFailed:
		return stepsv3.Abort(v3beta1.ReasonUpgradeFailed, state.Message)
	case fluxstate.PhaseDeployed:
		return stepResultForDeployed(doguResource)
	default:
		return stepsv3.RequeueWithError(fmt.Errorf("unknown release phase %q", state.Phase), reason)
	}
}

// stepResultForDeployed derives the Ready condition of a deployed dogu from its Healthy condition.
func stepResultForDeployed(doguResource *v3beta1.Dogu) stepsv3.StepResult {
	healthy := metautil.FindStatusCondition(doguResource.Status.Conditions, v3beta1.ConditionHealthy)
	if healthy == nil {
		return stepsv3.RequeueAfter(longWaitRequeueAfter, v3beta1.ReasonWorkloadsNotReady, "Waiting for the health check of the dogu")
	}

	if healthy.Status != metav1.ConditionTrue || healthy.ObservedGeneration < doguResource.Generation {
		return stepsv3.RequeueAfter(longWaitRequeueAfter, v3beta1.ReasonWorkloadsNotReady, healthy.Message)
	}

	return stepsv3.Continue()
}
