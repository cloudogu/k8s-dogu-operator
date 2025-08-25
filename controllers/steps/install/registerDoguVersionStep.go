package install

import (
	"context"
	"fmt"

	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

type RegisterDoguVersionStep struct {
	resourceDoguFetcher resourceDoguFetcher
	doguRegistrator     doguRegistrator
	localDoguFetcher    localDoguFetcher
}

func NewRegisterDoguVersionStep(mgrSet *util.ManagerSet) *RegisterDoguVersionStep {
	return &RegisterDoguVersionStep{
		resourceDoguFetcher: mgrSet.ResourceDoguFetcher,
		localDoguFetcher:    mgrSet.LocalDoguFetcher,
		doguRegistrator:     mgrSet.DoguRegistrator,
	}
}

func (rdvs *RegisterDoguVersionStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	remoteDogu, _, err := rdvs.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to fetch dogu descriptor: %w", err))
	}
	installed, err := rdvs.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if installed != nil {
		return steps.StepResult{}
	}
	if err != nil && cloudoguerrors.IsNotFoundError(err) {
		err = rdvs.doguRegistrator.RegisterNewDogu(ctx, doguResource, remoteDogu)
		if err != nil {
			return steps.NewStepResultContinueIsTrueAndRequeueIsZero(fmt.Errorf("failed to register dogu: %w", err))
		}
		return steps.StepResult{}
	}
	return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
}
