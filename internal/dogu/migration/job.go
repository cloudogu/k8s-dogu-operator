package migration

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const migrationJobLabel = "k8s.cloudogu.com/migration-job"

// JobsPhase summarizes the migration jobs of a dogu.
type JobsPhase int

const (
	// JobsPending means the jobs have not started yet or are still running.
	JobsPending JobsPhase = iota
	// JobsSucceeded means all jobs completed.
	JobsSucceeded
	// JobsFailed means a job failed or a timeout was exceeded.
	JobsFailed
)

// JobsStatus is the phase of all migration jobs of a dogu with a human-readable message.
type JobsStatus struct {
	Phase   JobsPhase
	Message string
}

// JobMonitor evaluates the migration jobs of a dogu against the start and completion timeout.
type JobMonitor struct {
	client            client.Reader
	startTimeout      time.Duration
	completionTimeout time.Duration
	now               func() time.Time
}

// NewJobMonitor creates a JobMonitor.
func NewJobMonitor(reader client.Reader, startTimeout, completionTimeout time.Duration) *JobMonitor {
	return &JobMonitor{client: reader, startTimeout: startTimeout, completionTimeout: completionTimeout, now: time.Now}
}

// Status returns the JobsStatus of the dogu's migration jobs.
func (m *JobMonitor) Status(ctx context.Context, dogu *v3beta1.Dogu) (JobsStatus, error) {
	jobs, err := m.jobs(ctx, dogu)
	if err != nil {
		return JobsStatus{}, fmt.Errorf("failed to get migration jobs of dogu %q: %w", dogu.Spec.Name, err)
	}

	if len(jobs) == 0 {
		return m.missingJobsStatus(ctx, dogu)
	}

	completed := 0
	for _, job := range jobs {
		if condition := trueCondition(job, batchv1.JobFailed); condition != nil {
			return JobsStatus{Phase: JobsFailed, Message: fmt.Sprintf("Migration job %q failed: %s", job.Name, condition.Message)}, nil
		}

		if trueCondition(job, batchv1.JobComplete) != nil {
			completed++
			continue
		}

		if job.Status.StartTime != nil && m.now().After(job.Status.StartTime.Add(m.completionTimeout)) {
			return JobsStatus{Phase: JobsFailed, Message: fmt.Sprintf("Migration job %q did not complete within %s", job.Name, m.completionTimeout)}, nil
		}
	}

	if completed < len(jobs) {
		return JobsStatus{Phase: JobsPending, Message: fmt.Sprintf("Waiting for %d of %d migration jobs to complete", len(jobs)-completed, len(jobs))}, nil
	}

	return JobsStatus{Phase: JobsSucceeded, Message: fmt.Sprintf("All %d migration jobs completed", len(jobs))}, nil
}

// trueCondition returns the job's condition of the given type, if it is true.
func trueCondition(job batchv1.Job, conditionType batchv1.JobConditionType) *batchv1.JobCondition {
	for _, condition := range job.Status.Conditions {
		if condition.Type == conditionType && condition.Status == corev1.ConditionTrue {
			return &condition
		}
	}

	return nil
}

// missingJobsStatus fails after the start timeout, so that a chart without labeled jobs does not hang silently.
// The timeout starts with the HelmRelease, because it renders the jobs.
func (m *JobMonitor) missingJobsStatus(ctx context.Context, dogu *v3beta1.Dogu) (JobsStatus, error) {
	release := &helmv2.HelmRelease{}
	if err := m.client.Get(ctx, client.ObjectKey{Namespace: dogu.Namespace, Name: dogu.Spec.Name}, release); err != nil {
		return JobsStatus{}, fmt.Errorf("failed to get HelmRelease of dogu %q: %w", dogu.Spec.Name, err)
	}

	if m.now().Before(release.CreationTimestamp.Add(m.startTimeout)) {
		return JobsStatus{Phase: JobsPending, Message: "Waiting for the migration jobs to start"}, nil
	}

	return JobsStatus{
		Phase:   JobsFailed,
		Message: fmt.Sprintf("No migration job of dogu %q found within %s", dogu.Spec.Name, m.startTimeout),
	}, nil
}

// jobs lists the dogu's migration jobs. Both labels are required, because all dogus share the namespace.
func (m *JobMonitor) jobs(ctx context.Context, dogu *v3beta1.Dogu) ([]batchv1.Job, error) {
	list := &batchv1.JobList{}
	err := m.client.List(ctx, list, client.InNamespace(dogu.Namespace), client.MatchingLabels{
		migrationJobLabel:     "true",
		v3beta1.DoguLabelName: dogu.Spec.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	return list.Items, nil
}
