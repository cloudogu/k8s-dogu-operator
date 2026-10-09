package values

import (
	"encoding/json"
	"reflect"
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
	assert.Equal(t, "yes", got["on"])
	assert.Equal(t, "no", got["off"])
	assert.Equal(t, "on", got["yes"])
	assert.Equal(t, "off", got["no"])
	assert.Equal(t, "distinct", got["true"])
	encoded, err := json.Marshal(got)
	require.NoError(t, err)
	assert.Equal(t, `{"integer":9007199254740993,"no":"off","off":"no","on":"yes","signed":-9223372036854775808,"true":"distinct","unsigned":18446744073709551615,"yes":"on"}`, string(encoded))
}

func TestRuntimePatchesDocumentStart(t *testing.T) {
	for _, body := range []string{"patches: {}\n", "runtimePatches:\n  values.yaml: {host: example}\n"} {
		for _, prefix := range []string{"---\n", "# comment\n--- # single document\r\n"} {
			got, err := renderRuntimePatches([]byte(prefix+"apiVersion: v1\n"+body), nil)
			require.NoError(t, err)
			if strings.HasPrefix(body, "runtimePatches") {
				assert.Equal(t, Values{"host": "example"}, got)
			} else {
				assert.Nil(t, got)
			}
		}
	}
	for _, source := range []string{"---\n---\napiVersion: v1\npatches: {}", "---\napiVersion: v1\npatches: {}\n---\nother: {}"} {
		_, err := renderRuntimePatches([]byte(source), nil)
		require.Error(t, err, "multiple documents remain unsupported")
	}
}

func TestRenderRuntimePatchesRawAndIsolation(t *testing.T) {
	global := Values{"fqdn": "host: #quoted", "certificates": Values{"server.crt": "BEGIN\nEND\n", "number": "14"}}
	body := `runtimePatches:
  values.yaml:
    host: {{ globalConfig "fqdn" | quote }}
    certConfig:
      {{ globalConfig "certificates" | toYaml | nindent 6 }}
    leaf: {{ globalConfig "certificates/server.crt" | quote }}
    helper: {{ list "a" "b" | join ":" | upper | quote }}
{{ $copy := globalConfig "certificates" }}
{{ $_ := set $copy "number" "changed" }}
    original: {{ globalConfig "certificates/number" | quote }}
`
	image := "patches:\n  values.yaml:\n    image: {{ registryFrom .images.foo }}\n"
	want := Values{"host": "host: #quoted", "certConfig": Values{"server.crt": "BEGIN\nEND\n", "number": "14"}, "leaf": "BEGIN\nEND\n", "helper": "A:B", "original": "14"}
	for _, source := range []string{"apiVersion: v1\n" + image + body, body + "apiVersion: v1\n" + image, image + body + "apiVersion: v1\n"} {
		for _, crlf := range []bool{false, true} {
			if crlf {
				source = strings.ReplaceAll(source, "\n", "\r\n")
			}
			got, err := renderRuntimePatches([]byte(source), global)
			if err != nil || !reflect.DeepEqual(want, got) {
				t.Fatalf("got %#v, error %v", got, err)
			}
		}
	}
	if global["certificates"].(Values)["number"] != "14" {
		t.Fatal("helper mutated original global config")
	}
}

func TestRenderRuntimePatchesDefaultsAndControls(t *testing.T) {
	global := Values{"scalar": "value", "empty": "", "zero": 0, "false": false, "root.dot": "literal"}
	for _, path := range []string{"missing", "missing/child", "scalar/child", "empty", "zero", "false"} {
		source := "apiVersion: v1\nruntimePatches:\n  values.yaml:\n    value: {{ globalConfig " + `"` + path + `"` + " | default \"fallback\" | quote }}\n"
		got, err := renderRuntimePatches([]byte(source), global)
		if err != nil || got["value"] != "fallback" {
			t.Fatalf("path %s: %#v, %v", path, got, err)
		}
	}
	source := `apiVersion: v1
runtimePatches:
  values.yaml:
{{ if not (empty (globalConfig "root.dot")) }}
    dotted: {{ globalConfig "root.dot" | quote }}
{{ end }}
{{ range $i, $value := list "one" "two" }}
    key{{ $i }}: {{ $value | quote }}
{{ end }}
# boundary comment
patches:
  ignored: {{ fail "must never execute" }}
`
	got, err := renderRuntimePatches([]byte(source), global)
	if err != nil || !reflect.DeepEqual(got, Values{"dotted": "literal", "key0": "one", "key1": "two"}) {
		t.Fatalf("controls: %#v, %v", got, err)
	}
}

func TestRenderRuntimePatchesRejected(t *testing.T) {
	for _, tt := range []struct {
		name, source, errorContains string
	}{
		{"empty", "", "envelope is empty"},
		{"comments", "# comment\n", "envelope is empty"},
		{"api", "apiVersion: v2\nruntimePatches: {}\n", "requires apiVersion v1"},
		{"no api", "runtimePatches: {}", "requires apiVersion v1"},
		{"duplicate section", "apiVersion: v1\nruntimePatches: {}\nruntimePatches: {}", "duplicate runtimePatches section"},
		{"null runtime", "apiVersion: v1\nruntimePatches:", "runtimePatches must be a mapping"},
		{"list runtime", "apiVersion: v1\nruntimePatches: []", "must be a block mapping"},
		{"null target", "  values.yaml: null", "values.yaml must be a mapping"},
		{"scalar target", "  values.yaml: string", "values.yaml must be a mapping"},
		{"list target", "  values.yaml: []", "values.yaml must be a mapping"},
		{"wrong target", "  other.yaml: {}", "unsupported runtimePatches target"},
		{"duplicate values", "  values.yaml: {key: a, key: b}", "already defined"},
		{"numeric key", "  values.yaml: {1: a}", "mapping keys must be strings"},
		{"complex key", "  values.yaml: {? [a, b]: c}", "mapping keys must be strings"},
		{"invalid YAML", "  values.yaml: [", "decode runtimePatches"},
		{"non JSON", "  values.yaml: {value: .nan}", "unsupported value: NaN"},
		{"documents", "  values.yaml: {}\n---\nother: {}", "expected literal block-style root key"},
		{"alias", "  values.yaml: {original: &value example, copy: *value}", "aliases and anchors are unsupported"},
		{"bare lookup", "  values.yaml: {key: '{{ globalConfig fqdn }}'}", `function "fqdn" not defined`},
		{"empty lookup", "  values.yaml: {key: '{{ globalConfig \"\" | default \"fallback\" }}'}", "path contains an empty segment"},
		{"bad lookup", "  values.yaml: {key: '{{ globalConfig \"absent//child\" | default \"fallback\" }}'}", "path contains an empty segment"},
		{"fail", "  values.yaml:\n{{ if empty (globalConfig \"absent\") }}{{ fail \"mandatory\" }}{{ end }}", "mandatory"},
		{"marshal error", "  values.yaml: {key: '{{ globalConfig \"invalid\" | toYaml }}'}", "error calling toYaml"},
		{"cross boundary", "  values.yaml:\n{{ if true }}\n    key: value\npatches:\n{{ end }}", "root template actions outside runtimePatches"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := tt.source
			if strings.HasPrefix(source, "  ") {
				source = "apiVersion: v1\nruntimePatches:\n" + source
			}
			_, err := renderRuntimePatches([]byte(source), Values{"invalid": make(chan int)})
			require.ErrorContains(t, err, tt.errorContains)
		})
	}
	for _, helper := range []string{"env", "expandenv", "getHostByName", "required", "mustGlobalConfig", "include", "tpl", "lookup"} {
		t.Run(helper, func(t *testing.T) {
			_, err := renderRuntimePatches([]byte("apiVersion: v1\nruntimePatches:\n  values.yaml:\n    key: {{ "+helper+" \"x\" }}"), nil)
			if err == nil || !strings.Contains(err.Error(), "not defined") {
				t.Fatalf("expected unavailable function: %v", err)
			}
		})
	}
}

func TestRenderRuntimePatchesCompatibility(t *testing.T) {
	for _, source := range [][]byte{nil, []byte("apiVersion: v1\npatches:\n  image: {{ registryFrom .images }}")} {
		got, err := renderRuntimePatches(source, nil)
		if err != nil || got != nil {
			t.Fatalf("expected no overlay: %#v, %v", got, err)
		}
	}
	for _, body := range []string{"runtimePatches: {}", "runtimePatches:\n  values.yaml: {}"} {
		got, err := renderRuntimePatches([]byte("apiVersion: v1\n"+body), nil)
		if err != nil || got == nil || len(got) != 0 {
			t.Fatalf("expected empty overlay: %#v, %v", got, err)
		}
	}
}
