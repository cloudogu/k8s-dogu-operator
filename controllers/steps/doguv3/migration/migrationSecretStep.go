package migration

import (
	"bytes"
	"context"
	"fmt"
	"maps"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	stepsv3 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	dogumigration "github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/migration"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const migrationSecretStepFieldManager = "k8s-dogu-operator-migration-secret-step"

// MigrationSecretStep copies the service account credentials of the v2 sensitive config into the migration secret,
// so they outlive the v2 resources.
type MigrationSecretStep struct {
	k8sClient     K8sClient
	eventRecorder EventRecorder
}

func NewMigrationSecretStep(k8sClient K8sClient, recorder EventRecorder) *MigrationSecretStep {
	return &MigrationSecretStep{k8sClient: k8sClient, eventRecorder: recorder}
}

func (s *MigrationSecretStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) stepsv3.StepResult {
	sensitiveConfig, err := dogumigration.SensitiveConfig(ctx, s.k8sClient, doguResource)
	if err != nil {
		return s.fail(doguResource, err)
	}

	// Without sensitive config the migration secret is kept, because it may hold the last copy of the credentials.
	if sensitiveConfig == nil {
		return stepsv3.Continue()
	}

	credentials, err := dogumigration.ServiceAccountCredentials(sensitiveConfig)
	if err != nil {
		// The sensitive config has to be corrected manually. Its change triggers a new reconciliation, because the dogu owns it.
		return s.abort(doguResource, err)
	}

	existing, err := dogumigration.MigrationSecret(ctx, s.k8sClient, doguResource)
	if err != nil {
		return s.fail(doguResource, err)
	}

	secret := dogumigration.NewMigrationSecret(doguResource, credentials)
	if aErr := s.k8sClient.Apply(ctx, secret, client.FieldOwner(migrationSecretStepFieldManager), client.ForceOwnership); aErr != nil {
		return s.fail(doguResource, fmt.Errorf("failed to apply migration secret %q: %w", *secret.Name, aErr))
	}

	// The use case requeues periodically, so only changes are recorded to avoid flooding the events.
	if existing != nil && maps.EqualFunc(existing.Data, credentials, bytes.Equal) {
		return stepsv3.Continue()
	}

	action := "created"
	if existing != nil {
		action = "updated"
	}

	msg := fmt.Sprintf("Migration secret %q %s", *secret.Name, action)
	s.eventRecorder.Event(doguResource, corev1.EventTypeNormal, v3beta1.ReasonMigrationPending, msg)

	return stepsv3.Continue()
}

// abort records a warning and stops the reconciliation, because the error cannot be solved by a retry.
func (s *MigrationSecretStep) abort(doguResource *v3beta1.Dogu, err error) stepsv3.StepResult {
	s.eventRecorder.Event(doguResource, corev1.EventTypeWarning, v3beta1.ReasonMigrationFailed, err.Error())

	return stepsv3.Abort(v3beta1.ReasonMigrationFailed, err.Error())
}

// fail records a warning, because the use case emits no event when it requeues.
func (s *MigrationSecretStep) fail(doguResource *v3beta1.Dogu, err error) stepsv3.StepResult {
	s.eventRecorder.Event(doguResource, corev1.EventTypeWarning, v3beta1.ReasonMigrationPending, err.Error())

	return stepsv3.RequeueWithError(err, v3beta1.ReasonMigrationPending)
}
