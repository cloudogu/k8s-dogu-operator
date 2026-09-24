package values

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chartutil"
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

type Values map[string]any

type Assembler struct {
	k8s client.Client
}

func (a Assembler) Assemble(ctx context.Context, cr *v3beta1.Dogu, patchTpl []byte) (Values, error) {
	logger := log.FromContext(ctx)

	crValues, err := getDoguCRValues(cr)
	if err != nil {
		return nil, fmt.Errorf("failed to dogu spec values: %w", err)
	}

	doguMetaValues, err := getDoguMetaValues(cr, patchTpl, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get values from dogu values meta: %w", err)
	}

	globalConfigValues, err := getGlobalConfigValues(ctx, cr, a.k8s)
	if err != nil {
		return nil, fmt.Errorf("failed to get values from global config:%w", err)
	}

	finalValues := mergeValues(crValues, globalConfigValues, doguMetaValues)

	return finalValues, nil
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
		if _, ok := mappings.Metavalues[k]; !ok {
			continue
		}

		for _, key := range mappings.Metavalues[k].Keys {
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

	return mappedValues, nil
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

// mergeValues merges maps in order. Later Values override earlier Values.
func mergeValues(values ...Values) Values {
	mergedValues := make(Values)

	for _, vMap := range values {
		if vMap == nil {
			continue
		}
		// Merge current map on top of accumulated result
		mergedValues = chartutil.CoalesceTables(vMap, mergedValues)
	}

	return mergedValues
}
