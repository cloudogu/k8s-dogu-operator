package serviceaccount

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"

	mocks2 "github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount/mocks"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
		readyPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "postgresql-xyz", Labels: postgresqlCr.GetPodLabels()},
			Status:     v1.PodStatus{Conditions: []v1.PodCondition{{Type: v1.ContainersReady, Status: v1.ConditionTrue}}},
		}
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(readyPod).
			Build()
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)
		doguConfig.Mock.On("DeleteRecursive", "sa-postgresql").Return(nil)

		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)

		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)

		postgresCreateSAShellCmd := &resource.ShellCommand{Command: postgresRemoveCmd.Command, Args: []string{"redmine"}}

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommandForPod", ctx, readyPod, postgresCreateSAShellCmd, resource.PodReady).Return(nil, nil)

		localFetcher := &mocks2.LocalDoguFetcher{}
		localFetcher.Mock.On("FetchInstalled", "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := remover{
			client:      cli,
			registry:    registry,
			doguFetcher: localFetcher,
			executor:    commandExecutorMock,
		}

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineDescriptor)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, doguConfig, doguRegistry, registry, localFetcher, commandExecutorMock)
	})

	t.Run("failure during first sa deletion should not interrupt second sa deletion", func(t *testing.T) {
		// given
		readyPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "postgresql-xyz", Labels: postgresqlCr.GetPodLabels()},
			Status:     v1.PodStatus{Conditions: []v1.PodCondition{{Type: v1.ContainersReady, Status: v1.ConditionTrue}}},
		}
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(readyPod).
			Build()
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, assert.AnError)

		doguConfig.Mock.On("Exists", "sa-cas").Return(true, nil)
		doguConfig.Mock.On("DeleteRecursive", "sa-cas").Return(nil)

		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "cas").Return(true, nil)

		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		postgresCreateSAShellCmd := &resource.ShellCommand{Command: postgresRemoveCmd.Command, Args: []string{"redmine"}}

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommandForPod", ctx, readyPod, postgresCreateSAShellCmd, resource.PodReady).Return(nil, nil)

		localFetcher := &mocks2.LocalDoguFetcher{}
		// todo cas has an LDAP SA but receives postgresql. Maybe we should write proper tests
		localFetcher.Mock.On("FetchInstalled", "cas").Return(casDescriptor, nil)
		serviceAccountCreator := remover{
			client:      cli,
			registry:    registry,
			doguFetcher: localFetcher,
			executor:    commandExecutorMock,
		}

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineDescriptorTwoSa)

		// then
		require.Error(t, err)
		multiError, ok := err.(*multierror.Error)
		require.True(t, ok, "expected a different error than: "+err.Error())
		assert.Equal(t, 1, len(multiError.Errors))
		assert.ErrorIs(t, multiError.Errors[0], assert.AnError)
		mock.AssertExpectationsForObjects(t, doguConfig, doguRegistry, registry, localFetcher, commandExecutorMock)
	})

	t.Run("sa dogu does not exist", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, assert.AnError)

		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)

		serviceAccountCreator := remover{registry: registry}

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
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

		serviceAccountCreator := remover{registry: registry}

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineDescriptor)

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

		serviceAccountCreator := remover{registry: registry}

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineDescriptor)

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

		serviceAccountCreator := remover{registry: registry}

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineDescriptor)

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

		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)

		localFetcher := &mocks2.LocalDoguFetcher{}
		localFetcher.Mock.On("FetchInstalled", "postgresql").Return(nil, assert.AnError)
		serviceAccountCreator := remover{registry: registry, doguFetcher: localFetcher}

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get service account dogu.json")
		mock.AssertExpectationsForObjects(t, doguConfig, registry, localFetcher, doguRegistry)
	})

	t.Run("failed because sa dogu does not expose remote command", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)

		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)

		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)

		localFetcher := &mocks2.LocalDoguFetcher{}
		localFetcher.Mock.On("FetchInstalled", "postgresql").Return(invalidPostgresqlDescriptor, nil)
		serviceAccountCreator := remover{registry: registry, doguFetcher: localFetcher}

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service account dogu postgresql does not expose service-account-remove command")
		mock.AssertExpectationsForObjects(t, doguConfig, registry, localFetcher, doguRegistry)
	})

	t.Run("failed to execute command", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)

		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)

		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)

		postgresCreateSAShellCmd := &resource.ShellCommand{Command: postgresRemoveCmd.Command, Args: []string{"redmine"}}

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommandForDogu", mock.Anything, "postgresql", "test", postgresCreateSAShellCmd).Return(nil, assert.AnError)

		localFetcher := &mocks2.LocalDoguFetcher{}
		localFetcher.Mock.On("FetchInstalled", "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := remover{registry: registry, doguFetcher: localFetcher, executor: commandExecutorMock}

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to execute service account remove command")
		mock.AssertExpectationsForObjects(t, doguConfig, registry, doguRegistry, localFetcher, commandExecutorMock)
	})

	t.Run("failed to delete sa credentials from dogu config", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)
		doguConfig.Mock.On("DeleteRecursive", "sa-postgresql").Return(assert.AnError)

		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)

		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)

		postgresCreateSAShellCmd := &resource.ShellCommand{Command: postgresRemoveCmd.Command, Args: []string{"redmine"}}

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommandForDogu", mock.Anything, "postgresql", "test", postgresCreateSAShellCmd).Return(nil, nil)

		localFetcher := &mocks2.LocalDoguFetcher{}
		localFetcher.Mock.On("FetchInstalled", "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := remover{registry: registry, doguFetcher: localFetcher, executor: commandExecutorMock}

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove service account from config")
		mock.AssertExpectationsForObjects(t, doguConfig, registry, doguRegistry, localFetcher, commandExecutorMock)
	})
}
