package controllers

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

type RegisterDoguVersionStep struct {
	resourceDoguFetcher resourceDoguFetcher
}

func NewRegisterDoguVersionStep(mgrSet *util.ManagerSet) *RegisterDoguVersionStep {
	return &RegisterDoguVersionStep{
		resourceDoguFetcher: mgrSet.ResourceDoguFetcher,
	}
}

func (rdvs *RegisterDoguVersionStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	_, _, err = rdvs.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	return 0, err
}
