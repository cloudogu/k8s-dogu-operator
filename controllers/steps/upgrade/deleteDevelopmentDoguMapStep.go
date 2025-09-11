package upgrade

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

type DeleteDevelopmentDoguMapStep struct {
	resourceDoguFetcher resourceDoguFetcher
	client              k8sClient
}

func NewDeleteDevelopmentDoguMapStep(client k8sClient, mgrSet *util.ManagerSet) *DeleteDevelopmentDoguMapStep {
	return &DeleteDevelopmentDoguMapStep{
		resourceDoguFetcher: mgrSet.ResourceDoguFetcher,
		client:              client,
	}
}

func (ddms *DeleteDevelopmentDoguMapStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	_, developmentDoguMap, err := ddms.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("dogu upgrade failed: %w", err))
	}

	if developmentDoguMap != nil {
		err = developmentDoguMap.DeleteFromCluster(ctx, ddms.client)
		if err != nil {
			return steps.RequeueWithError(fmt.Errorf("dogu upgrade %s:%s failed: %w", doguResource.Name, doguResource.Spec.Version, err))
		}
	}

	return steps.Continue()
}
