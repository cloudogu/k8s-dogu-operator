package serviceaccount

import (
	"testing"

	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	opConfig "github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-registry-lib/config"
)

func TestNewRemover(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// when
		result := NewRemover(nil, nil, nil, nil, nil, &opConfig.OperatorConfig{})

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

	availablePostgresqlDoguResource := &k8sv2.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "postgresql"},
		Spec:       k8sv2.DoguSpec{Resources: k8sv2.DoguResources{MinDataVolumeSize: resource.MustParse("0")}},
		Status:     k8sv2.DoguStatus{Health: k8sv2.AvailableHealthStatus},
	}
	availableCasDoguResource := &k8sv2.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "cas"},
		Spec:       k8sv2.DoguSpec{Resources: k8sv2.DoguResources{MinDataVolumeSize: resource.MustParse("0")}},
		Status:     k8sv2.DoguStatus{Health: k8sv2.AvailableHealthStatus},
	}

	t.Run("success with dogu sa", func(t *testing.T) {
		// given
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(availablePostgresqlDoguResource).
			Build()

		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)
		sensitiveConfigRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, nil)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresRemoveCmd.Command, "redmine")

		commandExecutorMock := &mockCommandExecutor{}
		commandExecutorMock.EXPECT().ExecCommandForDogu(testCtx, availablePostgresqlDoguResource, postgresCreateSAShellCmd).Return(nil, nil)

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(postgresqlDescriptor, nil)
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
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(availableCasDoguResource, availablePostgresqlDoguResource).
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

		commandExecutorMock := &mockCommandExecutor{}
		commandExecutorMock.EXPECT().ExecCommandForDogu(testCtx, availableCasDoguResource, casRemoveSAShellCmd).Return(nil, nil)

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("cas")).Return(true, nil)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(false, assert.AnError)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("cas")).Return(casDescriptor, nil)
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

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(false, assert.AnError)

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

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(false, nil)

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

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(nil, assert.AnError)
		serviceAccountCreator := remover{sensitiveDoguRepo: sensitiveConfigRepoMock, doguFetcher: localFetcher}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get service account dogu.json")
	})

	t.Run("failed to get service account producer dogu", func(t *testing.T) {
		// given
		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(postgresqlDescriptor, nil)
		cliWithoutReadyPod := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		serviceAccountCreator := remover{client: cliWithoutReadyPod, sensitiveDoguRepo: sensitiveConfigRepoMock, doguFetcher: localFetcher}

		// when
		err := serviceAccountCreator.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to fetch dogu resource")
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

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(invalidPostgresqlDescriptor, nil)
		serviceAccountRemover := remover{client: cli, sensitiveDoguRepo: sensitiveConfigRepoMock, doguFetcher: localFetcher}

		// when
		err := serviceAccountRemover.RemoveAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "service account dogu postgresql does not expose service-account-remove command")
	})

	t.Run("failed to execute command", func(t *testing.T) {
		// given
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(availablePostgresqlDoguResource).
			Build()

		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)

		postgresRemoveSAShellCmd := exec.NewShellCommand(postgresRemoveCmd.Command, "redmine")

		commandExecutorMock := &mockCommandExecutor{}
		commandExecutorMock.EXPECT().ExecCommandForDogu(testCtx, availablePostgresqlDoguResource, postgresRemoveSAShellCmd).Return(nil, assert.AnError)

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(postgresqlDescriptor, nil)
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
		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(availablePostgresqlDoguResource).
			Build()

		doguCfg := config.CreateDoguConfig("test", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(doguCfg, nil)
		sensitiveConfigRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, assert.AnError)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresRemoveCmd.Command, "redmine")

		commandExecutorMock := &mockCommandExecutor{}
		commandExecutorMock.EXPECT().ExecCommandForDogu(testCtx, availablePostgresqlDoguResource, postgresCreateSAShellCmd).Return(nil, nil)

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(postgresqlDescriptor, nil)
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
		fakeClient := fake.NewClientset()

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

func TestRemover_RemoveAllFromComponents(t *testing.T) {
	t.Run("success remove component service account", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientset(
			&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa-provider-svc",
					Namespace: "testNs",
					Annotations: map[string]string{
						"ces.cloudogu.com/serviceaccount-port":        "9977",
						"ces.cloudogu.com/serviceaccount-path":        "/sa-management",
						"ces.cloudogu.com/serviceaccount-secret-name": "k8s-prometheus-api-key",
						"ces.cloudogu.com/serviceaccount-secret-key":  "theApiKey",
					},
					Labels: map[string]string{
						"app": "ces",
						"ces.cloudogu.com/serviceaccount-provider": "k8s-prometheus",
					},
				},
				Spec: v1.ServiceSpec{
					ClusterIP: "1.2.3.4",
				},
			},
			&v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "k8s-prometheus-api-key",
					Namespace: "testNs",
				},
				Data: map[string][]byte{
					"theApiKey": []byte("secretKey"),
				},
			},
		)

		doguCfg := config.CreateDoguConfig("grafana", config.Entries{
			"sa-k8s-prometheus/username": "testUser",
			"sa-k8s-prometheus/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, cescommons.SimpleName("grafana")).Return(doguCfg, nil)
		sensitiveConfigRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, nil)

		mockApiClient := newMockServiceAccountApiClient(t)
		mockApiClient.EXPECT().deleteServiceAccount(mock.Anything, "http://1.2.3.4:9977/sa-management", "secretKey", "grafana").Return(nil)

		r := remover{
			clientSet:         fakeClient,
			sensitiveDoguRepo: sensitiveConfigRepoMock,
			apiClient:         mockApiClient,
			namespace:         "testNs",
		}

		dogu := &core.Dogu{
			Name: "official/grafana",
			ServiceAccounts: []core.ServiceAccount{
				{Kind: "component", Type: "k8s-prometheus"},
			},
		}

		// when
		err := r.RemoveAllFromComponents(testCtx, dogu)

		// then
		require.NoError(t, err)
	})

	t.Run("skip removal because sensitive config not found", func(t *testing.T) {
		// given
		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, cescommons.SimpleName("grafana")).
			Return(config.DoguConfig{}, cloudoguerrors.NewNotFoundError(assert.AnError))

		r := remover{sensitiveDoguRepo: sensitiveConfigRepoMock}

		dogu := &core.Dogu{
			Name: "official/grafana",
			ServiceAccounts: []core.ServiceAccount{
				{Kind: "component", Type: "k8s-prometheus"},
			},
		}

		// when
		err := r.RemoveAllFromComponents(testCtx, dogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to get sensitive config", func(t *testing.T) {
		// given
		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, cescommons.SimpleName("grafana")).
			Return(config.DoguConfig{}, assert.AnError)

		r := remover{sensitiveDoguRepo: sensitiveConfigRepoMock}

		dogu := &core.Dogu{
			Name: "official/grafana",
			ServiceAccounts: []core.ServiceAccount{
				{Kind: "component", Type: "k8s-prometheus"},
			},
		}

		// when
		err := r.RemoveAllFromComponents(testCtx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to get sensitive config for dogu grafana")
	})

	t.Run("skip non-component service accounts", func(t *testing.T) {
		// given
		doguCfg := config.CreateDoguConfig("redmine", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, cescommons.SimpleName("redmine")).Return(doguCfg, nil)

		r := remover{sensitiveDoguRepo: sensitiveConfigRepoMock}

		dogu := &core.Dogu{
			Name: "official/redmine",
			ServiceAccounts: []core.ServiceAccount{
				{Kind: "dogu", Type: "postgresql"},
			},
		}

		// when
		err := r.RemoveAllFromComponents(testCtx, dogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to remove component service account", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientset()

		doguCfg := config.CreateDoguConfig("grafana", config.Entries{
			"sa-k8s-prometheus/username": "testUser",
			"sa-k8s-prometheus/password": "testPassword",
		})

		sensitiveConfigRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveConfigRepoMock.EXPECT().Get(mock.Anything, cescommons.SimpleName("grafana")).Return(doguCfg, nil)

		r := remover{
			clientSet:         fakeClient,
			sensitiveDoguRepo: sensitiveConfigRepoMock,
			namespace:         "testNs",
		}

		dogu := &core.Dogu{
			Name: "official/grafana",
			ServiceAccounts: []core.ServiceAccount{
				{Kind: "component", Type: "k8s-prometheus"},
			},
		}

		// when
		err := r.RemoveAllFromComponents(testCtx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "unable to remove service account for component k8s-prometheus")
	})
}
