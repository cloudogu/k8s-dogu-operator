package values

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimePatchesYAMLScalarFidelity(t *testing.T) {
	got, err := renderRuntimePatches([]byte(`on: yes
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
host: '{{ globalConfig "fqdn" }}'
prefixed: '1{{ globalConfig "fqdn" }}'
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
				assert.Equal(t, Values{
					"host": text, "prefixed": "1" + text, "nested": Values{"cert": text},
					"list": []any{text, false, 17, nil}, "dotted": "literal",
				}, got)
			}
			assert.Equal(t, text, global["fqdn"])
		})
	}
}

func TestRenderRuntimePatchesLookupErrors(t *testing.T) {
	global := Values{"scalar": "value", "null": nil, "number": 14, "bool": false, "map": Values{"key": "value"}, "list": []any{"value"}}
	for _, path := range []string{"missing", "missing/child", "scalar/child", "null", "number", "bool", "map", "list", "", "/scalar", "scalar/", "absent//child"} {
		t.Run(path, func(t *testing.T) {
			source := "value: '{{ globalConfig " + `"` + path + `"` + " }}'\n"
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
		{"empty", "", "decode runtime patch template"},
		{"comments", "# comment\n", "decode runtime patch template"},
		{"null", "null", "must be a mapping"},
		{"root list", "[]", "decode runtime patch template"},
		{"root scalar", "string", "decode runtime patch template"},
		{"invalid YAML", "key: [", "decode runtime patch template"},
		{"duplicate values", "key: a\nkey: b", "already defined"},
		{"complex key", "? [a, b]\n: c", "decode runtime patch template"},
		{"dynamic key", "'{{ globalConfig \"key\" }}': value", "keys must not contain templates"},
		{"unquoted template", "key: {{ globalConfig \"fqdn\" }}", "invalid map key"},
		{"non JSON", "value: .nan", "unsupported value: NaN"},
		{"documents", "{}\n---\nother: {}", "exactly one YAML document"},
		{"bare lookup", "key: '{{ globalConfig fqdn }}'", `function "fqdn" not defined`},
		{"syntax", "key: '{{ if }}'", "parse runtime-patch-tpl.yaml"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := renderRuntimePatches([]byte(tt.source), nil)
			require.ErrorContains(t, err, tt.errorContains)
		})
	}
	for _, helper := range []string{"quote", "default", "fail", "toYaml", "env", "include", "tpl", "lookup"} {
		t.Run(helper, func(t *testing.T) {
			_, err := renderRuntimePatches([]byte("key: '{{ "+helper+" \"x\" }}'"), nil)
			require.ErrorContains(t, err, "not defined")
		})
	}
}

func TestRenderRuntimePatchesCompatibility(t *testing.T) {
	got, err := renderRuntimePatches(nil, nil)
	require.NoError(t, err)
	assert.Nil(t, got)
	for _, source := range []string{"{}", "---\n{}"} {
		got, err := renderRuntimePatches([]byte(source), nil)
		require.NoError(t, err)
		assert.Equal(t, Values{}, got)
	}
	got, err = renderRuntimePatches([]byte(`original: &host '{{ globalConfig "fqdn" }}'
copy: *host
`), Values{"fqdn": "O'Reilly\nexample"})
	require.NoError(t, err)
	assert.Equal(t, Values{"original": "O'Reilly\nexample", "copy": "O'Reilly\nexample"}, got)
}
