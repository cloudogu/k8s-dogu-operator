package annotation

import (
	"encoding/json"
	"fmt"
	doguv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"

	corev1 "k8s.io/api/core/v1"
)

// AdditionalIngressAnnotationsAnnotation contains additional ingress annotations to be appended to the ingress object for this service.
const AdditionalIngressAnnotationsAnnotation = "k8s-dogu-operator.cloudogu.com/additional-ingress-annotations"

type IngressAnnotator struct{}

func (i IngressAnnotator) AppendIngressAnnotationsToService(service *corev1.Service, additionalIngressAnnotations doguv1.IngressAnnotations) error {
	err := appendAdditionalIngressAnnotations(service, additionalIngressAnnotations)
	if err != nil {
		return fmt.Errorf("failed to append annotation [%s] to service [%s]: %w", AdditionalIngressAnnotationsAnnotation, service.GetName(), err)
	}

	return nil

}

func appendAdditionalIngressAnnotations(service *corev1.Service, ingressAnnotations doguv1.IngressAnnotations) error {
	if len(ingressAnnotations) < 1 {
		return nil
	}

	if service.Annotations == nil {
		service.Annotations = map[string]string{}
	}

	ingressAnnotationsJson, err := json.Marshal(ingressAnnotations)
	if err != nil {
		return fmt.Errorf("failed to marshal additional ingress annotations: %w", err)
	}

	service.Annotations[AdditionalIngressAnnotationsAnnotation] = string(ingressAnnotationsJson)
	return nil
}
