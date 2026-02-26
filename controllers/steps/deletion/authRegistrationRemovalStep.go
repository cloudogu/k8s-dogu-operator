package deletion

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/authregistration"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

// AuthRegistrationRemoverStep removes AuthRegistration resources for a dogu.
type AuthRegistrationRemoverStep struct {
	authRegistrationManager authregistration.Manager
}

func NewAuthRegistrationRemoverStep(authRegistrationManager authregistration.Manager) *AuthRegistrationRemoverStep {
	return &AuthRegistrationRemoverStep{
		authRegistrationManager: authRegistrationManager,
	}
}

func (ars *AuthRegistrationRemoverStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	// FIXME only remove if the CAS-Dogu is NOT installed
	if err := ars.authRegistrationManager.RemoveAuthRegistration(ctx, doguResource.GetSimpleDoguName()); err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
