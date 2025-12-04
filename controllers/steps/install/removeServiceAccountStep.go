package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The RemoveServiceAccountStep removes service accounts to components that the dogu might have
// these service accounts need to be recreated if the dogu was restored from a backup
type RemoveServiceAccountStep struct {
	serviceAccountRemover serviceAccountRemover
	localDoguFetcher      localDoguFetcher
	doguInterface         doguInterface
}

func NewRemoveServiceAccountStep(creator serviceaccount.ServiceAccountRemover, fetcher cesregistry.LocalDoguFetcher, doguInterface doguClient.DoguInterface) *RemoveServiceAccountStep {
	return &RemoveServiceAccountStep{
		serviceAccountRemover: creator,
		localDoguFetcher:      fetcher,
		doguInterface:         doguInterface,
	}
}

func (sas *RemoveServiceAccountStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	const restoredAnnotation = "wasRestored"
	if len(doguResource.Annotations) == 0 {
		return steps.Continue()
	}
	if _, wasRestored := doguResource.Annotations[restoredAnnotation]; !wasRestored {
		return steps.Continue()
	}

	doguDescriptor, err := sas.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if doguResource.Annotations != nil {
		if _, wasRestored := doguResource.Annotations[restoredAnnotation]; wasRestored {
			err = sas.serviceAccountRemover.RemoveAllFromComponents(ctx, doguDescriptor)
			// remove annotation afterward to prevent endless requeue
			delete(doguResource.Annotations, restoredAnnotation)
			_, err = sas.doguInterface.Update(ctx, doguResource, metav1.UpdateOptions{})
		}
	}

	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
