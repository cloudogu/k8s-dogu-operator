package upgrade

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

type RegisterDoguVersionStep struct {
	resourceDoguFetcher resourceDoguFetcher
	doguRegistrator     doguRegistrator
}

func NewRegisterDoguVersionStep(mgrSet *util.ManagerSet) *RegisterDoguVersionStep {
	return &RegisterDoguVersionStep{
		resourceDoguFetcher: mgrSet.ResourceDoguFetcher,
		doguRegistrator:     mgrSet.DoguRegistrator,
	}
}

func (rdvs *RegisterDoguVersionStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, _, err := rdvs.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to fetch dogu descriptor: %w", err))
	}

	err = rdvs.doguRegistrator.RegisterDoguVersion(ctx, dogu)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to register dogu: %w", err))
	}
	return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
}
