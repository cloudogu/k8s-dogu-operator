package deletion

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type ServiceAccountRemoverStep struct {
	serviceAccountRemover serviceAccountRemover
	resourceDoguFetcher   resourceDoguFetcher
}

func NewServiceAccountRemoverStep(
	serviceAccountRemover serviceaccount.ServiceAccountRemover,
	resourceDoguFetcher cesregistry.ResourceDoguFetcher,
) *ServiceAccountRemoverStep {
	return &ServiceAccountRemoverStep{
		serviceAccountRemover: serviceAccountRemover,
		resourceDoguFetcher:   resourceDoguFetcher,
	}
}

func (sas *ServiceAccountRemoverStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	doguDescriptor, _, err := sas.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	err = sas.serviceAccountRemover.RemoveAll(ctx, doguDescriptor)
	if err != nil {
		return steps.RequeueWithError(err)
	}
	return steps.Continue()
}
