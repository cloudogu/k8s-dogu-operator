package migration

import (
	"context"
	"testing"
	"time"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testStartTimeout      = 5 * time.Minute
	testCompletionTimeout = time.Hour
)

var testNow = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

func jobStatus(t *testing.T, objects ...client.Object) (JobsStatus, error) {
	t.Helper()
	scheme := testScheme(t)
	require.NoError(t, helmv2.AddToScheme(scheme))
	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()

	monitor := NewJobMonitor(cl, testStartTimeout, testCompletionTimeout)
	monitor.now = func() time.Time { return testNow }

	return monitor.Status(context.Background(), v3Dogu("nexus"))
}

func helmRelease(createdAgo time.Duration) *helmv2.HelmRelease {
	return &helmv2.HelmRelease{
		Name:              "nexus",
		Namespace:         testNamespace,
		CreationTimestamp: metav1.NewTime(testNow.Add(-createdAgo))}
}

func migrationJob(name, doguName string, startedAgo time.Duration, conditions ...batchv1.JobCondition) *batchv1.Job {
	return &batchv1.Job{
		Name:      name,
		Namespace: testNamespace,
		Labels: map[string]string{
			"k8s.cloudogu.com/migration-job": "true",
			"k8s.cloudogu.com/dogu.name":     doguName,
		},
		Status: batchv1.JobStatus{
			StartTime:  new(metav1.NewTime(testNow.Add(-startedAgo))),
			Conditions: conditions,
		},
	}
}

func jobCondition(conditionType batchv1.JobConditionType, message string) batchv1.JobCondition {
	return batchv1.JobCondition{Type: conditionType, Status: corev1.ConditionTrue, Message: message}
}

func TestJobMonitor_Status(t *testing.T) {
	t.Run("missing jobs are pending within the start timeout", func(t *testing.T) {
		status, err := jobStatus(t, helmRelease(time.Minute))

		require.NoError(t, err)
		assert.Equal(t, JobsPending, status.Phase)
		assert.Equal(t, "Waiting for the migration jobs to start", status.Message)
	})

	t.Run("missing jobs fail after the start timeout with the HelmRelease state", func(t *testing.T) {
		release := helmRelease(10 * time.Minute)

		status, err := jobStatus(t, release)

		require.NoError(t, err)
		assert.Equal(t, JobsFailed, status.Phase)
		assert.Equal(t, `No migration job of dogu "nexus" found within 5m0s`, status.Message)
	})

	t.Run("jobs of other dogus are ignored", func(t *testing.T) {
		status, err := jobStatus(t, helmRelease(10*time.Minute), migrationJob("cas-migration", "cas", time.Minute))

		require.NoError(t, err)
		assert.Equal(t, JobsFailed, status.Phase)
	})

	t.Run("running jobs are pending", func(t *testing.T) {
		status, err := jobStatus(t,
			migrationJob("nexus-db", "nexus", time.Minute, jobCondition(batchv1.JobComplete, "")),
			migrationJob("nexus-blobs", "nexus", time.Minute),
		)

		require.NoError(t, err)
		assert.Equal(t, JobsPending, status.Phase)
		assert.Equal(t, "Waiting for 1 of 2 migration jobs to complete", status.Message)
	})

	t.Run("all completed jobs succeed", func(t *testing.T) {
		status, err := jobStatus(t,
			migrationJob("nexus-db", "nexus", time.Minute, jobCondition(batchv1.JobComplete, "")),
			migrationJob("nexus-blobs", "nexus", time.Minute, jobCondition(batchv1.JobComplete, "")),
		)

		require.NoError(t, err)
		assert.Equal(t, JobsSucceeded, status.Phase)
		assert.Equal(t, "All 2 migration jobs completed", status.Message)
	})

	t.Run("a failed job fails the migration", func(t *testing.T) {
		status, err := jobStatus(t,
			migrationJob("nexus-db", "nexus", time.Minute, jobCondition(batchv1.JobFailed, "BackoffLimitExceeded")),
			migrationJob("nexus-blobs", "nexus", time.Minute),
		)

		require.NoError(t, err)
		assert.Equal(t, JobsFailed, status.Phase)
		assert.Equal(t, `Migration job "nexus-db" failed: BackoffLimitExceeded`, status.Message)
	})

	t.Run("a job running longer than the completion timeout fails the migration", func(t *testing.T) {
		status, err := jobStatus(t, migrationJob("nexus-db", "nexus", 2*time.Hour))

		require.NoError(t, err)
		assert.Equal(t, JobsFailed, status.Phase)
		assert.Equal(t, `Migration job "nexus-db" did not complete within 1h0m0s`, status.Message)
	})

	t.Run("a completed job exceeding the completion timeout succeeds", func(t *testing.T) {
		status, err := jobStatus(t, migrationJob("nexus-db", "nexus", 2*time.Hour, jobCondition(batchv1.JobComplete, "")))

		require.NoError(t, err)
		assert.Equal(t, JobsSucceeded, status.Phase)
	})
}
