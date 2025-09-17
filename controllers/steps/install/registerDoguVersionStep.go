package install

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type RegisterDoguVersionStep struct {
	resourceDoguFetcher resourceDoguFetcher
	doguRegistrator     doguRegistrator
	localDoguFetcher    localDoguFetcher
}

func NewRegisterDoguVersionStep(resourceDoguFetcher cesregistry.ResourceDoguFetcher, localDoguFetcher cesregistry.LocalDoguFetcher, registrator cesregistry.DoguRegistrator) *RegisterDoguVersionStep {
	return &RegisterDoguVersionStep{
		resourceDoguFetcher: resourceDoguFetcher,
		localDoguFetcher:    localDoguFetcher,
		doguRegistrator:     registrator,
	}
}

func (rdvs *RegisterDoguVersionStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	enabled, err := rdvs.localDoguFetcher.Enabled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to check if dogu is enabled: %w", err))
	}

	if enabled {
		return steps.Continue()
	}

	remoteDogu, _, err := rdvs.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor: %w", err))
	}

	err = rdvs.doguRegistrator.RegisterNewDogu(ctx, doguResource, remoteDogu)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to register dogu: %w", err))
	}

	return steps.Continue()
}
