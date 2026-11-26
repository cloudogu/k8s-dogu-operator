package upgrade

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3/install"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ReasonUpgradeDetected is the reason of the event recorded when a dogu upgrade is detected.
	ReasonUpgradeDetected = "UpgradeDetected"
	// ReasonDowngradeNotAllowed is the reason for ConditionValid when an older version than the installed one is requested.
	ReasonDowngradeNotAllowed = "DowngradeNotAllowed"
	// ReasonInvalidVersion is the reason for ConditionValid when the requested or the installed version is not a valid semantic version.
	ReasonInvalidVersion = "InvalidVersion"
	// ReasonUpgradeValid is the reason for ConditionValid when the upgrade descriptor allows the upgrade.
	ReasonUpgradeValid = "UpgradeValid"
	// ReasonUpgradeValidationSkipped is the reason for ConditionValid when the upgrade of a dogu with a dev chart is not validated.
	ReasonUpgradeValidationSkipped = "UpgradeValidationSkipped"
	// ReasonInvalidUpgradeDescriptor is the reason for ConditionValid when the upgrade descriptor cannot be parsed or is ambiguous.
	ReasonInvalidUpgradeDescriptor = "InvalidUpgradeDescriptor"
	// ReasonUpgradeDescriptorUnavailable is the reason for ConditionValid when the dogu registry does not provide the upgrade descriptor.
	ReasonUpgradeDescriptorUnavailable = "UpgradeDescriptorUnavailable"
)

// mapRefusedReasons maps the cause of a refused version change to the reason for ConditionValid.
func mapRefusedReasons(up upgrade.Cause) string {
	switch up {
	case upgrade.CauseDowngrade:
		return ReasonDowngradeNotAllowed
	case upgrade.CauseInvalidVersion:
		return ReasonInvalidVersion
	case upgrade.CauseInvalidUpgradePath:
		return v3beta1.ReasonNoUpgradePath
	case upgrade.CauseInvalidDescriptor:
		return ReasonInvalidUpgradeDescriptor
	case upgrade.CauseDescriptorUnavailable:
		return ReasonUpgradeDescriptorUnavailable
	default:
		return ""
	}
}

// ValidateUpgradeStep validates the version change of a dogu and records the result in ConditionValid.
// A refused version change aborts the reconciliation, so the release of the dogu stays untouched.
type ValidateUpgradeStep struct {
	validator        UpgradeValidator
	k8sClient        K8sClient
	eventRecorder    EventRecorder
	devChartsEnabled bool
}

// NewValidateUpgradeStep creates the step. If devChartsEnabled is set, the upgrade of a dogu with a dev chart is not validated.
func NewValidateUpgradeStep(validator UpgradeValidator, k8sClient K8sClient, recorder EventRecorder, devChartsEnabled bool) *ValidateUpgradeStep {
	return &ValidateUpgradeStep{
		validator:        validator,
		k8sClient:        k8sClient,
		eventRecorder:    recorder,
		devChartsEnabled: devChartsEnabled,
	}
}

func (s *ValidateUpgradeStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	if doguResource.Spec.PauseReconciliation {
		logging.Debug(ctx, "Skipping ValidateUpgradeStep because pausedReconciliation is set for dogu", "dogu", doguResource.Name)
		return stepsv3.Continue()
	}

	devChart := s.devChartsEnabled && doguResource.Annotations[install.AnnotationDevChartURL] != ""
	result, err := s.validator.Validate(ctx, doguResource, devChart)
	if err != nil {
		return s.retry(ctx, doguResource, err)
	}

	switch result.Verdict {
	case upgrade.Valid:
		return s.accept(ctx, doguResource, result, ReasonUpgradeValid)
	case upgrade.Skipped:
		return s.accept(ctx, doguResource, result, ReasonUpgradeValidationSkipped)
	case upgrade.Refused:
		return s.refuse(ctx, doguResource, result)
	default:
		return stepsv3.Continue()
	}
}

// recordDetectedUpgrade records the event of a detected upgrade.
func (s *ValidateUpgradeStep) recordDetectedUpgrade(ctx context.Context, doguResource *v3beta1.Dogu, change upgrade.Change) {
	msg := fmt.Sprintf("Upgrade from version %s to %s detected", change.Installed, change.Target)
	s.eventRecorder.Event(doguResource, corev1.EventTypeNormal, ReasonUpgradeDetected, msg)

	logging.Info(ctx, msg, "dogu", doguResource.Name)
}

// retry requeues the reconciliation. A temporarily unavailable upgrade descriptor invalidates the dogu until it is available.
func (s *ValidateUpgradeStep) retry(ctx context.Context, doguResource *v3beta1.Dogu, err error) stepsv3.StepResult {
	if errors.Is(err, upgrade.ErrDescriptorUnavailable) {
		if _, updateErr := s.updateValidCondition(ctx, doguResource, metav1.ConditionFalse, ReasonUpgradeDescriptorUnavailable, err.Error()); updateErr != nil {
			err = errors.Join(err, updateErr)
		}
	}

	return stepsv3.RequeueWithError(err, v3beta1.ReasonUpgrading)
}

// refuse aborts the reconciliation, so the release of the dogu stays untouched.
// A refused upgrade is recorded only when ConditionValid changes, so requeues do not repeat the event.
func (s *ValidateUpgradeStep) refuse(ctx context.Context, doguResource *v3beta1.Dogu, result upgrade.Result) stepsv3.StepResult {
	reason := mapRefusedReasons(result.Cause)
	changed, err := s.updateValidCondition(ctx, doguResource, metav1.ConditionFalse, reason, result.Message)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonUpgrading)
	}

	if changed {
		if result.Change.Kind == upgrade.Upgrade {
			s.recordDetectedUpgrade(ctx, doguResource, result.Change)
		}

		logging.Info(ctx, result.Message, "dogu", doguResource.Name)
	}

	return stepsv3.Abort(reason, result.Message)
}

// accept marks the dogu as valid and continues the reconciliation.
// The upgrade is recorded only when ConditionValid changes, so requeues do not repeat the events.
func (s *ValidateUpgradeStep) accept(ctx context.Context, doguResource *v3beta1.Dogu, result upgrade.Result, reason string) stepsv3.StepResult {
	changed, err := s.updateValidCondition(ctx, doguResource, metav1.ConditionTrue, reason, result.Message)
	if err != nil {
		return stepsv3.RequeueWithError(err, v3beta1.ReasonUpgrading)
	}

	if changed {
		s.recordDetectedUpgrade(ctx, doguResource, result.Change)
		if result.Verdict == upgrade.Skipped {
			s.eventRecorder.Event(doguResource, corev1.EventTypeNormal, ReasonUpgradeValidationSkipped, result.Message)
		}

		logging.Info(ctx, result.Message, "dogu", doguResource.Name)
	}

	return stepsv3.Continue()
}

// updateValidCondition sets ConditionValid of the dogu and persists it if it changed. It reports whether the condition changed.
func (s *ValidateUpgradeStep) updateValidCondition(ctx context.Context, doguResource *v3beta1.Dogu, status metav1.ConditionStatus, reason, message string) (bool, error) {
	changed := meta.SetStatusCondition(&doguResource.Status.Conditions, metav1.Condition{
		Type:               v3beta1.ConditionValid,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: doguResource.Generation,
	})

	if !changed {
		return false, nil
	}

	if err := s.k8sClient.Status().Update(ctx, doguResource); err != nil {
		return false, fmt.Errorf("failed to update valid condition of dogu %q: %w", doguResource.Name, err)
	}

	return true, nil
}
