package install

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

const requeueAfterServiceAccount = 5 * time.Second

type ServiceAccountStep struct {
	serviceAccountCreator serviceaccount.ServiceAccountCreator
	resourceDoguFetcher   resourceDoguFetcher
}

func NewServiceAccountStep(mgrSet util.ManagerSet) *ServiceAccountStep {
	return &ServiceAccountStep{
		serviceAccountCreator: mgrSet.ServiceAccountCreator,
		resourceDoguFetcher:   mgrSet.ResourceDoguFetcher,
	}
}

func (sas *ServiceAccountStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	doguDescriptor, err := sas.getDoguDescriptor(ctx, doguResource)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}
	// Existing service accounts will be skipped.
	err = sas.serviceAccountCreator.CreateAll(ctx, doguDescriptor)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}
	return steps.StepResult{}
}

func (sas *ServiceAccountStep) getDoguDescriptor(ctx context.Context, doguResource *v2.Dogu) (*core.Dogu, error) {
	doguDescriptor, _, err := sas.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dogu descriptor: %w", err)
	}

	return doguDescriptor, nil
}
