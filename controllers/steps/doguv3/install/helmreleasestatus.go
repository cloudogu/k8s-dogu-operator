package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/health"
	fluxstate "github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
	flux "github.com/fluxcd/helm-controller/api/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metautil "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ReasonInstallFailed is the reason for the Ready condition if the installation of the dogu's release failed.
	// TODO replace with the constant from k8s-dogu-lib once it is available there.
	ReasonInstallFailed = "InstallFailed"
	// ReasonUpgradeFailed is the reason for the Ready condition if the upgrade of the dogu's release failed.
	// TODO replace with the constant from k8s-dogu-lib once it is available there.
	ReasonUpgradeFailed = "UpgradeFailed"
)

type healthChecker interface {
	Check(ctx context.Context, dogu *v3beta1.Dogu) (health.State, error)
}

// HelmReleaseStatusStep translates the state of the dogu's HelmRelease and its workloads into the conditions of the dogu resource.
type HelmReleaseStatusStep struct {
	k8sClient     K8sClient
	eventRecorder EventRecorder
	healthChecker healthChecker
}

func NewHelmReleaseStatusStep(k8sClient K8sClient, recorder EventRecorder, checker healthChecker) *HelmReleaseStatusStep {
	return &HelmReleaseStatusStep{
		k8sClient:     k8sClient,
		eventRecorder: recorder,
		healthChecker: checker,
	}
}

func (hrs *HelmReleaseStatusStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
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

	desired := fluxstate.NewDesiredRelease(doguResource.Spec.Version, release)

	state := fluxstate.EvaluateRelease(release, desired)

	return hrs.stepResultForRelease(ctx, state, doguResource)
}

func (hrs *HelmReleaseStatusStep) stepResultForRelease(ctx context.Context, state fluxstate.ReleaseState, doguResource *v3beta1.Dogu) stepsv3.StepResult {
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
		return stepsv3.Abort(ReasonInstallFailed, state.Message)
	case fluxstate.PhaseUpgradeFailed:
		return stepsv3.Abort(ReasonUpgradeFailed, state.Message)
	case fluxstate.PhaseDeployed:
		healthy, err := hrs.checkHealthy(ctx, doguResource)
		if err != nil {
			if errors.IsConflict(err) {
				// the dogu resource changed in the meantime, retry with the current version
				return stepsv3.RequeueAfter(defaultRequeueAfter, reason, err.Error())
			}

			return stepsv3.RequeueWithError(err, reason)
		}

		if !healthy {
			return stepsv3.RequeueAfter(defaultRequeueAfter, reason, "Dogu is not healthy")
		}

		// Dogu is healthy
		return stepsv3.Continue()
	default:
		return stepsv3.RequeueWithError(fmt.Errorf("unknown release phase %q", state.Phase), reason)
	}
}

// checkHealthy derives the health of the dogu from its workloads and persists it as Healthy condition.
func (hrs *HelmReleaseStatusStep) checkHealthy(ctx context.Context, doguResource *v3beta1.Dogu) (bool, error) {
	healthState, err := hrs.healthChecker.Check(ctx, doguResource)
	if err != nil {
		return false, fmt.Errorf("error checking dogu health: %w", err)
	}

	status := metav1.ConditionTrue
	if !healthState.Healthy {
		status = metav1.ConditionFalse
	}

	condition := metav1.Condition{
		Type:               v3beta1.ConditionHealthy,
		Status:             status,
		Reason:             healthState.Reason,
		Message:            healthState.Message,
		ObservedGeneration: doguResource.Generation,
	}

	if metautil.SetStatusCondition(&doguResource.Status.Conditions, condition) {
		if err := hrs.k8sClient.Status().Update(ctx, doguResource); err != nil {
			return false, fmt.Errorf("failed to update dogu status resource: %w", err)
		}
	}

	return healthState.Healthy, nil
}
