package postinstall

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v4 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RestartDoguStep struct {
	client                  client.Client
	doguConfigRepository    doguConfigRepository
	sensitiveDoguRepository doguConfigRepository
	podInterface            v4.PodInterface
	doguRestartManager      doguRestartManager
}

func NewRestartDoguStep(client client.Client, mgrSet *util.ManagerSet, namespace string, configRepos util.ConfigRepositories, manager doguRestartManager) *RestartDoguStep {
	return &RestartDoguStep{
		client:                  client,
		podInterface:            mgrSet.ClientSet.CoreV1().Pods(namespace),
		doguConfigRepository:    configRepos.DoguConfigRepository,
		sensitiveDoguRepository: configRepos.SensitiveDoguRepository,
		doguRestartManager:      manager,
	}
}

func (rds *RestartDoguStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	logger := log.FromContext(ctx)
	deployment, err := doguResource.GetDeployment(ctx, rds.client)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	startingTime, err := rds.getDeploymentLastStartingTime(ctx, deployment)
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
	if startingTime != nil {
		logger.Info(fmt.Sprintf("Starting time: %q, sensitive update: %q, sensitive after starting time: %t, dogu update: %q, dogu after starting time: %t", startingTime, sensConfig.LastUpdated.Time, startingTime.Before(sensConfig.LastUpdated.Time), doguConfig.LastUpdated.Time, startingTime.Before(doguConfig.LastUpdated.Time)))
	}

	if startingTime != nil && (startingTime.Before(sensConfig.LastUpdated.Time) || startingTime.Before(doguConfig.LastUpdated.Time)) {
		err := rds.doguRestartManager.RestartDogu(ctx, doguResource)
		if err != nil {
			return steps.RequeueWithError(err)
		}
	}

	return steps.Continue()
}
func (rds RestartDoguStep) getDeploymentLastStartingTime(ctx context.Context, deployment *v1.Deployment) (*time.Time, error) {
	labelSelector := metav1.FormatLabelSelector(deployment.Spec.Selector)

	pods, err := rds.podInterface.List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, err
	}

	var lastTimeStarted *time.Time
	for _, pod := range pods.Items {
		if pod.Status.StartTime != nil {
			startTime := pod.Status.StartTime.Time
			if lastTimeStarted == nil || startTime.After(*lastTimeStarted) {
				lastTimeStarted = &startTime
			}
		}
	}
	return lastTimeStarted, nil
}
