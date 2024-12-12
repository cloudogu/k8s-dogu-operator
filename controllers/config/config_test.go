package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOperatorConfig(t *testing.T) {
	_ = os.Unsetenv("NAMESPACE")
	_ = os.Unsetenv("DOGU_REGISTRY_ENDPOINT")
	_ = os.Unsetenv("DOGU_REGISTRY_USERNAME")
	_ = os.Unsetenv("DOGU_REGISTRY_PASSWORD")
	_ = os.Unsetenv("DOGU_REGISTRY_URLSCHEMA")

	expectedNamespace := "myNamespace"
	expectedDoguRegistryData := DoguRegistryData{
		Endpoint: "myEndpoint",
		Username: "myUsername",
		Password: "myPassword",
	}

	t.Run("Error on missing namespace env var", func(t *testing.T) {
		// when
		operatorConfig, err := NewOperatorConfig("0.0.0")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get env var [NAMESPACE]: environment variable NAMESPACE must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("NAMESPACE", expectedNamespace)
	t.Run("Error on missing dogu registry endpoint var", func(t *testing.T) {
		// when
		operatorConfig, err := NewOperatorConfig("0.0.0")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get env var [DOGU_REGISTRY_ENDPOINT]: environment variable DOGU_REGISTRY_ENDPOINT must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("DOGU_REGISTRY_ENDPOINT", expectedDoguRegistryData.Endpoint)
	t.Run("Error on missing dogu registry username var", func(t *testing.T) {
		// when
		operatorConfig, err := NewOperatorConfig("0.0.0")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get env var [DOGU_REGISTRY_USERNAME]: environment variable DOGU_REGISTRY_USERNAME must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("DOGU_REGISTRY_USERNAME", expectedDoguRegistryData.Username)
	t.Run("Error on missing dogu registry password var", func(t *testing.T) {
		// when
		operatorConfig, err := NewOperatorConfig("0.0.0")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get env var [DOGU_REGISTRY_PASSWORD]: environment variable DOGU_REGISTRY_PASSWORD must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("DOGU_REGISTRY_PASSWORD", expectedDoguRegistryData.Password)
	t.Setenv("DOGU_REGISTRY_URLSCHEMA", "")
	t.Setenv("NETWORK_POLICIES_ENABLED", "true")

	t.Run("Create config successfully", func(t *testing.T) {
		// when
		operatorConfig, err := NewOperatorConfig("0.1.0")

		// then
		require.NoError(t, err)
		require.NotNil(t, operatorConfig)
		assert.Equal(t, expectedNamespace, operatorConfig.Namespace)
		assert.Equal(t, expectedDoguRegistryData, operatorConfig.DoguRegistry)
		assert.Equal(t, "0.1.0", operatorConfig.Version.Raw)
	})
}

func TestOperatorConfig_GetRemoteConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		inputEndpoint string
		stage         string
		urlSchemaEnv  string
		wantUrlSchema string
		wantEndpoint  string
	}{
		{name: "get remote configuration with correct url and production mode", inputEndpoint: "https://dogu.cloudogu.com/api/v2/", stage: StageProduction, wantUrlSchema: "default", wantEndpoint: "https://dogu.cloudogu.com/api/v2/", urlSchemaEnv: ""},
		{name: "get remote configuration with correct url and development mode", inputEndpoint: "https://dogu.cloudogu.com/api/v2/", stage: StageDevelopment, wantUrlSchema: "default", wantEndpoint: "https://dogu.cloudogu.com/api/v2/", urlSchemaEnv: "invalid"},
		{name: "get remote configuration with 'dogus' suffix url", inputEndpoint: "https://dogu.cloudogu.com/api/v2/dogus", stage: StageProduction, wantUrlSchema: "default", wantEndpoint: "https://dogu.cloudogu.com/api/v2/", urlSchemaEnv: "default"},
		{name: "get remote configuration with 'dogus/' suffix url", inputEndpoint: "https://dogu.cloudogu.com/api/v2/dogus/", stage: StageProduction, wantUrlSchema: "default", wantEndpoint: "https://dogu.cloudogu.com/api/v2/", urlSchemaEnv: "default"},
		{name: "get remote configuration with 'dogus' suffix url and non-default url schema", inputEndpoint: "https://dogu.cloudogu.com/api/v2/dogus", stage: StageProduction, wantUrlSchema: "index", wantEndpoint: "https://dogu.cloudogu.com/api/v2/dogus", urlSchemaEnv: "index"},
		{name: "get remote configuration with correct url and production mode", inputEndpoint: "https://dogu.cloudogu.com/api/v2/", stage: StageProduction, wantUrlSchema: "index", wantEndpoint: "https://dogu.cloudogu.com/api/v2/", urlSchemaEnv: "index"},
	}

	t.Setenv(envVarNamespace, "test")
	t.Setenv(envVarDoguRegistryEndpoint, "myEndpoint")
	t.Setenv(envVarDoguRegistryUsername, "user")
	t.Setenv(envVarDoguRegistryPassword, "password")
	t.Setenv(envVarDoguRegistryPassword, "password")
	t.Setenv(envVarNetworkPolicyEnabled, "true")

	defer func() {
		_ = os.Unsetenv(envVarNamespace)
		_ = os.Unsetenv(envVarDoguRegistryEndpoint)
		_ = os.Unsetenv(envVarDoguRegistryUsername)
		_ = os.Unsetenv(envVarDoguRegistryPassword)
	}()

	for _, tt := range tests {
		t.Setenv(envVarDoguRegistryURLSchema, tt.urlSchemaEnv)
		t.Run(tt.name, func(t *testing.T) {
			// given
			t.Setenv(StageEnvironmentVariable, tt.stage)
			defer func() {
				_ = os.Unsetenv(StageEnvironmentVariable)
			}()

			o, err := NewOperatorConfig("1.0.0")
			require.NoError(t, err)
			o.DoguRegistry = DoguRegistryData{Endpoint: tt.inputEndpoint, URLSchema: tt.urlSchemaEnv}

			// when
			remoteConfig := o.GetRemoteConfiguration()

			// then
			assert.NotNil(t, remoteConfig)
			assert.Equal(t, tt.wantEndpoint, remoteConfig.Endpoint)
			assert.Equal(t, "/tmp/dogu-registry-cache", remoteConfig.CacheDir)
			assert.Equal(t, tt.wantUrlSchema, remoteConfig.URLSchema)
			_ = os.Unsetenv(envVarDoguRegistryURLSchema)
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
