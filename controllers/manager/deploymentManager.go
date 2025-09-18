package manager

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type deploymentManager struct {
	deploymentInterface deploymentInterface
	podInterface        podInterface
}

func NewDeploymentManager(
	podInterface corev1.PodInterface,
	deploymentInterface appsv1.DeploymentInterface,
) DeploymentManager {
	return &deploymentManager{
		podInterface:        podInterface,
		deploymentInterface: deploymentInterface,
	}
}

func (dm *deploymentManager) GetLastStartingTime(ctx context.Context, deploymentName string) (*time.Time, error) {
	deployment, err := dm.deploymentInterface.Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	labelSelector := metav1.FormatLabelSelector(deployment.Spec.Selector)

	pods, err := dm.podInterface.List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
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
