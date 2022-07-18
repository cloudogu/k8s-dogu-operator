package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestNewOperatorConfig(t *testing.T) {
	_ = os.Unsetenv("NAMESPACE")
	_ = os.Unsetenv("DOGU_REGISTRY_ENDPOINT")
	_ = os.Unsetenv("DOGU_REGISTRY_USERNAME")
	_ = os.Unsetenv("DOGU_REGISTRY_PASSWORD")
	_ = os.Unsetenv("DOCKER_REGISTRY")

	expectedNamespace := "myNamepsace"
	expectedDoguRegistryData := DoguRegistryData{
		Endpoint: "myEndpoint",
		Username: "myUsername",
		Password: "myPassword",
	}
	inputDockerRegistrySecretData := `{"auths":{"your.private.registry.example.com":{"username":"myDockerUsername","password":"myDockerPassword","email":"jdoe@example.com","auth":"c3R...zE2"}}}`
	expectedDockerRegistryData := DockerRegistryData{
		Username: "myDockerUsername",
		Password: "myDockerPassword",
		Email:    "jdoe@example.com",
		Auth:     "c3R...zE2",
	}

	t.Run("Error on missing namespace env var", func(t *testing.T) {
		// when
		operatorConfig, err := NewOperatorConfig("0.0.0")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get env var [NAMESPACE]: environment variable NAMESPACE must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("NAMESPACE", expectedNamespace)
	t.Run("Error on missing dogu registry endpoint var", func(t *testing.T) {
		// when
		operatorConfig, err := NewOperatorConfig("0.0.0")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get env var [DOGU_REGISTRY_ENDPOINT]: environment variable DOGU_REGISTRY_ENDPOINT must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("DOGU_REGISTRY_ENDPOINT", expectedDoguRegistryData.Endpoint)
	t.Run("Error on missing dogu registry username var", func(t *testing.T) {
		// when
		operatorConfig, err := NewOperatorConfig("0.0.0")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get env var [DOGU_REGISTRY_USERNAME]: environment variable DOGU_REGISTRY_USERNAME must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("DOGU_REGISTRY_USERNAME", expectedDoguRegistryData.Username)
	t.Run("Error on missing dogu registry password var", func(t *testing.T) {
		// when
		operatorConfig, err := NewOperatorConfig("0.0.0")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get env var [DOGU_REGISTRY_PASSWORD]: environment variable DOGU_REGISTRY_PASSWORD must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("DOGU_REGISTRY_PASSWORD", expectedDoguRegistryData.Password)
	t.Run("Error on missing docker registry data var", func(t *testing.T) {
		// when
		operatorConfig, err := NewOperatorConfig("0.0.0")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get env var [DOCKER_REGISTRY]: environment variable DOCKER_REGISTRY must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("DOCKER_REGISTRY", inputDockerRegistrySecretData)
	t.Run("Create config successfully", func(t *testing.T) {
		// when
		operatorConfig, err := NewOperatorConfig("0.1.0")

		// then
		require.NoError(t, err)
		require.NotNil(t, operatorConfig)
		assert.Equal(t, expectedNamespace, operatorConfig.Namespace)
		assert.Equal(t, expectedDoguRegistryData, operatorConfig.DoguRegistry)
		assert.Equal(t, expectedDockerRegistryData, operatorConfig.DockerRegistry)
		assert.Equal(t, "0.1.0", operatorConfig.Version.Raw)
		assert.False(t, operatorConfig.DevelopmentLogMode)
	})

	t.Run("Error on parsing wrong value for zap log level", func(t *testing.T) {
		// given
		t.Setenv("ZAP_DEVELOPMENT_MODE", "invalid value")

		// when
		operatorConfig, err := NewOperatorConfig("0.0.0")

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "strconv.ParseBool: parsing \"invalid value\"")
		assert.Nil(t, operatorConfig)
	})
}

func TestOperatorConfig_GetRemoteConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		inputEndpoint string
		stage         string
	}{
		{name: "get remote configuration with correct url and production mode", inputEndpoint: "https://dogu.cloudogu.com/api/v2/", stage: StageProduction},
		{name: "get remote configuration with correct url and development mode", inputEndpoint: "https://dogu.cloudogu.com/api/v2/", stage: StageDevelopment},
		{name: "get remote configuration with 'dogus' suffix url", inputEndpoint: "https://dogu.cloudogu.com/api/v2/dogus", stage: StageProduction},
		{name: "get remote configuration with 'dogus/' suffix url", inputEndpoint: "https://dogu.cloudogu.com/api/v2/dogus/", stage: StageProduction},
	}

	t.Setenv(envVarNamespace, "test")
	t.Setenv("DOGU_REGISTRY_ENDPOINT", "myEndpoint")
	t.Setenv("DOGU_REGISTRY_USERNAME", "user")
	t.Setenv("DOGU_REGISTRY_PASSWORD", "password")
	t.Setenv("DOCKER_REGISTRY", `{"auths":{"your.private.registry.example.com":{"username":"myDockerUsername","password":"myDockerPassword","email":"jdoe@example.com","auth":"c3R...zE2"}}}`)

	defer func() {
		_ = os.Unsetenv(envVarNamespace)
		_ = os.Unsetenv("DOGU_REGISTRY_ENDPOINT")
		_ = os.Unsetenv("DOGU_REGISTRY_USERNAME")
		_ = os.Unsetenv("DOGU_REGISTRY_PASSWORD")
		_ = os.Unsetenv("DOCKER_REGISTRY")
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			t.Setenv(StageEnvironmentVariable, tt.stage)
			defer func() {
				_ = os.Unsetenv(StageEnvironmentVariable)
			}()

			o, err := NewOperatorConfig("1.0.0")
			require.NoError(t, err)
			o.DoguRegistry = DoguRegistryData{Endpoint: tt.inputEndpoint}

			// when
			remoteConfig := o.GetRemoteConfiguration()

			// then
			assert.NotNil(t, remoteConfig)
			assert.Equal(t, "https://dogu.cloudogu.com/api/v2/", remoteConfig.Endpoint)
			if tt.stage == StageProduction {
				assert.Equal(t, "/home/nonroot", remoteConfig.CacheDir)
			} else {
				assert.Equal(t, ".", remoteConfig.CacheDir)
			}
		})
	}
}

func TestOperatorConfig_GetRemoteCredentials(t *testing.T) {
	// given
	o := &OperatorConfig{
		DoguRegistry: DoguRegistryData{
			Username: "testUsername",
			Password: "testPassword",
		},
	}

	// when
	remoteCredentials := o.GetRemoteCredentials()

	// then
	assert.NotNil(t, remoteCredentials)
	assert.Equal(t, "testUsername", remoteCredentials.Username)
	assert.Equal(t, "testPassword", remoteCredentials.Password)
}
