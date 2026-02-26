package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/authregistration"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

// AuthRegistrationStep creates/updates AuthRegistration resources for a dogu.
type AuthRegistrationStep struct {
	localDoguFetcher        localDoguFetcher
	authRegistrationManager authregistration.Manager
}

func NewAuthRegistrationStep(authRegistrationManager authregistration.Manager, localDoguFetcher cesregistry.LocalDoguFetcher) *AuthRegistrationStep {
	return &AuthRegistrationStep{
		localDoguFetcher:        localDoguFetcher,
		authRegistrationManager: authRegistrationManager,
	}
}

func (ars *AuthRegistrationStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	doguDescriptor, err := ars.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(err)
	}

	// FIXME only ensure auth-registration if the dogu needs an CAS-Service Account and if the CAS-Dogu is NOT installed

	if err = ars.authRegistrationManager.EnsureAuthRegistration(ctx, doguDescriptor); err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
