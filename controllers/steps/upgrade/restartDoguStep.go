package upgrade

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/initfx"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/manager"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

// The RestartAfterConfigChangeStep restarts the dogu if the secret config, dogu config or global config is updated after the last start / restart of the dogu.
type RestartAfterConfigChangeStep struct {
	doguConfigRepository    doguConfigRepository
	sensitiveDoguRepository doguConfigRepository
	doguRestartManager      doguRestartManager
	deploymentManager       deploymentManager
	globalConfigRepository  globalConfigRepository
}

func NewRestartAfterConfigChangeStep(
	doguConfigRepo initfx.DoguConfigRepository,
	sensitiveDoguConfigRepo initfx.DoguConfigRepository,
	restartManager manager.DoguRestartManager,
	deploymentManager manager.DeploymentManager,
	globalConfigRepository resource.GlobalConfigRepository,
) *RestartAfterConfigChangeStep {
	return &RestartAfterConfigChangeStep{
		doguConfigRepository:    doguConfigRepo,
		sensitiveDoguRepository: sensitiveDoguConfigRepo,
		doguRestartManager:      restartManager,
		deploymentManager:       deploymentManager,
		globalConfigRepository:  globalConfigRepository,
	}
}

func (rds *RestartAfterConfigChangeStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
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
