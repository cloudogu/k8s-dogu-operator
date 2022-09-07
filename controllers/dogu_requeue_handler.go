package controllers

import (
	"context"
	"errors"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// statusReporter is responsible to save information in the dogu status via messages.
type statusReporter interface {
	ReportMessage(ctx context.Context, doguResource *k8sv1.Dogu, message string) error
	ReportError(ctx context.Context, doguResource *k8sv1.Dogu, err error) error
}

// requeuableError indicates that the current error requires the operator to requeue the dogu.
type requeuableError interface {
	// Requeue returns true when the error should produce a requeue for the current dogu resource operation.
	Requeue() bool
}

// doguRequeueHandler is responsible to requeue a dogu resource after it failed.
type doguRequeueHandler struct {
	doguStatusReporter statusReporter
	client             client.Client
	// nonCacheClient is required to list all events while filtering them by their fields.
	nonCacheClient kubernetes.Interface
	namespace      string
	recorder       record.EventRecorder
}

// NewDoguRequeueHandler creates a new dogu requeue handler.
func NewDoguRequeueHandler(client client.Client, reporter statusReporter, recorder record.EventRecorder, namespace string) (*doguRequeueHandler, error) {
	clusterConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load cluster configuration: %w", err)
	}

	clientSet, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot create kubernetes client: %w", err)
	}

	return &doguRequeueHandler{
		doguStatusReporter: reporter,
		client:             client,
		nonCacheClient:     clientSet,
		namespace:          namespace,
		recorder:           recorder,
	}, nil
}

// Handle takes an error and handles the requeue process for the current dogu operation.
func (d *doguRequeueHandler) Handle(ctx context.Context, contextMessage string, doguResource *k8sv1.Dogu, err error, onRequeue func(dogu *k8sv1.Dogu)) (ctrl.Result, error) {
	if err != nil {
		reportError := d.doguStatusReporter.ReportError(ctx, doguResource, err)
		if reportError != nil {
			return ctrl.Result{}, fmt.Errorf("failed to report error: %w", reportError)
		}
	}

	return d.handleRequeue(ctx, contextMessage, doguResource, err, onRequeue)
}

func (d *doguRequeueHandler) handleRequeue(ctx context.Context, contextMessage string, doguResource *k8sv1.Dogu, err error, onRequeue func(dogu *k8sv1.Dogu)) (ctrl.Result, error) {
	if shouldRequeue(err) {
		requeueTime := doguResource.Status.NextRequeue()

		if onRequeue != nil {
			onRequeue(doguResource)
		}

		updateError := doguResource.Update(ctx, d.client)
		if updateError != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update dogu status: %w", updateError)
		}

		result := ctrl.Result{RequeueAfter: requeueTime}
		err := d.fireRequeueEvent(ctx, doguResource, result)
		if err != nil {
			return ctrl.Result{}, err
		}
		log.FromContext(ctx).Error(err, fmt.Sprintf("%s: requeue in %s seconds", contextMessage, requeueTime))
		return result, nil
	}

	return ctrl.Result{}, err
}

func shouldRequeue(err error) bool {
	if err == nil {
		return false
	}

	errorList := resource.GetAllErrorsFromChain(err)
	for _, err := range errorList {
		var requeueableError requeuableError
		if errors.As(err, &requeueableError) {
			if requeueableError.Requeue() {
				return true
			}
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

	log.FromContext(ctx).Info(fmt.Sprintf("%+v", doguEvents.Items))

	for _, event := range doguEvents.Items {
		err = d.nonCacheClient.CoreV1().Events(d.namespace).Delete(ctx, event.Name, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete old requeue event: %w", err)
		}
	}

	d.recorder.Eventf(doguResource, v1.EventTypeNormal, RequeueEventReason, "Trying again in %s.", result.RequeueAfter.String())
	return nil
}
