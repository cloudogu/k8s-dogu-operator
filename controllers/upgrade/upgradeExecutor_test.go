package upgrade

import (
	"context"
	"testing"

	"github.com/cloudogu/cesapp-lib/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testCtx = context.TODO()

func Test_upgradeExecutor_Upgrade(t *testing.T) {

}

func Test_registerUpgradedDoguVersion(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = "4.2.3-11"

		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = "4.2.3-11"
		doguRegistryMock := new(mocks.DoguRegistry)
		registryMock := new(mocks.Registry)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		doguRegistryMock.On("IsEnabled", toDogu.GetSimpleName()).Return(true, nil)
		doguRegistryMock.On("Register", toDogu).Return(nil)
		doguRegistryMock.On("Enable", toDogu).Return(nil)

		cesreg := cesregistry.NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registerUpgradedDoguVersion(cesreg, toDogu)

		// then
		require.NoError(t, err)
		registryMock.AssertExpectations(t)
		doguRegistryMock.AssertExpectations(t)
	})
	t.Run("should fail", func(t *testing.T) {
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = "4.2.3-11"

		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = "4.2.3-11"
		doguRegistryMock := new(mocks.DoguRegistry)
		registryMock := new(mocks.Registry)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		doguRegistryMock.On("IsEnabled", toDogu.GetSimpleName()).Return(false, nil)

		cesreg := cesregistry.NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registerUpgradedDoguVersion(cesreg, toDogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to register upgrade: could not register dogu version: previous version not found")
		registryMock.AssertExpectations(t)
		doguRegistryMock.AssertExpectations(t)
	})
}
