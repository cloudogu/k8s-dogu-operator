package doguv3

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	v3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/deletion"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/install"
	coreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ReasonReconcileSuccess = "ReconcileSucceeded"
)

// DoguUseCase is a controlling structure responsible for handling all those actions that occur within the dogu's lifecycle phase.
// The step factory selects the steps to execute based on the desired Dogu state.
type DoguUseCase struct {
	stepFactory   StepFactory
	k8sClient     K8sClient
	eventRecorder EventRecorder
}

func NewDoguDeleteUseCase(
	unsuspendStep *deletion.UnsuspendHelmReleaseStep,
	deleteHelmReleaseStep *deletion.DeleteHelmReleaseStep,
	deleteOCIRepositoryStep *deletion.DeleteOCIRepositoryStep,
	removeFinalizerStep *deletion.RemoveFinalizerStep,
	client K8sClient, recorder EventRecorder) *DoguUseCase {
	return &DoguUseCase{
		stepFactory: StepFactory{deletionSteps: []Step{
			// Unsuspend first so a suspended HelmRelease is actually processed (and thus uninstalled)
			// by flux once we delete it. Then delete the HelmRelease and wait for flux to finish the
			// uninstall, delete the chart source, and finally drop the finalizer so the dogu cr is removed.
			unsuspendStep,
			deleteHelmReleaseStep,
			deleteOCIRepositoryStep,
			removeFinalizerStep,
		}},
		k8sClient:     client,
		eventRecorder: recorder,
	}
}

// NewDoguInstallOrChangeUseCase creates a new DoguUseCase instance with the given steps.
// The step factory defines their execution order for running and stopped Dogus.
func NewDoguInstallOrChangeUseCase(
	finalizerStep *install.CreateFinalizerStep,
	ociStep *install.EnsureOCIRepositoryStep,
	waitOCIStep *install.WaitForOCIRepositoryReadyStep,
	validateChartStep *install.ValidateChartStep,
	helmReleaseStep *install.EnsureHelmReleaseStep,
	helmReleaseStatusStep *install.HelmReleaseStatusStep,
	client K8sClient, recorder EventRecorder) *DoguUseCase {
	stopStep := install.NewStopWorkloadsStep(client)
	return &DoguUseCase{
		stepFactory:   NewStepFactory(finalizerStep, stopStep, ociStep, waitOCIStep, validateChartStep, helmReleaseStep, helmReleaseStatusStep),
		k8sClient:     client,
		eventRecorder: recorder,
	}
}

// HandleUntilApplied acts as the core control function which runs all use-case steps, independently of the dogu at hand. This function will wait until the Step was fully executed, resulting in either
//   - continuing to the next step (if any),
//   - a requeue if a step needs more time or errored in a way that might be handled programmatically
//   - an abortion of the dogus reconciliation
//
// If the dogu executed all steps successfully, the duration and the error will contain null values indicating the dogu phase is done.
func (duc *DoguUseCase) HandleUntilApplied(ctx context.Context, doguResource *v3beta1.Dogu) (time.Duration, error) {
	releaseCompared := false
	for _, s := range duc.stepFactory.StepsFor(doguResource) {
		// Name the logger after the step type so every log line carries the step that produced it
		// (e.g. "logger":"DeleteHelmReleaseStep"), without each step having to name itself.
		stepCtx := log.IntoContext(ctx, log.FromContext(ctx).WithName(stepName(s)))
		result := s.Run(stepCtx, doguResource)

		// Abort, requeue or error
		if !result.Continue {
			return duc.stopStepExecution(ctx, doguResource, result, releaseCompared)
		}
		if _, comparesRelease := s.(*install.EnsureHelmReleaseStep); comparesRelease {
			releaseCompared = true
		}
	}

	// During deletion the final step removes the finalizer, after which the dogu cr is gone. There is
	// no Ready condition to set (the resource is disappearing) and a Status().Update would race the
	// object's removal and fail with NotFound. So once all deletion steps completed, we are simply done.
	if !doguResource.GetDeletionTimestamp().IsZero() {
		return 0, nil
	}

	// success
	successMessage := "Dogu reconciliation completed successfully"
	if doguResource.Spec.Stopped {
		successMessage = "Dogu workloads are stopped; release changes remain deferred"
	}

	stoppedChanged := clearStoppedStatusAfterStart(doguResource) // Save Stopped status changes even if Ready is unchanged.
	readyChanged := setReadyCondition(doguResource, metav1.ConditionTrue, v3beta1.ReasonSucceeded, successMessage)
	var statusErr error
	if stoppedChanged || readyChanged {
		statusErr = duc.saveStatus(ctx, doguResource)
	}
	if apierrors.IsConflict(statusErr) {
		return 5 * time.Second, nil
	}
	if statusErr != nil {
		return 0, statusErr
	}
	duc.eventRecorder.Event(doguResource, coreV1.EventTypeNormal, ReasonReconcileSuccess, successMessage)
	log.FromContext(ctx).Info(successMessage)

	return 0, nil
}

// stepName derives a human-readable logger name from a step's concrete type (e.g. "DeleteHelmReleaseStep").
// The pointer guard keeps a value-type step from panicking on Elem().
func stepName(s Step) string {
	t := reflect.TypeOf(s)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
}

func (duc *DoguUseCase) stopStepExecution(ctx context.Context, doguResource *v3beta1.Dogu, result v3.StepResult, releaseCompared bool) (time.Duration, error) {
	reason, msg := deriveReasonAndMessage(result)
	logger := log.FromContext(ctx)
	pendingChanged := markPendingComparisonUnknown(doguResource, releaseCompared)
	readyChanged := setReadyCondition(doguResource, metav1.ConditionFalse, reason, msg)
	var statusErr error
	if pendingChanged || readyChanged {
		statusErr = duc.saveStatus(ctx, doguResource)
	}
	if apierrors.IsConflict(statusErr) {
		logger.V(1).Info("requeueing due to conflict", "dogu", doguResource.Name)
		return 5 * time.Second, nil
	}
	if statusErr != nil {
		return result.RequeueAfter, errors.Join(result.Err, statusErr)
	}
	// Only send events on abort and do not send events on requeues because this would flood the apiserver.
	if result.Err == nil && result.RequeueAfter == 0 && reason != v3beta1.ReasonPauseReconciliation && reason != v3beta1.ReasonStopped {
		logger.Error(errors.New(result.ReadyMessage), "dogu reconciliation aborted")
		duc.eventRecorder.Event(doguResource, coreV1.EventTypeWarning, result.ReadyReason, result.ReadyMessage)
	}

	if result.Err != nil {
		logger.Error(result.Err, "dogu reconciliation failed")
	} else {
		logger.V(1).Info("requeueing dogu reconciliation", "reason", reason, "requeueAfter", result.RequeueAfter)
	}

	return result.RequeueAfter, result.Err
}

// clearStoppedStatusAfterStart resets the previous stop status after a successful start and reports any change.
func clearStoppedStatusAfterStart(dogu *v3beta1.Dogu) bool {
	if dogu.Spec.Stopped {
		return false
	}
	if !dogu.Status.Stopped && meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionStopped) == nil {
		return false
	}

	stoppedChanged := dogu.Status.Stopped
	dogu.Status.Stopped = false

	conditionChanged := meta.SetStatusCondition(&dogu.Status.Conditions, metav1.Condition{
		Type:               v3beta1.ConditionStopped,
		Status:             metav1.ConditionFalse,
		Reason:             v3beta1.ReasonNotStopped,
		Message:            "Dogu workloads are running",
		ObservedGeneration: dogu.Generation,
	})
	return stoppedChanged || conditionChanged
}

// markPendingComparisonUnknown invalidates an incomplete comparison, preserving current True or Unknown results, and reports any change.
func markPendingComparisonUnknown(dogu *v3beta1.Dogu, releaseCompared bool) bool {
	if !dogu.Spec.Stopped || releaseCompared {
		return false
	}
	if dogu.Spec.PauseReconciliation || !dogu.GetDeletionTimestamp().IsZero() {
		return false
	}

	pending := meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionUpdatePending)
	if pending != nil && pending.ObservedGeneration == dogu.Generation &&
		(pending.Status == metav1.ConditionTrue || pending.Status == metav1.ConditionUnknown) {
		return false
	}

	// Global inputs can change without advancing the Dogu generation.
	return meta.SetStatusCondition(&dogu.Status.Conditions, metav1.Condition{
		Type:               v3beta1.ConditionUpdatePending,
		Status:             metav1.ConditionUnknown,
		Reason:             v3beta1.ReasonStopped,
		Message:            "The desired release cannot yet be fully compared",
		ObservedGeneration: dogu.Generation,
	})
}

// setReadyCondition updates Ready in memory and reports whether it changed.
func setReadyCondition(doguResource *v3beta1.Dogu, status metav1.ConditionStatus, reason, message string) bool {
	return meta.SetStatusCondition(&doguResource.Status.Conditions, metav1.Condition{
		Type:               v3beta1.ConditionReady,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: doguResource.Generation,
	})
}

// saveStatus persists the Dogu status through the Kubernetes API.
func (duc *DoguUseCase) saveStatus(ctx context.Context, doguResource *v3beta1.Dogu) error {
	if err := duc.k8sClient.Status().Update(ctx, doguResource); err != nil {
		return fmt.Errorf("failed to update dogu status resource: %w", err)
	}

	return nil
}

func deriveReasonAndMessage(result v3.StepResult) (string, string) {
	reason := result.ReadyReason
	msg := result.ReadyMessage

	if reason == "" {
		if result.Err != nil {
			reason = "StepFailed"
		} else if result.RequeueAfter != 0 {
			reason = v3beta1.ReasonInstalling
		} else {
			reason = "ReconciliationAborted"
		}
	}

	if msg == "" {
		if result.Err != nil {
			msg = result.Err.Error()
		} else if result.RequeueAfter != 0 {
			msg = "Waiting for background operation to complete"
		}
	}

	return reason, msg
}
