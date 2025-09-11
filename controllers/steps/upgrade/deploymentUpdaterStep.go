package upgrade

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentUpdaterStep struct {
	upserter                 resource.ResourceUpserter
	localDoguFetcher         localDoguFetcher
	client                   client.Client
	deploymentInterface      deploymentInterface
	securityContextGenerator *resource.SecurityContextGenerator
}

func NewDeploymentUpdaterStep(client client.Client, mgrSet *util.ManagerSet, namespace string) *DeploymentUpdaterStep {
	return &DeploymentUpdaterStep{
		upserter:                 mgrSet.ResourceUpserter,
		localDoguFetcher:         mgrSet.LocalDoguFetcher,
		client:                   client,
		deploymentInterface:      mgrSet.ClientSet.AppsV1().Deployments(namespace),
		securityContextGenerator: resource.NewSecurityContextGenerator(),
	}
}

func (dus *DeploymentUpdaterStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	deployment, err := doguResource.GetDeployment(ctx, dus.client)
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
