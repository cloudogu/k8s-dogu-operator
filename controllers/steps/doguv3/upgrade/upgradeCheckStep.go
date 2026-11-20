package upgrade

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
)

const (
	// ReasonUpgradeDetected is the reason of the event recorded when a dogu upgrade is detected.
	ReasonUpgradeDetected = "UpgradeDetected"
	// ReasonDowngradeNotAllowed is the reason for ConditionValid when an older version than the installed one is requested.
	ReasonDowngradeNotAllowed = "DowngradeNotAllowed"
	// ReasonInvalidVersion is the reason for ConditionValid when the requested or the installed version is not a valid semantic version.
	ReasonInvalidVersion = "InvalidVersion"
)

// CheckStep detects the version change of a dogu and refuses a downgrade.
type CheckStep struct {
	detector      UpgradeDetector
	k8sClient     K8sClient
	eventRecorder EventRecorder
}

func NewUpgradeCheckStep(detector UpgradeDetector, k8sClient K8sClient, recorder EventRecorder) *CheckStep {
	return &CheckStep{detector: detector, k8sClient: k8sClient, eventRecorder: recorder}
}

func (s *CheckStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	if doguResource.Spec.PauseReconciliation {
		logging.Debug(ctx, "Skipping because pausedReconciliation is set for dogu", "dogu", doguResource.Name)
		return stepsv3.Continue()
	}

	change, err := s.detector.Detect(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to detect version change of dogu %q: %w", doguResource.Name, err), v3beta1.ReasonUpgrading)
	}

	switch change.Kind {
	case upgrade.Upgrade:
		s.recordDetectedUpgrade(ctx, doguResource, change)
		return stepsv3.Continue()

	case upgrade.Downgrade:
		return refuse(ctx, s.k8sClient, doguResource, ReasonDowngradeNotAllowed, fmt.Sprintf("Downgrade from version %s to %s is not allowed", change.Installed, change.Target))

	case upgrade.InvalidVersion:
		msg := fmt.Sprintf("Version change from %s to %s is not allowed because a version is not a valid semantic version", change.Installed, change.Target)
		return refuse(ctx, s.k8sClient, doguResource, ReasonInvalidVersion, msg)

	default:
		return stepsv3.Continue()
	}
}

// recordDetectedUpgrade records the upgrade once per generation of the dogu resource.
func (s *CheckStep) recordDetectedUpgrade(ctx context.Context, doguResource *v3beta1.Dogu, change upgrade.Change) {
	valid := meta.FindStatusCondition(doguResource.Status.Conditions, v3beta1.ConditionValid)
	if valid != nil && valid.ObservedGeneration == doguResource.Generation {
		return
	}

	msg := fmt.Sprintf("Upgrade from version %s to %s detected", change.Installed, change.Target)
	s.eventRecorder.Event(doguResource, corev1.EventTypeNormal, ReasonUpgradeDetected, msg)
	logging.Info(ctx, msg, "dogu", doguResource.Name)
}
