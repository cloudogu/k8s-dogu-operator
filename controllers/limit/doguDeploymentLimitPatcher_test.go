package limit

import (
	"github.com/cloudogu/cesapp-lib/registry/mocks"
	v13 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
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
	doguResource := &v13.Dogu{
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

	t.Run("successfully create limits with some of the keys", func(t *testing.T) {
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

		assert.Equal(t, "100m", doguLimitObject.CpuLimit)
		assert.Equal(t, "1Gi", doguLimitObject.MemoryLimit)
		assert.Equal(t, "4Gi", doguLimitObject.EphemeralStorageLimit)
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

		assert.Equal(t, "100m", doguLimitObject.CpuLimit)
		assert.Equal(t, "1Gi", doguLimitObject.MemoryLimit)
		assert.Equal(t, "4Gi", doguLimitObject.EphemeralStorageLimit)
	})
}

func Test_doguDeploymentLimitPatcher_PatchDeployment(t *testing.T) {
	t.Run("patch deployment without containers", func(t *testing.T) {
		// given
		deployment := &v1.Deployment{
			Spec: v1.DeploymentSpec{Template: v12.PodTemplateSpec{Spec: v12.PodSpec{}}},
		}

		regMock := &mocks.Registry{}
		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		// when
		err := patcher.PatchDeployment(deployment, DoguLimits{})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "given deployment cannot be patched, no containers are defined, at least one container is required")
	})

	t.Run("receives error when patching memory limits", func(t *testing.T) {
		// given
		deployment := &v1.Deployment{
			Spec: v1.DeploymentSpec{Template: v12.PodTemplateSpec{Spec: v12.PodSpec{
				Containers: []v12.Container{
					{Name: "testContainer"},
				},
			}}},
		}

		regMock := &mocks.Registry{}
		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		// when
		err := patcher.PatchDeployment(deployment, DoguLimits{MemoryLimit: "12e890uq209er"})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse memory request quantity")
	})

	t.Run("receives error when patching cpu limits", func(t *testing.T) {
		// given
		deployment := &v1.Deployment{
			Spec: v1.DeploymentSpec{Template: v12.PodTemplateSpec{Spec: v12.PodSpec{
				Containers: []v12.Container{
					{Name: "testContainer"},
				},
			}}},
		}

		regMock := &mocks.Registry{}
		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		// when
		err := patcher.PatchDeployment(deployment, DoguLimits{CpuLimit: "12e890uq209er"})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse cpu request quantity")
	})

	t.Run("receives error when patching storageEphemeral limits", func(t *testing.T) {
		// given
		deployment := &v1.Deployment{
			Spec: v1.DeploymentSpec{Template: v12.PodTemplateSpec{Spec: v12.PodSpec{
				Containers: []v12.Container{
					{Name: "testContainer"},
				},
			}}},
		}

		regMock := &mocks.Registry{}
		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		// when
		err := patcher.PatchDeployment(deployment, DoguLimits{EphemeralStorageLimit: "12e890uq209er"})

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse storageEphemeral request quantity")
	})

	t.Run("successful patch resources with one limit", func(t *testing.T) {
		// given
		deployment := &v1.Deployment{
			Spec: v1.DeploymentSpec{Template: v12.PodTemplateSpec{Spec: v12.PodSpec{
				Containers: []v12.Container{
					{Name: "testContainer"},
				},
			}}},
		}

		regMock := &mocks.Registry{}
		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		doguLimits := DoguLimits{
			CpuLimit:    "100m",
			MemoryLimit: "2Gi",
		}

		// when
		err := patcher.PatchDeployment(deployment, doguLimits)

		// then
		require.NoError(t, err)

		cpuLimitQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Limits[v12.ResourceCPU]
		cpuRequestQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Requests[v12.ResourceCPU]
		assert.Equal(t, cpuLimitQuantity.String(), "100m")
		assert.Equal(t, cpuRequestQuantity.String(), "100m")

		memoryLimitQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Limits[v12.ResourceMemory]
		memoryRequestQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Requests[v12.ResourceMemory]
		assert.Equal(t, memoryLimitQuantity.String(), "2Gi")
		assert.Equal(t, memoryRequestQuantity.String(), "2Gi")
	})

	t.Run("successful patch resources with multiple limits", func(t *testing.T) {
		// given
		deployment := &v1.Deployment{
			Spec: v1.DeploymentSpec{Template: v12.PodTemplateSpec{Spec: v12.PodSpec{
				Containers: []v12.Container{
					{Name: "testContainer"},
				},
			}}},
		}

		regMock := &mocks.Registry{}
		patcher := doguDeploymentLimitPatcher{}
		patcher.registry = regMock

		doguLimits := DoguLimits{
			CpuLimit:              "100m",
			MemoryLimit:           "2Gi",
			EphemeralStorageLimit: "4Gi",
		}

		// when
		err := patcher.PatchDeployment(deployment, doguLimits)

		// then
		require.NoError(t, err)

		cpuLimitQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Limits[v12.ResourceCPU]
		cpuRequestQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Requests[v12.ResourceCPU]
		assert.Equal(t, cpuLimitQuantity.String(), "100m")
		assert.Equal(t, cpuRequestQuantity.String(), "100m")

		memoryLimitQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Limits[v12.ResourceMemory]
		memoryRequestQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Requests[v12.ResourceMemory]
		assert.Equal(t, memoryLimitQuantity.String(), "2Gi")
		assert.Equal(t, memoryRequestQuantity.String(), "2Gi")

		empheralStorageLimitQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Limits[v12.ResourceEphemeralStorage]
		empheralStorageResourceQuantity := deployment.Spec.Template.Spec.Containers[0].Resources.Requests[v12.ResourceEphemeralStorage]
		assert.Equal(t, empheralStorageLimitQuantity.String(), "4Gi")
		assert.Equal(t, empheralStorageResourceQuantity.String(), "4Gi")
	})
}
