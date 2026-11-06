package migration

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	dogumigration "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	corev1 "k8s.io/api/core/v1"
)

// PrepareMigrationJobsStep lets the migration jobs of the dogu's chart reach the shared v2 dogus.
// It stops the migration if the chart ships no migration job for the migration data.
type PrepareMigrationJobsStep struct {
	k8sClient              K8sClient
	charts                 ChartService
	eventRecorder          EventRecorder
	networkPoliciesEnabled bool
}

func NewPrepareMigrationJobsStep(k8sClient K8sClient, charts ChartService, recorder EventRecorder, networkPoliciesEnabled bool) *PrepareMigrationJobsStep {
	return &PrepareMigrationJobsStep{k8sClient: k8sClient, charts: charts, eventRecorder: recorder, networkPoliciesEnabled: networkPoliciesEnabled}
}

func (s *PrepareMigrationJobsStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	hasMigrationJob, err := s.charts.HasMigrationJob(ctx, doguResource)
	if err != nil {
		return s.fail(doguResource, fmt.Errorf("failed to check if chart %q has migration job: %w", doguResource.Spec.Name, err))
	}

	if !hasMigrationJob {
		found, lErr := dogumigration.HasMigrationVolumeClaim(ctx, s.k8sClient, doguResource)
		if lErr != nil {
			return s.fail(doguResource, fmt.Errorf("failed to check if migration volume claim exists for chart %q: %w", doguResource.Spec.Name, lErr))
		}

		if found {
			return stepsv3.Abort(v3beta1.ReasonMigrationFailed, fmt.Sprintf("the chart ships no migration job, so the data of the migration volume claim %q would be lost", doguResource.Spec.Name+"-migration"))
		}

		return stepsv3.Continue()
	}

	// Without network policies there is no deny-all policy to pass.
	if !s.networkPoliciesEnabled {
		return stepsv3.Continue()
	}

	secret, err := dogumigration.MigrationSecret(ctx, s.k8sClient, doguResource)
	if err != nil {
		return s.fail(doguResource, fmt.Errorf("failed to get migration secret for chart %q: %w", doguResource.Spec.Name, err))
	}

	if secret == nil {
		return stepsv3.Continue()
	}

	// The step runs on every requeue, so only created policies are recorded.
	created, err := dogumigration.ApplyNetworkPolicies(ctx, s.k8sClient, doguResource, secret)
	for _, name := range created {
		s.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, fmt.Sprintf("Migration network policy %q created", name))
	}

	if err != nil {
		return s.fail(doguResource, fmt.Errorf("failed to apply migration network policies for chart %q: %w", doguResource.Spec.Name, err))
	}

	return stepsv3.Continue()
}

// fail records a warning, because the use case emits no event when it requeues.
func (s *PrepareMigrationJobsStep) fail(doguResource *v3beta1.Dogu, err error) stepsv3.StepResult {
	s.eventRecorder.Event(doguResource, corev1.EventTypeWarning, v3beta1.ReasonMigrationPending, err.Error())

	return stepsv3.RequeueWithError(err, v3beta1.ReasonMigrationPending)
}
