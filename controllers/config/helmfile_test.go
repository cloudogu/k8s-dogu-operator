package config

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExperimentalHelmfileSupport(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		unset, want bool
	}{
		{name: "unset", unset: true},
		{name: "enabled", value: "true", want: true},
		{name: "disabled", value: "false"},
		{name: "invalid", value: "invalid"},
		{name: "empty", value: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(envVarExperimentalHelmfileSupport, tc.value)
			if tc.unset {
				require.NoError(t, os.Unsetenv(envVarExperimentalHelmfileSupport))
			}
			assert.Equal(t, tc.want, getExperimentalHelmfileSupport())
		})
	}
}

func TestHelmfileConfiguration(t *testing.T) {
	global := HelmfileGlobalConfig{
		HelmfileBin: "/bin/helmfile", HelmBin: "/bin/helm", HelmPluginHome: "/plugins",
		HelmCacheHome: "/cache", HelmConfigHome: "/config", HelmDataHome: "/data", HelmfileUnpackDir: "/unpack",
	}
	openDesk := HelmfileOpenDeskConfig{
		HelmfileSource: "https://example.org/source.tar.gz", Environment: "prod", ValuesFilePath: "values.yaml",
		Namespace: "office", DomainConfigMap: K8sConfigRef{Name: "domain", Key: "fqdn"},
		MasterkeySecret: K8sConfigRef{Name: "password", Key: "masterkey"},
		ExtraValues:     map[string]any{"global": map[string]any{"enabled": true}}, ExtraEnvVars: map[string]string{"EXTRA": "value"},
	}
	for _, tc := range []struct {
		name, env, wrongType string
		full, empty          any
		read                 func() any
	}{
		{"global", envVarHelmfileGlobalConfig, `{"helmBin":123}`, global, HelmfileGlobalConfig{}, func() any { return getHelmfileGlobalConfig() }},
		{"openDesk", envVarHelmfileOpenDeskConfig, `{"extraEnvVars":123}`, openDesk, HelmfileOpenDeskConfig{}, func() any { return getHelmfileOpenDeskConfig() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(tc.full)
			require.NoError(t, err)
			for _, scenario := range []struct {
				name, value string
				unset       bool
				want        any
			}{
				{"unset", "", true, tc.empty}, {"empty", "", false, tc.empty},
				{"invalid JSON", "{", false, tc.empty}, {"wrong type", tc.wrongType, false, tc.empty},
				{"complete", string(data), false, tc.full}, {"empty object", "{}", false, tc.empty},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					t.Setenv(tc.env, scenario.value)
					if scenario.unset {
						require.NoError(t, os.Unsetenv(tc.env))
					}
					assert.Equal(t, scenario.want, tc.read())
				})
			}
		})
	}
}
