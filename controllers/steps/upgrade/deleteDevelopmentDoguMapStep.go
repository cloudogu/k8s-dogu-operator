package upgrade

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeleteDevelopmentDoguMapStep struct {
	resourceDoguFetcher resourceDoguFetcher
	client              client.Client
}

func NewDeleteDevelopmentDoguMapStep(client client.Client, mgrSet *util.ManagerSet) *DeleteDevelopmentDoguMapStep {
	return &DeleteDevelopmentDoguMapStep{
		resourceDoguFetcher: mgrSet.ResourceDoguFetcher,
		client:              client,
	}
}

func (ddms *DeleteDevelopmentDoguMapStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	_, developmentDoguMap, err := ddms.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("dogu upgrade failed: %w", err))
	}
	if developmentDoguMap != nil {
		err = developmentDoguMap.DeleteFromCluster(ctx, ddms.client)
		if err != nil {
			// an error during deleting the developmentDoguMap is not critical, so we change the dogu state as installed earlier
			return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("dogu upgrade %s:%s failed: %w", doguResource.Name, doguResource.Spec.Version, err))
		}
	}
	return steps.StepResult{}
}
