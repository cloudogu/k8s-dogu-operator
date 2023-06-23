package resource

import (
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.etcd.io/etcd/client/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"
)

func Test_convertCesUnitToQuantity(t *testing.T) {
	tests := []struct {
		name         string
		cesUnit      string
		resourceType resourceType
		want         resource.Quantity
		wantErr      func(t *testing.T, err error)
	}{
		{
			"should succeed to convert CPU quantity",
			"0.5",
			cpuCoreType,
			resource.MustParse("0.5"),
			func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			"should fail to convert CPU quantity for wrong suffix",
			"0.5TTT",
			cpuCoreType,
			resource.Quantity{},
			func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "failed to convert cpu cores with value '0.5TTT' to quantity")
				assert.ErrorContains(t, err, "unable to parse quantity's suffix")
			},
		},
		{
			"should succeed to convert Memory quantity as bytes",
			"100b",
			memoryType,
			resource.MustParse("100"),
			func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			"should succeed to convert Memory quantity as kiloBytes",
			"200k",
			memoryType,
			resource.MustParse("200Ki"),
			func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			"should succeed to convert Memory quantity as megaBytes",
			"300m",
			memoryType,
			resource.MustParse("300Mi"),
			func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			"should succeed to convert Memory quantity as gigabytes",
			"400g",
			memoryType,
			resource.MustParse("400Gi"),
			func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			"should fail to convert Memory quantity with wrong suffix",
			"500gg",
			memoryType,
			resource.Quantity{},
			func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "failed to convert ces unit '500Gig' of type 'memory' to quantity")
				assert.ErrorContains(t, err, "quantities must match the regular expression")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertCesUnitToQuantity(tt.cesUnit, tt.resourceType)
			tt.wantErr(t, err)
			assert.Equalf(t, tt.want, got, "convertCesUnitToQuantity(%v, %v)", tt.cesUnit, tt.resourceType)
		})
	}
}

func Test_appendRequirementsForResourceType(t *testing.T) {

	t.Run("should fail to read configured limit", func(t *testing.T) {
		requirements := corev1.ResourceRequirements{
			Limits:   corev1.ResourceList{},
			Requests: corev1.ResourceList{},
		}
		configurationContext := mocks.NewConfigurationContext(t)
		configurationContext.EXPECT().Get(fmt.Sprintf("container_config/%s_limit", memoryType)).Return("", assert.AnError)
		configurationContext.EXPECT().Get(fmt.Sprintf("container_config/%s_request", memoryType)).Return("200m", nil)

		err := appendRequirementsForResourceType(memoryType, requirements, configurationContext, &core.Dogu{Name: "official/ldap"})
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "errors occured while appending requirements for resource type 'memory'")
		assert.ErrorContains(t, err, "failed to read value of key 'container_config/memory_limit' from registry config of dogu 'official/ldap'")
		assert.Empty(t, requirements.Limits)
		assert.Equal(t, resource.MustParse("200Mi"), requirements.Requests[corev1.ResourceMemory])
	})

	t.Run("should fail to read configured request", func(t *testing.T) {
		requirements := corev1.ResourceRequirements{
			Limits:   corev1.ResourceList{},
			Requests: corev1.ResourceList{},
		}
		configurationContext := mocks.NewConfigurationContext(t)
		configurationContext.EXPECT().Get(fmt.Sprintf("container_config/%s_limit", memoryType)).Return("200m", nil)
		configurationContext.EXPECT().Get(fmt.Sprintf("container_config/%s_request", memoryType)).Return("", assert.AnError)

		err := appendRequirementsForResourceType(memoryType, requirements, configurationContext, &core.Dogu{Name: "official/ldap"})
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "errors occured while appending requirements for resource type 'memory'")
		assert.ErrorContains(t, err, "failed to read value of key 'container_config/memory_request' from registry config of dogu 'official/ldap'")
		assert.Equal(t, resource.MustParse("200Mi"), requirements.Limits[corev1.ResourceMemory])
		assert.Empty(t, requirements.Requests)
	})

	t.Run("should fail to read configured limit and request", func(t *testing.T) {
		requirements := corev1.ResourceRequirements{
			Limits:   corev1.ResourceList{},
			Requests: corev1.ResourceList{},
		}
		configurationContext := mocks.NewConfigurationContext(t)
		configurationContext.EXPECT().Get(fmt.Sprintf("container_config/%s_limit", memoryType)).Return("", fmt.Errorf("test error limit memory"))
		configurationContext.EXPECT().Get(fmt.Sprintf("container_config/%s_request", memoryType)).Return("", fmt.Errorf("test error request memory"))

		err := appendRequirementsForResourceType(memoryType, requirements, configurationContext, &core.Dogu{Name: "official/ldap"})
		assert.ErrorContains(t, err, "errors occured while appending requirements for resource type 'memory'")
		assert.ErrorContains(t, err, "failed to read value of key 'container_config/memory_limit' from registry config of dogu 'official/ldap'")
		assert.ErrorContains(t, err, "test error limit memory")
		assert.ErrorContains(t, err, "test error request memory")
		assert.Empty(t, requirements.Limits)
		assert.Empty(t, requirements.Requests)
	})

	t.Run("should not set limit or request when no value configures", func(t *testing.T) {
		requirements := corev1.ResourceRequirements{
			Limits:   corev1.ResourceList{},
			Requests: corev1.ResourceList{},
		}
		configurationContext := mocks.NewConfigurationContext(t)
		configurationContext.EXPECT().Get(fmt.Sprintf("container_config/%s_limit", memoryType)).Return("", nil)
		configurationContext.EXPECT().Get(fmt.Sprintf("container_config/%s_request", memoryType)).Return("", nil)

		err := appendRequirementsForResourceType(memoryType, requirements, configurationContext, &core.Dogu{})
		assert.NoError(t, err)
		assert.Empty(t, requirements.Limits)
		assert.Empty(t, requirements.Requests)
	})
}

func Test_readFromConfigOrDefault(t *testing.T) {
	t.Run("should fail to read config", func(t *testing.T) {
		key := "container_config/memory_limit"

		configurationContext := mocks.NewConfigurationContext(t)
		configurationContext.EXPECT().Get(key).Return("", assert.AnError)

		val, err := readFromConfigOrDefault(key, configurationContext, &core.Dogu{Name: "official/ldap"})
		require.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to read value of key 'container_config/memory_limit' from registry config of dogu 'official/ldap'")
		assert.Empty(t, val)
	})

	t.Run("should return default-config from dogu when not configured ", func(t *testing.T) {
		key := "container_config/memory_limit"

		configurationContext := mocks.NewConfigurationContext(t)
		configurationContext.EXPECT().Get(key).Return("", client.Error{Code: client.ErrorCodeKeyNotFound})

		val, err := readFromConfigOrDefault(key, configurationContext, &core.Dogu{
			Configuration: []core.ConfigurationField{
				{
					Name:    key,
					Default: "500k",
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "500k", val)
	})

	t.Run("should return empty-string when not configured and no default-value is present", func(t *testing.T) {
		key := "container_config/memory_limit"

		configurationContext := mocks.NewConfigurationContext(t)
		configurationContext.EXPECT().Get(key).Return("", client.Error{Code: client.ErrorCodeKeyNotFound})

		val, err := readFromConfigOrDefault(key, configurationContext, &core.Dogu{
			Configuration: []core.ConfigurationField{
				{
					Name:    "something/other",
					Default: "40g",
				},
			},
		})
		require.NoError(t, err)
		assert.Empty(t, val)
	})

	t.Run("should return configured-value when present", func(t *testing.T) {
		key := "container_config/memory_limit"

		configurationContext := mocks.NewConfigurationContext(t)
		configurationContext.EXPECT().Get(key).Return("20g", nil)

		val, err := readFromConfigOrDefault(key, configurationContext, &core.Dogu{})
		require.NoError(t, err)
		assert.Equal(t, "20g", val)
	})
}

func Test_requirementsGenerator_Generate(t *testing.T) {
	t.Run("should fail to generate requirements for error in configRegistry", func(t *testing.T) {
		dogu := &core.Dogu{
			Name: "official/ldap",
		}

		configurationContext := mocks.NewConfigurationContext(t)
		configurationContext.EXPECT().Get("container_config/memory_limit").Return("", fmt.Errorf("error memory limit"))
		configurationContext.EXPECT().Get("container_config/cpu_core_limit").Return("", fmt.Errorf("error cpu_core limit"))
		configurationContext.EXPECT().Get(mock.Anything).Return("200m", nil)

		registry := mocks.NewConfigurationRegistry(t)
		registry.EXPECT().DoguConfig(dogu.GetSimpleName()).Return(configurationContext)

		generator := NewRequirementsGenerator(registry)

		requirements, err := generator.Generate(dogu)
		assert.ErrorContains(t, err, "errors occured during requirements generation")
		assert.ErrorContains(t, err, "errors occured while appending requirements for resource type 'memory'")
		assert.ErrorContains(t, err, "errors occured while appending requirements for resource type 'cpu_core'")
		require.ErrorContains(t, err, "error memory limit")
		require.ErrorContains(t, err, "error cpu_core limit")
		assert.Empty(t, requirements)
	})

	t.Run("should succeed to generate requirements", func(t *testing.T) {
		dogu := &core.Dogu{
			Name: "official/ldap",
		}

		configurationContext := mocks.NewConfigurationContext(t)
		configurationContext.EXPECT().Get("container_config/memory_limit").Return("500m", nil)
		configurationContext.EXPECT().Get("container_config/memory_request").Return("200m", nil)
		configurationContext.EXPECT().Get("container_config/cpu_core_request").Return("0.5", nil)
		configurationContext.EXPECT().Get("container_config/cpu_core_limit").Return("2", nil)
		configurationContext.EXPECT().Get("container_config/storage_limit").Return("20g", nil)
		configurationContext.EXPECT().Get("container_config/storage_request").Return("1g", nil)

		registry := mocks.NewConfigurationRegistry(t)
		registry.EXPECT().DoguConfig(dogu.GetSimpleName()).Return(configurationContext)

		generator := NewRequirementsGenerator(registry)

		requirements, err := generator.Generate(dogu)
		require.NoError(t, err)
		assert.Equal(t, corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory:           resource.MustParse("500Mi"),
				corev1.ResourceCPU:              resource.MustParse("2"),
				corev1.ResourceEphemeralStorage: resource.MustParse("20Gi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory:           resource.MustParse("200Mi"),
				corev1.ResourceCPU:              resource.MustParse("0.5"),
				corev1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
			},
		}, requirements)
	})
}
