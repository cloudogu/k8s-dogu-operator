package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/authregistration"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AuthRegistrationStep creates/updates AuthRegistration resources for a dogu.
type AuthRegistrationStep struct {
	authRegistrationManager authRegistrationManager
	authRegistrationEnabled bool
}

func NewAuthRegistrationStep(authRegistrationManager authregistration.Manager, operatorConfig *config.OperatorConfig) *AuthRegistrationStep {
	return &AuthRegistrationStep{
		authRegistrationManager: authRegistrationManager,
		authRegistrationEnabled: operatorConfig.AuthRegistrationEnabled,
	}
}

func (ars *AuthRegistrationStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	logger := log.FromContext(ctx).WithName("authRegistrationStep")

	if !ars.authRegistrationEnabled {
		logger.Info("Auth registration is disabled, skipping auth registration")
		return steps.Continue()
	}

	if err := ars.authRegistrationManager.EnsureAuthRegistration(ctx, doguResource); err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
