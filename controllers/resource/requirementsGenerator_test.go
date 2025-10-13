package resource

import (
	"context"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-registry-lib/config"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
	t.Run("should not set limit or request when no value configures", func(t *testing.T) {
		requirements := corev1.ResourceRequirements{
			Limits:   corev1.ResourceList{},
			Requests: corev1.ResourceList{},
		}

		entries := config.Entries{}
		doguConfig := config.CreateDoguConfig("test", entries)

		err := appendRequirementsForResourceType(context.TODO(), memoryType, requirements, doguConfig, &core.Dogu{})
		assert.NoError(t, err)
		assert.Empty(t, requirements.Limits)
		assert.Empty(t, requirements.Requests)
	})
}

func Test_readFromConfigOrDefault(t *testing.T) {
	t.Run("should return default-config from dogu when not configured ", func(t *testing.T) {
		key := "container_config/memory_limit"

		doguConfig := config.CreateDoguConfig("test", config.Entries{})

		val := readFromConfigOrDefault(key, doguConfig, &core.Dogu{
			Configuration: []core.ConfigurationField{
				{
					Name:    key,
					Default: "500k",
				},
			},
		})
		assert.Equal(t, "500k", val)
	})

	t.Run("should return empty-string when not configured and no default-value is present", func(t *testing.T) {
		key := "container_config/memory_limit"

		doguConfig := config.CreateDoguConfig("test", config.Entries{})

		val := readFromConfigOrDefault(key, doguConfig, &core.Dogu{
			Configuration: []core.ConfigurationField{
				{
					Name:    "something/other",
					Default: "40g",
				},
			},
		})
		assert.Empty(t, val)
	})

	t.Run("should return configured-value when present", func(t *testing.T) {
		key := "container_config/memory_limit"

		doguConfig := config.CreateDoguConfig("test", config.Entries{
			config.Key(key): "20g",
		})

		val := readFromConfigOrDefault(key, doguConfig, &core.Dogu{})
		assert.Equal(t, "20g", val)
	})
}

func Test_requirementsGenerator_Generate(t *testing.T) {
	t.Run("should fail to generate requirements for error in configRegistry", func(t *testing.T) {
		dogu := &core.Dogu{
			Name: "official/ldap",
		}

		doguConfig := config.CreateDoguConfig("test", config.Entries{
			"container_config/memory_limit":   "500ß",
			"container_config/cpu_core_limit": "",
		})

		doguConfigRepoMock := NewMockDoguConfigRepository(t)
		doguConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguConfig, nil)

		generator := NewRequirementsGenerator(doguConfigRepoMock)

		requirements, err := generator.Generate(context.TODO(), dogu)
		assert.ErrorContains(t, err, "errors occured during requirements generation")
		assert.ErrorContains(t, err, "errors occured while appending requirements for resource type 'memory'")
		require.ErrorContains(t, err, "failed to convert ces unit")
		assert.Empty(t, requirements)
	})

	t.Run("should succeed to generate requirements", func(t *testing.T) {
		dogu := &core.Dogu{
			Name: "official/ldap",
		}

		doguConfig := config.CreateDoguConfig("test", config.Entries{
			"container_config/memory_limit":     "500m",
			"container_config/memory_request":   "200m",
			"container_config/cpu_core_request": "0.5",
			"container_config/cpu_core_limit":   "2",
			"container_config/storage_limit":    "20g",
			"container_config/storage_request":  "1g",
		})

		doguConfigRepoMock := NewMockDoguConfigRepository(t)
		doguConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguConfig, nil)

		generator := NewRequirementsGenerator(doguConfigRepoMock)

		requirements, err := generator.Generate(context.TODO(), dogu)
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
