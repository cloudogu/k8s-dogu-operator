package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	fluxstate "github.com/cloudogu/k8s-dogu-operator/v3/internal/flux"
	flux "github.com/fluxcd/helm-controller/api/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// HelmReleaseStatusStep translates the state of the dogu's HelmRelease and its workloads into the conditions of the dogu resource.
type HelmReleaseStatusStep struct {
	k8sClient     K8sClient
	eventRecorder EventRecorder
}

func NewHelmReleaseStatusStep(k8sClient K8sClient, recorder EventRecorder) *HelmReleaseStatusStep {
	return &HelmReleaseStatusStep{
		k8sClient:     k8sClient,
		eventRecorder: recorder,
	}
}

func (hrs *HelmReleaseStatusStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	logger := log.FromContext(ctx)

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

	// TODO remove once the result is used in the next steps
	logger.Info("evaluated HelmRelease", "name", release.Name, "phase", state.Phase, "message", state.Message,
		"desiredChartVersion", desired.ChartVersion, "desiredGeneration", desired.Generation,
		"observedGeneration", release.Status.ObservedGeneration)

	// 4. Check the health of the dogu's workloads (StatefulSets and Deployments with label k8s.cloudogu.com/dogu.name):
	//  stopped dogu => Healthy=False/Stopped, missing or not ready workloads => Healthy=False/WorkloadsNotReady,
	//  otherwise Healthy=True/Succeeded.

	// 5. Set the Healthy condition and, if the HelmRelease reports ArtifactFailed, the ChartAvailable condition.
	//  Persist the status only if a condition changed; a conflict results in a requeue.
	//  Never touch conditions owned by other steps.

	// 6. Return the StepResult depending on the release phase (the use-case derives the Ready condition from it):
	//  ChartUnavailable                => Abort(ChartAvailable reason, msg)
	//  Installing / Upgrading          => RequeueAfter(fallback, ReasonInstalling / ReasonUpgrading, msg)
	//  InstallFailed / UpgradeFailed   => Abort(InstallFailed / UpgradeFailed, msg)
	//  Deployed, workloads not healthy => RequeueAfter(fallback, ReasonWorkloadsNotReady / ReasonStopped, msg)
	//  Deployed, workloads healthy     => Continue()

	return stepsv3.Continue()
}
