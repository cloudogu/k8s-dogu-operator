package annotation

import (
	"encoding/json"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	corev1 "k8s.io/api/core/v1"
)

type CesExposedPort struct {
	Protocol   string `json:"protocol"`
	Port       int    `json:"port"`
	TargetPort int    `json:"targetPort"`
}

const CesExposedPortAnnotation = "k8s-dogu-operator.cloudogu.com/ces-exposed-ports"

// CesExposedPortAnnotator adds information about the exposedPorts of a dogu to dogu services
type CesExposedPortAnnotator struct{}

// AnnotateService annotates a given service with exposed ports.
// Can bes used to identify services with exposed ports and get information about these ports
// Services are annotated like this:
// k8s-dogu-operator.cloudogu.com/ces-exposed-ports = [{"protocol":"tcp","port":2222,"targetPort":2222},{"protocol":"udp","port":8080,"targetPort":80}]
func (c *CesExposedPortAnnotator) AnnotateService(service *corev1.Service, dogu *core.Dogu) error {
	exposedPorts := parseExposedPorts(dogu.ExposedPorts)

	err := appendAnnotations(service, exposedPorts)
	if err != nil {
		return fmt.Errorf("failed to append annotation [%s] to service [%s]: %w", CesExposedPortAnnotation, service.GetName(), err)
	}

	return nil
}

func parseExposedPorts(exposedPorts []core.ExposedPort) []CesExposedPort {
	if len(exposedPorts) < 1 {
		return []CesExposedPort{}
	}
	var annotationExposedPorts []CesExposedPort

	for _, exposedPort := range exposedPorts {
		annotationExposedPorts = append(annotationExposedPorts, CesExposedPort{
			Protocol:   exposedPort.Type,
			Port:       exposedPort.Container,
			TargetPort: exposedPort.Host,
		})
	}
	return annotationExposedPorts
}

func appendAnnotations(service *corev1.Service, exposedPorts []CesExposedPort) error {
	if len(exposedPorts) < 1 {
		return nil
	}

	if service.Annotations == nil {
		service.Annotations = map[string]string{}
	}

	exposedPortsJson, err := json.Marshal(&exposedPorts)
	if err != nil {
		return fmt.Errorf("failed to marshal exposed ports: %w", err)
	}

	service.Annotations[CesExposedPortAnnotation] = string(exposedPortsJson)
	return nil
}
