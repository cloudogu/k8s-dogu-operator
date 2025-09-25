package upgrade

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/initfx"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/manager"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type RestartDoguStep struct {
	doguConfigRepository    doguConfigRepository
	sensitiveDoguRepository doguConfigRepository
	doguRestartManager      doguRestartManager
	deploymentManager       deploymentManager
	configMapInterface      configMapInterface
}

func NewRestartDoguStep(
	doguConfigRepo initfx.DoguConfigRepository,
	sensitiveDoguConfigRepo initfx.DoguConfigRepository,
	restartManager manager.DoguRestartManager,
	deploymentManager manager.DeploymentManager,
	mapInterface v1.ConfigMapInterface,
) *RestartDoguStep {
	return &RestartDoguStep{
		doguConfigRepository:    doguConfigRepo,
		sensitiveDoguRepository: sensitiveDoguConfigRepo,
		doguRestartManager:      restartManager,
		deploymentManager:       deploymentManager,
		configMapInterface:      mapInterface,
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

	globalConfig, err := rds.configMapInterface.Get(ctx, "global-config", metav1.GetOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	globalConfigLastUpdateTime := rds.getConfigMapLastUpdatedTime(globalConfig)

	if startingTime != nil && (startingTime.Before(sensConfig.LastUpdated.Time) || startingTime.Before(doguConfig.LastUpdated.Time) || startingTime.Before(globalConfigLastUpdateTime.Time)) {
		err = rds.doguRestartManager.RestartDogu(ctx, doguResource)
		if err != nil {
			return steps.RequeueWithError(err)
		}
	}

	return steps.Continue()
}

func (rds *RestartDoguStep) getConfigMapLastUpdatedTime(cm *corev1.ConfigMap) *metav1.Time {
	timestamp := cm.GetCreationTimestamp()
	latest := &timestamp

	for _, managedFields := range cm.GetManagedFields() {
		if managedFields.Time != nil && managedFields.Time.After(latest.Time) {
			latest = managedFields.Time
		}
	}

	return latest
}
