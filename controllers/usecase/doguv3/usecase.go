package doguv3

import (
	"context"
	"errors"
	"fmt"
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
// The actions are set up as [v3.Step]s during the use-case construction, independently of the dogu at hand, i. e. different
// dogus within the same lifecycle phase traverse the same list of steps.
type DoguUseCase struct {
	steps         []Step
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
// The steps are executed in the order they are provided.
// With new steps the parameter list will grow.
// Unfortunately, this is the only way to provide the steps in the correct order.
// Uber fx provides value groups to use variadic parameters like NewDoguInstallOrChangeUseCase(steps ...doguv3.Step),
// but these are unordered.
func NewDoguInstallOrChangeUseCase(
	finalizerStep *install.CreateFinalizerStep,
	ociStep *install.EnsureOCIRepositoryStep,
	waitOCIStep *install.WaitForOCIRepositoryReadyStep,
	validateChartStep *install.ValidateChartStep,
	helmReleaseStep *install.EnsureHelmReleaseStep,
	client K8sClient, recorder EventRecorder) *DoguUseCase {
	return &DoguUseCase{
		steps: []Step{
			// CreateFinalizerStep MUST run before any owned resource (OCIRepository, HelmRelease) is
			// created, so a delete racing an install can never orphan those resources. (In v2 the
			// finalizer is added after its read-only gate steps, but likewise before resource creation;
			// v3 has no such gate steps, so "first" is the equivalent placement.)
			finalizerStep,
			ociStep,
			waitOCIStep,
			validateChartStep,
			helmReleaseStep,
		},
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
	for _, s := range duc.steps {
		result := s.Run(ctx, doguResource)

		// Abort, requeue or error
		if !result.Continue {
			return duc.stopStepExecution(ctx, doguResource, result)
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
	err := duc.updateReadyCondition(ctx, doguResource, metav1.ConditionTrue, v3beta1.ReasonSucceeded, successMessage)
	if err != nil {
		if apierrors.IsConflict(err) {
			return 5 * time.Second, nil
		}
		return 0, err
	}
	duc.eventRecorder.Event(doguResource, coreV1.EventTypeNormal, ReasonReconcileSuccess, successMessage)
	log.FromContext(ctx).Info(successMessage)

	return 0, nil
}

func (duc *DoguUseCase) stopStepExecution(ctx context.Context, doguResource *v3beta1.Dogu, result v3.StepResult) (time.Duration, error) {
	reason, msg := deriveReasonAndMessage(result)
	logger := log.FromContext(ctx)
	if err := duc.updateReadyCondition(ctx, doguResource, metav1.ConditionFalse, reason, msg); err != nil {
		if apierrors.IsConflict(err) {
			logger.V(1).Info("requeueing due to conflict", "dogu", doguResource.Name)
			return 5 * time.Second, nil
		}

		return result.RequeueAfter, errors.Join(result.Err, err)
	}
	// Only send events on abort and do not send events on requeues because this would flood the apiserver.
	if result.Err == nil && result.RequeueAfter == 0 {
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

// updateReadyCondition kapselt rein das Aktualisieren der K8s-Ressource
func (duc *DoguUseCase) updateReadyCondition(ctx context.Context, doguResource *v3beta1.Dogu, status metav1.ConditionStatus, reason, message string) error {
	condition := metav1.Condition{
		Type:               v3beta1.ConditionReady,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: doguResource.Generation,
	}

	if meta.SetStatusCondition(&doguResource.Status.Conditions, condition) {
		if err := duc.k8sClient.Status().Update(ctx, doguResource); err != nil {
			return fmt.Errorf("failed to update dogu status resource: %w", err)
		}
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
