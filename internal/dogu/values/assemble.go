package values

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/strvals"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	globalConfigName     = "global-config"
	globalConfigFileName = "config.yaml"
)

type Mapping struct {
	Path    string            `yaml:"path"`
	Mapping map[string]string `yaml:"mapping"`
}

type MetaValue struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Keys        []Mapping
}

type MetadataMapping struct {
	ApiVersion string               `yaml:"apiVersion"`
	Metavalues map[string]MetaValue `yaml:"metavalues"`
}

type Values = map[string]any

// Assembler combines values in increasing precedence: global.cesConfig, runtime patches,
// spec.values, spec.mappedValues. Helm applies the chart defaults separately.
type Assembler struct {
	k8s client.Client
}

// NewAssembler creates an Assembler that reads the global config from the cluster via the given client.
func NewAssembler(k8s client.Client) *Assembler {
	return &Assembler{k8s: k8s}
}

// Assemble reads global configuration once, renders the optional runtime patch template and
// merges the value sources. A nil runtimePatchTemplate means the file is absent.
func (a Assembler) Assemble(ctx context.Context, cr *v3beta1.Dogu, valuesMeta, runtimePatchTemplate []byte) (Values, error) {
	crValues, err := getDoguCRValues(cr)
	if err != nil {
		return nil, fmt.Errorf("failed to dogu spec values: %w", err)
	}

	doguMetaValues, err := getDoguMetaValues(ctx, cr, valuesMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to get values from dogu values meta: %w", err)
	}

	globalConfigValues, err := getGlobalConfigValues(ctx, cr, a.k8s)
	if err != nil {
		return nil, fmt.Errorf("failed to get values from global config: %w", err)
	}

	runtimeValues, err := renderRuntimePatches(runtimePatchTemplate, globalConfigValues)
	if err != nil {
		return nil, fmt.Errorf("failed to render %s: %w", RuntimePatchTemplateFileName, err)
	}

	// Place global config under global.cesConfig.
	nestedGlobalConfig := Values{"global": Values{"cesConfig": globalConfigValues}}

	// Later sources win: global config < runtime patches < CR values < mapped values.
	finalValues := make(Values)
	for _, source := range []struct {
		name   string
		values Values
	}{
		{"global.cesConfig", nestedGlobalConfig},
		{"runtimePatches", runtimeValues},
		{"spec.values", crValues},
		{"spec.mappedValues", doguMetaValues},
	} {
		finalValues = mergeValues(ctx, finalValues, source.values, source.name, "")
	}

	return finalValues, nil
}

func getDoguCRValues(cr *v3beta1.Dogu) (Values, error) {
	var doguCRValues Values
	if pErr := yaml.Unmarshal(cr.Spec.Values.Raw, &doguCRValues); pErr != nil {
		return nil, fmt.Errorf("failed to parse values from dogu spec: %w", pErr)
	}

	return doguCRValues, nil
}

func getDoguMetaValues(ctx context.Context, cr *v3beta1.Dogu, valuesMeta []byte) (Values, error) {
	mappedValues := make(Values)

	if len(cr.Spec.MappedValues) == 0 {
		return mappedValues, nil
	}

	var mappings MetadataMapping
	if mErr := yaml.Unmarshal(valuesMeta, &mappings); mErr != nil {
		return nil, fmt.Errorf("failed to unmarshal mapping meta data: %w", mErr)
	}

	for k, v := range cr.Spec.MappedValues {
		metaValue, ok := mappings.Metavalues[k]
		if !ok {
			continue
		}

		applyMappedValueKeys(ctx, metaValue.Keys, v, mappedValues)
	}

	return mappedValues, nil
}

// applyMappedValueKeys resolves each mapped key for the given CR value and parses the resulting
// "path=value" expressions into mappedValues. Parsing and lookup errors are logged, not returned.
func applyMappedValueKeys(ctx context.Context, keys []Mapping, v string, mappedValues Values) {
	for _, key := range keys {
		var mappedValue string

		if key.Mapping == nil {
			mappedValue = v
		} else {
			val, ok := key.Mapping[v]
			if !ok {
				logging.Error(ctx, errors.New("missing mapping"), "no Mapping found in mapping meta data", "key", v)
			}
			mappedValue = val
		}

		strval := fmt.Sprintf("%s=%s", key.Path, mappedValue)
		if pErr := strvals.ParseInto(strval, mappedValues); pErr != nil {
			logging.Error(ctx, pErr, "error parsing key path from mapping meta data", "path", key.Path)
		}
	}
}

func getGlobalConfigValues(ctx context.Context, cr *v3beta1.Dogu, s client.Client) (Values, error) {
	gcm := &coreV1.ConfigMap{}
	if gErr := s.Get(ctx, types.NamespacedName{Namespace: cr.Namespace, Name: globalConfigName}, gcm); gErr != nil {
		return nil, fmt.Errorf("failed to get global config: %w", gErr)
	}

	cfgYamlStr, ok := gcm.Data[globalConfigFileName]
	if !ok {
		return nil, fmt.Errorf("did not found key %s in %s", globalConfigFileName, globalConfigName)
	}

	globalCfgValues := make(Values)
	if pErr := yaml.Unmarshal([]byte(cfgYamlStr), globalCfgValues); pErr != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", globalConfigName, pErr)
	}

	return globalCfgValues, nil
}

// mergeValues merges higher-priority values into a copy of lower-priority values.
// Maps merge recursively; lists and scalars replace earlier values. A YAML null (Go nil)
// removes an existing non-nil value, but is retained when the key was absent or already nil.
// Only paths and the source name are logged: Helm's merge warnings can expose config contents.
func mergeValues(ctx context.Context, lower, higher Values, source, prefix string) Values {
	mergedValues := maps.Clone(lower)
	if mergedValues == nil {
		mergedValues = make(Values)
	}

	for key, value := range higher {
		previous, exists := mergedValues[key]
		path := prefix + key
		lowerMap, lowerIsMap := previous.(Values)
		higherMap, higherIsMap := value.(Values)
		if lowerIsMap && higherIsMap {
			mergedValues[key] = mergeValues(ctx, lowerMap, higherMap, source, path+".")
			continue
		}

		if exists {
			logging.Info(ctx, "Values overlap; higher-priority source wins", "path", path, "source", source)
		}
		if value == nil && exists && previous != nil {
			delete(mergedValues, key)
		} else {
			mergedValues[key] = value
		}
	}

	return mergedValues
}
