package install

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

const requeueAfterDeployment = 5 * time.Second

type DeploymentStep struct {
	upserter         resource.ResourceUpserter
	localDoguFetcher localDoguFetcher
}

func NewDeploymentStep(mgrSet *util.ManagerSet, upserter resource.ResourceUpserter) *DeploymentStep {
	return &DeploymentStep{
		upserter:         upserter,
		localDoguFetcher: mgrSet.LocalDoguFetcher,
	}
}

func (ds *DeploymentStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, err := ds.getLocalDogu(ctx, doguResource)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}
	// TODO Generate Resource-Limits
	// TODO Already done in resource generator: https://github.com/cloudogu/k8s-dogu-operator/blob/d289f34b58294461aa7249ceb1402f484ffd183c/controllers/resource/resource_generator.go#L198
	_, err = ds.upserter.UpsertDoguDeployment(ctx, doguResource, dogu, nil)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}
	return steps.StepResult{}
}

func (ds *DeploymentStep) getLocalDogu(ctx context.Context, doguResource *v2.Dogu) (*core.Dogu, error) {
	dogu, err := ds.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return nil, fmt.Errorf("dogu not found in local registry: %w", err)
	}

	return dogu, nil
}
