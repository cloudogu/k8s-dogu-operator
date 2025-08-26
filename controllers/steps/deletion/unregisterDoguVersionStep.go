package deletion

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

type UnregisterDoguVersionStep struct {
	resourceDoguFetcher resourceDoguFetcher
	doguRegistrator     doguRegistrator
	localDoguFetcher    localDoguFetcher
}

func NewUnregisterDoguVersionStep(mgrSet *util.ManagerSet) *UnregisterDoguVersionStep {
	return &UnregisterDoguVersionStep{
		resourceDoguFetcher: mgrSet.ResourceDoguFetcher,
		localDoguFetcher:    mgrSet.LocalDoguFetcher,
		doguRegistrator:     mgrSet.DoguRegistrator,
	}
}

func (udvs *UnregisterDoguVersionStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	err := udvs.doguRegistrator.UnregisterDogu(ctx, doguResource.Name)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to register dogu: %w", err))
	}
	return steps.Continue()
}
