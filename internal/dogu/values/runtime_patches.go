package values

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// RuntimePatchTemplateFileName is the optional runtime values template at the chart root.
const RuntimePatchTemplateFileName = "runtime-patch-tpl.yaml"

// renderRuntimePatches decodes and renders a YAML values map. A nil source means the file is absent;
// a present file must contain exactly one YAML mapping, which may be empty.
func renderRuntimePatches(source []byte, global Values) (Values, error) {
	if source == nil {
		return nil, nil
	}

	decoder := yaml.NewDecoder(bytes.NewReader(source))
	var result Values
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("decode runtime patch template: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("runtime patch template must contain exactly one YAML document")
	}
	if result == nil {
		return nil, fmt.Errorf("runtime patch template must be a mapping")
	}

	funcs := template.FuncMap{
		"globalConfig": func(path string) (string, error) {
			var value any = global
			for segment := range strings.SplitSeq(path, "/") {
				if segment == "" {
					return "", fmt.Errorf("globalConfig path contains an empty segment")
				}
				mapping, ok := value.(map[string]any)
				if !ok {
					return "", fmt.Errorf("globalConfig %q does not refer to a string", path)
				}
				value, ok = mapping[segment]
				if !ok {
					return "", fmt.Errorf("globalConfig %q is missing", path)
				}
			}

			text, ok := value.(string)
			if !ok {
				return "", fmt.Errorf("globalConfig %q does not refer to a string", path)
			}
			return text, nil
		},
	}
	if _, err := renderRuntimeValue(result, funcs, RuntimePatchTemplateFileName); err != nil {
		return nil, err
	}

	if _, err := json.Marshal(result); err != nil {
		return nil, fmt.Errorf("failed to marshal runtime patches to JSON: %w", err)
	}

	return result, nil
}

// renderRuntimeValue renders string leaves once and updates maps and lists in place.
// Rendered strings are not reparsed as YAML or templates, so inserted quotes, newlines and
// template delimiters remain literal data.
func renderRuntimeValue(value any, funcs template.FuncMap, path string) (any, error) {
	switch value := value.(type) {
	case map[string]any:
		for key, child := range value {
			if strings.Contains(key, "{{") {
				return nil, fmt.Errorf("runtimePatches mapping keys must not contain templates")
			}
			rendered, err := renderRuntimeValue(child, funcs, path+"."+key)
			if err != nil {
				return nil, err
			}
			value[key] = rendered
		}
	case []any:
		for i, child := range value {
			rendered, err := renderRuntimeValue(child, funcs, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			value[i] = rendered
		}
	case string:
		if !strings.Contains(value, "{{") {
			return value, nil
		}

		tpl, err := template.New(path).Option("missingkey=error").Funcs(funcs).Parse(value)
		if err != nil {
			return nil, fmt.Errorf("parse %s template: %w", path, err)
		}

		var output bytes.Buffer
		if err := tpl.Execute(&output, nil); err != nil {
			return nil, fmt.Errorf("execute %s template: %w", path, err)
		}
		return output.String(), nil
	}
	return value, nil
}
