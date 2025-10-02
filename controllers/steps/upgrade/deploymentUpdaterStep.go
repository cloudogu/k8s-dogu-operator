package upgrade

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type DeploymentUpdaterStep struct {
	upserter                 resourceUpserter
	localDoguFetcher         localDoguFetcher
	client                   k8sClient
	securityContextGenerator securityContextGenerator
}

func NewDeploymentUpdaterStep(
	upserter resourceUpserter,
	fetcher localDoguFetcher,
) *DeploymentUpdaterStep {
	return &DeploymentUpdaterStep{
		upserter:         upserter,
		localDoguFetcher: fetcher,
	}
}

func (dus *DeploymentUpdaterStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	// Only update the deployment if this is not a dogu upgrade.
	// The deployment was already updated from prior steps in an upgrade path.
	if doguResource.Status.InstalledVersion != doguResource.Spec.Version {
		return steps.Continue()
	}

	dogu, err := dus.localDoguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	_, err = dus.upserter.UpsertDoguDeployment(ctx, doguResource, dogu, nil)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
