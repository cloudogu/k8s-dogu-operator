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

type patchTemplate struct {
	APIVersion     string            `yaml:"apiVersion"`
	RuntimePatches map[string]Values `yaml:"runtimePatches"`
}

func renderRuntimePatches(source []byte, global Values) (Values, error) {
	if source == nil {
		return nil, nil
	}
	decoder := yaml.NewDecoder(bytes.NewReader(source))
	var envelope patchTemplate
	if err := decoder.Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode chart patch template: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("chart patch template must contain exactly one YAML document")
	}
	if envelope.RuntimePatches == nil {
		return nil, nil
	}
	if envelope.APIVersion != "v1" {
		return nil, fmt.Errorf("runtimePatches requires apiVersion v1")
	}
	result := Values{}
	for target := range envelope.RuntimePatches {
		if target != "values.yaml" {
			return nil, fmt.Errorf("unsupported runtimePatches target %q", target)
		}
	}
	if value, exists := envelope.RuntimePatches["values.yaml"]; exists {
		if value == nil {
			return nil, fmt.Errorf("runtimePatches values.yaml must be a mapping")
		}
		result = value
	}
	funcs := template.FuncMap{
		"globalConfig": func(path string) (string, error) {
			var value any = global
			for _, segment := range strings.Split(path, "/") {
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
	if _, err := renderRuntimeValue(result, funcs, "runtimePatches.values.yaml"); err != nil {
		return nil, err
	}
	if _, err := json.Marshal(result); err != nil {
		return nil, fmt.Errorf("runtimePatches JSON conversion: %w", err)
	}
	return result, nil
}

// Render string leaves directly: inserted quotes and newlines are data, never YAML syntax.
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
