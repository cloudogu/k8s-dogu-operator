package install

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	fluxstate "github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
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
	k8sClient             K8sClient
	eventRecorder         EventRecorder
	chartService          ChartService
	assembler             ValueAssembler
	releaseReader         *fluxstate.HelmReleaseReader
	helmReconcileInterval metav1.Duration
	retryInterval         metav1.Duration
}

func NewHelmReleaseStatusStep(
	k8sClient K8sClient,
	chartService ChartService,
	assembler ValueAssembler,
	operatorConfig *config.OperatorConfig,
	releaseReader *fluxstate.HelmReleaseReader,
	recorder EventRecorder,
) *HelmReleaseStatusStep {
	return &HelmReleaseStatusStep{
		k8sClient:             k8sClient,
		eventRecorder:         recorder,
		chartService:          chartService,
		assembler:             assembler,
		releaseReader:         releaseReader,
		helmReconcileInterval: metav1.Duration{Duration: operatorConfig.DoguHelmReconciliationInterval},
		retryInterval:         metav1.Duration{Duration: operatorConfig.DoguHelmRetryInterval},
	}
}

func (hrs *HelmReleaseStatusStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	// Get the HelmRelease
	release := &flux.HelmRelease{}
	err := hrs.k8sClient.Get(ctx, client.ObjectKey{Namespace: doguResource.Namespace, Name: doguResource.Spec.Name}, release)
	// cache lag
	if errors.IsNotFound(err) {
		if doguResource.Spec.PauseReconciliation || doguResource.Spec.Stopped {
			return hrs.checkPendingChanges(ctx, doguResource, nil)
		}
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonInstalling, err.Error())
	}
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to get HelmRelease: %w", err), v3beta1.ReasonInstalling)
	}

	if doguResource.Spec.PauseReconciliation || doguResource.Spec.Stopped {
		if result := hrs.checkPendingChanges(ctx, doguResource, release); !result.Continue {
			return result
		}
		if doguResource.Spec.PauseReconciliation {
			return stepsv3.Continue()
		}
	}

	if release.Spec.Suspend {
		if doguResource.Spec.Stopped && doguResource.Status.Stopped {
			return stepsv3.Continue()
		}
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonStarting, "Waiting for the suspension step to release the prepared HelmRelease")
	}
	if doguResource.Spec.Stopped {
		return stepsv3.RequeueAfter(defaultRequeueAfter, v3beta1.ReasonStopping, "Waiting for HelmRelease suspension")
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
		return stepsv3.Abort(v3beta1.ReasonInstallFailed, state.Message)
	case fluxstate.PhaseUpgradeFailed:
		return stepsv3.Abort(v3beta1.ReasonUpgradeFailed, state.Message)
	case fluxstate.PhaseDeployed:
		return stepResultForDeployed(ctx, doguResource)
	default:
		return stepsv3.RequeueWithError(fmt.Errorf("unknown release phase %q", state.Phase), reason)
	}
}

// stepResultForDeployed derives the Ready condition of a deployed dogu from its Healthy condition.
func stepResultForDeployed(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	healthy := metautil.FindStatusCondition(doguResource.Status.Conditions, v3beta1.ConditionHealthy)
	if healthy == nil {
		return stepsv3.RequeueAfter(longWaitRequeueAfter, v3beta1.ReasonWorkloadsNotReady, "Waiting for the health check of the dogu")
	}

	if healthy.Status != metav1.ConditionTrue || healthy.ObservedGeneration < doguResource.Generation {
		return stepsv3.RequeueAfter(longWaitRequeueAfter, v3beta1.ReasonWorkloadsNotReady, healthy.Message)
	}
	logging.Info(ctx, "Dogu's Helm release is deployed and healthy", "dogu", doguResource.Name)
	return stepsv3.Continue()
}
