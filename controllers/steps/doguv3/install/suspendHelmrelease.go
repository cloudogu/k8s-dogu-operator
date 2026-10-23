package install

import (
	"context"
	"encoding/json"
	"fmt"

	doguv3 "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	flux "github.com/fluxcd/helm-controller/api/v2"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	messageNotSuspended         = "HelmRelease reconciliation is enabled"
	messagePVCResizeInProgress  = "HelmRelease reconciliation is suspended while a PVC resize is in progress"
	messageDoguStopped          = "HelmRelease reconciliation is suspended because the Dogu is stopped"
	messageReconciliationPaused = "HelmRelease reconciliation is suspended because spec.pauseReconciliation is set"
)

// SuspendHelmReleaseStep ensures the HelmRelease is suspended under certain conditions.
type SuspendHelmReleaseStep struct {
	k8sClient     K8sClient
	eventRecorder EventRecorder
	chartService  ChartService
	assembler     ValueAssembler
	resizeChecker ResizeChecker
}

func NewSuspendHelmReleaseStep(k8sClient K8sClient, chartService ChartService, assembler ValueAssembler, resizeChecker ResizeChecker, recorder EventRecorder) *SuspendHelmReleaseStep {
	return &SuspendHelmReleaseStep{
		k8sClient:     k8sClient,
		eventRecorder: recorder,
		chartService:  chartService,
		assembler:     assembler,
		resizeChecker: resizeChecker,
	}
}

type desiredState struct {
	suspendHelmRelease bool
	reason             string
	message            string
}

func (shr *SuspendHelmReleaseStep) Run(ctx context.Context, doguResource *doguv3.Dogu) stepsv3.StepResult {
	// get Helm release if it exists
	helmRelease, err := shr.existingHelmRelease(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(err, doguv3.ReasonInstalling)
	}
	if helmRelease == nil {
		// if there is no helm release to suspend, continue to the next step
		return stepsv3.Continue()
	}

	targetState, err := shr.expectedState(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(err, doguv3.ReasonReconciliationPaused)
	}

	return shr.suspendOrRelease(ctx, doguResource, helmRelease, targetState)
}

func (shr *SuspendHelmReleaseStep) suspendOrRelease(ctx context.Context, doguResource *doguv3.Dogu, helmRelease *flux.HelmRelease, targetState desiredState) stepsv3.StepResult {
	err := shr.setSuspendValueOfHelmRelease(ctx, helmRelease, targetState.suspendHelmRelease)
	if err != nil {
		return stepsv3.RequeueWithError(err, doguv3.ReasonReconciliationPaused)
	}

	var result = stepsv3.Continue()
	if err := shr.setSuspendedConditionOnDoguCR(ctx, doguResource, targetState); err != nil {
		result = stepsv3.RequeueWithError(err, doguv3.ReasonReconciliationPaused)
	}
	return result
}

func (shr *SuspendHelmReleaseStep) existingHelmRelease(ctx context.Context, doguResource *doguv3.Dogu) (*flux.HelmRelease, error) {
	release := &flux.HelmRelease{}
	err := shr.k8sClient.Get(ctx, client.ObjectKey{Namespace: doguResource.Namespace, Name: doguResource.Spec.Name}, release)
	if apierrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return release, fmt.Errorf("failed to get HelmRelease %q before suspending: %w", doguResource.Spec.Name, err)
	}
	return release, nil
}

func (shr *SuspendHelmReleaseStep) expectedState(ctx context.Context, doguResource *doguv3.Dogu) (desiredState, error) {
	if doguResource.Spec.PauseReconciliation {
		return desiredState{
			suspendHelmRelease: true,
			reason:             doguv3.ReasonReconciliationPaused,
			message:            messageReconciliationPaused,
		}, nil
	}
	if doguResource.Spec.Stopped {
		return desiredState{
			suspendHelmRelease: true,
			reason:             doguv3.ReasonDoguStopped,
			message:            messageDoguStopped,
		}, nil
	}

	//Get the details required to check for pvc
	combinedValues, err := combineValues(ctx, doguResource, shr.chartService, shr.assembler)
	if err != nil {
		return desiredState{}, err
	}

	return shr.checkForPVCResize(ctx, doguResource, combinedValues)
}

func (shr *SuspendHelmReleaseStep) checkForPVCResize(ctx context.Context, doguResource *doguv3.Dogu, values *v1.JSON) (desiredState, error) {

	chartValues := map[string]any{}
	if err := json.Unmarshal(values.Raw, &chartValues); err != nil {
		return desiredState{}, fmt.Errorf("failed to decode assembled chart values for PVC resize check: %w", err)
	}
	renderedObjects, err := shr.chartService.Render(ctx, doguResource, chartValues)
	if err != nil {
		return desiredState{}, fmt.Errorf("failed to render chart for PVC resize check: %w", err)
	}

	checkResult, err := shr.resizeChecker.Check(ctx, doguResource.GetNamespace(), renderedObjects)
	if err != nil {
		return desiredState{}, fmt.Errorf("failed to check PVC resize requirements: %w", err)
	}
	if len(checkResult.ResizeRequests) > 0 {
		return desiredState{
			suspendHelmRelease: true,
			reason:             doguv3.ReasonPVCResizeInProgress,
			message:            messagePVCResizeInProgress,
		}, nil
	}

	return desiredState{
		suspendHelmRelease: false,
		reason:             doguv3.ReasonNotSuspended,
		message:            messageNotSuspended,
	}, nil
}

func (shr *SuspendHelmReleaseStep) setSuspendValueOfHelmRelease(ctx context.Context, release *flux.HelmRelease, suspend bool) (err error) {
	if release.Spec.Suspend == suspend {
		return nil
	}
	// only write to release now if suspend == true, as the unsuspend gets handled automatically later by helmRelease step
	if !suspend {
		logging.Info(ctx, "Suspension on HelmRelease for dogu will be released", "name", release.Name)
		return nil
	}
	releaseCR := release.DeepCopy()
	releaseCR.Spec.Suspend = suspend
	if err := shr.k8sClient.Patch(ctx, releaseCR, client.MergeFrom(release)); err != nil {
		return fmt.Errorf("failed to suspend HelmRelease %q: %w", release.Name, err)
	}
	logging.Info(ctx, "Suspended HelmRelease for dogu", "name", release.Name)
	return nil
}

func (shr *SuspendHelmReleaseStep) setSuspendedConditionOnDoguCR(ctx context.Context, doguResource *doguv3.Dogu, targetState desiredState) error {
	status := metav1.ConditionFalse
	if targetState.suspendHelmRelease {
		status = metav1.ConditionTrue
	}
	condition := metav1.Condition{
		Type:               doguv3.ConditionSuspended,
		Status:             status,
		Reason:             targetState.reason,
		Message:            targetState.message,
		ObservedGeneration: doguResource.Generation,
	}
	if !apimeta.SetStatusCondition(&doguResource.Status.Conditions, condition) {
		return nil
	}

	if err := shr.k8sClient.Status().Update(ctx, doguResource); err != nil {
		return fmt.Errorf("failed to update Dogu suspended condition: %w", err)
	}
	return nil
}
