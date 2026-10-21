package migration

import (
	"context"
	"fmt"
	"strings"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	dogumigration "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
)

// MigrationPreconditionsStep checks whether the v2 dogu can be migrated to v3.
// The preconditions are only checked before the migration starts, a migration in progress continues.
type MigrationPreconditionsStep struct {
	detector      MigrationDetector
	eventRecorder EventRecorder
}

func NewMigrationPreconditionsStep(detector MigrationDetector, recorder EventRecorder) *MigrationPreconditionsStep {
	return &MigrationPreconditionsStep{detector: detector, eventRecorder: recorder}
}

func (s *MigrationPreconditionsStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	state, err := s.detector.Detect(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to detect migration state of dogu %q: %w", doguResource.Name, err), ReasonMigrationPending)
	}
	if state != dogumigration.Required {
		return stepsv3.Continue()
	}

	unmet := unmetPreconditions(doguResource.Status)
	if len(unmet) == 0 {
		return stepsv3.Continue()
	}

	msg := "The migration to v3 is blocked by the v2 dogu: " + strings.Join(unmet, ", ")
	s.eventRecorder.Event(doguResource, corev1.EventTypeNormal, ReasonMigrationPending, msg)
	return stepsv3.RequeueAfter(pendingRequeueTime, ReasonMigrationPending, msg)
}

// unmetPreconditions compares against the v2 constants, because the status still holds the values the v2 reconciler wrote.
func unmetPreconditions(status v3beta1.DoguStatus) []string {
	var unmet []string
	if meta.IsStatusConditionTrue(status.Conditions, doguv2.ConditionSupportMode) {
		unmet = append(unmet, "support mode is active")
	}
	if status.ExportMode {
		unmet = append(unmet, "export mode is active")
	}
	if status.Status != doguv2.DoguStatusInstalled {
		unmet = append(unmet, fmt.Sprintf("operation %q is still running", status.Status))
	}
	// The v2 reconciler sets this condition to false while it resizes the data volume.
	if meta.IsStatusConditionFalse(status.Conditions, doguv2.ConditionMeetsMinVolumeSize) {
		unmet = append(unmet, "data volume is being resized")
	}
	return unmet
}
