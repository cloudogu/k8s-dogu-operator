package exposition

import (
	"testing"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCollectRoutes(t *testing.T) {
	t.Run("should collect webapp and additional routes", func(t *testing.T) {
		service := newTestService("cas", 80)
		config := &imagev1.Config{
			Labels: map[string]string{
				"SERVICE_TAGS": "webapp",
				"SERVICE_NAME": "admin",
			},
			Env: []string{
				"SERVICE_ADDITIONAL_SERVICES=[{\"name\":\"admin-api\",\"location\":\"api\",\"pass\":\"admin/api/v2/\"}]",
			},
			ExposedPorts: map[string]struct{}{
				"80/tcp": {},
			},
		}

		routes, err := CollectRoutes(service, config)

		require.NoError(t, err)
		assert.Equal(t, []Route{
			{
				Name:       "admin",
				Port:       80,
				Path:       "/admin",
				TargetPath: "/admin",
			},
			{
				Name:       "admin-api",
				Port:       80,
				Path:       "/api",
				TargetPath: "/admin/api/v2/",
			},
		}, routes)
	})

	t.Run("should apply port specific overrides and rewrite", func(t *testing.T) {
		service := newTestService("cas", 8080)
		config := &imagev1.Config{
			Env: []string{
				"SERVICE_8080_TAGS=webapp",
				"SERVICE_8080_NAME=console",
				"SERVICE_8080_LOCATION=ui",
				"SERVICE_8080_PASS=internal/ui/",
				"SERVICE_8080_REWRITE=/",
			},
			ExposedPorts: map[string]struct{}{
				"8080/tcp": {},
				"8081/udp": {},
			},
		}

		routes, err := CollectRoutes(service, config)

		require.NoError(t, err)
		assert.Equal(t, []Route{
			{
				Name:       "console",
				Port:       8080,
				Path:       "/ui",
				TargetPath: "/internal/ui/",
				Rewrite:    "/",
			},
		}, routes)
	})

	t.Run("should use service name as default route values", func(t *testing.T) {
		service := newTestService("cas", 80)
		config := &imagev1.Config{
			Labels: map[string]string{
				"SERVICE_TAGS": "webapp",
			},
			ExposedPorts: map[string]struct{}{
				"80/tcp": {},
			},
		}

		routes, err := CollectRoutes(service, config)

		require.NoError(t, err)
		assert.Equal(t, []Route{
			{
				Name:       "cas",
				Port:       80,
				Path:       "/cas",
				TargetPath: "/cas",
			},
		}, routes)
	})

	t.Run("should keep leading slashes in configured path and target path", func(t *testing.T) {
		service := newTestService("cas", 80)
		config := &imagev1.Config{
			Env: []string{
				"SERVICE_TAGS=webapp",
				"SERVICE_LOCATION=/ui",
				"SERVICE_PASS=/internal/ui",
			},
			ExposedPorts: map[string]struct{}{
				"80/tcp": {},
			},
		}

		routes, err := CollectRoutes(service, config)

		require.NoError(t, err)
		assert.Equal(t, []Route{
			{
				Name:       "cas",
				Port:       80,
				Path:       "/ui",
				TargetPath: "/internal/ui",
			},
		}, routes)
	})

	t.Run("should fail on invalid environment variable", func(t *testing.T) {
		service := newTestService("cas", 80)
		config := &imagev1.Config{
			Env: []string{
				"SERVICE_TAGS-invalidEnvironmentVariable",
			},
		}

		routes, err := CollectRoutes(service, config)

		require.Error(t, err)
		assert.Nil(t, routes)
		assert.ErrorContains(t, err, "environment variable [SERVICE_TAGS-invalidEnvironmentVariable] needs to be in form NAME=VALUE")
	})

	t.Run("should fail on invalid additional services json", func(t *testing.T) {
		service := newTestService("cas", 80)
		config := &imagev1.Config{
			Env: []string{
				"SERVICE_ADDITIONAL_SERVICES='bad json'",
			},
		}

		routes, err := CollectRoutes(service, config)

		require.Error(t, err)
		assert.Nil(t, routes)
		assert.ErrorContains(t, err, "failed to unmarshal additional services")
	})

	t.Run("should fail on invalid exposed port", func(t *testing.T) {
		service := newTestService("cas", 80)
		config := &imagev1.Config{
			Labels: map[string]string{
				"SERVICE_TAGS": "webapp",
			},
			ExposedPorts: map[string]struct{}{
				"tcp": {},
			},
		}

		routes, err := CollectRoutes(service, config)

		require.Error(t, err)
		assert.Nil(t, routes)
		assert.ErrorContains(t, err, "error parsing int")
	})
}

func TestBuildServiceForRoutes(t *testing.T) {
	t.Run("should skip invalid exposed ports", func(t *testing.T) {
		service := buildServiceForRoutes(newDoguResource(), &imagev1.ConfigFile{
			Config: imagev1.Config{
				ExposedPorts: map[string]struct{}{
					"80/tcp": {},
					"tcp":    {},
				},
			},
		})

		require.Len(t, service.Spec.Ports, 1)
		assert.Equal(t, int32(80), service.Spec.Ports[0].Port)
		assert.Equal(t, corev1.ProtocolTCP, service.Spec.Ports[0].Protocol)
	})
}

func newTestService(name string, port int32) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Port: port},
			},
		},
	}
}
