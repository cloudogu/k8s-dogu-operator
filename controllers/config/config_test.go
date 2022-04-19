package config_test

import (
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewOperatorConfig(t *testing.T) {
	expectedNamespace := "myNamepsace"
	expectedDoguRegistryData := config.DoguRegistryData{
		Endpoint: "myEndpoint",
		Username: "myUsername",
		Password: "myPassword",
	}
	expectedDockerRegistryData := config.DockerRegistryData{
		Username: "myDockerUsername",
		Password: "myDockerPassword",
	}

	t.Run("Error on missing namespace env var", func(t *testing.T) {
		// when
		operatorConfig, err := config.NewOperatorConfig()

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get env var [NAMESPACE]: environment variable NAMESPACE must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("NAMESPACE", expectedNamespace)
	t.Run("Error on missing dogu registry endpoint var", func(t *testing.T) {
		// when
		operatorConfig, err := config.NewOperatorConfig()

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get env var [DOGU_REGISTRY_ENDPOINT]: environment variable DOGU_REGISTRY_ENDPOINT must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("DOGU_REGISTRY_ENDPOINT", expectedDoguRegistryData.Endpoint)
	t.Run("Error on missing dogu registry username var", func(t *testing.T) {
		// when
		operatorConfig, err := config.NewOperatorConfig()

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get env var [DOGU_REGISTRY_USERNAME]: environment variable DOGU_REGISTRY_USERNAME must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("DOGU_REGISTRY_USERNAME", expectedDoguRegistryData.Username)
	t.Run("Error on missing dogu registry password var", func(t *testing.T) {
		// when
		operatorConfig, err := config.NewOperatorConfig()

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get env var [DOGU_REGISTRY_PASSWORD]: environment variable DOGU_REGISTRY_PASSWORD must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("DOGU_REGISTRY_PASSWORD", expectedDoguRegistryData.Password)
	t.Run("Error on missing docker registry username var", func(t *testing.T) {
		// when
		operatorConfig, err := config.NewOperatorConfig()

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get env var [DOCKER_REGISTRY_USERNAME]: environment variable DOCKER_REGISTRY_USERNAME must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("DOCKER_REGISTRY_USERNAME", expectedDockerRegistryData.Username)
	t.Run("Error on missing docker registry password var", func(t *testing.T) {
		// when
		operatorConfig, err := config.NewOperatorConfig()

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get env var [DOCKER_REGISTRY_PASSWORD]: environment variable DOCKER_REGISTRY_PASSWORD must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("DOCKER_REGISTRY_PASSWORD", expectedDockerRegistryData.Password)
	t.Run("Create config successfully", func(t *testing.T) {
		// when
		operatorConfig, err := config.NewOperatorConfig()

		// then
		require.NoError(t, err)
		require.NotNil(t, operatorConfig)
		assert.Equal(t, expectedNamespace, operatorConfig.Namespace)
		assert.Equal(t, expectedDoguRegistryData, operatorConfig.DoguRegistry)
		assert.False(t, operatorConfig.DevelopmentLogMode)
	})

	t.Run("Error on parsing wrong value for zap log level", func(t *testing.T) {
		// given
		t.Setenv("ZAP_DEVELOPMENT_MODE", "invalid value")

		// when
		operatorConfig, err := config.NewOperatorConfig()

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "strconv.ParseBool: parsing \"invalid value\"")
		assert.Nil(t, operatorConfig)
	})
}
