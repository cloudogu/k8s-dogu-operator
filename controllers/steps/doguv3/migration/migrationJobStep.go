package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	dogumigration "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
)

const jobPollInterval = 10 * time.Second

// MigrationJobStep waits until the migration jobs of the dogu's chart completed.
type MigrationJobStep struct {
	charts        ChartService
	monitor       JobMonitor
	eventRecorder EventRecorder
}

func NewMigrationJobStep(charts ChartService, monitor JobMonitor, recorder EventRecorder) *MigrationJobStep {
	return &MigrationJobStep{charts: charts, monitor: monitor, eventRecorder: recorder}
}

func (s *MigrationJobStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	hasMigrationJob, err := s.charts.HasMigrationJob(ctx, doguResource)
	if err != nil {
		return s.fail(doguResource, fmt.Errorf("failed to check for migration job: %w", err))
	}

	if !hasMigrationJob {
		return stepsv3.Continue()
	}

	status, err := s.monitor.Status(ctx, doguResource)
	if err != nil {
		return s.fail(doguResource, fmt.Errorf("failed to get migration job status: %w", err))
	}

	switch status.Phase {
	case dogumigration.JobsSucceeded:
		s.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, status.Message)
		return stepsv3.Continue()
	case dogumigration.JobsFailed:
		return stepsv3.Abort(v3beta1.ReasonMigrationFailed, status.Message)
	case dogumigration.JobsPending:
		// The step polls the jobs, so only progress changes are recorded to avoid flooding the events.
		if readyMessage(doguResource) != status.Message {
			s.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, status.Message)
		}

		return stepsv3.RequeueAfter(jobPollInterval, v3beta1.ReasonMigrationPending, status.Message)
	}
}

// fail records a warning, because the use case emits no event when it requeues.
func (s *MigrationJobStep) fail(doguResource *v3beta1.Dogu, err error) stepsv3.StepResult {
	s.eventRecorder.Event(doguResource, corev1.EventTypeWarning, v3beta1.ReasonMigrationPending, err.Error())

	return stepsv3.RequeueWithError(err, v3beta1.ReasonMigrationPending)
}

// readyMessage returns the message of the dogu's ready condition, which mirrors the last recorded progress.
func readyMessage(doguResource *v3beta1.Dogu) string {
	ready := meta.FindStatusCondition(doguResource.Status.Conditions, v3beta1.ConditionReady)
	if ready == nil {
		return ""
	}

	return ready.Message
}
