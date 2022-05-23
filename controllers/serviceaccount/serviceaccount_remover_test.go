package serviceaccount_test

import (
	"context"
	"github.com/cloudogu/cesapp/v4/core"
	cesmocks "github.com/cloudogu/cesapp/v4/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRemover_RemoveServiceAccounts(t *testing.T) {
	var postgresRemoveCmd core.ExposedCommand
	for _, command := range postgresqlDescriptor.ExposedCommands {
		if command.Name == "service-account-remove" {
			postgresRemoveCmd = command
			break
		}
	}
	require.NotNil(t, postgresRemoveCmd)

	t.Run("success", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)
		doguConfig.Mock.On("DeleteRecursive", "sa-postgresql").Return(nil)
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(postgresqlDescriptor, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommand", mock.Anything, "postgresql", "test", &postgresRemoveCmd, []string{"redmine"}).Return(nil, nil)
		serviceAccountCreator := serviceaccount.NewRemover(registry, commandExecutorMock)

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineCr.Namespace, redmineDescriptor)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, doguConfig, doguRegistry, registry, commandExecutorMock)
	})

	t.Run("sa dogu does not exist", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, assert.AnError)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		serviceAccountCreator := serviceaccount.NewRemover(registry, nil)

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineCr.Namespace, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check if service account already exists")
		mock.AssertExpectationsForObjects(t, doguConfig, registry)
	})

	t.Run("skip routine because serviceaccount does not exist in dogu config", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		serviceAccountCreator := serviceaccount.NewRemover(registry, nil)

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineCr.Namespace, redmineDescriptor)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, doguConfig, registry)
	})

	t.Run("failed to check if sa dogu is enabled", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(false, assert.AnError)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		serviceAccountCreator := serviceaccount.NewRemover(registry, nil)

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineCr.Namespace, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check if dogu postgresql is enabled")
		mock.AssertExpectationsForObjects(t, doguConfig, registry, doguRegistry)
	})

	t.Run("skip routine because sa dogu is not enabled", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(false, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		serviceAccountCreator := serviceaccount.NewRemover(registry, nil)

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineCr.Namespace, redmineDescriptor)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, doguConfig, registry, doguRegistry)
	})

	t.Run("failed to get dogu.json from service account dogu", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(nil, assert.AnError)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		serviceAccountCreator := serviceaccount.NewRemover(registry, nil)

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineCr.Namespace, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get service account dogu.json")
		mock.AssertExpectationsForObjects(t, doguConfig, registry, doguRegistry)
	})

	t.Run("failed because sa dogu does not expose remote command", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(invalidPostgresqlDescriptor, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		serviceAccountCreator := serviceaccount.NewRemover(registry, nil)

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineCr.Namespace, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service account dogu postgresql does not expose service-account-remove command")
		mock.AssertExpectationsForObjects(t, doguConfig, registry, doguRegistry)
	})

	t.Run("failed to execute command", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(postgresqlDescriptor, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommand", mock.Anything, "postgresql", "test", &postgresRemoveCmd, []string{"redmine"}).Return(nil, assert.AnError)
		serviceAccountCreator := serviceaccount.NewRemover(registry, commandExecutorMock)

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineCr.Namespace, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to execute service account remove command")
		mock.AssertExpectationsForObjects(t, doguConfig, registry, doguRegistry, commandExecutorMock)
	})

	t.Run("failed to delete sa credentials from dogu config", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)
		doguConfig.Mock.On("DeleteRecursive", "sa-postgresql").Return(assert.AnError)
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(postgresqlDescriptor, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommand", mock.Anything, "postgresql", "test", &postgresRemoveCmd, []string{"redmine"}).Return(nil, nil)
		serviceAccountCreator := serviceaccount.NewRemover(registry, commandExecutorMock)

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineCr.Namespace, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove service account from config")
		mock.AssertExpectationsForObjects(t, doguConfig, registry, doguRegistry, commandExecutorMock)
	})
}
