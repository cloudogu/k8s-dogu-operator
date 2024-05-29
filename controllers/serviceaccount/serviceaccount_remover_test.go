package serviceaccount

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRemover(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// when
		result := NewRemover(nil, nil, nil, nil, nil, nil, "")

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

	t.Run("success with dogu and host config sa", func(t *testing.T) {
		// given
		readyPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "postgresql-xyz", Labels: postgresqlCr.GetPodLabels()},
			Status:     v1.PodStatus{Conditions: []v1.PodCondition{{Type: v1.ContainersReady, Status: v1.ConditionTrue}}},
		}
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(readyPod).
			Build()
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)
		doguConfig.Mock.On("DeleteRecursive", "sa-postgresql").Return(nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		hostConfig := cesmocks.NewConfigurationContext(t)
		hostConfig.On("Exists", "redmine").Return(true, nil)
		hostConfig.On("DeleteRecursive", "redmine").Return(nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("HostConfig", "k8s-ces-control").Return(hostConfig)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresRemoveCmd.Command, "redmine")

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(nil, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := remover{
			client:            cli,
			registry:          registry,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptorCesSa)

		// then
		require.NoError(t, err)
	})

	t.Run("failure during first SA deletion should not interrupt second SA deletion", func(t *testing.T) {
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
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, assert.AnError)

		doguConfig.Mock.On("Exists", "sa-cas").Return(true, nil)
		doguConfig.Mock.On("DeleteRecursive", "sa-cas").Return(nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "cas").Return(true, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		casRemoveSAShellCmd := exec.NewShellCommand(casRemoveCmd.Command, "redmine")

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.
			On("ExecCommandForPod", testCtx, readyCasPod, casRemoveSAShellCmd, cloudogu.PodReady).Return(nil, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "cas").Return(casDescriptor, nil)
		serviceAccountRemover := remover{
			client:            cli,
			registry:          registry,
			doguFetcher:       localFetcher,
			localDoguRegistry: localDoguRegMock,
			executor:          commandExecutorMock,
		}

		// when
		err := serviceAccountRemover.RemoveAll(testCtx, redmineDescriptorTwoSa)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("sa dogu does not exist", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, assert.AnError)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)

		serviceAccountCreator := remover{registry: registry}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if service account already exists")
	})

	t.Run("skip routine because serviceaccount does not exist in dogu config", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)

		serviceAccountCreator := remover{registry: registry}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.NoError(t, err)
	})

	t.Run("failed to check if sa dogu is enabled", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(false, assert.AnError)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)

		serviceAccountCreator := remover{registry: registry, localDoguRegistry: localDoguRegMock}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to check if dogu postgresql is enabled")
	})

	t.Run("skip routine because sa dogu is not enabled", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(false, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)

		serviceAccountCreator := remover{registry: registry, localDoguRegistry: localDoguRegMock}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.NoError(t, err)
	})

	t.Run("failed to get dogu.json from service account dogu", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(nil, assert.AnError)
		serviceAccountCreator := remover{registry: registry, doguFetcher: localFetcher, localDoguRegistry: localDoguRegMock}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get service account dogu.json")
	})

	t.Run("failed to get service account producer pod", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		cliWithoutReadyPod := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		serviceAccountCreator := remover{client: cliWithoutReadyPod, registry: registry, doguFetcher: localFetcher, localDoguRegistry: localDoguRegMock}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "could not find service account producer pod postgresql")
	})

	t.Run("failed because sa dogu does not expose remote command", func(t *testing.T) {
		// given
		readyPostgresPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "postgresql-xyz", Labels: postgresqlCr.GetPodLabels()},
			Status:     v1.PodStatus{Conditions: []v1.PodCondition{{Type: v1.ContainersReady, Status: v1.ConditionTrue}}},
		}
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(readyPostgresPod).
			Build()
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(invalidPostgresqlDescriptor, nil)
		serviceAccountRemover := remover{client: cli, registry: registry, doguFetcher: localFetcher, localDoguRegistry: localDoguRegMock}

		// when
		err := serviceAccountRemover.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "service account dogu postgresql does not expose service-account-remove command")
	})

	t.Run("failed to execute command", func(t *testing.T) {
		// given
		readyPostgresPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "postgresql-xyz", Labels: postgresqlCr.GetPodLabels()},
			Status:     v1.PodStatus{Conditions: []v1.PodCondition{{Type: v1.ContainersReady, Status: v1.ConditionTrue}}},
		}
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(readyPostgresPod).
			Build()
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.
			On("Exists", "sa-postgresql").Return(true, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.
			On("DoguConfig", "redmine").Return(doguConfig)

		postgresRemoveSAShellCmd := exec.NewShellCommand(postgresRemoveCmd.Command, "redmine")

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.
			On("ExecCommandForPod", testCtx, readyPostgresPod, postgresRemoveSAShellCmd, cloudogu.PodReady).Return(nil, assert.AnError)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountRemover := remover{client: cli, registry: registry, doguFetcher: localFetcher, executor: commandExecutorMock, localDoguRegistry: localDoguRegMock}

		// when
		err := serviceAccountRemover.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to execute service account remove command")
	})

	t.Run("failed to delete SA credentials from dogu config", func(t *testing.T) {
		// given
		readyPostgresPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "postgresql-xyz", Labels: postgresqlCr.GetPodLabels()},
			Status:     v1.PodStatus{Conditions: []v1.PodCondition{{Type: v1.ContainersReady, Status: v1.ConditionTrue}}},
		}
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(readyPostgresPod).
			Build()
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)
		doguConfig.Mock.On("DeleteRecursive", "sa-postgresql").Return(assert.AnError)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresRemoveCmd.Command, "redmine")

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPostgresPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(nil, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := remover{
			client:            cli,
			registry:          registry,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to remove service account from config")
	})

	t.Run("failed to create host config sa", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		hostConfig := cesmocks.NewConfigurationContext(t)
		hostConfig.On("Exists", "redmine").Return(true, assert.AnError)
		registry := cesmocks.NewRegistry(t)
		registry.On("DoguConfig", "redmine").Return(doguConfig)
		registry.On("HostConfig", "k8s-ces-control").Return(hostConfig)
		remover := remover{registry: registry}

		// when
		err := remover.RemoveAll(testCtx, redmineDescriptorCesSa)

		// then
		require.Error(t, err)
	})

	t.Run("failed to remove components sa", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-k8s-prometheus").Return(false, assert.AnError)
		registry := cesmocks.NewRegistry(t)
		registry.On("DoguConfig", "grafana").Return(doguConfig)
		remover := remover{registry: registry}

		dogu := &core.Dogu{
			Name: "official/grafana",
			ServiceAccounts: []core.ServiceAccount{
				{Kind: "component", Type: "k8s-prometheus"},
			},
		}

		// when
		err := remover.RemoveAll(testCtx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if service account already exists:")
	})
}

func Test_remover_removeCesControlServiceAccount(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		hostConfig := cesmocks.NewConfigurationContext(t)
		hostConfig.On("Exists", "redmine").Return(true, nil)
		registry := cesmocks.NewRegistry(t)
		registry.On("HostConfig", "k8s-ces-control").Return(hostConfig)
		hostConfig.On("DeleteRecursive", "redmine").Return(nil)
		remover := remover{registry: registry}

		// when
		err := remover.removeCesControlServiceAccount(redmineDescriptorCesSa)

		// then
		require.NoError(t, err)
	})

	t.Run("failed to read if dogu sa is in host config", func(t *testing.T) {
		// given
		hostConfig := cesmocks.NewConfigurationContext(t)
		hostConfig.On("Exists", "redmine").Return(true, assert.AnError)
		registry := cesmocks.NewRegistry(t)
		registry.On("HostConfig", "k8s-ces-control").Return(hostConfig)
		remover := remover{registry: registry}

		// when
		err := remover.removeCesControlServiceAccount(redmineDescriptorCesSa)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read host config for dogu")
	})

	t.Run("failed to delete s from host config", func(t *testing.T) {
		// given
		hostConfig := cesmocks.NewConfigurationContext(t)
		hostConfig.On("Exists", "redmine").Return(true, nil)
		hostConfig.On("DeleteRecursive", "redmine").Return(assert.AnError)
		registry := cesmocks.NewRegistry(t)
		registry.On("HostConfig", "k8s-ces-control").Return(hostConfig)
		remover := remover{registry: registry}

		// when
		err := remover.removeCesControlServiceAccount(redmineDescriptorCesSa)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to delete host config for dogu")
	})
}
