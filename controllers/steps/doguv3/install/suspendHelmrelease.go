package install

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	doguv3 "github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	values3 "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/values"
	flux "github.com/fluxcd/helm-controller/api/v2"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConditionSuspended          = "Suspended"
	ReasonNotSuspended          = "NotSuspended"
	ReasonPVCResizeInProgress   = "PVCResizeInProgress"
	ReasonDoguStopped           = "DoguStopped"
	ReasonReconciliationPaused  = "ReconciliationPaused"
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
	resizeChecker pvc.ResizeChecker
}

func NewSuspendHelmReleaseStep(k8sClient K8sClient, chartService ChartService, resizeChecker pvc.ResizeChecker, recorder EventRecorder) *SuspendHelmReleaseStep {
	return &SuspendHelmReleaseStep{
		k8sClient:     k8sClient,
		eventRecorder: recorder,
		chartService:  chartService,
		resizeChecker: resizeChecker,
	}
}

func (shr *SuspendHelmReleaseStep) Run(ctx context.Context, doguResource *doguv3.Dogu) stepsv3.StepResult {

	//Get Helm release if it exists
	helmRelease, hrErr := shr.existingHelmRelease(ctx, doguResource)
	if hrErr != nil {
		return stepsv3.RequeueWithError(hrErr, doguv3.ReasonInstalling)
	}
	//There is no helm release to suspend, so continue to the next step
	if helmRelease == nil {
		return stepsv3.Continue()
	}

	//Get the details required to check for pvc
	templateAsm := values3.NewAssembler(shr.k8sClient)
	combinedValues, err := combineValues(ctx, doguResource, shr.chartService, templateAsm)
	if err != nil {
		return stepsv3.RequeueWithError(err, doguv3.ReasonReconciliationPaused)
	}

	//Get the state of the suspension
	reason, message, err := shr.suspensionReason(ctx, doguResource, combinedValues)

	if err != nil {
		return stepsv3.RequeueWithError(err, doguv3.ReasonReconciliationPaused)
	}

	if helmRelease.Spec.Suspend {
		//We need to update if the reason for suspension has changed
		if reason == ReasonNotSuspended {
			if err := shr.ChangeSuspendValueOfHelmRelease(ctx, helmRelease, false); err != nil {
				return stepsv3.RequeueWithError(err, doguv3.ReasonReconciliationPaused)
			}
			if err := shr.setSuspendedConditionOnDoguCR(ctx, doguResource, reason, message); err != nil {
				return stepsv3.RequeueWithError(err, doguv3.ReasonReconciliationPaused)
			}
			return stepsv3.RequeueAfter(time.Duration(0), reason, message)
		}
		//TODO: what if the reason for suspension has changed pauseReconciliation <=> stop <=> PVC-Resize
	} else {
		if reason != ReasonNotSuspended {
			if err := shr.ChangeSuspendValueOfHelmRelease(ctx, helmRelease, true); err != nil {
				return stepsv3.RequeueWithError(err, doguv3.ReasonReconciliationPaused)
			}
			if ReasonReconciliationPaused == reason || ReasonDoguStopped == reason {
				if err := shr.setSuspendedConditionOnDoguCR(ctx, doguResource, reason, message); err != nil {
					return stepsv3.RequeueWithError(err, doguv3.ReasonReconciliationPaused)
				}
			}
			//TODO: what if the reason for suspension has changed pauseReconciliation / stop => PVC-Resize, then we are not updating the status of the dogu CR
			return stepsv3.RequeueAfter(time.Duration(0), reason, message)
		}
	}
	return stepsv3.Continue()
}

func (shr *SuspendHelmReleaseStep) shouldSuspendResult(ctx context.Context, doguResource *doguv3.Dogu) (bool, string, string, error) {
	suspendHelmRelease := false
	var reason, message string
	if doguResource.Spec.PauseReconciliation {
		suspendHelmRelease = true
		reason, message = ReasonReconciliationPaused, messageReconciliationPaused
	} else if doguResource.Spec.Stopped {
		suspendHelmRelease = true
		reason, message = ReasonDoguStopped, messageDoguStopped
	} else {
		resize, err := shr.resizeInProgress(ctx, doguResource)
		if err != nil {
			return false, "", "", err
		}
		if resize {
			suspendHelmRelease = true
			reason, message = ReasonPVCResizeInProgress, messagePVCResizeInProgress
		}
	}
	return suspendHelmRelease, reason, message, nil
}

func (shr *SuspendHelmReleaseStep) resizeInProgress(ctx context.Context, doguResource *doguv3.Dogu) (bool, error) {

	//TODO: return correct value
	return false, nil

	chartValues := map[string]any{}

	templateAsm := values3.NewAssembler(shr.k8sClient)
	values, err := combineValues(ctx, doguResource, shr.chartService, templateAsm)
	if err != nil {
		return false, err
	}

	if err := json.Unmarshal(values.Raw, &chartValues); err != nil {
		return false, fmt.Errorf("failed to decode assembled chart values for PVC resize check: %w", err)
	}
	renderedObjects, err := shr.chartService.Render(ctx, doguResource, chartValues)
	if err != nil {
		return false, fmt.Errorf("failed to render chart for PVC resize check: %w", err)
	}
	checkResult, err := shr.resizeChecker.Check(ctx, doguResource.Spec.DoguNamespace, renderedObjects)
	if err != nil {
		return false, fmt.Errorf("failed to check PVC resize requirements: %w", err)
	}
	if len(checkResult.ResizeRequests) > 0 {
		return true, nil
	}
	return false, nil
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

func (shr *SuspendHelmReleaseStep) ChangeSuspendValueOfHelmRelease(ctx context.Context, release *flux.HelmRelease, suspend bool) error {
	release.Spec.Suspend = suspend
	if err := shr.k8sClient.Update(ctx, release); err != nil {
		return fmt.Errorf("failed to suspend HelmRelease %q: %w", release.Name, err)
	}
	return nil
}

func (shr *SuspendHelmReleaseStep) suspensionReason(ctx context.Context, doguResource *doguv3.Dogu, values *apiext.JSON) (string, string, error) {
	if doguResource.Spec.PauseReconciliation {
		return ReasonReconciliationPaused, messageReconciliationPaused, nil
	}
	if doguResource.Spec.Stopped {
		return ReasonDoguStopped, messageDoguStopped, nil
	}

	chartValues := map[string]any{}
	if err := json.Unmarshal(values.Raw, &chartValues); err != nil {
		return "", "", fmt.Errorf("failed to decode assembled chart values for PVC resize check: %w", err)
	}
	renderedObjects, err := shr.chartService.Render(ctx, doguResource, chartValues)
	if err != nil {
		return "", "", fmt.Errorf("failed to render chart for PVC resize check: %w", err)
	}
	checkResult, err := shr.resizeChecker.Check(ctx, doguResource.Spec.DoguNamespace, renderedObjects)
	if err != nil {
		return "", "", fmt.Errorf("failed to check PVC resize requirements: %w", err)
	}
	if len(checkResult.ResizeRequests) > 0 {
		return ReasonPVCResizeInProgress, messagePVCResizeInProgress, nil
	}

	return ReasonNotSuspended, messageNotSuspended, nil
}

func (shr *SuspendHelmReleaseStep) setSuspendedConditionOnDoguCR(ctx context.Context, doguResource *doguv3.Dogu, reason, message string) error {
	status := metav1.ConditionFalse
	if reason != ReasonNotSuspended {
		status = metav1.ConditionTrue
	}
	condition := metav1.Condition{
		Type:               ConditionSuspended,
		Status:             status,
		Reason:             reason,
		Message:            message,
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
