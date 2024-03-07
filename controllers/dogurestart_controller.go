/*
This file was generated with "make generate-deepcopy".
*/

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
	start                      restartOperation = "start"
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
	instruction := r.evaluate(ctx, req)
	return r.execute(ctx, instruction)
}

func (r *DoguRestartReconciler) evaluate(ctx context.Context, req ctrl.Request) (instruction restartInstruction) {
	logger := log.FromContext(ctx)

	instruction.req = req

	doguRestart, err := r.clientSet.DoguRestarts(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		instruction.op = handleGetDoguRestartFailed
		instruction.err = err
		return
	}

	instruction.restart = doguRestart
	logger.Info(fmt.Sprintf("dogu restart ressource %q has been found", req.NamespacedName))

	// ignore cases
	switch doguRestart.Status.Phase {
	case k8sv1.RestartStatusPhaseCompleted,
		k8sv1.RestartStatusPhaseDoguNotFound:
		instruction.op = ignore
		return
	}

	dogu, err := r.clientSet.Dogus(doguRestart.Namespace).Get(ctx, doguRestart.Name, metav1.GetOptions{})
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

	// wait if any other operations are running
	switch doguRestart.Status.Phase {
	case k8sv1.RestartStatusPhaseStarting,
		k8sv1.RestartStatusPhaseStopping:
		instruction.op = wait
		return
	}
	if dogu.Status.Status != k8sv1.DoguStatusInstalled {
		instruction.op = wait
		return
	}

	switch doguRestart.Status.Phase {
	// check if stop is necessary
	case k8sv1.RestartStatusPhaseNew,
		k8sv1.RestartStatusPhaseFailedStop:
		instruction.op = stop
		return
	// check if start is necessary
	case k8sv1.RestartStatusPhaseStopped,
		k8sv1.RestartStatusPhaseFailedStart:
		instruction.op = start
		return
	}

	// TODO check if other operations are necessary or status phases have to be synchronized
	instruction.op = ignore
	return
}

func (r *DoguRestartReconciler) execute(ctx context.Context, instruction restartInstruction) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	switch instruction.op {
	case ignore:
		logger.Info(fmt.Sprintf("nothing to do for restart %q, ignoring", instruction.restart.Name))
		return ctrl.Result{}, nil
	case wait:
		logger.Info(fmt.Sprintf("restart %q or its dogu have running operations, requeue scheduled", instruction.restart.Name))
		return ctrl.Result{RequeueAfter: requeueWaitTimeout}, nil
	case stop:
		// TODO implement
		return ctrl.Result{}, nil
	case start:
		// TODO implement
		return ctrl.Result{}, nil
	case handleGetDoguRestartFailed:
		logger.Error(instruction.err, fmt.Sprintf("failed to get dogu restart %q", instruction.req.NamespacedName))
		return ctrl.Result{}, client.IgnoreNotFound(instruction.err)
	case handleDoguNotFound:
		r.recorder.Event(instruction.restart, v1.EventTypeWarning, "DoguNotFound", "Dogu to restart was not found.")

		_, statusErr := r.clientSet.DoguRestarts(instruction.req.Namespace).UpdateStatusWithRetry(ctx, instruction.restart, func(status k8sv1.DoguRestartStatus) k8sv1.DoguRestartStatus {
			status.Phase = k8sv1.RestartStatusPhaseDoguNotFound
			return status
		}, metav1.UpdateOptions{})
		if statusErr != nil {
			logger.Error(statusErr, fmt.Sprintf("failed to update status of dogu restart %q", instruction.req.NamespacedName))
		}

		// cannot restart, no retry necessary
		return ctrl.Result{}, nil
	case handleGetDoguFailed:
		r.recorder.Event(instruction.restart, v1.EventTypeWarning, "FailedGetDogu", "Could not get ressource of dogu to restart.")

		_, statusErr := r.clientSet.DoguRestarts(instruction.req.Namespace).UpdateStatusWithRetry(ctx, instruction.restart, func(status k8sv1.DoguRestartStatus) k8sv1.DoguRestartStatus {
			status.Phase = k8sv1.RestartStatusPhaseFailedGetDogu
			return status
		}, metav1.UpdateOptions{})

		// retry
		return ctrl.Result{}, errors.Join(instruction.err, statusErr)
	default:
		logger.Info(fmt.Sprintf("unknown restart operation %q, ignoring", instruction.op))
		return ctrl.Result{}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *DoguRestartReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.DoguRestart{}).
		Complete(r)
}
