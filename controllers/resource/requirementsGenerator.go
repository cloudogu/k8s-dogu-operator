package resource

import (
	"context"
	"errors"
	"fmt"
	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/k8s-registry-lib/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"strings"

	"github.com/cloudogu/cesapp-lib/core"
)

type resourceType string

const (
	memoryType  = resourceType("memory")
	cpuCoreType = resourceType("cpu_core")
	storageType = resourceType("storage")
)

var resourceTypeMapping = map[resourceType]corev1.ResourceName{
	memoryType:  corev1.ResourceMemory,
	cpuCoreType: corev1.ResourceCPU,
	storageType: corev1.ResourceEphemeralStorage,
}

type RequirementsGenerator struct {
	doguConfigGetter doguConfigGetter
}

func NewRequirementsGenerator(doguConfigGetter doguConfigGetter) *RequirementsGenerator {
	return &RequirementsGenerator{doguConfigGetter: doguConfigGetter}
}

func (r RequirementsGenerator) Generate(ctx context.Context, dogu *core.Dogu) (corev1.ResourceRequirements, error) {
	doguConfig, err := r.doguConfigGetter.Get(ctx, cescommons.SimpleName(dogu.GetSimpleName()))
	if err != nil {
		return corev1.ResourceRequirements{}, fmt.Errorf("unable to get config for dogu %s: %w", dogu.GetSimpleName(), err)
	}

	requirements := corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{},
		Requests: corev1.ResourceList{},
	}

	var errList []error
	for resourceType := range resourceTypeMapping {
		err := appendRequirementsForResourceType(ctx, resourceType, requirements, doguConfig, dogu)
		if err != nil {
			errList = append(errList, err)
		}
	}
	if len(errList) > 0 {
		return corev1.ResourceRequirements{},
			fmt.Errorf("errors occured during requirements generation: %w", errors.Join(errList...))
	}

	return requirements, nil
}

func readFromConfigOrDefault(key string, doguConfig config.DoguConfig, dogu *core.Dogu) string {
	configValue, ok := doguConfig.Get(config.Key(key))
	if !ok {
		for _, field := range dogu.Configuration {
			if field.Name == key {
				return field.Default
			}
		}

		return ""
	}

	return configValue.String()
}

func appendRequirementsForResourceType(_ context.Context, resourceType resourceType, requirements corev1.ResourceRequirements, doguConfig config.DoguConfig, dogu *core.Dogu) error {
	resourceName := resourceTypeMapping[resourceType]

	limitKey := fmt.Sprintf("container_config/%s_limit", resourceType)
	limit := readFromConfigOrDefault(limitKey, doguConfig, dogu)
	var limitConversionErr error
	if limit != "" {
		requirements.Limits[resourceName], limitConversionErr = convertCesUnitToQuantity(limit, resourceType)
	}

	requestKey := fmt.Sprintf("container_config/%s_request", resourceType)
	request := readFromConfigOrDefault(requestKey, doguConfig, dogu)
	var requestConversionErr error
	if request != "" {
		requirements.Requests[resourceName], requestConversionErr = convertCesUnitToQuantity(request, resourceType)
	}

	err := errors.Join(limitConversionErr, requestConversionErr)
	if err != nil {
		return fmt.Errorf("errors occured while appending requirements for resource type '%s': %w", resourceType, err)
	}

	return nil
}

func convertCesUnitToQuantity(cesUnit string, resourceType resourceType) (resource.Quantity, error) {
	if resourceType == cpuCoreType {
		quantity, err := resource.ParseQuantity(cesUnit)
		if err != nil {
			return resource.Quantity{},
				fmt.Errorf("failed to convert cpu cores with value '%s' to quantity: %w", cesUnit, err)
		}
		return quantity, nil
	}

	// otherwise this is a binary measurement and converted as follows
	// b->""
	// k->Ki
	// m->Mi
	// g->Gi
	cesUnit = strings.Replace(cesUnit, "b", "", 1)
	cesUnit = strings.Replace(cesUnit, "k", "Ki", 1)
	cesUnit = strings.Replace(cesUnit, "m", "Mi", 1)
	cesUnit = strings.Replace(cesUnit, "g", "Gi", 1)
	quantity, err := resource.ParseQuantity(cesUnit)
	if err != nil {
		return resource.Quantity{},
			fmt.Errorf("failed to convert ces unit '%s' of type '%s' to quantity: %w", cesUnit, resourceType, err)
	}
	return quantity, nil
}
