package serviceaccount

import (
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/client-go/kubernetes/fake"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/exec"
	"github.com/stretchr/testify/require"
)

func TestNewRemover(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// when
		result := NewRemover(nil, nil, nil, nil, nil, "")

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

		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)
		sensitiveConfigRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, nil)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresRemoveCmd.Command, "redmine")

		commandExecutorMock := &MockCommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, exec.PodReady).Return(nil, nil)

		localFetcher := NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := remover{
			client:            cli,
			sensitiveDoguRepo: sensitiveConfigRepoMock,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
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

		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
			"sa-cas/username":        "testUser",
			"sa-cas/password":        "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)
		sensitiveConfigRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, nil)

		casRemoveSAShellCmd := exec.NewShellCommand(casRemoveCmd.Command, "redmine")

		commandExecutorMock := &MockCommandExecutor{}
		commandExecutorMock.Mock.
			On("ExecCommandForPod", testCtx, readyCasPod, casRemoveSAShellCmd, exec.PodReady).Return(nil, nil)

		localFetcher := NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "cas").Return(true, nil)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(false, assert.AnError)
		localFetcher.EXPECT().FetchInstalled(testCtx, "cas").Return(casDescriptor, nil)
		serviceAccountRemover := remover{
			client:            cli,
			sensitiveDoguRepo: sensitiveConfigRepoMock,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
		}

		// when
		err := serviceAccountRemover.RemoveAll(testCtx, redmineDescriptorTwoSa)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("skip routine because serviceaccount does not exist in dogu config", func(t *testing.T) {
		// given
		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-cas/username": "testUser",
			"sa-cas/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)

		serviceAccountCreator := remover{sensitiveDoguRepo: sensitiveConfigRepoMock}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.NoError(t, err)
	})

	t.Run("failed to check if sa dogu is enabled", func(t *testing.T) {
		// given
		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)

		localFetcher := NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(false, assert.AnError)

		serviceAccountCreator := remover{sensitiveDoguRepo: sensitiveConfigRepoMock, doguFetcher: localFetcher}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to check if dogu postgresql is enabled")
	})

	t.Run("skip routine because sa dogu is not enabled", func(t *testing.T) {
		// given
		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)

		localFetcher := NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(false, nil)

		serviceAccountCreator := remover{sensitiveDoguRepo: sensitiveConfigRepoMock, doguFetcher: localFetcher}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.NoError(t, err)
	})

	t.Run("failed to get dogu.json from service account dogu", func(t *testing.T) {
		// given
		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)

		localFetcher := NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(nil, assert.AnError)
		serviceAccountCreator := remover{sensitiveDoguRepo: sensitiveConfigRepoMock, doguFetcher: localFetcher}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get service account dogu.json")
	})

	t.Run("failed to get service account producer pod", func(t *testing.T) {
		// given
		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)

		localFetcher := NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		cliWithoutReadyPod := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		serviceAccountCreator := remover{client: cliWithoutReadyPod, sensitiveDoguRepo: sensitiveConfigRepoMock, doguFetcher: localFetcher}

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

		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)

		localFetcher := NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(invalidPostgresqlDescriptor, nil)
		serviceAccountRemover := remover{client: cli, sensitiveDoguRepo: sensitiveConfigRepoMock, doguFetcher: localFetcher}

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

		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)

		postgresRemoveSAShellCmd := exec.NewShellCommand(postgresRemoveCmd.Command, "redmine")

		commandExecutorMock := &MockCommandExecutor{}
		commandExecutorMock.Mock.
			On("ExecCommandForPod", testCtx, readyPostgresPod, postgresRemoveSAShellCmd, exec.PodReady).Return(nil, assert.AnError)

		localFetcher := NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountRemover := remover{client: cli, sensitiveDoguRepo: sensitiveConfigRepoMock, doguFetcher: localFetcher, executor: commandExecutorMock}

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

		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)
		sensitiveConfigRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, assert.AnError)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresRemoveCmd.Command, "redmine")

		commandExecutorMock := &MockCommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPostgresPod, postgresCreateSAShellCmd, exec.PodReady).Return(nil, nil)

		localFetcher := NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := remover{
			client:            cli,
			sensitiveDoguRepo: sensitiveConfigRepoMock,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
		}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed write config for dogu test after updating")
	})

	t.Run("failed to remove components sa", func(t *testing.T) {
		// given
		fakeClient := fake.NewSimpleClientset()

		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-k8s-prometheus/username": "testUser",
			"sa-k8s-prometheus/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)

		r := remover{clientSet: fakeClient, sensitiveDoguRepo: sensitiveConfigRepoMock}

		dogu := &core.Dogu{
			Name: "official/grafana",
			ServiceAccounts: []core.ServiceAccount{
				{Kind: "component", Type: "k8s-prometheus"},
			},
		}

		// when
		err := r.RemoveAll(testCtx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get service:")
	})
}
