package upgrade

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

type DeleteExecPodStep struct {
	execPodFactory   execPodFactory
	localDoguFetcher localDoguFetcher
}

func NewDeleteExecPodStep(mgrSet *util.ManagerSet) *DeleteExecPodStep {
	return &DeleteExecPodStep{
		localDoguFetcher: mgrSet.LocalDoguFetcher,
		execPodFactory:   mgrSet.ExecPodFactory,
	}
}

func (deps *DeleteExecPodStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, err := deps.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("dogu not found in local registry: %w", err))
	}

	err = deps.execPodFactory.Delete(ctx, doguResource, dogu)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to delete exec pod for dogu %q: %w", dogu.GetSimpleName(), err))
	}

	return steps.Continue()
}
