package annotation

import (
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestCesExposedPortAnnotator_AnnotateService(t *testing.T) {
	tests := []struct {
		name                      string
		exposedPorts              []core.ExposedPort
		expectedAnnotation        string
		updatedExposedPorts       []core.ExposedPort
		updatedExpectedAnnotation string
	}{
		{
			name:               "No exposedPorts existing",
			exposedPorts:       []core.ExposedPort{},
			expectedAnnotation: "",
		},
		{
			name: "Successfully added annotations",
			exposedPorts: []core.ExposedPort{
				{
					Type:      "tcp",
					Host:      2222,
					Container: 2222,
				},
				{
					Type:      "udp",
					Host:      80,
					Container: 8080,
				},
			},
			expectedAnnotation: "[{\"protocol\":\"tcp\",\"port\":2222,\"targetPort\":2222},{\"protocol\":\"udp\",\"port\":8080,\"targetPort\":80}]",
		},
		{
			name: "Successfully remove unnecessary annotations",
			exposedPorts: []core.ExposedPort{
				{
					Type:      "tcp",
					Host:      2222,
					Container: 2222,
				},
				{
					Type:      "udp",
					Host:      80,
					Container: 8080,
				},
			},
			expectedAnnotation: "[{\"protocol\":\"tcp\",\"port\":2222,\"targetPort\":2222},{\"protocol\":\"udp\",\"port\":8080,\"targetPort\":80}]",
			updatedExposedPorts: []core.ExposedPort{
				{
					Type:      "tcp",
					Host:      2222,
					Container: 2222,
				},
			},
			updatedExpectedAnnotation: "[{\"protocol\":\"tcp\",\"port\":2222,\"targetPort\":2222}]",
		},
		{
			name: "Successfully remove all unnecessary annotations",
			exposedPorts: []core.ExposedPort{
				{
					Type:      "tcp",
					Host:      2222,
					Container: 2222,
				},
				{
					Type:      "udp",
					Host:      80,
					Container: 8080,
				},
				{
					Type:      "tcp",
					Host:      22,
					Container: 22,
				},
				{
					Type:      "tcp",
					Host:      443,
					Container: 443,
				},
			},
			expectedAnnotation:        "[{\"protocol\":\"tcp\",\"port\":2222,\"targetPort\":2222},{\"protocol\":\"udp\",\"port\":8080,\"targetPort\":80},{\"protocol\":\"tcp\",\"port\":22,\"targetPort\":22},{\"protocol\":\"tcp\",\"port\":443,\"targetPort\":443}]",
			updatedExposedPorts:       []core.ExposedPort{},
			updatedExpectedAnnotation: "",
		},
		{
			name: "Successfully update annotations",
			exposedPorts: []core.ExposedPort{
				{
					Type:      "tcp",
					Host:      2222,
					Container: 2222,
				},
				{
					Type:      "udp",
					Host:      80,
					Container: 8080,
				},
			},
			expectedAnnotation: "[{\"protocol\":\"tcp\",\"port\":2222,\"targetPort\":2222},{\"protocol\":\"udp\",\"port\":8080,\"targetPort\":80}]",
			updatedExposedPorts: []core.ExposedPort{
				{
					Type:      "tcp",
					Host:      2222,
					Container: 2222,
				},
				{
					Type:      "udp",
					Host:      80,
					Container: 8080,
				},
				{
					Type:      "tcp",
					Host:      22,
					Container: 22,
				},
				{
					Type:      "tcp",
					Host:      443,
					Container: 443,
				},
			},
			updatedExpectedAnnotation: "[{\"protocol\":\"tcp\",\"port\":2222,\"targetPort\":2222},{\"protocol\":\"udp\",\"port\":8080,\"targetPort\":80},{\"protocol\":\"tcp\",\"port\":22,\"targetPort\":22},{\"protocol\":\"tcp\",\"port\":443,\"targetPort\":443}]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CesExposedPortAnnotator{}
			service := &corev1.Service{}
			dogu := &core.Dogu{ExposedPorts: tt.exposedPorts}

			err := c.AnnotateService(service, dogu)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedAnnotation, service.Annotations[CesExposedPortAnnotation])

			if tt.updatedExposedPorts != nil {
				dogu.ExposedPorts = tt.updatedExposedPorts
				err := c.AnnotateService(service, dogu)
				require.NoError(t, err)
				assert.Equal(t, tt.updatedExpectedAnnotation, service.Annotations[CesExposedPortAnnotation])
			}
		})
	}
}
