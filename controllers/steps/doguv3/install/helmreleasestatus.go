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

	// TODO remove once the release is evaluated in the next steps
	logValues := []any{"name", release.Name,
		"desiredChartVersion", desired.ChartVersion, "desiredGeneration", desired.Generation,
		"observedGeneration", release.Status.ObservedGeneration,
		"lastAttemptedGeneration", release.Status.LastAttemptedGeneration,
		"lastAttemptedRevision", release.Status.LastAttemptedRevision,
		"lastAttemptedConfigDigest", release.Status.LastAttemptedConfigDigest}
	if latest := release.Status.History.Latest(); latest != nil {
		logValues = append(logValues,
			"latestVersion", latest.Version, "latestStatus", latest.Status,
			"latestChartVersion", latest.ChartVersion, "latestConfigDigest", latest.ConfigDigest)
	}
	logger.Info("found HelmRelease", logValues...)

	// 3. Evaluate the release phase from the desired release, status.observedGeneration, status.history and the
	//  Flux Ready condition (pure function in internal/flux):
	//  ChartUnavailable, Installing, Upgrading, InstallFailed, UpgradeFailed or Deployed.
	//  observedGeneration < generation means Flux has not finished the current spec yet; only then are history and
	//  the Flux Ready condition up to date. Compare the chart version without the build metadata Flux appends ("+<digest>").
	//  Distinguish install from upgrade by whether any revision was ever deployed,
	//  because with RetryOnFailure Flux retries a failed install as an upgrade.

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
