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
	}

	requestedExternalPort := int32(32222)
	assert.Equal(t, []expv1.TCPEntry{
		{
			Name:                  "port-2222-32222",
			Service:               "cas",
			Port:                  2222,
			RequestedExternalPort: &requestedExternalPort,
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
	}

	requestedExternalPort := int32(32222)
	assert.Equal(t, []expv1.UDPEntry{
		{
			Name:                  "port-2222-32222",
			Service:               "cas",
			Port:                  2222,
			RequestedExternalPort: &requestedExternalPort,
		},
	}, buildUDPEntries("cas", exposedPorts))
}
