package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	dogumigration "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	corev1 "k8s.io/api/core/v1"
)

// ReasonMigrationPending is the Ready reason while a dogu is being migrated from v2 to v3.
// TODO: replace with v3beta1.ReasonMigrationPending once k8s-dogu-lib provides it.
const ReasonMigrationPending = "MigrationPending"

const pendingRequeueTime = time.Minute

// MigrationPendingStep is a no-op placeholder for the actual migration steps, which will replace it.
// It only keeps the dogu in the MigrationPending state.
type MigrationPendingStep struct {
	detector      MigrationDetector
	eventRecorder EventRecorder
}

func NewMigrationPendingStep(detector MigrationDetector, recorder EventRecorder) *MigrationPendingStep {
	return &MigrationPendingStep{detector: detector, eventRecorder: recorder}
}

func (s *MigrationPendingStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	state, err := s.detector.Detect(ctx, doguResource)
	if err != nil {
		return stepsv3.RequeueWithError(fmt.Errorf("failed to detect migration state of dogu %q: %w", doguResource.Name, err), ReasonMigrationPending)
	}

	msg := "V2 resources found, the dogu has to be migrated to v3"
	if state == dogumigration.InProgress {
		msg = "Migration resources found, the migration to v3 has to be resumed"
	}
	s.eventRecorder.Event(doguResource, corev1.EventTypeNormal, ReasonMigrationPending, msg)
	return stepsv3.RequeueAfter(pendingRequeueTime, ReasonMigrationPending, msg)
}
