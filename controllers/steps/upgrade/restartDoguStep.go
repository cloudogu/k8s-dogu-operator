package upgrade

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/initfx"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/manager"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type RestartDoguStep struct {
	doguConfigRepository    doguConfigRepository
	sensitiveDoguRepository doguConfigRepository
	doguRestartManager      doguRestartManager
	deploymentManager       deploymentManager
	configMapInterface      configMapInterface
	globalConfigRepository  globalConfigRepository
}

func NewRestartDoguStep(
	doguConfigRepo initfx.DoguConfigRepository,
	sensitiveDoguConfigRepo initfx.DoguConfigRepository,
	restartManager manager.DoguRestartManager,
	deploymentManager manager.DeploymentManager,
	globalConfigRepository resource.GlobalConfigRepository,
) *RestartDoguStep {
	return &RestartDoguStep{
		doguConfigRepository:    doguConfigRepo,
		sensitiveDoguRepository: sensitiveDoguConfigRepo,
		doguRestartManager:      restartManager,
		deploymentManager:       deploymentManager,
		globalConfigRepository:  globalConfigRepository,
	}
}

func (rds *RestartDoguStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	startingTime, err := rds.deploymentManager.GetLastStartingTime(ctx, doguResource.Name)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	sensConfig, err := rds.sensitiveDoguRepository.Get(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(err)
	}

	doguConfig, err := rds.doguConfigRepository.Get(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return steps.RequeueWithError(err)
	}

	globalConfig, err := rds.globalConfigRepository.Get(ctx)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if startingTime != nil && (startingTime.Before(sensConfig.LastUpdated.Time) || startingTime.Before(doguConfig.LastUpdated.Time) || startingTime.Before(globalConfig.LastUpdated.Time)) {
		err = rds.doguRestartManager.RestartDogu(ctx, doguResource)
		if err != nil {
			return steps.RequeueWithError(err)
		}
	}

	return steps.Continue()
}
