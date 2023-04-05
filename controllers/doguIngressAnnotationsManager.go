package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/annotation"
)

const (
	// AdditionalIngressAnnotationsChangeEventReason is the reason string for firing additional ingress annotations change events.
	AdditionalIngressAnnotationsChangeEventReason = "AdditionalIngressAnnotationsChange"
	// ErrorOnAdditionalIngressAnnotationsChangeEventReason is the error string for firing additional ingress annotations change error events.
	ErrorOnAdditionalIngressAnnotationsChangeEventReason = "ErrAdditionalIngressAnnotationsChange"
)

func NewDoguAdditionalIngressAnnotationsManager(client client.Client, eventRecorder record.EventRecorder) *doguAdditionalIngressAnnotationsManager {
	return &doguAdditionalIngressAnnotationsManager{client: client, eventRecorder: eventRecorder}
}

type doguAdditionalIngressAnnotationsManager struct {
	client        client.Client
	eventRecorder record.EventRecorder
}

func (d *doguAdditionalIngressAnnotationsManager) SetDoguAdditionalIngressAnnotations(ctx context.Context, doguResource *k8sv1.Dogu) error {
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

	return nil
}
