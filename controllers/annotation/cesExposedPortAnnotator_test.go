package annotation

import (
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestCesExposedPortAnnotator_AnnotateService(t *testing.T) {
	t.Run("No exposedPorts existing", func(t *testing.T) {
		// given
		service := &corev1.Service{}
		dogu := &core.Dogu{}

		annotator := CesExposedPortAnnotator{}

		// when
		err := annotator.AnnotateService(service, dogu)

		// then
		require.NoError(t, err)
		assert.Equal(t, "", service.Annotations[CesExposedPortAnnotation])
	})

	t.Run("Successfully added annotations", func(t *testing.T) {
		// given
		service := &corev1.Service{}
		exposedPorts := &[]core.ExposedPort{
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
		}
		dogu := &core.Dogu{
			ExposedPorts: *exposedPorts,
		}

		annotator := CesExposedPortAnnotator{}

		// when
		err := annotator.AnnotateService(service, dogu)

		// then
		require.NoError(t, err)
		assert.Equal(t, "[{\"protocol\":\"tcp\",\"port\":2222,\"targetPort\":2222},{\"protocol\":\"udp\",\"port\":8080,\"targetPort\":80}]", service.Annotations[CesExposedPortAnnotation])
	})
}
