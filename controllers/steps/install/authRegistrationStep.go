package install

import (
	"context"
	"fmt"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/authregistration"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const casDoguName = "cas"

// AuthRegistrationStep creates/updates AuthRegistration resources for a dogu.
type AuthRegistrationStep struct {
	doguFetcher             localDoguFetcher
	authRegistrationManager authRegistrationManager
}

func NewAuthRegistrationStep(authRegistrationManager authregistration.Manager, localDoguFetcher cesregistry.LocalDoguFetcher) *AuthRegistrationStep {
	return &AuthRegistrationStep{
		doguFetcher:             localDoguFetcher,
		authRegistrationManager: authRegistrationManager,
	}
}

func (ars *AuthRegistrationStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	logger := log.FromContext(ctx).WithName("authRegistrationStep")

	casEnabled, err := ars.doguFetcher.Enabled(ctx, cescommons.SimpleName(casDoguName))
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to check if CAS is enabled: %w", err))
	}

	if !casEnabled {
		logger.Info("CAS is not enabled, skipping auth registration")
		return steps.Continue()
	}

	doguDescriptor, err := ars.doguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if err = ars.authRegistrationManager.EnsureAuthRegistration(ctx, doguDescriptor); err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
