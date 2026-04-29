package exposition

import (
	"fmt"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccess"
	expv1 "github.com/cloudogu/k8s-exposition-lib/api/v1"
)

func buildTCPEntries(serviceName string, exposedPorts []serviceaccess.ExposedPort) []expv1.TCPEntry {
	entries := make([]expv1.TCPEntry, 0, len(exposedPorts))

	for _, exposedPort := range exposedPorts {
		entries = append(entries, expv1.TCPEntry{
			Name:                  buildExposedPortEntryName(exposedPort),
			Service:               serviceName,
			Port:                  int32(exposedPort.Port),
			RequestedExternalPort: new(int32(exposedPort.TargetPort)),
		})
	}

	return entries
}

func buildUDPEntries(serviceName string, exposedPorts []serviceaccess.ExposedPort) []expv1.UDPEntry {
	entries := make([]expv1.UDPEntry, 0, len(exposedPorts))

	for _, exposedPort := range exposedPorts {
		entries = append(entries, expv1.UDPEntry{
			Name:                  buildExposedPortEntryName(exposedPort),
			Service:               serviceName,
			Port:                  int32(exposedPort.Port),
			RequestedExternalPort: new(int32(exposedPort.TargetPort)),
		})
	}

	return entries
}

func buildExposedPortEntryName(exposedPort serviceaccess.ExposedPort) string {
	return fmt.Sprintf("port-%d-%d", exposedPort.Port, exposedPort.TargetPort)
}
