package resource

import (
	"context"
	"errors"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/hashicorp/go-multierror"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ReportableError is used to identify all errors that are designed to report something into the dogu resource status.
type ReportableError interface {
	// Report constructs a human readable message for the dogu resource status.
	Report() string
}

// DoguStatusReporter is responsible to add messages to a dogu resource.
type DoguStatusReporter struct {
	KubernetesClient client.Client `json:"kubernetes_client"`
}

// NewDoguStatusReporter create a new instance of a dogu error reporter.
func NewDoguStatusReporter(client client.Client) *DoguStatusReporter {
	return &DoguStatusReporter{KubernetesClient: client}
}

// ReportMessage adds the given message to the status of the dogu resource.
func (der DoguStatusReporter) ReportMessage(ctx context.Context, doguResource *k8sv1.Dogu, message string) error {
	doguResource.Status.AddMessage(message)
	return der.updateDoguState(ctx, doguResource)
}

// ReportError adds the or all errors from a multi error to the status of the dogu resource.
func (der DoguStatusReporter) ReportError(ctx context.Context, doguResource *k8sv1.Dogu, reportError error) error {
	errorList := GetAllErrorsFromChain(reportError)

	for _, err := range errorList {
		var reportableError ReportableError
		if errors.As(err, &reportableError) {
			doguResource.Status.AddMessage(reportableError.Report())
		}
	}

	return der.updateDoguState(ctx, doguResource)
}

func (der DoguStatusReporter) updateDoguState(ctx context.Context, doguResource *k8sv1.Dogu) error {
	err := der.KubernetesClient.Status().Update(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	return nil
}

func GetAllErrorsFromChain(err error) []error {
	multiError, ok := err.(*multierror.Error)
	if !ok {
		return []error{err}
	}

	return multiError.Errors
}
