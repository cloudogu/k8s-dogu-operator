package migration

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	dogumigration "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	corev1 "k8s.io/api/core/v1"
)

// MigrationCleanupStep removes the migration resources after the migration jobs completed.
// Without them the dogu is no longer detected as migrating, so the next reconciliation installs it the regular way.
type MigrationCleanupStep struct {
	k8sClient     K8sClient
	eventRecorder EventRecorder
}

func NewMigrationCleanupStep(k8sClient K8sClient, recorder EventRecorder) *MigrationCleanupStep {
	return &MigrationCleanupStep{k8sClient: k8sClient, eventRecorder: recorder}
}

func (s *MigrationCleanupStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	if err := dogumigration.CleanupMigrationResources(ctx, s.k8sClient, doguResource); err != nil {
		cErr := fmt.Errorf("failed to cleanup migration resources: %w", err)
		s.eventRecorder.Event(doguResource, corev1.EventTypeWarning, v3beta1.ReasonMigrationPending, cErr.Error())

		return stepsv3.RequeueWithError(cErr, v3beta1.ReasonMigrationPending)
	}

	s.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, "Migration resources removed")

	logging.Info(ctx, "Migration resources removed, completed migration.")

	// The regular installation resets the migration job flag and sets the dogu ready.
	return stepsv3.RequeueAfter(requeueTime, v3beta1.ReasonMigrationPending, "Migration completed")
}
