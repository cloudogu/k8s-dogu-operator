package controllers

import (
	"context"
	"fmt"
	"time"

	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"sigs.k8s.io/controller-runtime/pkg/log"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	ReasonReconcileStarted = "ReconcileStarted"
	ReasonReconcileOK      = "ReconcileSucceeded"
)

const requeueTime = 5 * time.Second

// requeuableError indicates that the current error requires the operator to requeue the dogu.
type requeuableError interface {
	// Requeue returns true when the error should produce a requeue for the current dogu resource operation.
	Requeue() bool
}

// requeuableError indicates that the current error requires the operator to requeue the dogu.
type requeuableErrorWithTime interface {
	requeuableError
	// GetRequeueTime return the time to wait before the next reconciliation. The constant ExponentialRequeueTime indicates
	// that the requeue time increased exponentially.
	GetRequeueTime() time.Duration
}

// requeuableErrorWithState indicates that the current error requires the operator to requeue the dogu and set the state
// in dogu status.
type requeuableErrorWithState interface {
	requeuableErrorWithTime
	// GetState returns the current state of the reconciled resource.
	// In most cases it can be empty if no async state mechanism is used.
	GetState() string
}

// doguRequeueHandler is responsible to requeue a dogu resource after it failed.
type doguRequeueHandler struct {
	// nonCacheClient is required to list all events while filtering them by their fields.
	nonCacheClient kubernetes.Interface
	namespace      string
	recorder       record.EventRecorder
	doguInterface  doguClient.DoguInterface
}

// NewDoguRequeueHandler creates a new dogu requeue handler.
func NewDoguRequeueHandler(doguInterface doguClient.DoguInterface, recorder record.EventRecorder, operatorConfig *config.OperatorConfig) (*doguRequeueHandler, error) {
	clusterConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load cluster configuration: %w", err)
	}

	clientSet, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot create kubernetes client: %w", err)
	}

	return &doguRequeueHandler{
		doguInterface:  doguInterface,
		nonCacheClient: clientSet,
		namespace:      operatorConfig.Namespace,
		recorder:       recorder,
	}, nil
}

// Handle takes an error and handles the requeue process for the current dogu operation.
func (d *doguRequeueHandler) Handle(ctx context.Context, doguResource *doguv2.Dogu, err error, reqTime time.Duration) (ctrl.Result, error) {
	var timeErr error
	logger := log.FromContext(ctx)
	result := ctrl.Result{Requeue: true, RequeueAfter: reqTime}
	if err == nil && reqTime == 0 {
		doguResource.Status.RequeueTime = reqTime
		newDoguResource, timeErr := d.doguInterface.UpdateStatus(ctx, doguResource, metav1.UpdateOptions{})
		if timeErr != nil {
			result.RequeueAfter = requeueTime
			d.fireRequeueEvent(doguResource, result, err)
			logger.Error(err, fmt.Sprintf("failed to set requeue time: %w", timeErr))
			return result, nil
		}
		d.recorder.Event(newDoguResource, v1.EventTypeNormal, ReasonReconcileOK, "resource synced")
		return ctrl.Result{}, nil
	}

	if reqTime == 0 {
		reqTime = requeueTime
	}

	doguResource.Status.RequeueTime = reqTime
	newDoguResource, timeErr := d.doguInterface.UpdateStatus(ctx, doguResource, metav1.UpdateOptions{})
	if timeErr != nil {
		result.RequeueAfter = requeueTime
		d.fireRequeueEvent(doguResource, result, err)
		logger.Error(err, fmt.Sprintf("failed to set requeue time: %w", timeErr))
	} else {
		d.fireRequeueEvent(newDoguResource, result, err)
	}

	if err != nil {
		logger.Error(err, fmt.Sprintf("requeue in %s seconds because of: %s", reqTime, err.Error()))
	} else {
		logger.Info(fmt.Sprintf("requeue in %s seconds", reqTime))
	}

	return result, nil

}

func (d *doguRequeueHandler) fireRequeueEvent(doguResource *doguv2.Dogu, result ctrl.Result, reconcileError error) {
	if reconcileError != nil {
		d.recorder.Eventf(doguResource, v1.EventTypeWarning, ReasonReconcileFail, "Trying again in %s because of: %s", result.RequeueAfter.String(), reconcileError.Error())
	} else {
		d.recorder.Eventf(doguResource, v1.EventTypeNormal, RequeueEventReason, "Trying again in %s.", result.RequeueAfter.String())
	}
}
