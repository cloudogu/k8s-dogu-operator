package postinstall

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/annotation"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	v1 "k8s.io/api/core/v1"
)

type AdditionalIngressAnnotationsStep struct {
	client k8sClient
}

func NewAdditionalIngressAnnotationsStep(client k8sClient) *AdditionalIngressAnnotationsStep {
	return &AdditionalIngressAnnotationsStep{
		client: client,
	}
}

func (aias *AdditionalIngressAnnotationsStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	ingressAnnotationsChanged, err := aias.checkForAdditionalIngressAnnotations(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if ingressAnnotationsChanged {
		err = aias.setDoguAdditionalIngressAnnotations(ctx, doguResource)
		if err != nil {
			return steps.RequeueWithError(err)
		}
	}

	return steps.Continue()
}

func (aias *AdditionalIngressAnnotationsStep) checkForAdditionalIngressAnnotations(ctx context.Context, doguResource *v2.Dogu) (bool, error) {
	doguService := &v1.Service{}
	err := aias.client.Get(ctx, doguResource.GetObjectKey(), doguService)
	if err != nil {
		return false, fmt.Errorf("failed to get service of dogu [%s]: %w", doguResource.Name, err)
	}

	annotationsJson, exists := doguService.Annotations[annotation.AdditionalIngressAnnotationsAnnotation]
	annotations := v2.IngressAnnotations(nil)
	if exists {
		err = json.Unmarshal([]byte(annotationsJson), &annotations)
		if err != nil {
			return false, fmt.Errorf("failed to get additional ingress annotations from service of dogu [%s]: %w", doguResource.Name, err)
		}
	}

	if reflect.DeepEqual(annotations, doguResource.Spec.AdditionalIngressAnnotations) {
		return false, nil
	} else {
		return true, nil
	}
}

// setDoguAdditionalIngressAnnotations reads the additional ingress annotations from the dogu resource and appends them to the dogu service.
// These annotations are then to be read by the service discovery and appended to the ingress object for the dogu.
func (aias *AdditionalIngressAnnotationsStep) setDoguAdditionalIngressAnnotations(ctx context.Context, doguResource *v2.Dogu) error {
	doguService := &v1.Service{}
	err := aias.client.Get(ctx, doguResource.GetObjectKey(), doguService)
	if err != nil {
		return fmt.Errorf("failed to fetch service for dogu '%s': %w", doguResource.Name, err)
	}

	annotator := &annotation.IngressAnnotator{}
	err = annotator.AppendIngressAnnotationsToService(doguService, doguResource.Spec.AdditionalIngressAnnotations)
	if err != nil {
		return fmt.Errorf("failed to add additional ingress annotations to service of dogu '%s': %w", doguResource.Name, err)
	}

	err = aias.client.Update(ctx, doguService)
	if err != nil {
		return fmt.Errorf("failed to update dogu service '%s' with ingress annotations: %w", doguService.Name, err)
	}

	return nil
}
