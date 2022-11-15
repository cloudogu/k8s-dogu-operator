package limit

import (
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudogu/cesapp-lib/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

func TestNewDoguDeploymentLimitPatcher(t *testing.T) {
	// given
	regMock := &mocks.Registry{}

	// when
	patcher := NewDoguDeploymentLimitPatcher(regMock)

	// then
	assert.NotNil(t, patcher)
}

func Test_doguDeploymentLimitPatch_RetrieveMemoryLimits(t *testing.T) {
	// given
	doguResource := &k8sv1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "testDogu"},
	}

	t.Run("return error when retrieving cpu limit key", func(t *testing.T) {
		// given
		regMock := &mocks.Registry{}
		testDoguRegistry := &mocks.ConfigurationContext{}
		regMock.On("DoguConfig", "testDogu").Return(testDoguRegistry)

		testDoguRegistry.On("Get", cpuLimitKey).Return("", assert.AnError)

		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		// when
		_, err := patcher.RetrievePodLimits(doguResource)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		mock.AssertExpectationsForObjects(t, regMock, testDoguRegistry)
	})

	t.Run("return error when retrieving memory limit key", func(t *testing.T) {
		// given
		regMock := &mocks.Registry{}
		testDoguRegistry := &mocks.ConfigurationContext{}
		regMock.On("DoguConfig", "testDogu").Return(testDoguRegistry)

		testDoguRegistry.On("Get", cpuLimitKey).Return("100m", nil)
		testDoguRegistry.On("Get", memoryLimitKey).Return("", assert.AnError)

		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		// when
		_, err := patcher.RetrievePodLimits(doguResource)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		mock.AssertExpectationsForObjects(t, regMock, testDoguRegistry)
	})

	t.Run("return error when retrieving ephemeral storage limit key", func(t *testing.T) {
		// given
		regMock := &mocks.Registry{}
		testDoguRegistry := &mocks.ConfigurationContext{}
		regMock.On("DoguConfig", "testDogu").Return(testDoguRegistry)

		testDoguRegistry.On("Get", cpuLimitKey).Return("100m", nil)
		testDoguRegistry.On("Get", memoryLimitKey).Return("1Gi", nil)
		testDoguRegistry.On("Get", ephemeralStorageLimitKey).Return("", assert.AnError)

		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		// when
		_, err := patcher.RetrievePodLimits(doguResource)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		mock.AssertExpectationsForObjects(t, regMock, testDoguRegistry)
	})

	t.Run("receives error when parsing memory limits", func(t *testing.T) {
		// given
		regMock := mocks.NewRegistry(t)
		testDoguRegistry := mocks.NewConfigurationContext(t)
		regMock.On("DoguConfig", "testDogu").Return(testDoguRegistry)

		testDoguRegistry.On("Get", cpuLimitKey).Return("100m", nil)
		testDoguRegistry.On("Get", memoryLimitKey).Return("12e890uq209er", nil)

		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		// when
		_, err := patcher.RetrievePodLimits(doguResource)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse memory limit '12e890uq209er'")
	})

	t.Run("receives error when parsing cpu limits", func(t *testing.T) {
		// given
		regMock := &mocks.Registry{}
		testDoguRegistry := &mocks.ConfigurationContext{}
		regMock.On("DoguConfig", "testDogu").Return(testDoguRegistry)

		testDoguRegistry.On("Get", cpuLimitKey).Return("12e890uq209er", nil)

		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		// when
		_, err := patcher.RetrievePodLimits(doguResource)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse cpu limit '12e890uq209er'")
	})

	t.Run("receives error when parsing ephemeral storage limits", func(t *testing.T) {
		// given
		regMock := &mocks.Registry{}
		testDoguRegistry := &mocks.ConfigurationContext{}
		regMock.On("DoguConfig", "testDogu").Return(testDoguRegistry)

		testDoguRegistry.On("Get", cpuLimitKey).Return("100m", nil)
		testDoguRegistry.On("Get", memoryLimitKey).Return("3Gi", nil)
		testDoguRegistry.On("Get", ephemeralStorageLimitKey).Return("12e890uq209er", nil)

		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		// when
		_, err := patcher.RetrievePodLimits(doguResource)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse ephemeral storage limit '12e890uq209er'")
	})

	t.Run("successfully create limits with some of the keys", func(t *testing.T) {
		// given
		regMock := &mocks.Registry{}
		testDoguRegistry := &mocks.ConfigurationContext{}
		regMock.On("DoguConfig", "testDogu").Return(testDoguRegistry)

		testDoguRegistry.On("Get", cpuLimitKey).Return("100m", nil)
		testDoguRegistry.On("Get", memoryLimitKey).Return("", nil)
		testDoguRegistry.On("Get", ephemeralStorageLimitKey).Return("4Gi", nil)

		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		// when
		doguLimitObject, err := patcher.RetrievePodLimits(doguResource)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, regMock, testDoguRegistry)

		assert.Equal(t, "100m", doguLimitObject.CpuLimit().String())
		assert.Equal(t, "4Gi", doguLimitObject.EphemeralStorageLimit().String())
	})

	t.Run("successfully create limits with all keys", func(t *testing.T) {
		// given
		regMock := &mocks.Registry{}
		testDoguRegistry := &mocks.ConfigurationContext{}
		regMock.On("DoguConfig", "testDogu").Return(testDoguRegistry)

		testDoguRegistry.On("Get", cpuLimitKey).Return("100m", nil)
		testDoguRegistry.On("Get", memoryLimitKey).Return("1Gi", nil)
		testDoguRegistry.On("Get", ephemeralStorageLimitKey).Return("4Gi", nil)

		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		// when
		doguLimitObject, err := patcher.RetrievePodLimits(doguResource)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, regMock, testDoguRegistry)

		assert.Equal(t, "100m", doguLimitObject.CpuLimit().String())
		assert.Equal(t, "1Gi", doguLimitObject.MemoryLimit().String())
		assert.Equal(t, "4Gi", doguLimitObject.EphemeralStorageLimit().String())
	})
}

func Test_doguDeploymentLimitPatcher_PatchDeployment(t *testing.T) {
	t.Run("patch deployment without containers", func(t *testing.T) {
		// given
		deployment := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{}}},
		}

		regMock := &mocks.Registry{}
		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		// when
		err := patcher.PatchDeployment(deployment, &doguLimits{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "given deployment cannot be patched, no containers are defined, at least one container is required")
	})

	t.Run("successful patch resources with one limit", func(t *testing.T) {
		// given
		deployment := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "testContainer"},
				},
			}}},
		}

		regMock := &mocks.Registry{}
		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		cpuLimit, err := resource.ParseQuantity("100m")
		require.NoError(t, err)
		doguLimits := &doguLimits{
			cpuLimit: &cpuLimit,
		}

		// when
		err = patcher.PatchDeployment(deployment, doguLimits)

		// then
		require.NoError(t, err)

		cpuLimitQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU]
		cpuRequestQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU]
		assert.Equal(t, cpuLimitQuantity.String(), "100m")
		assert.Equal(t, cpuRequestQuantity.String(), "100m")
	})

	t.Run("successful patch resources with multiple limits", func(t *testing.T) {
		// given
		deployment := &appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "testContainer"},
				},
			}}},
		}

		regMock := &mocks.Registry{}
		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		cpuLimit, err := resource.ParseQuantity("100m")
		require.NoError(t, err)
		memoryLimit, err := resource.ParseQuantity("2Gi")
		require.NoError(t, err)
		ephemeralStorageLimit, err := resource.ParseQuantity("4Gi")
		require.NoError(t, err)
		doguLimits := &doguLimits{
			cpuLimit:              &cpuLimit,
			memoryLimit:           &memoryLimit,
			ephemeralStorageLimit: &ephemeralStorageLimit,
		}

		// when
		err = patcher.PatchDeployment(deployment, doguLimits)

		// then
		require.NoError(t, err)

		cpuLimitQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceCPU]
		cpuRequestQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU]
		assert.Equal(t, cpuLimitQuantity.String(), "100m")
		assert.Equal(t, cpuRequestQuantity.String(), "100m")

		memoryLimitQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory]
		memoryRequestQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceMemory]
		assert.Equal(t, memoryLimitQuantity.String(), "2Gi")
		assert.Equal(t, memoryRequestQuantity.String(), "2Gi")

		empheralStorageLimitQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceEphemeralStorage]
		empheralStorageResourceQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceEphemeralStorage]
		assert.Equal(t, empheralStorageLimitQuantity.String(), "4Gi")
		assert.Equal(t, empheralStorageResourceQuantity.String(), "4Gi")
	})
}
