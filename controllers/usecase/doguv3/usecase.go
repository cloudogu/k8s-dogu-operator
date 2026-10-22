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
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/migration"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pending"
	flux "github.com/fluxcd/helm-controller/api/v2"
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
type DoguUseCase struct {
	steps                 []Step
	k8sClient             K8sClient
	eventRecorder         EventRecorder
	ociStep               *install.EnsureOCIRepositoryStep
	helmReleaseStep       *install.EnsureHelmReleaseStep
	helmReleaseStatusStep *install.HelmReleaseStatusStep
}

func NewDoguDeleteUseCase(
	unsuspendStep *deletion.UnsuspendHelmReleaseStep,
	deleteHelmReleaseStep *deletion.DeleteHelmReleaseStep,
	deleteOCIRepositoryStep *deletion.DeleteOCIRepositoryStep,
	removeFinalizerStep *deletion.RemoveFinalizerStep,
	client K8sClient, recorder EventRecorder) *DoguUseCase {
	return &DoguUseCase{
		steps: []Step{
			// Unsuspend first so a suspended HelmRelease is actually processed (and thus uninstalled)
			// by flux once we delete it. Then delete the HelmRelease and wait for flux to finish the
			// uninstall, delete the chart source, and finally drop the finalizer so the dogu cr is removed.
			unsuspendStep,
			deleteHelmReleaseStep,
			deleteOCIRepositoryStep,
			removeFinalizerStep,
		},
		k8sClient:     client,
		eventRecorder: recorder,
	}
}

// NewDoguInstallOrChangeUseCase creates a new DoguUseCase instance with the given steps.
func NewDoguInstallOrChangeUseCase(
	finalizerStep *install.CreateFinalizerStep,
	ociStep *install.EnsureOCIRepositoryStep,
	waitOCIStep *install.WaitForOCIRepositoryReadyStep,
	validateChartStep *install.ValidateChartStep,
	suspendHelmReleaseStep *install.SuspendHelmReleaseStep,
	helmReleaseStep *install.EnsureHelmReleaseStep,
	helmReleaseStatusStep *install.HelmReleaseStatusStep,
	exportModeStep *install.ExportModeStep,
	client K8sClient, recorder EventRecorder) *DoguUseCase {
	stopStep := install.NewStopWorkloadsStep(client)
	return &DoguUseCase{
		steps:                 []Step{exportModeStep, finalizerStep, ociStep, waitOCIStep, validateChartStep, suspendHelmReleaseStep, stopStep, helmReleaseStep, helmReleaseStatusStep},
		k8sClient:             client,
		eventRecorder:         recorder,
		ociStep:               ociStep,
		helmReleaseStep:       helmReleaseStep,
		helmReleaseStatusStep: helmReleaseStatusStep,
	}
}

// NewDoguMigrationUseCase creates a new DoguUseCase instance that migrates a v2 dogu to v3.
// The steps are executed in the order they are provided.
func NewDoguMigrationUseCase(
	finalizerStep *install.CreateFinalizerStep,
	migrationPreconditionsStep *migration.MigrationPreconditionsStep,
	ociStep *install.EnsureOCIRepositoryStep,
	waitOCIStep *install.WaitForOCIRepositoryReadyStep,
	validateChartStep *install.ValidateChartStep,
	migrationPendingStep *migration.MigrationPendingStep,
	client K8sClient,
	recorder EventRecorder,
) *DoguUseCase {
	return &DoguUseCase{
		steps: []Step{
			finalizerStep,
			migrationPreconditionsStep,
			// The chart is validated before the migration, so that a v2 dogu is never removed for a chart that cannot be installed.
			ociStep,
			waitOCIStep,
			validateChartStep,
			// MigrationPendingStep stands in for the actual migration steps and halts the use-case,
			// so that no chart is installed next to the v2 resources.
			migrationPendingStep,
		},
		k8sClient:     client,
		eventRecorder: recorder,
	}
}

// HandleUntilApplied runs the selected chain until a step stops execution or all steps complete.
// It returns any retry delay and error to the controller rather than waiting here.
func (duc *DoguUseCase) HandleUntilApplied(ctx context.Context, doguResource *v3beta1.Dogu) (time.Duration, error) {
	result := duc.runSteps(ctx, doguResource)
	// Pending describes desired input even if chart validation or another step stopped execution.
	if doguResource.GetDeletionTimestamp().IsZero() && duc.helmReleaseStatusStep != nil {
		if err := duc.updatePending(ctx, doguResource); err != nil {
			if apierrors.IsConflict(err) && result.Err == nil {
				return 5 * time.Second, nil
			}
			result.Err = errors.Join(result.Err, err)
			result.Continue = false
		}
	}
	if !result.Continue {
		return duc.stopStepExecution(ctx, doguResource, result)
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
		successMessage = "Dogu workloads are stopped; the desired HelmRelease is prepared"
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

func (duc *DoguUseCase) runSteps(ctx context.Context, dogu *v3beta1.Dogu) v3.StepResult {
	if dogu.GetDeletionTimestamp().IsZero() && dogu.Spec.PauseReconciliation {
		return v3.Abort(v3beta1.ReasonPauseReconciliation, "Dogu reconciliation is paused")
	}
	for _, step := range duc.steps {
		stepCtx := log.IntoContext(ctx, log.FromContext(ctx).WithName(stepName(step)))
		if result := step.Run(stepCtx, dogu); !result.Continue {
			return result
		}
	}
	return v3.Continue()
}

// updatePending combines fresh source and release comparisons, never a previous aggregate condition.
func (duc *DoguUseCase) updatePending(ctx context.Context, dogu *v3beta1.Dogu) error {
	sourceChange, sourceErr := duc.ociStep.PendingChange(ctx, dogu)
	var desired *flux.HelmRelease
	var valuesErr error
	// A paused source may still contain the old chart's metadata. Only assemble against current input.
	if sourceChange.Status == metav1.ConditionFalse {
		desired, valuesErr = duc.helmReleaseStep.DesiredRelease(ctx, dogu)
	}
	releaseChange, releaseErr := duc.helmReleaseStatusStep.PendingChange(ctx, dogu, desired)
	change := pending.CombinePendingChanges(sourceChange, releaseChange)
	reason := v3beta1.ReasonReconciliationEnabled
	if dogu.Spec.Stopped {
		reason = v3beta1.ReasonStopped
	}
	if dogu.Spec.PauseReconciliation {
		reason = v3beta1.ReasonPauseReconciliation
	}
	changed := meta.SetStatusCondition(&dogu.Status.Conditions, change.ToCondition(reason, dogu.Generation))
	var statusErr error
	if changed {
		statusErr = duc.saveStatus(ctx, dogu)
	}
	return errors.Join(sourceErr, valuesErr, releaseErr, statusErr)
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

func (duc *DoguUseCase) stopStepExecution(ctx context.Context, doguResource *v3beta1.Dogu, result v3.StepResult) (time.Duration, error) {
	reason, msg := deriveReasonAndMessage(result)
	logger := log.FromContext(ctx)
	readyChanged := setReadyCondition(doguResource, metav1.ConditionFalse, reason, msg)
	var statusErr error
	if readyChanged {
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
