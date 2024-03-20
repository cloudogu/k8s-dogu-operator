package controllers

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	"time"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

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
	doguInterface  ecoSystem.DoguInterface
}

// NewDoguRequeueHandler creates a new dogu requeue handler.
func NewDoguRequeueHandler(doguInterface ecoSystem.DoguInterface, recorder record.EventRecorder, namespace string) (*doguRequeueHandler, error) {
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
		namespace:      namespace,
		recorder:       recorder,
	}, nil
}

// Handle takes an error and handles the requeue process for the current dogu operation.
func (d *doguRequeueHandler) Handle(ctx context.Context, contextMessage string, doguResource *k8sv1.Dogu, originalErr error, onRequeue func(dogu *k8sv1.Dogu) error) (ctrl.Result, error) {
	if !shouldRequeue(originalErr) {
		return ctrl.Result{}, nil
	}

	requeueTime, timeErr := getRequeueTime(ctx, doguResource, d.doguInterface, originalErr)
	if timeErr != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get requeue time: %w", timeErr)
	}
	if onRequeue != nil {
		onRequeueErr := onRequeue(doguResource)
		if onRequeueErr != nil {
			return ctrl.Result{}, fmt.Errorf("failed to call onRequeue handler: %w", onRequeueErr)
		}
	}

	_, updateError := d.doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status k8sv1.DoguStatus) k8sv1.DoguStatus {
		status.RequeuePhase = getRequeuePhase(originalErr)
		return status
	}, metav1.UpdateOptions{})
	if updateError != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update dogu status: %w", updateError)
	}

	result := ctrl.Result{Requeue: true, RequeueAfter: requeueTime}
	err := d.fireRequeueEvent(ctx, doguResource, result)
	if err != nil {
		return ctrl.Result{}, err
	}

	log.FromContext(ctx).Error(err, fmt.Sprintf("%s: requeue in %s seconds because of: %s", contextMessage, requeueTime, originalErr.Error()))

	return result, nil

}

func getRequeuePhase(err error) string {
	var errorWithState requeuableErrorWithState
	if errors.As(err, &errorWithState) {
		return errorWithState.GetState()
	}

	return ""
}

func getRequeueTime(ctx context.Context, dogu *k8sv1.Dogu, doguInterface ecoSystem.DoguInterface, err error) (time.Duration, error) {
	var errorWithTime requeuableErrorWithTime
	if errors.As(err, &errorWithTime) {
		return errorWithTime.GetRequeueTime(), nil
	}

	var requeueTime time.Duration
	_, timeErr := doguInterface.UpdateStatusWithRetry(ctx, dogu, func(status k8sv1.DoguStatus) k8sv1.DoguStatus {
		requeueTime = dogu.Status.NextRequeue()
		status.RequeueTime = requeueTime
		return status
	}, metav1.UpdateOptions{})
	if timeErr != nil {
		return 0, timeErr
	}

	return requeueTime, nil
}

func shouldRequeue(err error) bool {
	if err == nil {
		return false
	}

	var requeueableError requeuableError
	if errors.As(err, &requeueableError) {
		if requeueableError.Requeue() {

			return true
		}
	}

	return false
}

func (d *doguRequeueHandler) fireRequeueEvent(ctx context.Context, doguResource *k8sv1.Dogu, result ctrl.Result) error {
	doguEvents, err := d.nonCacheClient.CoreV1().Events(d.namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("reason=%s,involvedObject.name=%s", RequeueEventReason, doguResource.Name),
	})
	if err != nil {
		return fmt.Errorf("failed to get all requeue errors: %w", err)
	}

	for _, event := range doguEvents.Items {
		err = d.nonCacheClient.CoreV1().Events(d.namespace).Delete(ctx, event.Name, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete old requeue event: %w", err)
		}
	}

	d.recorder.Eventf(doguResource, v1.EventTypeNormal, RequeueEventReason, "Trying again in %s.", result.RequeueAfter.String())
	return nil
}
