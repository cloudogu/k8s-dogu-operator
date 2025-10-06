package upgrade

import (
	"context"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v3 "k8s.io/client-go/kubernetes/typed/apps/v1"
)

type DeploymentUpdaterStep struct {
	localDoguFetcher    localDoguFetcher
	deploymentInterface deploymentInterface
	resourceGenerator   resourceGenerator
}

func NewDeploymentUpdaterStep(
	fetcher cesregistry.LocalDoguFetcher,
	deploymentInterface v3.DeploymentInterface,
	resourceGenerator resource.DoguResourceGenerator,
) *DeploymentUpdaterStep {
	return &DeploymentUpdaterStep{
		localDoguFetcher:    fetcher,
		deploymentInterface: deploymentInterface,
		resourceGenerator:   resourceGenerator,
	}
}

func (dus *DeploymentUpdaterStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	// Only update the deployment if this is not a dogu upgrade.
	// The deployment was already updated from prior steps in an upgrade path.
	if doguResource.Status.InstalledVersion != doguResource.Spec.Version {
		return steps.Continue()
	}

	actual, err := dus.deploymentInterface.Get(ctx, doguResource.Name, v1.GetOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	dogu, err := dus.localDoguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	target, err := dus.resourceGenerator.UpdateDoguDeployment(ctx, actual, doguResource, dogu)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	_, err = dus.deploymentInterface.Update(ctx, target, v1.UpdateOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
