package serviceaccess

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
				Name:     "admin",
				Port:     80,
				Location: "/admin",
				Pass:     "/admin",
			},
			{
				Name:     "admin-api",
				Port:     80,
				Location: "/api",
				Pass:     "/admin/api/v2/",
			},
		}, routes)
	})

	t.Run("should apply port specific overrides", func(t *testing.T) {
		service := newTestService("cas", 8080)
		config := &imagev1.Config{
			Env: []string{
				"SERVICE_8080_TAGS=webapp",
				"SERVICE_8080_NAME=console",
				"SERVICE_8080_LOCATION=ui",
				"SERVICE_8080_PASS=internal/ui/",
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
				Name:     "console",
				Port:     8080,
				Location: "/ui",
				Pass:     "/internal/ui/",
			},
		}, routes)
	})

	t.Run("should prefer port specific webapp tags over global webapp tag", func(t *testing.T) {
		service := newTestService("jenkins", 8080)
		config := &imagev1.Config{
			Env: []string{
				"SERVICE_TAGS=webapp",
				"SERVICE_8080_TAGS=webapp",
				"SERVICE_8080_NAME=jenkins",
			},
			ExposedPorts: map[string]struct{}{
				"8080/tcp":  {},
				"50000/tcp": {},
			},
		}

		routes, err := CollectRoutes(service, config)

		require.NoError(t, err)
		assert.Equal(t, []Route{
			{
				Name:     "jenkins",
				Port:     8080,
				Location: "/jenkins",
				Pass:     "/jenkins",
			},
		}, routes)
	})

	t.Run("should collect legacy rewrite config", func(t *testing.T) {
		service := newTestService("cas", 80)
		config := &imagev1.Config{
			Env: []string{
				"SERVICE_TAGS=webapp",
				"SERVICE_REWRITE='{\"pattern\":\"portainer\",\"rewrite\":\"\"}'",
			},
			ExposedPorts: map[string]struct{}{
				"80/tcp": {},
			},
		}

		routes, err := CollectRoutes(service, config)

		require.NoError(t, err)
		assert.Equal(t, []Route{
			{
				Name:     "cas",
				Port:     80,
				Location: "/cas",
				Pass:     "/cas",
				Rewrite:  `'{"pattern":"portainer","rewrite":""}'`,
			},
		}, routes)
	})

	t.Run("should collect pass differing from location unchanged", func(t *testing.T) {
		service := newTestService("cas", 80)
		config := &imagev1.Config{
			Env: []string{
				"SERVICE_TAGS=webapp",
				"SERVICE_LOCATION=api",
				"SERVICE_PASS=internal/api/",
			},
			ExposedPorts: map[string]struct{}{
				"80/tcp": {},
			},
		}

		routes, err := CollectRoutes(service, config)

		require.NoError(t, err)
		assert.Equal(t, []Route{
			{
				Name:     "cas",
				Port:     80,
				Location: "/api",
				Pass:     "/internal/api/",
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
				Name:     "cas",
				Port:     80,
				Location: "/cas",
				Pass:     "/cas",
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
				Name:     "cas",
				Port:     80,
				Location: "/ui",
				Pass:     "/internal/ui",
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

func TestSplitImagePortConfig(t *testing.T) {
	t.Run("should parse port without protocol", func(t *testing.T) {
		port, protocol, err := SplitImagePortConfig("8080")

		assert.NoError(t, err)
		assert.Equal(t, int32(8080), port)
		assert.Equal(t, "TCP", string(protocol))
	})

	t.Run("should parse port with protocol", func(t *testing.T) {
		port, protocol, err := SplitImagePortConfig("53/udp")

		assert.NoError(t, err)
		assert.Equal(t, int32(53), port)
		assert.Equal(t, "UDP", string(protocol))
	})

	t.Run("should fail for invalid port", func(t *testing.T) {
		_, _, err := SplitImagePortConfig("http/tcp")

		assert.Error(t, err)
		assert.ErrorContains(t, err, "error parsing int")
	})
}
