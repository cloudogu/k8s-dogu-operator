package exposition

import (
	"testing"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/serviceaccess"
	expv1 "github.com/cloudogu/k8s-exposition-lib/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestBuildTCPEntries(t *testing.T) {
	exposedPorts := []serviceaccess.ExposedPort{
		{
			Protocol:   "tcp",
			Port:       2222,
			TargetPort: 32222,
		},
		{
			Protocol:   "udp",
			Port:       5353,
			TargetPort: 3053,
		},
		{
			Port:       8443,
			TargetPort: 30443,
		},
	}

	requestedExternalPort := int32(32222)
	defaultRequestedExternalPort := int32(30443)
	assert.Equal(t, []expv1.TCPEntry{
		{
			Name:                  "port-2222-32222",
			Service:               "cas",
			Port:                  2222,
			RequestedExternalPort: &requestedExternalPort,
		},
		{
			Name:                  "port-8443-30443",
			Service:               "cas",
			Port:                  8443,
			RequestedExternalPort: &defaultRequestedExternalPort,
		},
	}, buildTCPEntries("cas", exposedPorts))
}

func TestBuildUDPEntries(t *testing.T) {
	exposedPorts := []serviceaccess.ExposedPort{
		{
			Protocol:   "tcp",
			Port:       2222,
			TargetPort: 32222,
		},
		{
			Protocol:   "UDP",
			Port:       5353,
			TargetPort: 3053,
		},
	}

	requestedExternalPort := int32(3053)
	assert.Equal(t, []expv1.UDPEntry{
		{
			Name:                  "port-5353-3053",
			Service:               "cas",
			Port:                  5353,
			RequestedExternalPort: &requestedExternalPort,
		},
	}, buildUDPEntries("cas", exposedPorts))
}
