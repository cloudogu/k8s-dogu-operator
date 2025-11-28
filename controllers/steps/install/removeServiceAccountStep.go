package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

// The RemoveServiceAccountStep removes service accounts to components that the dogu might have
type RemoveServiceAccountStep struct {
	serviceAccountRemover serviceAccountRemover
	localDoguFetcher      localDoguFetcher
}

func NewRemoveServiceAccountStep(creator serviceaccount.ServiceAccountRemover, fetcher cesregistry.LocalDoguFetcher) *RemoveServiceAccountStep {
	return &RemoveServiceAccountStep{
		serviceAccountRemover: creator,
		localDoguFetcher:      fetcher,
	}
}

func (sas *RemoveServiceAccountStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	const restoredAnnotation = "k8s.cloudogu.com/was-restored"
	doguDescriptor, err := sas.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if doguResource.Annotations != nil {
		if _, wasRestored := doguResource.Annotations[restoredAnnotation]; wasRestored {
			err = sas.serviceAccountRemover.RemoveAllFromComponents(ctx, doguDescriptor)
		}
	}

	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
