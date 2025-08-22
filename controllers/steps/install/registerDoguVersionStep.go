package install

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
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

func (rdvs *RegisterDoguVersionStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	dogu, _, err := rdvs.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch dogu descriptor: %w", err)
	}
	// TODO Can this function be used here? It should be used for new installations of a dogu.
	err = rdvs.doguRegistrator.RegisterNewDogu(ctx, doguResource, dogu)
	if err != nil {
		return 0, fmt.Errorf("failed to register dogu: %w", err)
	}
	return 0, err
}
