package upgrade

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

// The DeleteExecPodStep deletes the exec pod if currently exist.
type DeleteExecPodStep struct {
	execPodFactory   execPodFactory
	localDoguFetcher localDoguFetcher
}

func NewDeleteExecPodStep(fetcher cesregistry.LocalDoguFetcher, factory exec.ExecPodFactory) *DeleteExecPodStep {
	return &DeleteExecPodStep{
		localDoguFetcher: fetcher,
		execPodFactory:   factory,
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
