package serviceaccount

import (
	"context"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"

	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	mocks2 "github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount/mocks"

	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewRemover(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)

		// when
		result := NewRemover(registryMock, nil, nil)

		// then
		require.NotNil(t, result)
	})
}

func TestRemover_RemoveServiceAccounts(t *testing.T) {
	var postgresRemoveCmd core.ExposedCommand
	for _, command := range postgresqlDescriptor.ExposedCommands {
		if command.Name == "service-account-remove" {
			postgresRemoveCmd = command
			break
		}
	}
	require.NotNil(t, postgresRemoveCmd)

	var casRemoveCmd core.ExposedCommand
	for _, command := range casDescriptor.ExposedCommands {
		if command.Name == "service-account-remove" {
			casRemoveCmd = command
			break
		}
	}
	require.NotNil(t, casRemoveCmd)

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

		postgresCreateSAShellCmd := &exec.ShellCommand{Command: postgresRemoveCmd.Command, Args: []string{"redmine"}}

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommandForPod", ctx, readyPod, postgresCreateSAShellCmd, exec.PodReady).Return(nil, nil)

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
		readyPostgresPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "postgresql-xyz", Labels: postgresqlCr.GetPodLabels()},
			Status:     v1.PodStatus{Conditions: []v1.PodCondition{{Type: v1.ContainersReady, Status: v1.ConditionTrue}}},
		}
		readyCasPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "cas-xyz", Labels: casCr.GetPodLabels()},
			Status:     v1.PodStatus{Conditions: []v1.PodCondition{{Type: v1.ContainersReady, Status: v1.ConditionTrue}}},
		}
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(readyPostgresPod, readyCasPod).
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
		casRemoveSAShellCmd := &exec.ShellCommand{Command: casRemoveCmd.Command, Args: []string{"redmine"}}

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.
			On("ExecCommandForPod", ctx, readyCasPod, casRemoveSAShellCmd, exec.PodReady).Return(nil, nil)

		localFetcher := &mocks2.LocalDoguFetcher{}
		localFetcher.Mock.On("FetchInstalled", "cas").Return(casDescriptor, nil)
		serviceAccountRemover := remover{
			client:      cli,
			registry:    registry,
			doguFetcher: localFetcher,
			executor:    commandExecutorMock,
		}

		// when
		err := serviceAccountRemover.RemoveAll(ctx, redmineDescriptorTwoSa)

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
		assert.ErrorContains(t, err, "failed to check if service account already exists")
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
		assert.ErrorContains(t, err, "failed to check if dogu postgresql is enabled")
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
		assert.ErrorContains(t, err, "failed to get service account dogu.json")
		mock.AssertExpectationsForObjects(t, doguConfig, registry, localFetcher, doguRegistry)
	})

	t.Run("failed to get service account producer pod", func(t *testing.T) {
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
		localFetcher.Mock.On("FetchInstalled", "postgresql").Return(postgresqlDescriptor, nil)
		cliWithoutReadyPod := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		serviceAccountCreator := remover{client: cliWithoutReadyPod, registry: registry, doguFetcher: localFetcher}

		// when
		err := serviceAccountCreator.RemoveAll(ctx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "could not find service account producer pod postgresql")
		mock.AssertExpectationsForObjects(t, doguConfig, registry, localFetcher, doguRegistry)
	})

	t.Run("failed because sa dogu does not expose remote command", func(t *testing.T) {
		// given
		ctx := context.TODO()

		readyPostgresPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "postgresql-xyz", Labels: postgresqlCr.GetPodLabels()},
			Status:     v1.PodStatus{Conditions: []v1.PodCondition{{Type: v1.ContainersReady, Status: v1.ConditionTrue}}},
		}
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(readyPostgresPod).
			Build()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)

		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)

		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)

		localFetcher := &mocks2.LocalDoguFetcher{}
		localFetcher.Mock.On("FetchInstalled", "postgresql").Return(invalidPostgresqlDescriptor, nil)
		serviceAccountRemover := remover{client: cli, registry: registry, doguFetcher: localFetcher}

		// when
		err := serviceAccountRemover.RemoveAll(ctx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "service account dogu postgresql does not expose service-account-remove command")
		mock.AssertExpectationsForObjects(t, doguConfig, registry, localFetcher, doguRegistry)
	})

	t.Run("failed to execute command", func(t *testing.T) {
		// given
		ctx := context.TODO()

		readyPostgresPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "postgresql-xyz", Labels: postgresqlCr.GetPodLabels()},
			Status:     v1.PodStatus{Conditions: []v1.PodCondition{{Type: v1.ContainersReady, Status: v1.ConditionTrue}}},
		}
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(readyPostgresPod).
			Build()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.
			On("Exists", "sa-postgresql").Return(true, nil)

		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)

		registry := &cesmocks.Registry{}
		registry.Mock.
			On("DoguConfig", "redmine").Return(doguConfig).
			On("DoguRegistry").Return(doguRegistry)

		postgresRemoveSAShellCmd := &exec.ShellCommand{Command: postgresRemoveCmd.Command, Args: []string{"redmine"}}

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.
			On("ExecCommandForPod", ctx, readyPostgresPod, postgresRemoveSAShellCmd, exec.PodReady).Return(nil, assert.AnError)

		localFetcher := &mocks2.LocalDoguFetcher{}
		localFetcher.Mock.On("FetchInstalled", "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountRemover := remover{client: cli, registry: registry, doguFetcher: localFetcher, executor: commandExecutorMock}

		// when
		err := serviceAccountRemover.RemoveAll(ctx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to execute service account remove command")
		mock.AssertExpectationsForObjects(t, doguConfig, registry, doguRegistry, localFetcher, commandExecutorMock)
	})

	t.Run("failed to delete SA credentials from dogu config", func(t *testing.T) {
		// given
		ctx := context.TODO()
		readyPostgresPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "postgresql-xyz", Labels: postgresqlCr.GetPodLabels()},
			Status:     v1.PodStatus{Conditions: []v1.PodCondition{{Type: v1.ContainersReady, Status: v1.ConditionTrue}}},
		}
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(readyPostgresPod).
			Build()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)
		doguConfig.Mock.On("DeleteRecursive", "sa-postgresql").Return(assert.AnError)

		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)

		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)

		postgresCreateSAShellCmd := &exec.ShellCommand{Command: postgresRemoveCmd.Command, Args: []string{"redmine"}}

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommandForPod", ctx, readyPostgresPod, postgresCreateSAShellCmd, exec.PodReady).Return(nil, nil)

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
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to remove service account from config")
		mock.AssertExpectationsForObjects(t, doguConfig, registry, doguRegistry, localFetcher, commandExecutorMock)
	})
}
