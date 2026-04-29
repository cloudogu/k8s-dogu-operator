package serviceaccess

import (
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/stretchr/testify/assert"
)

func TestCollectExposedPorts(t *testing.T) {
	t.Run("should return empty slice when dogu has no exposed ports", func(t *testing.T) {
		dogu := &core.Dogu{}

		exposedPorts := CollectExposedPorts(dogu)

		assert.Empty(t, exposedPorts)
	})

	t.Run("should map exposed ports from dogu", func(t *testing.T) {
		dogu := &core.Dogu{
			ExposedPorts: []core.ExposedPort{
				{
					Type:      "tcp",
					Container: 2222,
					Host:      32222,
				},
				{
					Type:      "udp",
					Container: 8080,
					Host:      30080,
				},
			},
		}

		exposedPorts := CollectExposedPorts(dogu)

		assert.Equal(t, []ExposedPort{
			{
				Protocol:   "tcp",
				Port:       2222,
				TargetPort: 32222,
			},
			{
				Protocol:   "udp",
				Port:       8080,
				TargetPort: 30080,
			},
		}, exposedPorts)
	})
}
