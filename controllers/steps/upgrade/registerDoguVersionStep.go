package upgrade

import (
	"context"
	"fmt"

	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type RegisterDoguVersionStep struct {
	localDoguFetcher LocalDoguFetcher
	doguRegistrator  doguRegistrator
}

func NewRegisterDoguVersionStep(fetcher cesregistry.LocalDoguFetcher, registrator cesregistry.DoguRegistrator) *RegisterDoguVersionStep {
	return &RegisterDoguVersionStep{
		localDoguFetcher: fetcher,
		doguRegistrator:  registrator,
	}
}

func (rdvs *RegisterDoguVersionStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, err := rdvs.localDoguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor: %w", err))
	}

	err = rdvs.doguRegistrator.RegisterDoguVersion(ctx, dogu)
	if err != nil {
		if cloudoguerrors.IsAlreadyExistsError(err) {
			return steps.Continue()
		}
		return steps.RequeueWithError(fmt.Errorf("failed to register dogu: %w", err))
	}

	return steps.Continue()
}
