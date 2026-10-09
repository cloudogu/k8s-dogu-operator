package values

import (
	"bytes"
	"context"
	stdlog "log"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const testNamespace = "ecosystem"

func testDiscardLogger() logr.Logger {
	return logr.Discard()
}

// newTestDogu builds a Dogu CR with the given namespace, raw spec values and mapped values.
func newTestDogu(namespace string, rawValues []byte, mappedValues map[string]string) *v3beta1.Dogu {
	return &v3beta1.Dogu{
		Namespace: namespace,
		Spec: v3beta1.DoguSpec{
			Values:       apiextensionsv1.JSON{Raw: rawValues},
			MappedValues: mappedValues,
		},
	}
}

// newFakeClientWithGlobalConfig returns a fake client holding a global-config ConfigMap
// in the given namespace with the given data map.
func newFakeClientWithGlobalConfig(namespace string, data map[string]string) client.Client {
	cm := &coreV1.ConfigMap{
		Name:      globalConfigName,
		Namespace: namespace,
		Data:      data,
	}
	return fake.NewClientBuilder().WithObjects(cm).Build()
}

func Test_getDoguCRValues(t *testing.T) {
	tests := []struct {
		name    string
		raw     []byte
		want    Values
		wantErr string
	}{
		{
			name: "nil raw values returns nil without error",
			raw:  nil,
			want: nil,
		},
		{
			name: "empty raw values returns nil without error",
			raw:  []byte(""),
			want: nil,
		},
		{
			name: "valid flat map",
			raw:  []byte("foo: bar\nbaz: qux\n"),
			want: Values{"foo": "bar", "baz": "qux"},
		},
		{
			name: "valid nested map",
			raw:  []byte("parent:\n  child: value\n"),
			// yaml.v3 decoding into a Values map produces nested Values, not map[string]any.
			want: Values{"parent": Values{"child": "value"}},
		},
		{
			name:    "invalid yaml (sequence into map)",
			raw:     []byte("- a\n- b\n"),
			wantErr: "failed to parse values from dogu spec",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := newTestDogu(testNamespace, tt.raw, nil)

			got, err := getDoguCRValues(cr)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_getDoguMetaValues(t *testing.T) {
	patchTpl := []byte(`
apiVersion: v1
metavalues:
  mainLogLevel:
    name: Log-Level
    description: sets the log level
    keys:
      - path: controllerManager.env.logLevel
        mapping:
          debug: trace
          info: info
  passthrough:
    name: Passthrough
    keys:
      - path: some.flat.key
  multiKey:
    name: MultiKey
    keys:
      - path: first.key
      - path: second.key
`)

	tests := []struct {
		name         string
		patchTpl     []byte
		mappedValues map[string]string
		want         Values
		wantErr      string
	}{
		{
			name:         "no mapped values returns empty map",
			patchTpl:     patchTpl,
			mappedValues: nil,
			want:         Values{},
		},
		{
			name:         "mapped value key absent from metavalues is skipped",
			patchTpl:     patchTpl,
			mappedValues: map[string]string{"unknown": "whatever"},
			want:         Values{},
		},
		{
			name:         "nil patch template with mapped values returns empty map",
			patchTpl:     nil,
			mappedValues: map[string]string{"mainLogLevel": "debug"},
			want:         Values{},
		},
		{
			name:         "empty patch template with mapped values returns empty map",
			patchTpl:     []byte{},
			mappedValues: map[string]string{"mainLogLevel": "debug"},
			want:         Values{},
		},
		{
			name:         "mapping present and value found is translated",
			patchTpl:     patchTpl,
			mappedValues: map[string]string{"mainLogLevel": "debug"},
			want: Values{
				"controllerManager": map[string]any{
					"env": map[string]any{"logLevel": "trace"},
				},
			},
		},
		{
			name:         "mapping present but value missing yields empty string",
			patchTpl:     patchTpl,
			mappedValues: map[string]string{"mainLogLevel": "unmapped"},
			want: Values{
				"controllerManager": map[string]any{
					"env": map[string]any{"logLevel": ""},
				},
			},
		},
		{
			name:         "no mapping table uses raw value",
			patchTpl:     patchTpl,
			mappedValues: map[string]string{"passthrough": "rawValue"},
			want: Values{
				"some": map[string]any{"flat": map[string]any{"key": "rawValue"}},
			},
		},
		{
			name:         "multiple keys for one metavalue are all set",
			patchTpl:     patchTpl,
			mappedValues: map[string]string{"multiKey": "v"},
			want: Values{
				"first":  map[string]any{"key": "v"},
				"second": map[string]any{"key": "v"},
			},
		},
		{
			// A comma is ParseInto's assignment separator. The value "a,b" is read as
			// "some.flat.key=a" followed by a bare "b" which has no value, so ParseInto
			// errors and only the truncated "a" is stored. getDoguMetaValues logs the
			// error and continues, so Assemble silently returns the corrupted value.
			// Known limitation: mapped values containing commas are lossy.
			name:         "value containing a comma is silently truncated",
			patchTpl:     patchTpl,
			mappedValues: map[string]string{"passthrough": "a,b"},
			want: Values{
				"some": map[string]any{"flat": map[string]any{"key": "a"}},
			},
		},
		{
			// ParseInto coerces numeric-looking values to int64, not string.
			name:         "numeric value is coerced to int64",
			patchTpl:     patchTpl,
			mappedValues: map[string]string{"passthrough": "123"},
			want: Values{
				"some": map[string]any{"flat": map[string]any{"key": int64(123)}},
			},
		},
		{
			// ParseInto coerces boolean-looking values to bool.
			name:         "boolean value is coerced to bool",
			patchTpl:     patchTpl,
			mappedValues: map[string]string{"passthrough": "true"},
			want: Values{
				"some": map[string]any{"flat": map[string]any{"key": true}},
			},
		},
		{
			// ParseInto coerces the literal "null" to nil.
			name:         "null value is coerced to nil",
			patchTpl:     patchTpl,
			mappedValues: map[string]string{"passthrough": "null"},
			want: Values{
				"some": map[string]any{"flat": map[string]any{"key": nil}},
			},
		},
		{
			// Brace syntax is parsed by ParseInto into a slice.
			name:         "brace list value is parsed into a slice",
			patchTpl:     patchTpl,
			mappedValues: map[string]string{"passthrough": "{x,y}"},
			want: Values{
				"some": map[string]any{"flat": map[string]any{"key": []any{"x", "y"}}},
			},
		},
		{
			// An equals sign after the first one is kept verbatim in the value.
			name:         "value containing an equals sign is preserved",
			patchTpl:     patchTpl,
			mappedValues: map[string]string{"passthrough": "k=v"},
			want: Values{
				"some": map[string]any{"flat": map[string]any{"key": "k=v"}},
			},
		},
		{
			name:         "invalid patch template yaml returns error",
			patchTpl:     []byte("\tnot: valid: yaml:"),
			mappedValues: map[string]string{"mainLogLevel": "debug"},
			wantErr:      "failed to unmarshal mapping meta data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := newTestDogu(testNamespace, nil, tt.mappedValues)

			got, err := getDoguMetaValues(cr, tt.patchTpl, testDiscardLogger())

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_getGlobalConfigValues(t *testing.T) {
	tests := []struct {
		name     string
		clientFn func() client.Client
		crNs     string
		want     Values
		wantErr  string
	}{
		{
			name: "config map not present returns error",
			clientFn: func() client.Client {
				return fake.NewClientBuilder().Build()
			},
			crNs:    testNamespace,
			wantErr: "failed to get global config",
		},
		{
			name: "config map in a different namespace is not found",
			clientFn: func() client.Client {
				return newFakeClientWithGlobalConfig("other", map[string]string{globalConfigFileName: "key: value"})
			},
			crNs:    testNamespace,
			wantErr: "failed to get global config",
		},
		{
			name: "config map without config.yaml key returns error",
			clientFn: func() client.Client {
				return newFakeClientWithGlobalConfig(testNamespace, map[string]string{"other.yaml": "key: value"})
			},
			crNs:    testNamespace,
			wantErr: "did not found key config.yaml in global-config",
		},
		{
			name: "valid config yaml is parsed",
			clientFn: func() client.Client {
				return newFakeClientWithGlobalConfig(testNamespace, map[string]string{
					globalConfigFileName: "foo: bar\nnested:\n  a: b\n",
				})
			},
			crNs: testNamespace,
			want: Values{"foo": "bar", "nested": Values{"a": "b"}},
		},
		{
			name: "invalid config yaml returns error",
			clientFn: func() client.Client {
				return newFakeClientWithGlobalConfig(testNamespace, map[string]string{
					globalConfigFileName: "- a\n- b\n",
				})
			},
			crNs:    testNamespace,
			wantErr: "failed to parse global-config",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := newTestDogu(tt.crNs, nil, nil)

			got, err := getGlobalConfigValues(context.Background(), cr, tt.clientFn())

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_mergeValues(t *testing.T) {
	tests := []struct {
		name   string
		values []Values
		want   Values
	}{
		{
			name:   "no input returns empty non-nil map",
			values: nil,
			want:   Values{},
		},
		{
			name:   "all nil inputs returns empty non-nil map",
			values: []Values{nil, nil},
			want:   Values{},
		},
		{
			name:   "nil entries are skipped",
			values: []Values{nil, {"a": "1"}, nil},
			want:   Values{"a": "1"},
		},
		{
			name:   "disjoint keys are unioned",
			values: []Values{{"a": "1"}, {"b": "2"}},
			want:   Values{"a": "1", "b": "2"},
		},
		{
			name:   "later value wins on scalar conflict",
			values: []Values{{"a": "first"}, {"a": "second"}},
			want:   Values{"a": "second"},
		},
		{
			name: "nested maps are deep merged with later leaf winning",
			values: []Values{
				{"parent": map[string]any{"keep": "x", "override": "old"}},
				{"parent": map[string]any{"override": "new", "add": "y"}},
			},
			want: Values{"parent": map[string]any{"keep": "x", "override": "new", "add": "y"}},
		},
		{
			name:   "higher scalar replaces lower map",
			values: []Values{{"key": Values{"child": "lower"}}, {"key": "higher"}},
			want:   Values{"key": "higher"},
		},
		{
			name:   "higher map replaces lower scalar",
			values: []Values{{"key": "lower"}, {"key": Values{"child": "higher"}}},
			want:   Values{"key": Values{"child": "higher"}},
		},
		{
			name:   "null removes an existing key but retains a new null key",
			values: []Values{{"existing": "lower", "sibling": "keep"}, {"existing": nil, "new": nil}},
			want:   Values{"sibling": "keep", "new": nil},
		},
		{
			name:   "nested null removes only the existing leaf",
			values: []Values{{"parent": Values{"child": "lower", "sibling": "keep"}}, {"parent": Values{"child": nil}}},
			want:   Values{"parent": Values{"sibling": "keep"}},
		},
		{
			name:   "null over null remains explicit",
			values: []Values{{"key": nil}, {"key": nil}},
			want:   Values{"key": nil},
		},
		{
			name:   "later source can restore a removed key",
			values: []Values{{"key": "lower"}, {"key": nil}, {"key": "restored"}},
			want:   Values{"key": "restored"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeValues(tt.values...)

			assert.NotNil(t, got)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAssembler_Assemble(t *testing.T) {
	patchTpl := []byte(`
apiVersion: v1
metavalues:
  mainLogLevel:
    name: Log-Level
    keys:
      - path: shared
        mapping:
          debug: trace
`)

	// cesConfig extracts the global.cesConfig subtree the global config is nested under.
	cesConfig := func(t *testing.T, got Values) map[string]any {
		t.Helper()
		global, ok := got["global"].(map[string]any)
		require.True(t, ok, "global key must be a map")
		ces, ok := global["cesConfig"].(map[string]any)
		require.True(t, ok, "global.cesConfig must be a map")
		return ces
	}

	t.Run("global config is nested under global.cesConfig and does not override top-level dogu values", func(t *testing.T) {
		cr := newTestDogu(testNamespace,
			[]byte("shared: fromCR\ncrOnly: crValue\n"),
			map[string]string{"mainLogLevel": "debug"},
		)
		k8s := newFakeClientWithGlobalConfig(testNamespace, map[string]string{
			globalConfigFileName: "shared: fromGlobal\nglobalOnly: globalValue\n",
		})
		a := Assembler{k8s: k8s}

		got, err := a.Assemble(context.Background(), cr, patchTpl, nil)

		require.NoError(t, err)
		// Top level: meta overrides cr; global lives in its own subtree and does not touch it.
		assert.Equal(t, "trace", got["shared"], "meta values must override CR values")
		assert.Equal(t, "crValue", got["crOnly"])
		// Global config is namespaced under global.cesConfig.
		assert.Equal(t, "fromGlobal", cesConfig(t, got)["shared"])
		assert.Equal(t, "globalValue", cesConfig(t, got)["globalOnly"])
	})

	t.Run("cr values win over global config on a shared path", func(t *testing.T) {
		// Both set global.cesConfig.key; cr is merged after global, so cr wins.
		cr := newTestDogu(testNamespace, []byte("global:\n  cesConfig:\n    key: fromCR\n"), nil)
		k8s := newFakeClientWithGlobalConfig(testNamespace, map[string]string{
			globalConfigFileName: "key: fromGlobal\n",
		})
		a := Assembler{k8s: k8s}

		got, err := a.Assemble(context.Background(), cr, patchTpl, nil)

		require.NoError(t, err)
		assert.Equal(t, "fromCR", cesConfig(t, got)["key"])
	})

	t.Run("minimal cr with only global config", func(t *testing.T) {
		cr := newTestDogu(testNamespace, nil, nil)
		k8s := newFakeClientWithGlobalConfig(testNamespace, map[string]string{
			globalConfigFileName: "onlyGlobal: value\n",
		})
		a := Assembler{k8s: k8s}

		got, err := a.Assemble(context.Background(), cr, patchTpl, nil)

		require.NoError(t, err)
		assert.Equal(t, Values{"global": map[string]any{"cesConfig": Values{"onlyGlobal": "value"}}}, got)
	})

	t.Run("invalid cr values returns error", func(t *testing.T) {
		cr := newTestDogu(testNamespace, []byte("- a\n- b\n"), nil)
		a := Assembler{k8s: newFakeClientWithGlobalConfig(testNamespace, map[string]string{
			globalConfigFileName: "a: b",
		})}

		_, err := a.Assemble(context.Background(), cr, patchTpl, nil)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to dogu spec values")
	})

	t.Run("missing global config returns error", func(t *testing.T) {
		cr := newTestDogu(testNamespace, []byte("a: b\n"), nil)
		a := Assembler{k8s: fake.NewClientBuilder().Build()}

		_, err := a.Assemble(context.Background(), cr, patchTpl, nil)

		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get values from global config")
	})
}

func TestAssembler_RuntimePrecedence(t *testing.T) {
	cr := newTestDogu(testNamespace, []byte(`application:
  host: fromCR
  logging: fromCR
  list: [cr]
global:
  cesConfig:
    crOverride: fromCR
`), map[string]string{"logging": "debug"})
	metadata := []byte(`apiVersion: v1
metavalues:
  logging:
    keys:
      - path: application.logging
        mapping:
          debug: trace
`)
	patch := []byte(`apiVersion: v1
runtimePatches:
  values.yaml:
    application:
      host: fromRuntime
      logging: fromRuntime
      sibling: {{ globalConfig "fqdn" | quote }}
      list: [runtime]
    global:
      cesConfig:
        runtimeOverride: fromRuntime
        crOverride: fromRuntime
`)
	a := NewAssembler(newFakeClientWithGlobalConfig(testNamespace, map[string]string{
		globalConfigFileName: "fqdn: ces.example.org\nruntimeOverride: fromGlobal\ncrOverride: fromGlobal\nuntouched: original\n",
	}))
	got, err := a.Assemble(t.Context(), cr, metadata, patch)
	require.NoError(t, err)
	assert.Equal(t, Values{
		"application": Values{"host": "fromCR", "logging": "trace", "sibling": "ces.example.org", "list": []any{"cr"}},
		"global":      Values{"cesConfig": Values{"fqdn": "ces.example.org", "runtimeOverride": "fromRuntime", "crOverride": "fromCR", "untouched": "original"}},
	}, got)
}

func TestAssembler_RuntimeErrorsDoNotBecomeDefaults(t *testing.T) {
	patch := []byte("apiVersion: v1\nruntimePatches:\n  values.yaml:\n    host: {{ globalConfig \"absent\" | default \"fallback\" | quote }}\n")
	for _, config := range []map[string]string{nil, {globalConfigFileName: "["}} {
		a := NewAssembler(newFakeClientWithGlobalConfig(testNamespace, config))
		_, err := a.Assemble(t.Context(), newTestDogu(testNamespace, nil, nil), nil, patch)
		require.Error(t, err)
	}
	a := NewAssembler(newFakeClientWithGlobalConfig(testNamespace, map[string]string{globalConfigFileName: "{}"}))
	_, err := a.Assemble(t.Context(), newTestDogu(testNamespace, nil, nil), nil, []byte("apiVersion: v1\nruntimePatches:\n  values.yaml:\n    value: {{ fail \"required\" }}"))
	require.ErrorContains(t, err, "required")
}

func TestAssembler_ConflictsDoNotLogValues(t *testing.T) {
	var standardLogs, structuredLogs bytes.Buffer
	previousOutput := stdlog.Writer()
	stdlog.SetOutput(&standardLogs)
	t.Cleanup(func() { stdlog.SetOutput(previousOutput) })
	logger := funcr.New(func(prefix, message string) {
		structuredLogs.WriteString(prefix + message + "\n")
	}, funcr.Options{})
	cr := newTestDogu(testNamespace, []byte(`application:
  mapToScalar: replacement
  scalarToMap:
    enabled: true
`), nil)
	patch := []byte(`apiVersion: v1
runtimePatches:
  values.yaml:
    application:
      mapToScalar:
        token: {{ globalConfig "token" | quote }}
      scalarToMap: {{ globalConfig "token" | quote }}
`)
	a := NewAssembler(newFakeClientWithGlobalConfig(testNamespace, map[string]string{
		globalConfigFileName: "token: sensitive-sentinel\n",
	}))
	got, err := a.Assemble(log.IntoContext(t.Context(), logger), cr, nil, patch)
	require.NoError(t, err)
	assert.Equal(t, Values{"mapToScalar": "replacement", "scalarToMap": Values{"enabled": true}}, got["application"])
	assert.Contains(t, structuredLogs.String(), "application.mapToScalar")
	assert.Contains(t, structuredLogs.String(), "application.scalarToMap")
	assert.Contains(t, structuredLogs.String(), "spec.values")
	assert.NotContains(t, structuredLogs.String(), "sensitive-sentinel")
	assert.Empty(t, standardLogs.String(), "merging must not emit value-bearing Helm warnings")
}
