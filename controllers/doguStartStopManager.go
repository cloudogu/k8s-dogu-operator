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
	"github.com/cloudogu/k8s-dogu-operator/retry"
)

const (
	// StartDoguEventReason is the reason string for firing start dogu events.
	StartDoguEventReason = "StartDogu"
	// ErrorOnStartDoguEventReason is the error string for firing start dogu error events.
	ErrorOnStartDoguEventReason = "ErrStartDogu"

	// StopDoguEventReason is the reason string for firing stop dogu events.
	StopDoguEventReason = "StopDogu"
	// ErrorOnStopDoguEventReason is the error string for firing stop dogu error events.
	ErrorOnStopDoguEventReason = "ErrStopDogu"
)

const (
	scaleDeploymentWaitTimeout  = time.Minute * 10
	scaleDeploymentWaitInterval = time.Second * 5
)

const containerStateCrashLoop = "CrashLoopBackOff"

// DoguStartStopManager includes functionality to start and stop dogus.
type DoguStartStopManager struct {
	clientSet  thirdParty.ClientSet
	doguClient cloudogu.EcosystemInterface
}

// StartDogu scales a stopped dogu to 1.
func (m *DoguStartStopManager) StartDogu(ctx context.Context, doguResource *k8sv1.Dogu) error {
	err := m.updateStatusWithRetry(ctx, doguResource, k8sv1.DoguStatusStarting)
	if err != nil {
		return err
	}

	err = m.scaleDeployment(ctx, doguResource.GetObjectKey(), 1, true)
	if err != nil {
		return fmt.Errorf("failed while starting dogu %q: %w", doguResource.Name, err)
	}

	err = m.updateStatusWithRetry(ctx, doguResource, k8sv1.DoguStatusInstalled)
	if err != nil {
		return err
	}

	return nil
}

// StopDogu scales a running dogu to 0.
func (m *DoguStartStopManager) StopDogu(ctx context.Context, doguResource *k8sv1.Dogu) error {
	err := m.updateStatusWithRetry(ctx, doguResource, k8sv1.DoguStatusStopping)
	if err != nil {
		return err
	}

	err = m.scaleDeployment(ctx, doguResource.GetObjectKey(), 0, true)
	if err != nil {
		return fmt.Errorf("failed while stopping dogu %q: %w", doguResource.Name, err)
	}

	err = m.updateStatusWithRetry(ctx, doguResource, k8sv1.DoguStatusInstalled)
	if err != nil {
		return err
	}

	return nil
}

func (m *DoguStartStopManager) updateStatusWithRetry(ctx context.Context, doguResource *k8sv1.Dogu, status string) error {
	err := retry.OnConflict(func() error {
		latestDoguResource, err := m.doguClient.Dogus(doguResource.Namespace).Get(ctx, doguResource.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		latestDoguResource.Status.Status = status

		_, err = m.doguClient.Dogus(doguResource.Namespace).UpdateStatus(ctx, doguResource, metav1.UpdateOptions{})
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update status of dogu %q to %q: %w", doguResource.Name, status, err)
	}

	return nil
}

func (m *DoguStartStopManager) scaleDeployment(ctx context.Context, doguName types.NamespacedName, replicas int32, waitForRollout bool) error {
	scale := &scalingv1.Scale{ObjectMeta: metav1.ObjectMeta{Name: doguName.Name, Namespace: doguName.Namespace}, Spec: scalingv1.ScaleSpec{Replicas: replicas}}
	_, err := m.clientSet.AppsV1().Deployments(doguName.Namespace).UpdateScale(ctx, doguName.Name, scale, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to scale deployment %q to %d: %s", doguName, replicas, err.Error())
	}

	if waitForRollout {
		return m.waitForDeploymentRollout(ctx, doguName)
	}

	return nil
}

func (m *DoguStartStopManager) waitForDeploymentRollout(ctx context.Context, doguName types.NamespacedName) error {
	timeoutTimer := time.NewTimer(scaleDeploymentWaitTimeout)
	// Use a ticker instead of a kubernetes watch because the watch does not notify on status changes.
	ticker := time.NewTicker(scaleDeploymentWaitInterval)
	for {
		select {
		case <-ticker.C:
			rolledOut, stopWait, err := m.doWaitForDeploymentRollout(ctx, doguName)
			if err != nil {
				stopWaitChannels(timeoutTimer, ticker)
				return err
			}

			if stopWait || rolledOut {
				stopWaitChannels(timeoutTimer, ticker)
				return nil
			}
		case <-timeoutTimer.C:
			ticker.Stop()
			return fmt.Errorf("failed to wait for deployment %q rollout: timeout reached", doguName)
		}
	}
}

func (m *DoguStartStopManager) doWaitForDeploymentRollout(ctx context.Context, doguName types.NamespacedName) (rolledOut bool, stopWait bool, err error) {
	logrus.Info(fmt.Sprintf("check rollout status for deployment %s", doguName))
	deployment, getErr := m.clientSet.AppsV1().Deployments(doguName.Namespace).Get(ctx, doguName.Name, metav1.GetOptions{})
	if getErr != nil {
		return false, true, fmt.Errorf("failed to get deployment %s: %w", doguName, getErr)
	}

	isInCrashLoop, err := m.isDoguContainerInCrashLoop(ctx, doguName)
	if err != nil || isInCrashLoop {
		return false, true, err
	}

	if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
		logrus.Info(fmt.Sprintf("waiting for deployment %q rollout to finish: %d out of %d new replicas have been updated", deployment.Name, deployment.Status.UpdatedReplicas, *deployment.Spec.Replicas))
		return false, false, nil
	}
	if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
		logrus.Info(fmt.Sprintf("waiting for deployment %q rollout to finish: %d old replicas are pending termination", deployment.Name, deployment.Status.Replicas-deployment.Status.UpdatedReplicas))
		return false, false, nil
	}
	if deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas {
		logrus.Info(fmt.Sprintf("waiting for deployment %q rollout to finish: %d of %d updated replicas are available", deployment.Name, deployment.Status.AvailableReplicas, deployment.Status.UpdatedReplicas))
		return false, false, nil
	}
	logrus.Info(fmt.Sprintf("deployment %q successfully rolled out", deployment.Name))
	return true, true, nil
}

func stopWaitChannels(timer *time.Timer, ticker *time.Ticker) {
	timer.Stop()
	ticker.Stop()
}

func (m *DoguStartStopManager) isDoguContainerInCrashLoop(ctx context.Context, doguName types.NamespacedName) (bool, error) {
	list, getErr := m.clientSet.CoreV1().Pods(doguName.Namespace).List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("dogu.name=%s", doguName)})
	if getErr != nil {
		return false, fmt.Errorf("failed to get pods of deployment %q", doguName)
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
