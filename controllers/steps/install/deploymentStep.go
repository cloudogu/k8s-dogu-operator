package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	client2 "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const previousDoguVersionAnnotationKey = "k8s.cloudogu.com/dogu-previous-version"

type DeploymentStep struct {
	upserter                resource.ResourceUpserter
	localDoguFetcher        localDoguFetcher
	client                  client.Client
	ecosystemClient         client2.EcoSystemV2Interface
	doguHealthStatusUpdater health.DoguHealthStatusUpdater
}

func NewDeploymentStep(client client.Client, mgrSet *util.ManagerSet, doguHealthStatusUpdater health.DoguHealthStatusUpdater) *DeploymentStep {

	return &DeploymentStep{
		client:                  client,
		ecosystemClient:         mgrSet.EcosystemClient,
		upserter:                mgrSet.ResourceUpserter,
		localDoguFetcher:        mgrSet.LocalDoguFetcher,
		doguHealthStatusUpdater: doguHealthStatusUpdater,
	}
}

func (ds *DeploymentStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	deployment, err := doguResource.GetDeployment(ctx, ds.client)
	if err != nil && !errors.IsNotFound(err) {
		return steps.RequeueWithError(err)
	}

	dogu, err := ds.getLocalDogu(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if deployment != nil {
		return steps.Continue()
	}

	_, err = ds.upserter.UpsertDoguDeployment(ctx, doguResource, dogu, func(deployment *v1.Deployment) {
		setPreviousDoguVersionInAnnotations(dogu.Version, deployment)
	})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}

func (ds *DeploymentStep) getLocalDogu(ctx context.Context, doguResource *v2.Dogu) (*core.Dogu, error) {
	dogu, err := ds.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return nil, fmt.Errorf("dogu not found in local registry: %w", err)
	}

	return dogu, nil
}

func setPreviousDoguVersionInAnnotations(previousDoguVersion string, deployment *v1.Deployment) {
	if deployment.Annotations == nil {
		deployment.Annotations = map[string]string{}
	}
	deployment.Annotations[previousDoguVersionAnnotationKey] = previousDoguVersion
}
