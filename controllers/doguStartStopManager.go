package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	scalingv1 "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
	"github.com/cloudogu/k8s-dogu-operator/internal/thirdParty"
)

const (
	// StartDoguEventReason is the reason string for firing start dogu events.
	StartDoguEventReason = "StartDogu"
	// ErrorOnStartDoguEventReason is the error string for firing start dogu error events.
	ErrorOnStartDoguEventReason = "ErrStartDogu"

	// StopDoguEventReason is the reason string for firing stop dogu events.
	StopDoguEventReason = "StopDogu"
	// ErrorOnStopDoguEventReason is the error string for firing stop dogu error events.
	ErrorOnStopDoguEventReason     = "ErrStopDogu"
	CheckStartedEventReason        = "CheckStarted"
	ErrorOnCheckStartedEventReason = "ErrCheckStarted"
	CheckStoppedEventReason        = "CheckStopped"
	ErrorOnCheckStoppedEventReason = "ErrCheckStopped"
)

const containerStateCrashLoop = "CrashLoopBackOff"

// doguStartStopManager includes functionality to start and stop dogus.
type doguStartStopManager struct {
	clientSet  thirdParty.ClientSet
	doguClient cloudogu.EcosystemInterface
}

type deploymentNotYetScaledError struct {
	doguName string
}

func (n deploymentNotYetScaledError) Error() string {
	return fmt.Sprintf("the deployment of dogu %q has not yet been scaled to its desired number of replicas", n.doguName)
}

func (n deploymentNotYetScaledError) Requeue() bool {
	return true
}

func (n deploymentNotYetScaledError) GetRequeueTime() time.Duration {
	return requeueWaitTimeout
}

func (m *doguStartStopManager) CheckStarted(ctx context.Context, doguResource *k8sv1.Dogu) error {
	rolledOut, err := m.checkForDeploymentRollout(ctx, doguResource.GetObjectKey())
	if err != nil {
		return fmt.Errorf("failed to start dogu %q: %w", doguResource.GetObjectKey(), err)
	}

	if !rolledOut {
		return deploymentNotYetScaledError{doguName: doguResource.GetObjectKey().String()}
	}

	err = m.updateStatusWithRetry(ctx, doguResource, k8sv1.DoguStatusInstalled, false)
	if err != nil {
		return err
	}

	return nil
}

func (m *doguStartStopManager) CheckStopped(ctx context.Context, doguResource *k8sv1.Dogu) error {
	rolledOut, err := m.checkForDeploymentRollout(ctx, doguResource.GetObjectKey())
	if err != nil {
		return fmt.Errorf("failed to stop dogu %q: %w", doguResource.GetObjectKey(), err)
	}

	if !rolledOut {
		return deploymentNotYetScaledError{doguName: doguResource.GetObjectKey().String()}
	}

	err = m.updateStatusWithRetry(ctx, doguResource, k8sv1.DoguStatusInstalled, true)
	if err != nil {
		return err
	}

	return nil
}

func newDoguStartStopManager(clientSet thirdParty.ClientSet, doguClient cloudogu.EcosystemInterface) *doguStartStopManager {
	return &doguStartStopManager{clientSet: clientSet, doguClient: doguClient}
}

// StartDogu scales a stopped dogu to 1.
func (m *doguStartStopManager) StartDogu(ctx context.Context, doguResource *k8sv1.Dogu) error {
	err := m.updateStatusWithRetry(ctx, doguResource, k8sv1.DoguStatusStarting, doguResource.Status.Stopped)
	if err != nil {
		return err
	}

	err = m.scaleDeployment(ctx, doguResource.GetObjectKey(), 1)
	if err != nil {
		return fmt.Errorf("failed to start dogu %q: %w", doguResource.Name, err)
	}

	return nil
}

// StopDogu scales a running dogu to 0.
func (m *doguStartStopManager) StopDogu(ctx context.Context, doguResource *k8sv1.Dogu) error {
	err := m.updateStatusWithRetry(ctx, doguResource, k8sv1.DoguStatusStopping, doguResource.Status.Stopped)
	if err != nil {
		return err
	}

	err = m.scaleDeployment(ctx, doguResource.GetObjectKey(), 0)
	if err != nil {
		return fmt.Errorf("failed while stopping dogu %q: %w", doguResource.Name, err)
	}

	return nil
}

func (m *doguStartStopManager) updateStatusWithRetry(ctx context.Context, doguResource *k8sv1.Dogu, phase string, stopped bool) error {
	_, err := m.doguClient.Dogus(doguResource.Namespace).UpdateStatusWithRetry(ctx, doguResource, func(status k8sv1.DoguStatus) k8sv1.DoguStatus {
		status.Status = phase
		status.Stopped = stopped
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update status of dogu %q to %q: %w", doguResource.Name, phase, err)
	}

	return nil
}

func (m *doguStartStopManager) scaleDeployment(ctx context.Context, doguName types.NamespacedName, replicas int32) error {
	scale := &scalingv1.Scale{ObjectMeta: metav1.ObjectMeta{Name: doguName.Name, Namespace: doguName.Namespace}, Spec: scalingv1.ScaleSpec{Replicas: replicas}}
	_, err := m.clientSet.AppsV1().Deployments(doguName.Namespace).UpdateScale(ctx, doguName.Name, scale, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to scale deployment %q to %d: %s", doguName, replicas, err.Error())
	}

	return nil
}

func (m *doguStartStopManager) checkForDeploymentRollout(ctx context.Context, doguName types.NamespacedName) (rolledOut bool, err error) {
	logrus.Info(fmt.Sprintf("check rollout status for deployment %s", doguName))
	deployment, getErr := m.clientSet.AppsV1().Deployments(doguName.Namespace).Get(ctx, doguName.Name, metav1.GetOptions{})
	if getErr != nil {
		return false, fmt.Errorf("failed to get deployment %q: %w", doguName, getErr)
	}

	isInCrashLoop, err := m.isDoguContainerInCrashLoop(ctx, doguName)
	if err != nil || isInCrashLoop {
		return false, err
	}

	switch {
	case deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas:
		logrus.Info(fmt.Sprintf("waiting for deployment %q rollout to finish: %d out of %d new replicas have been updated", deployment.Name, deployment.Status.UpdatedReplicas, *deployment.Spec.Replicas))
	case deployment.Status.Replicas > deployment.Status.UpdatedReplicas:
		logrus.Info(fmt.Sprintf("waiting for deployment %q rollout to finish: %d old replicas are pending termination", deployment.Name, deployment.Status.Replicas-deployment.Status.UpdatedReplicas))
	case deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas:
		logrus.Info(fmt.Sprintf("waiting for deployment %q rollout to finish: %d of %d updated replicas are available", deployment.Name, deployment.Status.AvailableReplicas, deployment.Status.UpdatedReplicas))
	default:
		logrus.Info(fmt.Sprintf("deployment %q successfully rolled out", deployment.Name))
		return true, nil
	}

	return false, nil
}

func (m *doguStartStopManager) isDoguContainerInCrashLoop(ctx context.Context, doguName types.NamespacedName) (bool, error) {
	list, getErr := m.clientSet.CoreV1().Pods(doguName.Namespace).List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("dogu.name=%s", doguName.Name)})
	if getErr != nil {
		return false, fmt.Errorf("failed to get pods of deployment %q: %w", doguName, getErr)
	}

	for _, pod := range list.Items {
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name != doguName.Name {
				continue
			}

			containerWaitState := containerStatus.State.Waiting

			if containerWaitState != nil && containerWaitState.Reason == containerStateCrashLoop {
				logrus.Error(fmt.Errorf("some containers are in a crash loop"), fmt.Sprintf("skip waiting rollout for deployment %s", doguName))
				return true, nil
			}
		}
	}

	return false, nil
}
