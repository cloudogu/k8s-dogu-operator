package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RestartOperation string

const (
	ignore                     RestartOperation = "ignore"
	wait                       RestartOperation = "wait"
	stop                       RestartOperation = "stop"
	checkStopped               RestartOperation = "check if dogu is stopped"
	start                      RestartOperation = "start"
	checkStarted               RestartOperation = "check if dogu is started"
	handleDoguNotFound         RestartOperation = "handle dogu not found"
	handleGetDoguFailed        RestartOperation = "handle get dogu failed"
	handleGetDoguRestartFailed RestartOperation = "handle get dogu restart failed"
)

const (
	updateStatusErrorMessage = "failed to update status of dogu restart"
)

const requeueWaitTimeout = 5 * time.Second

func RestartOperationFromRestartStatusPhase(phase doguv2.RestartStatusPhase) RestartOperation {
	switch phase {
	case doguv2.RestartStatusPhaseCompleted, doguv2.RestartStatusPhaseDoguNotFound:
		return ignore
	case doguv2.RestartStatusPhaseStopping:
		return checkStopped
	case doguv2.RestartStatusPhaseStarting:
		return checkStarted
	case doguv2.RestartStatusPhaseStopped, doguv2.RestartStatusPhaseFailedStart:
		return start
	case doguv2.RestartStatusPhaseNew, doguv2.RestartStatusPhaseFailedStop:
		return stop
	default:
		return ignore
	}
}

type restartInstruction struct {
	op                   RestartOperation
	err                  error
	req                  ctrl.Request
	restart              *doguv2.DoguRestart
	dogu                 *doguv2.Dogu
	doguRestartInterface doguRestartInterface
	doguInterface        doguInterface
	recorder             eventRecorder
}

func (r *restartInstruction) execute(ctx context.Context) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("restartInstruction.execute").
		WithValues("doguRestart", r.req.NamespacedName)
	switch r.op {
	case ignore:
		logger.Info("nothing to do for dogu restart, ignoring")
		return ctrl.Result{}, nil
	case wait:
		logger.Info("dogu restart or its dogu have running operations, requeue scheduled")
		return ctrl.Result{RequeueAfter: requeueWaitTimeout}, nil
	case checkStopped:
		return r.checkStopped(ctx)
	case checkStarted:
		return r.checkStarted(ctx)
	case stop:
		return r.handleStop(ctx)
	case start:
		return r.handleStart(ctx)
	case handleGetDoguRestartFailed:
		logger.Error(r.err, "failed to get dogu restart")
		return ctrl.Result{}, client.IgnoreNotFound(r.err)
	case handleDoguNotFound:
		return r.handleDoguNotFound(ctx)
	case handleGetDoguFailed:
		return r.handleGetDoguFailed(ctx)
	default:
		logger.Info(fmt.Sprintf("unknown restart operation %q, ignoring", r.op))
		return ctrl.Result{}, nil
	}
}

func (r *restartInstruction) checkStopped(ctx context.Context) (ctrl.Result, error) {
	return r.checkStartStop(ctx, true)
}

func (r *restartInstruction) checkStarted(ctx context.Context) (ctrl.Result, error) {
	return r.checkStartStop(ctx, false)
}

func (r *restartInstruction) checkStartStop(ctx context.Context, shouldBeStopped bool) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("DoguRestartReconciler.checkStartStop").
		WithValues("doguRestart", r.req.NamespacedName).
		WithValues("dogu", r.dogu.Name)

	eventMessage, logMessage, notReadyLogMessage, eventReason, requeue, phase := getCheckStartStopAttributes(shouldBeStopped)

	if (r.dogu.Status.Stopped && shouldBeStopped) || (!r.dogu.Status.Stopped && !shouldBeStopped) {
		r.recorder.Event(r.restart, v1.EventTypeNormal, eventReason, eventMessage)
		logger.Info(logMessage)
		statusErr := r.updateDoguRestartPhase(ctx, phase)

		// directly start after stop
		return ctrl.Result{Requeue: requeue}, statusErr
	}

	logger.Info(notReadyLogMessage)
	return ctrl.Result{RequeueAfter: requeueWaitTimeout}, nil
}

func getCheckStartStopAttributes(stopped bool) (eventMessage, logMessage, notReadyLogMessage, eventReason string, requeue bool, phase doguv2.RestartStatusPhase) {
	eventReason = "Started"
	eventMessage = "dogu started, restart completed"
	logMessage = "dogu started, setting completed phase"
	notReadyLogMessage = "dogu not yet started, requeue"
	phase = doguv2.RestartStatusPhaseCompleted
	requeue = false
	if stopped {
		eventReason = "Stopped"
		eventMessage = "dogu stopped, restarting"
		logMessage = "dogu stopped, setting stopped phase"
		phase = doguv2.RestartStatusPhaseStopped
		notReadyLogMessage = "dogu not yet stopped, requeue"
		requeue = true
	}
	return
}

func (r *restartInstruction) handleStop(ctx context.Context) (ctrl.Result, error) {
	return r.handleStartStop(ctx, true)
}

func (r *restartInstruction) handleStart(ctx context.Context) (ctrl.Result, error) {
	return r.handleStartStop(ctx, false)
}

func (r *restartInstruction) handleStartStop(ctx context.Context, shouldStop bool) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("DoguRestartReconciler.handleStartStop").
		WithValues("doguRestart", r.req.NamespacedName).
		WithValues("dogu", r.dogu.Name)

	eventMessage, eventInitMessage, restartStatusErrorMessage, eventReason, requeue, phase, phaseOnFail := getHandleStartStopAttributes(shouldStop)

	// set stopped field in dogu
	err := r.updateDoguSpecStopped(ctx, shouldStop)
	if err != nil {
		r.recorder.Event(r.restart, v1.EventTypeWarning, eventReason, eventMessage)
		logger.Error(err, "failed to set stopped field in dogu")

		statusErr := r.updateDoguRestartPhase(ctx, phaseOnFail)
		if statusErr != nil {
			logger.Error(statusErr, "failed to set stop failed status in restart")
			return ctrl.Result{}, errors.Join(err, statusErr)
		}

		return ctrl.Result{}, err
	}

	// set stopping status
	err = r.updateDoguRestartPhase(ctx, phase)
	if err != nil {
		logger.Error(err, restartStatusErrorMessage)
		return ctrl.Result{}, err
	}

	r.recorder.Event(r.restart, v1.EventTypeNormal, eventReason, eventInitMessage)
	if requeue {
		return ctrl.Result{RequeueAfter: requeueWaitTimeout}, nil
	} else {
		return ctrl.Result{}, nil
	}
}

func getHandleStartStopAttributes(stopped bool) (eventMessage, eventInitMessage, restartStatusErrorMessage, eventReason string, requeue bool, phase, phaseOnFail doguv2.RestartStatusPhase) {
	eventReason = "Starting"
	eventMessage = "failed to start dogu"
	eventInitMessage = "initiated start of dogu"
	restartStatusErrorMessage = "failed to set starting status for restart"
	requeue = false
	phase = doguv2.RestartStatusPhaseStarting
	phaseOnFail = doguv2.RestartStatusPhaseFailedStart
	if stopped {
		eventReason = "Stopping"
		eventMessage = "failed to stop dogu"
		eventInitMessage = "initiated stop of dogu"
		restartStatusErrorMessage = "failed to set stopping status for restart"
		requeue = true
		phase = doguv2.RestartStatusPhaseStopping
		phaseOnFail = doguv2.RestartStatusPhaseFailedStop
	}

	return
}

func (r *restartInstruction) handleGetDoguFailed(ctx context.Context) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("DoguRestartReconciler.handleDoguNotFound").
		WithValues("doguRestart", r.req.NamespacedName)
	r.recorder.Event(r.restart, v1.EventTypeWarning, "FailedGetDogu", "Could not get ressource of dogu to restart.")

	statusErr := r.updateDoguRestartPhase(ctx, doguv2.RestartStatusPhaseFailedGetDogu)
	if statusErr != nil {
		logger.Error(statusErr, updateStatusErrorMessage)
	}

	// retry
	return ctrl.Result{}, errors.Join(r.err, statusErr)
}

func (r *restartInstruction) handleDoguNotFound(ctx context.Context) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("DoguRestartReconciler.handleDoguNotFound").
		WithValues("doguRestart", r.req.NamespacedName)
	r.recorder.Event(r.restart, v1.EventTypeWarning, "DoguNotFound", "Dogu to restart was not found.")

	statusErr := r.updateDoguRestartPhase(ctx, doguv2.RestartStatusPhaseDoguNotFound)
	if statusErr != nil {
		logger.Error(statusErr, updateStatusErrorMessage)
	}

	// cannot restart, no retry necessary
	return ctrl.Result{}, nil
}

func (r *restartInstruction) updateDoguRestartPhase(ctx context.Context, phase doguv2.RestartStatusPhase) error {
	_, statusErr := r.doguRestartInterface.UpdateStatusWithRetry(ctx, r.restart, func(status doguv2.DoguRestartStatus) doguv2.DoguRestartStatus {
		status.Phase = phase
		return status
	}, metav1.UpdateOptions{})

	return statusErr
}

func (r *restartInstruction) updateDoguSpecStopped(ctx context.Context, shouldStop bool) error {
	_, statusErr := r.doguInterface.UpdateSpecWithRetry(ctx, r.dogu, func(spec doguv2.DoguSpec) doguv2.DoguSpec {
		spec.Stopped = shouldStop
		return spec
	}, metav1.UpdateOptions{})

	return statusErr
}
