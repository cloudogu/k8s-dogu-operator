package resource

import (
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"strings"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
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

type requirementsGenerator struct {
	configRegistry registry.Registry
}

func NewRequirementsGenerator(configRegistry registry.Registry) *requirementsGenerator {
	return &requirementsGenerator{configRegistry: configRegistry}
}

func (r requirementsGenerator) Generate(dogu *core.Dogu) (corev1.ResourceRequirements, error) {
	doguConfig := r.configRegistry.DoguConfig(dogu.GetSimpleName())

	requirements := corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{},
		Requests: corev1.ResourceList{},
	}

	var errList []error
	for resourceType := range resourceTypeMapping {
		err := appendRequirementsForResourceType(resourceType, requirements, doguConfig, dogu)
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

func readFromConfigOrDefault(key string, doguConfig registry.ConfigurationContext, dogu *core.Dogu) (string, error) {
	configValue, err := doguConfig.Get(key)

	if err != nil {
		if registry.IsKeyNotFoundError(err) {
			for _, field := range dogu.Configuration {
				if field.Name == key {
					return field.Default, nil
				}
			}

			return "", nil
		}

		return "", fmt.Errorf("failed to read value of key '%s' from registry config of dogu '%s': %w", key, dogu.Name, err)
	}

	return configValue, nil
}

func appendRequirementsForResourceType(resourceType resourceType, requirements corev1.ResourceRequirements, doguConfig registry.ConfigurationContext, dogu *core.Dogu) error {
	resourceName := resourceTypeMapping[resourceType]

	limitKey := fmt.Sprintf("container_config/%s_limit", resourceType)
	limit, limitErr := readFromConfigOrDefault(limitKey, doguConfig, dogu)
	var limitConversionErr error
	if limit != "" {
		requirements.Limits[resourceName], limitConversionErr = convertCesUnitToQuantity(limit, resourceType)
	}

	requestKey := fmt.Sprintf("container_config/%s_request", resourceType)
	request, requestErr := readFromConfigOrDefault(requestKey, doguConfig, dogu)
	var requestConversionErr error
	if request != "" {
		requirements.Requests[resourceName], requestConversionErr = convertCesUnitToQuantity(request, resourceType)
	}

	err := errors.Join(limitErr, limitConversionErr, requestErr, requestConversionErr)
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
