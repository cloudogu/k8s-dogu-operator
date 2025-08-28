package install

import (
	"context"
	"fmt"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type VolumeGeneratorStep struct {
	localDoguFetcher localDoguFetcher
	resourceUpserter resource.ResourceUpserter
	pvcGetter        v1.PersistentVolumeClaimInterface
}

func NewVolumeGeneratorStep(mgrSet *util.ManagerSet, namespace string) *VolumeGeneratorStep {

	return &VolumeGeneratorStep{
		localDoguFetcher: mgrSet.LocalDoguFetcher,
		resourceUpserter: mgrSet.ResourceUpserter,
		pvcGetter:        mgrSet.ClientSet.CoreV1().PersistentVolumeClaims(namespace),
	}
}

func (vgs *VolumeGeneratorStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	_, err := vgs.pvcGetter.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return steps.RequeueWithError(err)
		}
	} else {
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
