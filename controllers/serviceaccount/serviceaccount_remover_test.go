package serviceaccount

import (
	"github.com/stretchr/testify/mock"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/cesapp-lib/core"
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

	t.Run("success with dogu sa", func(t *testing.T) {
		// given
		readyPod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "postgresql-xyz", Labels: postgresqlCr.GetPodLabels()},
			Status:     v1.PodStatus{Conditions: []v1.PodCondition{{Type: v1.ContainersReady, Status: v1.ConditionTrue}}},
		}
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(readyPod).
			Build()

		sensitiveConfig := NewMockSensitiveDoguConfig(t)
		sensitiveConfig.EXPECT().Exists(mock.Anything, "sa-postgresql").Return(true, nil)
		sensitiveConfig.EXPECT().DeleteRecursive(mock.Anything, "sa-postgresql").Return(nil)

		sensitiveConfigProvider := NewMockSensitiveDoguConfigProvider(t)
		sensitiveConfigProvider.EXPECT().GetSensitiveDoguConfig(mock.Anything, mock.Anything).Return(sensitiveConfig, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresRemoveCmd.Command, "redmine")

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(nil, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := remover{
			client:                   cli,
			sensitiveDoguCfgProvider: sensitiveConfigProvider,
			doguFetcher:              localFetcher,
			executor:                 commandExecutorMock,
			localDoguRegistry:        localDoguRegMock,
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

		sensitiveConfig := NewMockSensitiveDoguConfig(t)
		sensitiveConfig.EXPECT().Exists(mock.Anything, "sa-postgresql").Return(false, assert.AnError)
		sensitiveConfig.EXPECT().Exists(mock.Anything, "sa-cas").Return(true, nil)
		sensitiveConfig.EXPECT().DeleteRecursive(mock.Anything, "sa-cas").Return(nil)

		sensitiveConfigProvider := NewMockSensitiveDoguConfigProvider(t)
		sensitiveConfigProvider.EXPECT().GetSensitiveDoguConfig(mock.Anything, mock.Anything).Return(sensitiveConfig, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "cas").Return(true, nil)

		casRemoveSAShellCmd := exec.NewShellCommand(casRemoveCmd.Command, "redmine")

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.
			On("ExecCommandForPod", testCtx, readyCasPod, casRemoveSAShellCmd, cloudogu.PodReady).Return(nil, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "cas").Return(casDescriptor, nil)
		serviceAccountRemover := remover{
			client:                   cli,
			sensitiveDoguCfgProvider: sensitiveConfigProvider,
			doguFetcher:              localFetcher,
			localDoguRegistry:        localDoguRegMock,
			executor:                 commandExecutorMock,
		}

		// when
		err := serviceAccountRemover.RemoveAll(testCtx, redmineDescriptorTwoSa)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("sa dogu does not exist", func(t *testing.T) {
		// given
		sensitiveConfig := NewMockSensitiveDoguConfig(t)
		sensitiveConfig.EXPECT().Exists(mock.Anything, "sa-postgresql").Return(false, assert.AnError)

		sensitiveConfigProvider := NewMockSensitiveDoguConfigProvider(t)
		sensitiveConfigProvider.EXPECT().GetSensitiveDoguConfig(mock.Anything, mock.Anything).Return(sensitiveConfig, nil)

		serviceAccountCreator := remover{sensitiveDoguCfgProvider: sensitiveConfigProvider}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if service account already exists")
	})

	t.Run("skip routine because serviceaccount does not exist in dogu config", func(t *testing.T) {
		// given
		sensitiveConfig := NewMockSensitiveDoguConfig(t)
		sensitiveConfig.EXPECT().Exists(mock.Anything, "sa-postgresql").Return(false, nil)

		sensitiveConfigProvider := NewMockSensitiveDoguConfigProvider(t)
		sensitiveConfigProvider.EXPECT().GetSensitiveDoguConfig(mock.Anything, mock.Anything).Return(sensitiveConfig, nil)

		serviceAccountCreator := remover{sensitiveDoguCfgProvider: sensitiveConfigProvider}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.NoError(t, err)
	})

	t.Run("failed to check if sa dogu is enabled", func(t *testing.T) {
		// given
		sensitiveConfig := NewMockSensitiveDoguConfig(t)
		sensitiveConfig.EXPECT().Exists(mock.Anything, "sa-postgresql").Return(true, nil)

		sensitiveConfigProvider := NewMockSensitiveDoguConfigProvider(t)
		sensitiveConfigProvider.EXPECT().GetSensitiveDoguConfig(mock.Anything, mock.Anything).Return(sensitiveConfig, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(false, assert.AnError)

		serviceAccountCreator := remover{sensitiveDoguCfgProvider: sensitiveConfigProvider, localDoguRegistry: localDoguRegMock}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to check if dogu postgresql is enabled")
	})

	t.Run("skip routine because sa dogu is not enabled", func(t *testing.T) {
		// given
		sensitiveConfig := NewMockSensitiveDoguConfig(t)
		sensitiveConfig.EXPECT().Exists(mock.Anything, "sa-postgresql").Return(true, nil)

		sensitiveConfigProvider := NewMockSensitiveDoguConfigProvider(t)
		sensitiveConfigProvider.EXPECT().GetSensitiveDoguConfig(mock.Anything, mock.Anything).Return(sensitiveConfig, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(false, nil)

		serviceAccountCreator := remover{sensitiveDoguCfgProvider: sensitiveConfigProvider, localDoguRegistry: localDoguRegMock}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.NoError(t, err)
	})

	t.Run("failed to get dogu.json from service account dogu", func(t *testing.T) {
		// given
		sensitiveConfig := NewMockSensitiveDoguConfig(t)
		sensitiveConfig.EXPECT().Exists(mock.Anything, "sa-postgresql").Return(true, nil)

		sensitiveConfigProvider := NewMockSensitiveDoguConfigProvider(t)
		sensitiveConfigProvider.EXPECT().GetSensitiveDoguConfig(mock.Anything, mock.Anything).Return(sensitiveConfig, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(nil, assert.AnError)
		serviceAccountCreator := remover{sensitiveDoguCfgProvider: sensitiveConfigProvider, doguFetcher: localFetcher, localDoguRegistry: localDoguRegMock}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get service account dogu.json")
	})

	t.Run("failed to get service account producer pod", func(t *testing.T) {
		// given
		sensitiveConfig := NewMockSensitiveDoguConfig(t)
		sensitiveConfig.EXPECT().Exists(mock.Anything, "sa-postgresql").Return(true, nil)

		sensitiveConfigProvider := NewMockSensitiveDoguConfigProvider(t)
		sensitiveConfigProvider.EXPECT().GetSensitiveDoguConfig(mock.Anything, mock.Anything).Return(sensitiveConfig, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		cliWithoutReadyPod := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		serviceAccountCreator := remover{client: cliWithoutReadyPod, sensitiveDoguCfgProvider: sensitiveConfigProvider, doguFetcher: localFetcher, localDoguRegistry: localDoguRegMock}

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

		sensitiveConfig := NewMockSensitiveDoguConfig(t)
		sensitiveConfig.EXPECT().Exists(mock.Anything, "sa-postgresql").Return(true, nil)

		sensitiveConfigProvider := NewMockSensitiveDoguConfigProvider(t)
		sensitiveConfigProvider.EXPECT().GetSensitiveDoguConfig(mock.Anything, mock.Anything).Return(sensitiveConfig, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(invalidPostgresqlDescriptor, nil)
		serviceAccountRemover := remover{client: cli, sensitiveDoguCfgProvider: sensitiveConfigProvider, doguFetcher: localFetcher, localDoguRegistry: localDoguRegMock}

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

		sensitiveConfig := NewMockSensitiveDoguConfig(t)
		sensitiveConfig.EXPECT().Exists(mock.Anything, "sa-postgresql").Return(true, nil)

		sensitiveConfigProvider := NewMockSensitiveDoguConfigProvider(t)
		sensitiveConfigProvider.EXPECT().GetSensitiveDoguConfig(mock.Anything, mock.Anything).Return(sensitiveConfig, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		postgresRemoveSAShellCmd := exec.NewShellCommand(postgresRemoveCmd.Command, "redmine")

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.
			On("ExecCommandForPod", testCtx, readyPostgresPod, postgresRemoveSAShellCmd, cloudogu.PodReady).Return(nil, assert.AnError)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountRemover := remover{client: cli, sensitiveDoguCfgProvider: sensitiveConfigProvider, doguFetcher: localFetcher, executor: commandExecutorMock, localDoguRegistry: localDoguRegMock}

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

		sensitiveConfig := NewMockSensitiveDoguConfig(t)
		sensitiveConfig.EXPECT().Exists(mock.Anything, "sa-postgresql").Return(true, nil)
		sensitiveConfig.EXPECT().DeleteRecursive(mock.Anything, "sa-postgresql").Return(assert.AnError)

		sensitiveConfigProvider := NewMockSensitiveDoguConfigProvider(t)
		sensitiveConfigProvider.EXPECT().GetSensitiveDoguConfig(mock.Anything, mock.Anything).Return(sensitiveConfig, nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresRemoveCmd.Command, "redmine")

		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPostgresPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(nil, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := remover{
			client:                   cli,
			sensitiveDoguCfgProvider: sensitiveConfigProvider,
			doguFetcher:              localFetcher,
			executor:                 commandExecutorMock,
			localDoguRegistry:        localDoguRegMock,
		}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to remove service account from sensitive config")
	})

	t.Run("failed to remove components sa", func(t *testing.T) {
		// given
		sensitiveConfig := NewMockSensitiveDoguConfig(t)
		sensitiveConfig.EXPECT().Exists(mock.Anything, "sa-k8s-prometheus").Return(false, assert.AnError)

		sensitiveConfigProvider := NewMockSensitiveDoguConfigProvider(t)
		sensitiveConfigProvider.EXPECT().GetSensitiveDoguConfig(mock.Anything, mock.Anything).Return(sensitiveConfig, nil)

		remover := remover{sensitiveDoguCfgProvider: sensitiveConfigProvider}

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
