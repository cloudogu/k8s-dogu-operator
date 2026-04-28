package serviceaccess

import "github.com/cloudogu/cesapp-lib/core"

func CollectExposedPorts(dogu *core.Dogu) []ExposedPort {
	exposedPorts := dogu.ExposedPorts
	if len(exposedPorts) < 1 {
		return []ExposedPort{}
	}
	var annotationExposedPorts []ExposedPort

	for _, exposedPort := range exposedPorts {
		annotationExposedPorts = append(annotationExposedPorts, ExposedPort{
			Protocol:   exposedPort.Type,
			Port:       exposedPort.Container,
			TargetPort: exposedPort.Host,
		})
	}

	return annotationExposedPorts
}
