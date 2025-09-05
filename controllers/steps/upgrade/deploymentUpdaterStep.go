package upgrade

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentUpdaterStep struct {
	deploymentGenerator deploymentGenerator
	localDoguFetcher    localDoguFetcher
	client              client.Client
	deploymentInterface deploymentInterface
}

func NewDeploymentUpdaterStep(client client.Client, mgrSet *util.ManagerSet, namespace string) *DeploymentUpdaterStep {
	return &DeploymentUpdaterStep{
		deploymentGenerator: mgrSet.DoguResourceGenerator,
		localDoguFetcher:    mgrSet.LocalDoguFetcher,
		client:              client,
		deploymentInterface: mgrSet.ClientSet.AppsV1().Deployments(namespace),
	}
}

func (dus *DeploymentUpdaterStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, err := dus.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(err)
	}

	newDeployment, err := dus.deploymentGenerator.CreateDoguDeployment(ctx, doguResource, dogu)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	_, err = dus.deploymentInterface.Update(ctx, newDeployment, v1.UpdateOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
