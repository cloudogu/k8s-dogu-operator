package upgrade

import (
	"context"
	"testing"

	"github.com/cloudogu/cesapp-lib/registry/mocks"
	"github.com/stretchr/testify/require"
)

var testCtx = context.TODO()

func Test_upgradeExecutor_Upgrade(t *testing.T) {

}

func Test_registerUpgradedDoguVersion(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = "4.2.3-11"
		registryMock := new(mocks.Registry)
		registryMock.On("Set").Return(nil)

		// when
		err := registerUpgradedDoguVersion(testCtx, nil, toDogu)

		// then
		require.NoError(t, err)
		registryMock.AssertExpectations(t)
	})
}
