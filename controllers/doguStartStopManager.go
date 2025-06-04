package controllers

import (
	"context"
	"fmt"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// StartStopDoguEventReason is the reason string for firing start/stop dogu events.
	StartStopDoguEventReason = "StartStopDogu"
	// ErrorOnStartStopDoguEventReason is the error string for firing start/stop dogu error events.
	ErrorOnStartStopDoguEventReason = "ErrStartStopDogu"
)

// doguStartStopManager includes functionality to start and stop dogus.
type doguStartStopManager struct {
	resourceUpserter    resource.ResourceUpserter
	doguFetcher         localDoguFetcher
	doguInterface       doguClient.DoguInterface
	deploymentInterface deploymentInterface
}

type doguNotYetStartedStoppedError struct {
	err      error
	doguName string
	stopped  bool
}

func (n doguNotYetStartedStoppedError) Error() string {
	if n.err != nil {
		return fmt.Sprintf("error while starting/stopping dogu %q: %v", n.doguName, n.err)
	}

	desiredStateText := getDesiredStartStopStateText(n.stopped)
	return fmt.Sprintf("the dogu %q has not yet been changed to its desired state: %s", n.doguName, desiredStateText)
}

func (n doguNotYetStartedStoppedError) Requeue() bool {
	return true
}

func (n doguNotYetStartedStoppedError) GetRequeueTime() time.Duration {
	return requeueWaitTimeout
}

func newDoguStartStopManager(
	resourceUpserter resource.ResourceUpserter,
	doguFetcher localDoguFetcher,
	doguInterface doguClient.DoguInterface,
	deploymentInterface deploymentInterface,
) *doguStartStopManager {
	return &doguStartStopManager{
		resourceUpserter:    resourceUpserter,
		doguFetcher:         doguFetcher,
		doguInterface:       doguInterface,
		deploymentInterface: deploymentInterface,
	}
}

func (m *doguStartStopManager) StartStopDogu(ctx context.Context, doguResource *doguv2.Dogu) error {
	logger := log.FromContext(ctx)

	shouldStartStop, err := m.shouldStartStopDogu(ctx, doguResource)
	if shouldStartStop || err != nil {
		if err != nil {
			logger.Error(err, "error while checking if dogu should be started or stopped.")
		}

		if updateErr := m.startStopDogu(ctx, doguResource); updateErr != nil {
			return updateErr
		}

		return doguNotYetStartedStoppedError{doguName: doguResource.Name, stopped: doguResource.Spec.Stopped, err: err}
	}

	desiredStateText := getDesiredStartStopStateText(doguResource.Spec.Stopped)
	logger.Info(fmt.Sprintf("The dogu %q has changed to its desired state: %s", doguResource.Name, desiredStateText))
	return m.updateStatusWithRetry(ctx, doguResource, doguv2.DoguStatusInstalled, doguResource.Spec.Stopped)
}

func (m *doguStartStopManager) shouldStartStopDogu(ctx context.Context, doguResource *doguv2.Dogu) (bool, error) {
	var desiredReplicas int32 = resource.ReplicaCountStarted
	if doguResource.Spec.Stopped {
		desiredReplicas = resource.ReplicaCountStopped
	}

	deployment, getErr := m.deploymentInterface.Get(ctx, doguResource.GetObjectKey().Name, metav1.GetOptions{})
	if getErr != nil {
		return true, fmt.Errorf("failed to get deployment %q: %w", doguResource.Name, getErr)
	}

	return desiredReplicas != deployment.Status.Replicas, nil
}

func (m *doguStartStopManager) startStopDogu(ctx context.Context, doguResource *doguv2.Dogu) error {
	logger := log.FromContext(ctx)

	statusText := doguv2.DoguStatusStarting
	if doguResource.Spec.Stopped {
		statusText = doguv2.DoguStatusStopping
	}

	if err := m.updateStatusWithRetry(ctx, doguResource, statusText, doguResource.Status.Stopped); err != nil {
		return err
	}

	logger.Info("Getting local dogu descriptor...")
	dogu, err := m.doguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return fmt.Errorf("failed to get local descriptor for dogu %q: %w", doguResource.Name, err)
	}

	logger.Info("Upserting deployment...")
	_, err = m.resourceUpserter.UpsertDoguDeployment(ctx, doguResource, dogu, nil)
	if err != nil {
		return fmt.Errorf("failed to upsert deployment for starting/stopping dogu %q: %w", doguResource.Name, err)
	}

	return nil
}

func (m *doguStartStopManager) updateStatusWithRetry(ctx context.Context, doguResource *doguv2.Dogu, phase string, stopped bool) error {
	_, err := m.doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status doguv2.DoguStatus) doguv2.DoguStatus {
		status.Status = phase
		status.Stopped = stopped
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update status of dogu %q to %q: %w", doguResource.Name, phase, err)
	}

	return nil
}

func getDesiredStartStopStateText(stopped bool) string {
	desiredStateText := "started"
	if stopped {
		desiredStateText = "stopped"
	}
	return desiredStateText
}
