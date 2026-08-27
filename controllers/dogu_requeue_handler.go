package controllers

import (
	"context"
	"reflect"
	"time"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClientV2 "github.com/cloudogu/k8s-dogu-lib/v2/client/typed/api/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	ReasonReconcileStarted = "ReconcileStarted"
	ReasonReconcileOK      = "ReconcileSucceeded"
)

// doguRequeueHandler is responsible to requeue a dogu resource after it failed.
type doguRequeueHandler struct {
	namespace     string
	recorder      record.EventRecorder
	doguInterface doguClientV2.DoguInterface
	requeueTime   time.Duration
}

// NewDoguRequeueHandler creates a new dogu requeue handler.
func NewDoguRequeueHandler(doguInterface doguClientV2.DoguInterface, recorder record.EventRecorder, operatorConfig *config.OperatorConfig) *doguRequeueHandler {
	return &doguRequeueHandler{
		doguInterface: doguInterface,
		namespace:     operatorConfig.Namespace,
		recorder:      recorder,
		requeueTime:   operatorConfig.RequeueTimeForDoguReconciler,
	}
}

func (d *doguRequeueHandler) Handle(ctx context.Context, doguResource *doguv2.Dogu, reconcileError error, reqTime time.Duration) (ctrl.Result, error) {
	result := d.handleRequeue(doguResource, reconcileError, reqTime)
	d.handleRequeueEvent(doguResource, reconcileError, result.RequeueAfter)
	d.handleRequeueTime(ctx, doguResource, &result)
	return result, nil
}

func (d *doguRequeueHandler) handleRequeueTime(ctx context.Context, doguResource *doguv2.Dogu, result *ctrl.Result) {
	logger := log.FromContext(ctx)

	// A Get on an empty dogu would lead to "resource name may not be empty". Prevent by skipping.
	emptyDogu := &doguv2.Dogu{}
	if reflect.DeepEqual(doguResource, emptyDogu) || !doguResource.DeletionTimestamp.IsZero() {
		return
	}

	var err error
	doguResource, err = d.doguInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return
		}
		result.RequeueAfter = d.requeueTime
		logger.Error(err, "failed to get doguResource for setting requeue time")
		return
	}
	// check deletionTimestamp again to catch the race condition that the dogu was deleted in the meantime
	if !doguResource.DeletionTimestamp.IsZero() {
		return
	}

	updatedDoguResource, err := d.doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status doguv2.DoguStatus) doguv2.DoguStatus {
		status.RequeueTime = result.RequeueAfter
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		result.RequeueAfter = d.requeueTime
		logger.Error(err, "failed to set requeue time")
		return
	}
	*doguResource = *updatedDoguResource
}

func (d *doguRequeueHandler) handleRequeueEvent(doguResource *doguv2.Dogu, reconcileError error, reqTime time.Duration) {
	emptyDogu := &doguv2.Dogu{}
	if reflect.DeepEqual(doguResource, emptyDogu) {
		return
	}
	if reconcileError == nil && reqTime == 0 {
		d.recorder.Event(doguResource, v1.EventTypeNormal, ReasonReconcileOK, "resource synced")
	} else if reconcileError != nil {
		d.recorder.Eventf(doguResource, v1.EventTypeWarning, ReasonReconcileFail, "Trying again in %s because of: %s", reqTime.String(), reconcileError.Error())
	} else {
		d.recorder.Eventf(doguResource, v1.EventTypeNormal, RequeueEventReason, "Trying again in %s.", reqTime.String())
	}
}

func (d *doguRequeueHandler) handleRequeue(doguResource *doguv2.Dogu, reconcileError error, reqTime time.Duration) ctrl.Result {
	result := ctrl.Result{}
	emptyDogu := &doguv2.Dogu{}
	if reflect.DeepEqual(doguResource, emptyDogu) {
		return result
	}

	if reconcileError != nil {
		result.RequeueAfter = d.requeueTime
		return result
	}

	if reqTime > 0 {
		result.RequeueAfter = reqTime
	}

	return result
}
