package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type ServiceAccountStep struct {
	serviceAccountCreator serviceAccountCreator
	localDoguFetcher      localDoguFetcher
}

func (sas *ServiceAccountStep) Priority() int {
	return 4600
}

func NewServiceAccountStep(creator serviceAccountCreator, fetcher cesregistry.LocalDoguFetcher) *ServiceAccountStep {
	return &ServiceAccountStep{
		serviceAccountCreator: creator,
		localDoguFetcher:      fetcher,
	}
}

func (sas *ServiceAccountStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	doguDescriptor, err := sas.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(err)
	}

	// Existing service accounts will be skipped.
	err = sas.serviceAccountCreator.CreateAll(ctx, doguDescriptor)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
