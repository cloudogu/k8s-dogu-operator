package deletion

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

type ServiceAccountRemoverStep struct {
	serviceAccountRemover serviceAccountRemover
	resourceDoguFetcher   resourceDoguFetcher
}

func NewServiceAccountRemoverStep(
	client k8sClient,
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
	doguDescriptor, _, err := sas.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	err = sas.serviceAccountRemover.RemoveAll(ctx, doguDescriptor)
	if err != nil {
		return steps.RequeueWithError(err)
	}
	return steps.Continue()
}
