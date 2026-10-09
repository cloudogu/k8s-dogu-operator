package values

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/strvals"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

type Assembler struct {
	k8s client.Client
}

// NewAssembler creates an Assembler that reads the global config from the cluster via the given client.
func NewAssembler(k8s client.Client) *Assembler {
	return &Assembler{k8s: k8s}
}

func (a Assembler) Assemble(ctx context.Context, cr *v3beta1.Dogu, valuesMeta, patchTemplate []byte) (Values, error) {
	logger := log.FromContext(ctx)

	crValues, err := getDoguCRValues(cr)
	if err != nil {
		return nil, fmt.Errorf("failed to dogu spec values: %w", err)
	}

	doguMetaValues, err := getDoguMetaValues(cr, valuesMeta, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get values from dogu values meta: %w", err)
	}

	globalConfigValues, err := getGlobalConfigValues(ctx, cr, a.k8s)
	if err != nil {
		return nil, fmt.Errorf("failed to get values from global config: %w", err)
	}

	runtimeValues, err := renderRuntimePatches(patchTemplate, globalConfigValues)
	if err != nil {
		return nil, fmt.Errorf("failed to render chart-patch-tpl.yaml runtime patches: %w", err)
	}

	// Place global config under global.cesConfig.
	nestedGlobalConfig := Values{"global": Values{"cesConfig": globalConfigValues}}

	// Later sources win: global config < runtime patches < CR values < mapped values.
	finalValues := nestedGlobalConfig
	for _, source := range []struct {
		name   string
		values Values
	}{
		{"runtimePatches", runtimeValues},
		{"spec.values", crValues},
		{"spec.mappedValues", doguMetaValues},
	} {
		logValueOverlaps(logger, finalValues, source.values, source.name, "")
		finalValues = mergeValues(finalValues, source.values)
	}

	return finalValues, nil
}

// Log paths, never configuration contents, when a higher-priority source overrides values.
func logValueOverlaps(logger logr.Logger, lower, higher Values, source, prefix string) {
	for key, value := range higher {
		previous, exists := lower[key]
		if !exists {
			continue
		}
		path := prefix + key
		lowMap, lowOK := previous.(map[string]any)
		highMap, highOK := value.(map[string]any)
		if lowOK && highOK {
			logValueOverlaps(logger, lowMap, highMap, source, path+".")
		} else {
			logger.Info("Values overlap; higher-priority source wins", "path", path, "source", source)
		}
	}
}

func getDoguCRValues(cr *v3beta1.Dogu) (Values, error) {
	var doguCRValues Values
	if pErr := yaml.Unmarshal(cr.Spec.Values.Raw, &doguCRValues); pErr != nil {
		return nil, fmt.Errorf("failed to parse values from dogu spec: %w", pErr)
	}

	return doguCRValues, nil
}

func getDoguMetaValues(cr *v3beta1.Dogu, patchTpl []byte, logger logr.Logger) (Values, error) {
	mappedValues := make(Values)

	if len(cr.Spec.MappedValues) == 0 {
		return mappedValues, nil
	}

	var mappings MetadataMapping
	if mErr := yaml.Unmarshal(patchTpl, &mappings); mErr != nil {
		return nil, fmt.Errorf("failed to unmarshal mapping meta data: %w", mErr)
	}

	for k, v := range cr.Spec.MappedValues {
		metaValue, ok := mappings.Metavalues[k]
		if !ok {
			continue
		}

		applyMappedValueKeys(metaValue.Keys, v, mappedValues, logger)
	}

	return mappedValues, nil
}

// applyMappedValueKeys resolves each mapped key for the given CR value and parses the resulting
// "path=value" expressions into mappedValues. Parsing and lookup errors are logged, not returned.
func applyMappedValueKeys(keys []Mapping, v string, mappedValues Values, logger logr.Logger) {
	for _, key := range keys {
		var mappedValue string

		if key.Mapping == nil {
			mappedValue = v
		} else {
			val, ok := key.Mapping[v]
			if !ok {
				logger.Error(errors.New("missing mapping"), "no Mapping found in mapping meta data", "key", v)
			}
			mappedValue = val
		}

		strval := fmt.Sprintf("%s=%s", key.Path, mappedValue)
		if pErr := strvals.ParseInto(strval, mappedValues); pErr != nil {
			logger.Error(pErr, "error parsing key path from mapping meta data", "path", key.Path)
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

// mergeValues merges maps in order without Helm's value-bearing conflict warnings.
// Later Values win; maps merge recursively and lists replace earlier lists.
// A null removes an existing non-null key, but otherwise remains an explicit null.
func mergeValues(values ...Values) Values {
	mergedValues := make(Values)

	for _, vMap := range values {
		for key, value := range vMap {
			previous, exists := mergedValues[key]
			if value == nil && exists && previous != nil {
				delete(mergedValues, key)
				continue
			}
			lower, lowOK := previous.(Values)
			higher, highOK := value.(Values)
			if lowOK && highOK {
				mergedValues[key] = mergeValues(lower, higher)
			} else {
				mergedValues[key] = value
			}
		}
	}

	return mergedValues
}
