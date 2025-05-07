package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/annotation"
)

const (
	// AdditionalIngressAnnotationsChangeEventReason is the reason string for firing additional ingress annotations change events.
	AdditionalIngressAnnotationsChangeEventReason = "AdditionalIngressAnnotationsChange"
	// ErrorOnAdditionalIngressAnnotationsChangeEventReason is the error string for firing additional ingress annotations change error events.
	ErrorOnAdditionalIngressAnnotationsChangeEventReason = "ErrAdditionalIngressAnnotationsChange"
)

// NewDoguAdditionalIngressAnnotationsManager creates a new instance of a manager to append ingress annotations to a dogu service.
func NewDoguAdditionalIngressAnnotationsManager(client client.Client, eventRecorder record.EventRecorder) *doguAdditionalIngressAnnotationsManager {
	return &doguAdditionalIngressAnnotationsManager{client: client, eventRecorder: eventRecorder}
}

type doguAdditionalIngressAnnotationsManager struct {
	client        client.Client
	eventRecorder record.EventRecorder
}

// SetDoguAdditionalIngressAnnotations reads the additional ingress annotations from the dogu resource and appends them to the dogu service.
// These annotations are then to be read by the service discovery and appended to the ingress object for the dogu.
func (d *doguAdditionalIngressAnnotationsManager) SetDoguAdditionalIngressAnnotations(ctx context.Context, doguResource *k8sv2.Dogu) error {
	doguService := &corev1.Service{}
	err := d.client.Get(ctx, doguResource.GetObjectKey(), doguService)
	if err != nil {
		return fmt.Errorf("failed to fetch service for dogu '%s': %w", doguResource.Name, err)
	}

	annotator := &annotation.IngressAnnotator{}
	err = annotator.AppendIngressAnnotationsToService(doguService, doguResource.Spec.AdditionalIngressAnnotations)
	if err != nil {
		return fmt.Errorf("failed to add additional ingress annotations to service of dogu '%s': %w", doguResource.Name, err)
	}

	err = d.client.Update(ctx, doguService)
	if err != nil {
		return fmt.Errorf("failed to update dogu service '%s' with ingress annotations: %w", doguService.Name, err)
	}

	return nil
}
