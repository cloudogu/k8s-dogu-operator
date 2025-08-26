package deletion

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceAccountRemoverStep struct {
	serviceAccountRemover serviceAccountRemover
	resourceDoguFetcher   resourceDoguFetcher
}

func NewServiceAccountRemoverStep(
	client client.Client,
	mgrSet *util.ManagerSet,
	configRepos util.ConfigRepositories,
	operatorConfig *config.OperatorConfig,
) *ServiceAccountRemoverStep {
	return &ServiceAccountRemoverStep{
		serviceAccountRemover: serviceaccount.NewRemover(configRepos.SensitiveDoguRepository, mgrSet.LocalDoguFetcher, mgrSet.CommandExecutor, client, mgrSet.ClientSet, operatorConfig.Namespace),
		resourceDoguFetcher:   mgrSet.ResourceDoguFetcher,
	}
}

func (sas *ServiceAccountRemoverStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	doguDescriptor, err := sas.getDoguDescriptor(ctx, doguResource)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}

	err = sas.serviceAccountRemover.RemoveAll(ctx, doguDescriptor)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}
	return steps.StepResult{}
}

func (sas *ServiceAccountRemoverStep) getDoguDescriptor(ctx context.Context, doguResource *v2.Dogu) (*core.Dogu, error) {
	doguDescriptor, _, err := sas.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dogu descriptor: %w", err)
	}

	return doguDescriptor, nil
}
