package controllers

import (
	"context"
	"errors"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
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

// DoguRequeueHandler is responsible to requeue a dogu resource after it failed.
type DoguRequeueHandler struct {
	DoguStatusReporter statusReporter `json:"dogu_status_reporter"`
	KubernetesClient   client.Client  `json:"kubernetes_client"`
}

// NewDoguRequeueHandler creates a new dogu requeue handler.
func NewDoguRequeueHandler(client client.Client, reporter statusReporter) *DoguRequeueHandler {
	return &DoguRequeueHandler{
		DoguStatusReporter: reporter,
		KubernetesClient:   client,
	}
}

// Handle takes an error and handles the requeue process for the current dogu operation.
func (d *DoguRequeueHandler) Handle(ctx context.Context, contextMessage string, doguResource *k8sv1.Dogu, err error) (ctrl.Result, error) {
	if err != nil {
		reportError := d.DoguStatusReporter.ReportError(ctx, doguResource, err)
		if reportError != nil {
			return ctrl.Result{}, fmt.Errorf("failed to report error: %w", reportError)
		}
	}

	return d.handleRequeue(ctx, contextMessage, doguResource, err)
}

func (d *DoguRequeueHandler) handleRequeue(ctx context.Context, contextMessage string, doguResource *k8sv1.Dogu, err error) (ctrl.Result, error) {
	if shouldRequeue(err) {
		requeueTime := doguResource.Status.NextRequeue()
		updateError := doguResource.Update(ctx, d.KubernetesClient)
		if updateError != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update dogu status: %w", updateError)
		}

		log.FromContext(ctx).Error(err, fmt.Sprintf("%s: requeue in %s seconds", contextMessage, requeueTime))
		return ctrl.Result{RequeueAfter: requeueTime}, nil
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
