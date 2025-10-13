package deletion

import (
	"context"

	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

// The ServiceAccountRemoverStep removes all service accounts that are needed for the dogu.
type ServiceAccountRemoverStep struct {
	serviceAccountRemover serviceAccountRemover
	doguFetcher           localDoguFetcher
}

func NewServiceAccountRemoverStep(
	serviceAccountRemover serviceaccount.ServiceAccountRemover,
	doguFetcher cesregistry.LocalDoguFetcher,
) *ServiceAccountRemoverStep {
	return &ServiceAccountRemoverStep{
		serviceAccountRemover: serviceAccountRemover,
		doguFetcher:           doguFetcher,
	}
}

func (sas *ServiceAccountRemoverStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	doguDescriptor, err := sas.doguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if cloudoguerrors.IsNotFoundError(err) {
		return steps.Continue()
	} else if err != nil {
		return steps.RequeueWithError(err)
	}

	err = sas.serviceAccountRemover.RemoveAll(ctx, doguDescriptor)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
