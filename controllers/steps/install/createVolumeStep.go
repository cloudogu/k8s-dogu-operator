package install

import (
	"context"
	"fmt"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type CreateVolumeStep struct {
	localDoguFetcher localDoguFetcher
	resourceUpserter resourceUpserter
	pvcGetter        persistentVolumeClaimInterface
}

func NewCreateVolumeStep(fetcher cesregistry.LocalDoguFetcher, upserter resource.ResourceUpserter, pvcInterface v1.PersistentVolumeClaimInterface) *CreateVolumeStep {
	return &CreateVolumeStep{
		localDoguFetcher: fetcher,
		resourceUpserter: upserter,
		pvcGetter:        pvcInterface,
	}
}

func (vgs *CreateVolumeStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	pvc, err := vgs.pvcGetter.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return steps.RequeueWithError(err)
		}
	} else {
		err := vgs.resourceUpserter.SetControllerReferenceForPVC(ctx, pvc, doguResource)
		if err != nil {
			return steps.RequeueWithError(err)
		}

		return steps.Continue()
	}

	dogu, err := vgs.localDoguFetcher.FetchInstalled(ctx, cescommons.SimpleName(doguResource.Name))
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to get dogu descriptor for dogu %s: %w", doguResource.Name, err))
	}

	_, err = vgs.resourceUpserter.UpsertDoguPVCs(ctx, doguResource, dogu)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
