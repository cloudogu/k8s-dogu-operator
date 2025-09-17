package upgrade

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

type DeploymentUpdaterStep struct {
	upserter                 resourceUpserter
	localDoguFetcher         localDoguFetcher
	client                   k8sClient
	deploymentInterface      deploymentInterface
	securityContextGenerator securityContextGenerator
}

func NewDeploymentUpdaterStep(
	upserter resource.ResourceUpserter,
	fetcher cesregistry.LocalDoguFetcher,
	deploymentInterface appsv1.DeploymentInterface,
) *DeploymentUpdaterStep {
	return &DeploymentUpdaterStep{
		upserter:            upserter,
		localDoguFetcher:    fetcher,
		deploymentInterface: deploymentInterface,
	}
}

func (dus *DeploymentUpdaterStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	deployment, err := dus.deploymentInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return steps.RequeueWithError(err)
	}

	dogu, err := dus.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if deployment != nil {
		return steps.Continue()
	}

	_, err = dus.upserter.UpsertDoguDeployment(ctx, doguResource, dogu, func(deployment *v1.Deployment) {
		util.SetPreviousDoguVersionInAnnotations(dogu.Version, deployment)
	})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
