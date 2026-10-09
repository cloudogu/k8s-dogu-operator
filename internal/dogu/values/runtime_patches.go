package values

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

var runtimeRootKey = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_.-]*):(?:[ \t]*(.*))?$`)
var runtimeDocumentStart = regexp.MustCompile(`^---(?:[ \t]+(?:#.*)?)?$`)

// runtimePatchSection isolates only the literal runtimePatches section. The
// envelope deliberately supports block-style, unquoted root keys only; template
// actions at column zero within the section are retained, not section boundaries.
func runtimePatchSection(source []byte) ([]byte, error) {
	lines := strings.SplitAfter(string(source), "\n")
	firstContent := slices.IndexFunc(lines, func(line string) bool {
		return strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "#")
	})
	start, end := -1, len(lines)
	api := ""
	seen := map[string]bool{}
	for i, line := range lines {
		plain := strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if strings.TrimSpace(plain) == "" || strings.HasPrefix(plain, "#") || strings.HasPrefix(plain, " ") || strings.HasPrefix(plain, "\t") {
			continue
		}
		if i == firstContent && runtimeDocumentStart.MatchString(plain) {
			continue
		}
		if strings.HasPrefix(plain, "{{") {
			if start >= 0 && end == len(lines) {
				continue
			}
			return nil, fmt.Errorf("chart patch envelope: root template actions outside runtimePatches are unsupported")
		}
		match := runtimeRootKey.FindStringSubmatch(plain)
		if match == nil {
			return nil, fmt.Errorf("chart patch envelope: expected literal block-style root key at line %d", i+1)
		}
		key := match[1]
		if seen[key] {
			return nil, fmt.Errorf("chart patch envelope: duplicate %s section", key)
		}
		seen[key] = true
		if start >= 0 && end == len(lines) {
			end = i
		}
		if key == "runtimePatches" {
			start = i
			value := strings.TrimSpace(strings.SplitN(match[2], "#", 2)[0])
			if value != "" && value != "{}" {
				return nil, fmt.Errorf("runtimePatches must be a block mapping or {}")
			}
		}
		if key == "apiVersion" {
			if err := yaml.Unmarshal([]byte(match[2]), &api); err != nil {
				return nil, fmt.Errorf("chart patch apiVersion: %w", err)
			}
		}
	}
	if len(seen) == 0 {
		return nil, fmt.Errorf("chart patch envelope is empty")
	}
	if start < 0 {
		return nil, nil
	}
	if api != "v1" {
		return nil, fmt.Errorf("runtimePatches requires apiVersion v1")
	}
	return []byte(strings.Join(lines[start:end], "")), nil
}

func decodeRuntimePatches(output []byte) (Values, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(output))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode runtimePatches: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("runtimePatches must contain exactly one YAML document")
	}
	if err := validateRuntimeNode(&document); err != nil {
		return nil, err
	}
	var envelope Values
	if err := document.Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode runtimePatches mapping: %w", err)
	}
	if len(envelope) != 1 {
		return nil, fmt.Errorf("runtimePatches template must not generate additional root sections")
	}
	targets, ok := envelope["runtimePatches"].(Values)
	if !ok {
		return nil, fmt.Errorf("runtimePatches must be a mapping")
	}
	result := Values{}
	for target, value := range targets {
		if target != "values.yaml" {
			return nil, fmt.Errorf("unsupported runtimePatches target %q", target)
		}
		if result, ok = value.(Values); !ok {
			return nil, fmt.Errorf("runtimePatches values.yaml must be a mapping")
		}
	}
	if _, err := json.Marshal(result); err != nil {
		return nil, fmt.Errorf("runtimePatches JSON conversion: %w", err)
	}
	return result, nil
}

func validateRuntimeNode(node *yaml.Node) error {
	if node.Kind == yaml.AliasNode || node.Anchor != "" {
		return fmt.Errorf("runtimePatches aliases and anchors are unsupported")
	}
	if node.Kind == yaml.MappingNode {
		// Node.Decode rejects duplicate keys; enforce our additional string-key policy here.
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
				return fmt.Errorf("runtimePatches mapping keys must be strings")
			}
		}
	}
	for _, child := range node.Content {
		if err := validateRuntimeNode(child); err != nil {
			return err
		}
	}
	return nil
}
