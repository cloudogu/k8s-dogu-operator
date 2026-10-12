package values

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimePatchesYAMLScalarFidelity(t *testing.T) {
	got, err := renderRuntimePatches([]byte(`apiVersion: v1
runtimePatches:
  values.yaml:
    on: yes
    off: no
    yes: on
    no: off
    "true": distinct
    integer: 9007199254740993
    signed: -9223372036854775808
    unsigned: 18446744073709551615
`), nil)
	require.NoError(t, err)
	encoded, err := json.Marshal(got)
	require.NoError(t, err)
	assert.Equal(t, `{"integer":9007199254740993,"no":"off","off":"no","on":"yes","signed":-9223372036854775808,"true":"distinct","unsigned":18446744073709551615,"yes":"on"}`, string(encoded))
}

func TestRenderRuntimePatchesStringsAndIsolation(t *testing.T) {
	source := `---
apiVersion: v1
patches:
  values.yaml:
    image: '{{ registryFrom .images.application }}'
runtimePatches:
  values.yaml:
    host: '{{ globalConfig "fqdn" }}'
    nested:
      cert: '{{ globalConfig "certificates/server.crt" }}'
    list: ['{{ globalConfig "fqdn" }}', false, 17, null]
    dotted: '{{ globalConfig "root.dot" }}'
`
	for _, text := range []string{"", "O'Reilly", `a"b\c`, "a: b # c", "BEGIN\nEND\n", "a\r\nb\r\n", "true", "null", "9007199254740993", "{{ unknown }}\ninjected: yes"} {
		t.Run(text, func(t *testing.T) {
			global := Values{"fqdn": text, "certificates": Values{"server.crt": text}, "root.dot": "literal"}
			for _, input := range []string{source, strings.ReplaceAll(source, "\n", "\r\n")} {
				got, err := renderRuntimePatches([]byte(input), global)
				require.NoError(t, err)
				assert.Equal(t, Values{"host": text, "nested": Values{"cert": text}, "list": []any{text, false, 17, nil}, "dotted": "literal"}, got)
			}
			// Prefixes must not change a string into a YAML number or introduce new keys.
			got, err := renderRuntimePatches([]byte(`apiVersion: v1
runtimePatches:
  values.yaml:
    value: '1{{ globalConfig "fqdn" }}'
`), global)
			require.NoError(t, err)
			assert.Equal(t, Values{"value": "1" + text}, got)
			assert.Equal(t, text, global["fqdn"])
		})
	}
}

func TestRenderRuntimePatchesLookupErrors(t *testing.T) {
	global := Values{"scalar": "value", "null": nil, "number": 14, "bool": false, "map": Values{"key": "value"}, "list": []any{"value"}}
	for _, path := range []string{"missing", "missing/child", "scalar/child", "null", "number", "bool", "map", "list", "", "/scalar", "scalar/", "absent//child"} {
		t.Run(path, func(t *testing.T) {
			source := "apiVersion: v1\nruntimePatches:\n  values.yaml:\n    value: '{{ globalConfig " + `"` + path + `"` + " }}'\n"
			_, err := renderRuntimePatches([]byte(source), global)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "globalConfig")
		})
	}
}

func TestRenderRuntimePatchesRejected(t *testing.T) {
	for _, tt := range []struct {
		name, source, errorContains string
	}{
		{"empty", "", "decode chart patch template"},
		{"comments", "# comment\n", "decode chart patch template"},
		{"root list", "[]", "decode chart patch template"},
		{"api", "apiVersion: v2\nruntimePatches: {}", "requires apiVersion v1"},
		{"no api", "runtimePatches: {}", "requires apiVersion v1"},
		{"duplicate section", "apiVersion: v1\nruntimePatches: {}\nruntimePatches: {}", "already defined"},
		{"list runtime", "apiVersion: v1\nruntimePatches: []", "decode chart patch template"},
		{"invalid mirror YAML", "apiVersion: v1\npatches: [\nruntimePatches: {}", "decode chart patch template"},
		{"null target", "  values.yaml: null", "values.yaml must be a mapping"},
		{"scalar target", "  values.yaml: string", "decode chart patch template"},
		{"list target", "  values.yaml: []", "decode chart patch template"},
		{"wrong target", "  other.yaml: {}", "unsupported runtimePatches target"},
		{"duplicate values", "  values.yaml: {key: a, key: b}", "already defined"},
		{"complex key", "  values.yaml: {? [a, b]: c}", "decode chart patch template"},
		{"dynamic key", "  values.yaml: {'{{ globalConfig \"key\" }}': value}", "keys must not contain templates"},
		{"unquoted template", "  values.yaml: {key: {{ globalConfig \"fqdn\" }}}", "invalid map key"},
		{"non JSON", "  values.yaml: {value: .nan}", "unsupported value: NaN"},
		{"documents", "  values.yaml: {}\n---\nother: {}", "exactly one YAML document"},
		{"bare lookup", "  values.yaml: {key: '{{ globalConfig fqdn }}'}", `function "fqdn" not defined`},
		{"syntax", "  values.yaml: {key: '{{ if }}'}", "parse runtimePatches"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := tt.source
			if strings.HasPrefix(source, "  ") {
				source = "apiVersion: v1\nruntimePatches:\n" + source
			}
			_, err := renderRuntimePatches([]byte(source), nil)
			require.ErrorContains(t, err, tt.errorContains)
		})
	}
	for _, helper := range []string{"quote", "default", "fail", "toYaml", "env", "include", "tpl", "lookup"} {
		t.Run(helper, func(t *testing.T) {
			_, err := renderRuntimePatches([]byte("apiVersion: v1\nruntimePatches:\n  values.yaml:\n    key: '{{ "+helper+" \"x\" }}'"), nil)
			require.ErrorContains(t, err, "not defined")
		})
	}
}

func TestRenderRuntimePatchesCompatibility(t *testing.T) {
	for _, source := range [][]byte{nil, []byte("apiVersion: v1\npatches:\n  image: '{{ registryFrom .images }}'"), []byte("apiVersion: v1\nruntimePatches: null")} {
		got, err := renderRuntimePatches(source, nil)
		require.NoError(t, err)
		assert.Nil(t, got)
	}
	for _, source := range []string{"apiVersion: v1\nruntimePatches: {}", "apiVersion: v1\nruntimePatches:\n  values.yaml: {}", `{"apiVersion": v1, "runtimePatches": {values.yaml: {}}}`} {
		got, err := renderRuntimePatches([]byte(source), nil)
		require.NoError(t, err)
		assert.Equal(t, Values{}, got)
	}
	got, err := renderRuntimePatches([]byte(`apiVersion: v1
runtimePatches:
  values.yaml:
    original: &host '{{ globalConfig "fqdn" }}'
    copy: *host
`), Values{"fqdn": "O'Reilly\nexample"})
	require.NoError(t, err)
	assert.Equal(t, Values{"original": "O'Reilly\nexample", "copy": "O'Reilly\nexample"}, got)
}
