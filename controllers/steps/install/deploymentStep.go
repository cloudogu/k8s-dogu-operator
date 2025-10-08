package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentStep struct {
	upserter         resourceUpserter
	localDoguFetcher localDoguFetcher
	client           k8sClient
}

func NewDeploymentStep(client client.Client, upserter resource.ResourceUpserter, fetcher cesregistry.LocalDoguFetcher) *DeploymentStep {
	return &DeploymentStep{
		client:           client,
		upserter:         upserter,
		localDoguFetcher: fetcher,
	}
}

func (ds *DeploymentStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	_, err := doguResource.GetDeployment(ctx, ds.client)
	if err != nil && !errors.IsNotFound(err) {
		return steps.RequeueWithError(err)
	} else if err == nil {
		return steps.Continue()
	}

	dogu, err := ds.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(err)
	}

	_, err = ds.upserter.UpsertDoguDeployment(ctx, doguResource, dogu, nil)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
