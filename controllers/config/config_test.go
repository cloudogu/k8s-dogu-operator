package config

import (
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	dccv3 "github.com/cloudogu/dogu-lib/doguv3/doguregistry/dcc/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOperatorConfig(t *testing.T) {
	_ = os.Unsetenv("NAMESPACE")
	_ = os.Unsetenv("DOGU_REGISTRY_ENDPOINT")
	_ = os.Unsetenv("DOGU_V3_REGISTRY_ENDPOINT")
	_ = os.Unsetenv("DOGU_V3_REGISTRY_USERNAME")
	_ = os.Unsetenv("DOGU_V3_REGISTRY_PASSWORD")
	_ = os.Unsetenv("DOGU_V3_REGISTRY_INSECURE_SKIP_VERIFY")
	_ = os.Unsetenv("DOGU_REGISTRY_USERNAME")
	_ = os.Unsetenv("DOGU_REGISTRY_PASSWORD")
	_ = os.Unsetenv("DOGU_REGISTRY_URLSCHEMA")
	_ = os.Unsetenv("AUTH_REGISTRATION_ENABLED")
	_ = os.Unsetenv("EXPOSITION_ENABLED")
	_ = os.Unsetenv("WARP_MENU_ENTRY_ENABLED")
	_ = os.Unsetenv("HELM_RECONCILIATION_INTERVAL")

	expectedNamespace := "myNamespace"
	expectedDoguRegistryData := DoguRegistryData{
		Endpoint: "myEndpoint",
		Username: "myUsername",
		Password: "myPassword",
	}

	expectedV3DoguRegistryData := DoguRegistryData{
		Endpoint:  "v3Endpoint",
		Username:  "v3User",
		Password:  "v3Password",
		URLSchema: "default",
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
	t.Setenv("REQUEUE_TIME_FOR_DOGU_RESOURCE_IN_NANOSECONDS", "50000")
	t.Setenv("DOGU_REGISTRY_URLSCHEMA", "")
	t.Setenv("NETWORK_POLICIES_ENABLED", "true")
	t.Setenv("AUTH_REGISTRATION_ENABLED", "true")
	t.Setenv("EXPOSITION_ENABLED", "true")
	t.Setenv("WARP_MENU_ENTRY_ENABLED", "true")
	t.Setenv("DISABLE_POSTFIX_DEPENDENCY_CHECK", "true")
	t.Setenv("DOGU_V3_REGISTRY_ENDPOINT", expectedV3DoguRegistryData.Endpoint)
	t.Setenv("DOGU_V3_REGISTRY_USERNAME", expectedV3DoguRegistryData.Username)
	t.Setenv("DOGU_V3_REGISTRY_PASSWORD", expectedV3DoguRegistryData.Password)

	t.Run("Error on missing duration for helm reconciliation", func(t *testing.T) {
		// when
		operatorConfig, err := NewOperatorConfig("0.0.0")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get env var [HELM_RECONCILIATION_INTERVAL]: environment variable HELM_RECONCILIATION_INTERVAL must be set")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("HELM_RECONCILIATION_INTERVAL", "24")
	t.Run("Error on invalid duration for helm reconciliation", func(t *testing.T) {
		// when
		operatorConfig, err := NewOperatorConfig("0.0.0")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read helm reconciliation interval:")
		assert.ErrorContains(t, err, "value of env var [HELM_RECONCILIATION_INTERVAL] is no valid duration")
		assert.Nil(t, operatorConfig)
	})

	t.Setenv("HELM_RECONCILIATION_INTERVAL", "15s")
	t.Run("Create config successfully", func(t *testing.T) {
		// when
		operatorConfig, err := NewOperatorConfig("0.1.0")

		// then
		require.NoError(t, err)
		require.NotNil(t, operatorConfig)
		assert.Equal(t, expectedNamespace, operatorConfig.Namespace)
		assert.Equal(t, expectedDoguRegistryData, operatorConfig.DoguRegistry)
		assert.Equal(t, "0.1.0", operatorConfig.Version.Raw)
		assert.True(t, operatorConfig.AuthRegistrationEnabled)
		assert.True(t, operatorConfig.ExpositionEnabled)
		assert.True(t, operatorConfig.WarpMenuEntryEnabled)
		assert.Equal(t, expectedV3DoguRegistryData, operatorConfig.DoguV3Registry)
		assert.Equal(t, 15*time.Second, operatorConfig.HelmReconciliationInterval)
	})
}

func TestOperatorConfig_GetRemoteConfiguration(t *testing.T) {
	tests := []struct {
		name              string
		inputEndpoint     string
		stage             string
		urlSchemaEnv      string
		wantUrlSchema     string
		wantEndpoint      string
		wantProxySettings core.ProxySettings
		setEnv            func(t *testing.T)
	}{
		{name: "get remote configuration with correct url and production mode", inputEndpoint: "https://dogu.cloudogu.com/api/v2/", stage: StageProduction, wantUrlSchema: "default", wantEndpoint: "https://dogu.cloudogu.com/api/v2/", urlSchemaEnv: ""},
		{name: "get remote configuration with correct url and development mode", inputEndpoint: "https://dogu.cloudogu.com/api/v2/", stage: StageDevelopment, wantUrlSchema: "default", wantEndpoint: "https://dogu.cloudogu.com/api/v2/", urlSchemaEnv: "invalid"},
		{name: "get remote configuration with 'dogus' suffix url", inputEndpoint: "https://dogu.cloudogu.com/api/v2/dogus", stage: StageProduction, wantUrlSchema: "default", wantEndpoint: "https://dogu.cloudogu.com/api/v2/", urlSchemaEnv: "default"},
		{name: "get remote configuration with 'dogus/' suffix url", inputEndpoint: "https://dogu.cloudogu.com/api/v2/dogus/", stage: StageProduction, wantUrlSchema: "default", wantEndpoint: "https://dogu.cloudogu.com/api/v2/", urlSchemaEnv: "default"},
		{name: "get remote configuration with 'dogus' suffix url and non-default url schema", inputEndpoint: "https://dogu.cloudogu.com/api/v2/dogus", stage: StageProduction, wantUrlSchema: "index", wantEndpoint: "https://dogu.cloudogu.com/api/v2/dogus", urlSchemaEnv: "index"},
		{name: "get remote configuration with correct url and production mode", inputEndpoint: "https://dogu.cloudogu.com/api/v2/", stage: StageProduction, wantUrlSchema: "index", wantEndpoint: "https://dogu.cloudogu.com/api/v2/", urlSchemaEnv: "index"},
		{name: "get remote configuration with proxy", inputEndpoint: "https://dogu.cloudogu.com/api/v2/", stage: StageProduction, wantUrlSchema: "index", wantEndpoint: "https://dogu.cloudogu.com/api/v2/", urlSchemaEnv: "index", wantProxySettings: core.ProxySettings{Enabled: true, Server: "host", Port: 3128, Username: "user", Password: "pass"}, setEnv: func(t *testing.T) {
			t.Setenv("PROXY_URL", "http://user:pass@host:3128")
		}},
	}

	t.Setenv(envVarNamespace, "test")
	t.Setenv(envVarDoguRegistryEndpoint, "myEndpoint")
	t.Setenv(envVarDoguRegistryUsername, "user")
	t.Setenv(envVarDoguRegistryPassword, "password")
	t.Setenv(envVarNetworkPolicyEnabled, "true")
	t.Setenv(envVarAuthRegistrationEnabled, "false")
	t.Setenv(envVarExpositionEnabled, "false")
	t.Setenv(envVarWarpMenuEntryEnabled, "false")
	t.Setenv(envVarRequeueTimeForDoguResourceInNanoseconds, "5")
	t.Setenv(envVarHelmReconciliationInterval, "3m")

	for _, tt := range tests {
		t.Setenv(envVarDoguRegistryURLSchema, tt.urlSchemaEnv)
		if tt.setEnv != nil {
			tt.setEnv(t)
		}
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
			remoteConfig, err := o.GetRemoteConfiguration()

			// then
			require.NoError(t, err)
			assert.NotNil(t, remoteConfig)
			assert.Equal(t, tt.wantEndpoint, remoteConfig.Endpoint)
			assert.Equal(t, "/tmp/dogu-registry-cache", remoteConfig.CacheDir)
			assert.Equal(t, tt.wantUrlSchema, remoteConfig.URLSchema)
			assert.Equal(t, tt.wantProxySettings, remoteConfig.ProxySettings)
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

func TestOperatorConfig_GetV3RemoteCredentials(t *testing.T) {
	// given
	o := &OperatorConfig{
		DoguV3Registry: DoguRegistryData{
			Username: "testUsername",
			Password: "testPassword",
		},
	}

	// when
	remoteCredentials := o.GetV3RemoteCredentials()

	// then
	assert.NotNil(t, remoteCredentials)
	assert.Equal(t, "testUsername", remoteCredentials.Username)
	assert.Equal(t, "testPassword", remoteCredentials.Password)
}

func TestGetStage(t *testing.T) {

	t.Run("Error on missing stage env var", func(t *testing.T) {
		t.Setenv(StageEnvironmentVariable, "")
		require.NoError(t, os.Unsetenv(StageEnvironmentVariable))

		// when
		stage, err := GetStage()

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "environment variable STAGE must be set")
		assert.Empty(t, stage)
	})

	t.Run("Successfully get stage env var", func(t *testing.T) {
		// given
		t.Setenv(StageEnvironmentVariable, "development")

		// when
		stage, err := GetStage()

		// then
		require.NoError(t, err)
		assert.Equal(t, "development", stage)
	})

}

func TestGetImageConfigCacheSize(t *testing.T) {
	t.Run("should return default when env var is not set", func(t *testing.T) {
		// given
		t.Setenv(envVarImageConfigCacheSize, "")
		require.NoError(t, os.Unsetenv(envVarImageConfigCacheSize))

		// when
		size := getImageConfigCacheSize()

		// then
		assert.Equal(t, defaultImageConfigCacheSize, size)
	})

	tests := []struct {
		name  string
		value string
		want  int
	}{
		{name: "should return the configured positive size", value: "10", want: 10},
		{name: "should return zero to disable the cache", value: "0", want: 0},
		{name: "should return a negative value to disable the cache", value: "-1", want: -1},
		{name: "should return default on an unparseable value", value: "not-a-number", want: defaultImageConfigCacheSize},
		{name: "should return default on an empty value", value: " ", want: defaultImageConfigCacheSize},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			t.Setenv(envVarImageConfigCacheSize, tt.value)

			// when
			size := getImageConfigCacheSize()

			// then
			assert.Equal(t, tt.want, size)
		})
	}
}

func TestOperatorConfig_GetV3RemoteConfiguration(t *testing.T) {
	type fields struct {
		DoguV3Registry DoguRegistryData
	}
	endpoint := "https://v3.dogu.cloudogu.com"
	schema := "default"
	tests := []struct {
		name    string
		fields  fields
		want    *dccv3.DoguRegistryConfiguration
		wantErr assert.ErrorAssertionFunc
		setEnv  func(t *testing.T)
	}{
		{
			name: "success",
			fields: fields{
				DoguV3Registry: DoguRegistryData{
					Endpoint:  endpoint,
					URLSchema: schema,
				},
			},
			want: &dccv3.DoguRegistryConfiguration{
				BaseURL:       endpoint,
				ProxySettings: dccv3.ProxySettings{},
				UserAgent:     fmt.Sprintf("k8s-dogu-operator/%s (%s/%s)", "1.0.0", runtime.GOOS, runtime.GOARCH),
				URLSchema:     schema,
			},
			wantErr: assert.NoError,
		},
		{
			name: "success with insecure",
			fields: fields{
				DoguV3Registry: DoguRegistryData{
					Endpoint:  endpoint,
					URLSchema: schema,
				},
			},
			want: &dccv3.DoguRegistryConfiguration{
				BaseURL:            endpoint,
				ProxySettings:      dccv3.ProxySettings{},
				UserAgent:          fmt.Sprintf("k8s-dogu-operator/%s (%s/%s)", "1.0.0", runtime.GOOS, runtime.GOARCH),
				URLSchema:          schema,
				InsecureSkipVerify: true,
			},
			wantErr: assert.NoError,
			setEnv: func(t *testing.T) {
				t.Setenv("DOGU_V3_REGISTRY_INSECURE_SKIP_VERIFY", "true")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv != nil {
				tt.setEnv(t)
			}
			version, err := core.ParseVersion("1.0.0")
			require.NoError(t, err)
			o := &OperatorConfig{
				DoguV3Registry: tt.fields.DoguV3Registry,
				Version:        &version,
			}
			got, err := o.GetV3RemoteConfiguration()
			if !tt.wantErr(t, err, fmt.Sprintf("GetV3RemoteConfiguration()")) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetV3RemoteConfiguration()")
		})
	}
}
