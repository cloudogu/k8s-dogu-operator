package controllers

import (
	"context"
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// DoguRestartReconciler reconciles a DoguRestart object
type DoguRestartReconciler struct {
	clientSet ecoSystem.EcoSystemV1Alpha1Interface
	recorder  record.EventRecorder
}

type restartInstruction struct {
	op      restartOperation
	err     error
	req     ctrl.Request
	restart *k8sv1.DoguRestart
	dogu    *k8sv1.Dogu
}

type restartOperation string

const (
	ignore                     restartOperation = "ignore"
	wait                       restartOperation = "wait"
	stop                       restartOperation = "stop"
	checkStopped               restartOperation = "check if dogu is stopped"
	start                      restartOperation = "start"
	checkStarted               restartOperation = "check if dogu is started"
	handleDoguNotFound         restartOperation = "handle dogu not found"
	handleGetDoguFailed        restartOperation = "handle get dogu failed"
	handleGetDoguRestartFailed restartOperation = "handle get dogu restart failed"
)

//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=dogurestarts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=dogurestarts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.cloudogu.com,resources=dogurestarts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *DoguRestartReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO: add garbage collection (keep last 3 completed CRs of each dogu)
	instruction := r.evaluate(ctx, req)
	return r.execute(ctx, instruction)
}

func (r *DoguRestartReconciler) evaluate(ctx context.Context, req ctrl.Request) (instruction restartInstruction) {
	logger := log.FromContext(ctx).
		WithName("DoguRestartReconciler.evaluate").
		WithValues("doguRestart", req.NamespacedName)

	instruction.req = req

	doguRestart, err := r.clientSet.DoguRestarts(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		instruction.op = handleGetDoguRestartFailed
		instruction.err = err
		return
	}

	instruction.restart = doguRestart
	logger.Info("dogu restart ressource has been found")

	switch doguRestart.Status.Phase {
	case k8sv1.RestartStatusPhaseCompleted,
		k8sv1.RestartStatusPhaseDoguNotFound:
		instruction.op = ignore
		return // early exit to prevent unnecessary dogu loading
	case k8sv1.RestartStatusPhaseStopping:
		instruction.op = checkStopped
	case k8sv1.RestartStatusPhaseStarting:
		instruction.op = checkStarted
	case k8sv1.RestartStatusPhaseStopped,
		k8sv1.RestartStatusPhaseFailedStart:
		instruction.op = stop
	case k8sv1.RestartStatusPhaseNew,
		k8sv1.RestartStatusPhaseFailedStop:
		instruction.op = start
	default:
		logger.Info("no operation determined for dogu restart")
		instruction.op = ignore
		return
	}

	dogu, err := r.clientSet.Dogus(doguRestart.Namespace).Get(ctx, doguRestart.Spec.DoguName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		instruction.op = handleDoguNotFound
		instruction.err = err
		return
	}
	if err != nil {
		instruction.op = handleGetDoguFailed
		instruction.err = err
		return
	}

	instruction.dogu = dogu

	return
}

func (r *DoguRestartReconciler) execute(ctx context.Context, instruction restartInstruction) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("DoguRestartReconciler.execute").
		WithValues("doguRestart", instruction.req.NamespacedName).
		WithValues("dogu", instruction.dogu.Name)
	switch instruction.op {
	case ignore:
		logger.Info("nothing to do for dogu restart, ignoring")
		return ctrl.Result{}, nil
	case wait:
		logger.Info("dogu restart or its dogu have running operations, requeue scheduled")
		return ctrl.Result{RequeueAfter: requeueWaitTimeout}, nil
	case checkStopped:
		return r.checkStopped(ctx, instruction)
	case checkStarted:
		return r.checkStarted(ctx, instruction)
	case stop:
		return r.handleStop(ctx, instruction)
	case start:
		return r.handleStart(ctx, instruction)
	case handleGetDoguRestartFailed:
		logger.Error(instruction.err, "failed to get dogu restart")
		return ctrl.Result{}, client.IgnoreNotFound(instruction.err)
	case handleDoguNotFound:
		return r.handleDoguNotFound(ctx, instruction)
	case handleGetDoguFailed:
		return r.handleGetDoguFailed(ctx, instruction)
	default:
		logger.Info(fmt.Sprintf("unknown restart operation %q, ignoring", instruction.op))
		return ctrl.Result{}, nil
	}
}

func (r *DoguRestartReconciler) checkStopped(ctx context.Context, instruction restartInstruction) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("DoguRestartReconciler.checkStopped").
		WithValues("doguRestart", instruction.req.NamespacedName).
		WithValues("dogu", instruction.dogu.Name)

	if instruction.dogu.Status.Stopped {
		r.recorder.Event(instruction.restart, v1.EventTypeNormal, "Stopped", "dogu stopped, restarting")
		logger.Info("dogu stopped, setting stopped phase")

		_, statusErr := r.clientSet.DoguRestarts(instruction.req.Namespace).UpdateStatusWithRetry(ctx, instruction.restart, func(status k8sv1.DoguRestartStatus) k8sv1.DoguRestartStatus {
			status.Phase = k8sv1.RestartStatusPhaseStopped
			return status
		}, metav1.UpdateOptions{})

		return ctrl.Result{}, statusErr
	}

	logger.Info("dogu not yet stopped, requeue")
	return ctrl.Result{RequeueAfter: requeueWaitTimeout}, nil
}

func (r *DoguRestartReconciler) checkStarted(ctx context.Context, instruction restartInstruction) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("DoguRestartReconciler.checkStarted").
		WithValues("doguRestart", instruction.req.NamespacedName).
		WithValues("dogu", instruction.dogu.Name)

	if instruction.dogu.Status.Stopped {
		r.recorder.Event(instruction.restart, v1.EventTypeNormal, "Started", "dogu started, restart completed")
		logger.Info("dogu started, setting completed phase")

		_, statusErr := r.clientSet.DoguRestarts(instruction.req.Namespace).UpdateStatusWithRetry(ctx, instruction.restart, func(status k8sv1.DoguRestartStatus) k8sv1.DoguRestartStatus {
			status.Phase = k8sv1.RestartStatusPhaseCompleted
			return status
		}, metav1.UpdateOptions{})

		return ctrl.Result{}, statusErr
	}

	logger.Info("dogu not yet started, requeue")
	return ctrl.Result{RequeueAfter: requeueWaitTimeout}, nil
}

func (r *DoguRestartReconciler) handleStop(ctx context.Context, instruction restartInstruction) (ctrl.Result, error) {
	restartClient := r.clientSet.DoguRestarts(instruction.restart.Namespace)
	logger := log.FromContext(ctx).
		WithName("DoguRestartReconciler.handleStop").
		WithValues("doguRestart", instruction.req.NamespacedName).
		WithValues("dogu", instruction.dogu.Name)

	// set stopped field in dogu
	_, err := r.clientSet.Dogus(instruction.dogu.Namespace).UpdateSpecWithRetry(ctx, instruction.dogu, func(spec k8sv1.DoguSpec) k8sv1.DoguSpec {
		spec.Stopped = true
		return spec
	}, metav1.UpdateOptions{})
	if err != nil {
		r.recorder.Event(instruction.restart, v1.EventTypeWarning, "Stopping", "failed to stop dogu")
		logger.Error(err, "failed to set stopped field in dogu")

		_, statusErr := restartClient.UpdateStatusWithRetry(ctx, instruction.restart, func(status k8sv1.DoguRestartStatus) k8sv1.DoguRestartStatus {
			status.Phase = k8sv1.RestartStatusPhaseFailedStop
			return status
		}, metav1.UpdateOptions{})
		if statusErr != nil {
			logger.Error(statusErr, "failed to set stop failed status in restart")
			return ctrl.Result{}, errors.Join(err, statusErr)
		}

		return ctrl.Result{}, err
	}

	// set stopping status
	_, err = restartClient.UpdateStatusWithRetry(ctx, instruction.restart, func(status k8sv1.DoguRestartStatus) k8sv1.DoguRestartStatus {
		status.Phase = k8sv1.RestartStatusPhaseStopping
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		logger.Error(err, "failed to set stopping status for restart")
		return ctrl.Result{}, err
	}

	r.recorder.Event(instruction.restart, v1.EventTypeNormal, "Stopping", "initiated stop of dogu")
	return ctrl.Result{}, nil
}

func (r *DoguRestartReconciler) handleStart(ctx context.Context, instruction restartInstruction) (ctrl.Result, error) {
	restartClient := r.clientSet.DoguRestarts(instruction.restart.Namespace)
	logger := log.FromContext(ctx).
		WithName("DoguRestartReconciler.handleStart").
		WithValues("doguRestart", instruction.req.NamespacedName).
		WithValues("dogu", instruction.dogu.Name)

	// unset stopped field in dogu
	_, err := r.clientSet.Dogus(instruction.dogu.Namespace).UpdateSpecWithRetry(ctx, instruction.dogu, func(spec k8sv1.DoguSpec) k8sv1.DoguSpec {
		spec.Stopped = false
		return spec
	}, metav1.UpdateOptions{})
	if err != nil {
		r.recorder.Event(instruction.restart, v1.EventTypeWarning, "Starting", "failed to start dogu")
		logger.Error(err, "failed to unset stopped field in dogu")

		_, statusErr := restartClient.UpdateStatusWithRetry(ctx, instruction.restart, func(status k8sv1.DoguRestartStatus) k8sv1.DoguRestartStatus {
			status.Phase = k8sv1.RestartStatusPhaseFailedStart
			return status
		}, metav1.UpdateOptions{})
		if statusErr != nil {
			logger.Error(statusErr, "failed to set start failed status in restart")
			return ctrl.Result{}, errors.Join(err, statusErr)
		}

		return ctrl.Result{}, err
	}

	// set starting status
	_, err = restartClient.UpdateStatusWithRetry(ctx, instruction.restart, func(status k8sv1.DoguRestartStatus) k8sv1.DoguRestartStatus {
		status.Phase = k8sv1.RestartStatusPhaseStarting
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		logger.Error(err, "failed to set starting status for restart")
		return ctrl.Result{}, err
	}

	r.recorder.Event(instruction.restart, v1.EventTypeNormal, "Starting", "initiated start of dogu")
	return ctrl.Result{}, nil
}

func (r *DoguRestartReconciler) handleGetDoguFailed(ctx context.Context, instruction restartInstruction) (ctrl.Result, error) {
	r.recorder.Event(instruction.restart, v1.EventTypeWarning, "FailedGetDogu", "Could not get ressource of dogu to restart.")

	_, statusErr := r.clientSet.DoguRestarts(instruction.req.Namespace).UpdateStatusWithRetry(ctx, instruction.restart, func(status k8sv1.DoguRestartStatus) k8sv1.DoguRestartStatus {
		status.Phase = k8sv1.RestartStatusPhaseFailedGetDogu
		return status
	}, metav1.UpdateOptions{})

	// retry
	return ctrl.Result{}, errors.Join(instruction.err, statusErr)
}

func (r *DoguRestartReconciler) handleDoguNotFound(ctx context.Context, instruction restartInstruction) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("DoguRestartReconciler.handleDoguNotFound").
		WithValues("doguRestart", instruction.req.NamespacedName)
	r.recorder.Event(instruction.restart, v1.EventTypeWarning, "DoguNotFound", "Dogu to restart was not found.")

	_, statusErr := r.clientSet.DoguRestarts(instruction.req.Namespace).UpdateStatusWithRetry(ctx, instruction.restart, func(status k8sv1.DoguRestartStatus) k8sv1.DoguRestartStatus {
		status.Phase = k8sv1.RestartStatusPhaseDoguNotFound
		return status
	}, metav1.UpdateOptions{})
	if statusErr != nil {
		logger.Error(statusErr, "failed to update status of dogu restart")
	}

	// cannot restart, no retry necessary
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DoguRestartReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.DoguRestart{}).
		Complete(r)
}
