package annotation

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestIngressAnnotator_AppendIngressAnnotationsToService(t *testing.T) {
	t.Run("Should create additional ingress annotations as json object on service", func(t *testing.T) {
		// given
		service := &corev1.Service{}
		additionalIngressAnnotations := map[string]string{
			"nginx.org/client-max-body-size": "100m",
			"example-annotation":             "example-value",
		}
		annotator := IngressAnnotator{}

		// when
		err := annotator.AppendIngressAnnotationsToService(service, additionalIngressAnnotations)

		// then
		require.NoError(t, err)
		assert.Equal(t, "{\"example-annotation\":\"example-value\",\"nginx.org/client-max-body-size\":\"100m\"}", service.Annotations[AdditionalIngressAnnotationsAnnotation])
	})
	t.Run("Should append additional ingress annotations as json object to service", func(t *testing.T) {
		// given
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"existing": "value",
				},
			},
		}
		additionalIngressAnnotations := map[string]string{
			"nginx.org/client-max-body-size": "100m",
			"example-annotation":             "example-value",
		}
		annotator := IngressAnnotator{}

		// when
		err := annotator.AppendIngressAnnotationsToService(service, additionalIngressAnnotations)

		// then
		require.NoError(t, err)
		assert.Equal(t, "value", service.Annotations["existing"])
		assert.Equal(t, "{\"example-annotation\":\"example-value\",\"nginx.org/client-max-body-size\":\"100m\"}", service.Annotations[AdditionalIngressAnnotationsAnnotation])
	})

	t.Run("Should delete annotations from service and succeed if the length of annotation is < 1", func(t *testing.T) {
		// given
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"existing":                             "value",
					AdditionalIngressAnnotationsAnnotation: "bla bla bla",
				},
			},
		}
		additionalIngressAnnotations := map[string]string{}

		// when
		err := IngressAnnotator{}.AppendIngressAnnotationsToService(service, additionalIngressAnnotations)

		// then
		require.Nil(t, err)
		assert.Contains(t, service.Annotations, "existing")
		assert.NotContains(t, service.Annotations, AdditionalIngressAnnotationsAnnotation)
	})
}
