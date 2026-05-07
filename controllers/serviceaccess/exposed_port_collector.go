package serviceaccess

import (
	"strings"

	"github.com/cloudogu/cesapp-lib/core"
)

func CollectExposedPorts(dogu *core.Dogu) []ExposedPort {
	exposedPorts := dogu.ExposedPorts
	if len(exposedPorts) < 1 {
		return []ExposedPort{}
	}
	var annotationExposedPorts []ExposedPort

	for _, exposedPort := range exposedPorts {
		annotationExposedPorts = append(annotationExposedPorts, ExposedPort{
			Protocol:   strings.ToLower(exposedPort.GetType()),
			Port:       exposedPort.Container,
			TargetPort: exposedPort.Host,
		})
	}

	return annotationExposedPorts
}
