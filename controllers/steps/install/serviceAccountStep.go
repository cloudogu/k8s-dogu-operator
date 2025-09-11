package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

type ServiceAccountStep struct {
	serviceAccountCreator serviceAccountCreator
	localDoguFetcher      localDoguFetcher
}

func NewServiceAccountStep(mgrSet *util.ManagerSet) *ServiceAccountStep {
	return &ServiceAccountStep{
		serviceAccountCreator: mgrSet.ServiceAccountCreator,
		localDoguFetcher:      mgrSet.LocalDoguFetcher,
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
