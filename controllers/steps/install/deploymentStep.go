package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

type DeploymentStep struct {
	upserter         resourceUpserter
	localDoguFetcher localDoguFetcher
	client           k8sClient
}

func NewDeploymentStep(client k8sClient, mgrSet *util.ManagerSet) *DeploymentStep {
	return &DeploymentStep{
		client:           client,
		upserter:         mgrSet.ResourceUpserter,
		localDoguFetcher: mgrSet.LocalDoguFetcher,
	}
}

func (ds *DeploymentStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	deployment, err := doguResource.GetDeployment(ctx, ds.client)
	if err != nil && !errors.IsNotFound(err) {
		return steps.RequeueWithError(err)
	}

	dogu, err := ds.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if deployment != nil {
		return steps.Continue()
	}

	_, err = ds.upserter.UpsertDoguDeployment(ctx, doguResource, dogu, func(deployment *v1.Deployment) {
		util.SetPreviousDoguVersionInAnnotations(dogu.Version, deployment)
	})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
