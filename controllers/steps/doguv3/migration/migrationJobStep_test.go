package migration

import (
	"fmt"
	"testing"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	dogumigration "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMigrationJobStep_Run(t *testing.T) {
	t.Run("continues without waiting for a chart without migration jobs", func(t *testing.T) {
		dogu := testDogu()
		charts := NewMockChartService(t)
		charts.EXPECT().HasMigrationJob(testCtx, dogu).Return(false, nil)

		result := NewMigrationJobStep(charts, NewMockJobMonitor(t), NewMockEventRecorder(t)).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
	})

	t.Run("pending jobs requeue and record the changed progress", func(t *testing.T) {
		const pendingJobsMessage = "Waiting for 1 of 2 migration jobs to complete"

		dogu := testDogu()
		charts := NewMockChartService(t)
		charts.EXPECT().HasMigrationJob(testCtx, dogu).Return(true, nil)
		monitor := NewMockJobMonitor(t)
		monitor.EXPECT().Status(testCtx, dogu).Return(dogumigration.JobsStatus{Phase: dogumigration.JobsPending, Message: pendingJobsMessage}, nil)
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, pendingJobsMessage)

		result := NewMigrationJobStep(charts, monitor, recorder).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.RequeueAfter(jobPollInterval, v3beta1.ReasonMigrationPending, pendingJobsMessage), result)
	})

	t.Run("pending jobs record no event for an unchanged progress", func(t *testing.T) {
		const unchangedJobsMessage = "Waiting for the migration jobs to start"

		dogu := testDogu()
		dogu.Status.Conditions = []metav1.Condition{{Type: v3beta1.ConditionReady, Status: metav1.ConditionFalse, Reason: v3beta1.ReasonMigrationPending, Message: unchangedJobsMessage}}
		charts := NewMockChartService(t)
		charts.EXPECT().HasMigrationJob(testCtx, dogu).Return(true, nil)
		recorder := NewMockEventRecorder(t)
		monitor := NewMockJobMonitor(t)
		monitor.EXPECT().Status(testCtx, dogu).Return(dogumigration.JobsStatus{Phase: dogumigration.JobsPending, Message: unchangedJobsMessage}, nil)

		result := NewMigrationJobStep(charts, monitor, recorder).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.RequeueAfter(jobPollInterval, v3beta1.ReasonMigrationPending, unchangedJobsMessage), result)
		assert.Empty(t, recorder.Calls)
	})

	t.Run("failed jobs abort the migration", func(t *testing.T) {
		const failedJobsMessage = `Migration job "nexus-db" failed: BackoffLimitExceeded`

		dogu := testDogu()
		charts := NewMockChartService(t)
		charts.EXPECT().HasMigrationJob(testCtx, dogu).Return(true, nil)
		monitor := NewMockJobMonitor(t)
		monitor.EXPECT().Status(testCtx, dogu).Return(dogumigration.JobsStatus{Phase: dogumigration.JobsFailed, Message: failedJobsMessage}, nil)

		result := NewMigrationJobStep(charts, monitor, NewMockEventRecorder(t)).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Abort(v3beta1.ReasonMigrationFailed, failedJobsMessage), result)
	})

	t.Run("succeeded jobs continue and record the completion", func(t *testing.T) {
		const succeededJobsMessage = "All 2 migration jobs completed"

		dogu := testDogu()
		charts := NewMockChartService(t)
		charts.EXPECT().HasMigrationJob(testCtx, dogu).Return(true, nil)
		monitor := NewMockJobMonitor(t)
		monitor.EXPECT().Status(testCtx, dogu).Return(dogumigration.JobsStatus{Phase: dogumigration.JobsSucceeded, Message: succeededJobsMessage}, nil)
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, succeededJobsMessage)

		result := NewMigrationJobStep(charts, monitor, recorder).Run(testCtx, dogu)

		assert.Equal(t, stepsv3.Continue(), result)
	})

	t.Run("monitor error requeues with a warning", func(t *testing.T) {
		dogu := testDogu()
		charts := NewMockChartService(t)
		charts.EXPECT().HasMigrationJob(testCtx, dogu).Return(true, nil)
		monitor := NewMockJobMonitor(t)
		monitor.EXPECT().Status(testCtx, dogu).Return(dogumigration.JobsStatus{}, assert.AnError)
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeWarning, v3beta1.ReasonMigrationPending, fmt.Sprintf("failed to get migration job status: %v", assert.AnError))

		result := NewMigrationJobStep(charts, monitor, recorder).Run(testCtx, dogu)

		assert.False(t, result.Continue)
		assert.Equal(t, 0*time.Second, result.RequeueAfter)
		assert.ErrorContains(t, result.Err, "failed to get migration job status")
		assert.Equal(t, v3beta1.ReasonMigrationPending, result.ReadyReason)
	})

	t.Run("chart error requeues with a warning", func(t *testing.T) {
		dogu := testDogu()
		charts := NewMockChartService(t)
		charts.EXPECT().HasMigrationJob(testCtx, dogu).Return(false, assert.AnError)
		recorder := NewMockEventRecorder(t)
		recorder.EXPECT().Event(dogu, corev1.EventTypeWarning, v3beta1.ReasonMigrationPending, fmt.Sprintf("failed to check for migration job: %v", assert.AnError))

		result := NewMigrationJobStep(charts, NewMockJobMonitor(t), recorder).Run(testCtx, dogu)

		assert.False(t, result.Continue)
		assert.Equal(t, 0*time.Second, result.RequeueAfter)
		assert.ErrorContains(t, result.Err, "failed to check for migration job")
		assert.Equal(t, v3beta1.ReasonMigrationPending, result.ReadyReason)
	})
}
