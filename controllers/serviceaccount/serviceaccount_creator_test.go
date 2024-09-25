package serviceaccount

import (
	"bytes"
	"context"
	_ "embed"
	"github.com/cloudogu/k8s-registry-lib/config"
	"k8s.io/client-go/kubernetes/fake"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	eventV1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var testCtx = context.TODO()

//go:embed testdata/redmine-cr.yaml
var redmineBytes []byte
var redmineCr = &k8sv1.Dogu{}

//go:embed testdata/redmine-dogu.json
var redmineDescriptorBytes []byte
var redmineDescriptor = &core.Dogu{}

//go:embed testdata/redmine-dogu-two-sa.json
var redmineDescriptorTwoSaBytes []byte
var redmineDescriptorTwoSa = &core.Dogu{}

//go:embed testdata/redmine-dogu-optional.json
var redmineDescriptorOptionalBytes []byte
var redmineDescriptorOptional = &core.Dogu{}

//go:embed testdata/redmine-dogu-ces-sa.json
var redmineDescriptorCesSaBytes []byte
var redmineDescriptorCesSa = &core.Dogu{}

//go:embed testdata/postgresql-dogu.json
var postgresqlDescriptorBytes []byte
var postgresqlDescriptor = &core.Dogu{}

//go:embed testdata/postgresql-cr.yaml
var postgresqlCrBytes []byte
var postgresqlCr = &k8sv1.Dogu{}

//go:embed testdata/cas-cr.yaml
var casCrBytes []byte
var casCr = &k8sv1.Dogu{}

//go:embed testdata/invalid-sa-dogu.json
var invalidPostgresqlDescriptorBytes []byte
var invalidPostgresqlDescriptor = &core.Dogu{}

//go:embed testdata/cas-dogu.json
var casDescriptorBytes []byte
var casDescriptor = &core.Dogu{}

func init() {
	err := yaml.Unmarshal(redmineBytes, redmineCr)
	if err != nil {
		panic(err)
	}
	redmineCr.Namespace = "test"

	err = json.Unmarshal(redmineDescriptorBytes, redmineDescriptor)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(redmineDescriptorTwoSaBytes, redmineDescriptorTwoSa)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(redmineDescriptorOptionalBytes, redmineDescriptorOptional)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(redmineDescriptorCesSaBytes, redmineDescriptorCesSa)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(postgresqlDescriptorBytes, postgresqlDescriptor)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(invalidPostgresqlDescriptorBytes, invalidPostgresqlDescriptor)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(casDescriptorBytes, casDescriptor)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(postgresqlCrBytes, postgresqlCr)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(casCrBytes, casCr)
	if err != nil {
		panic(err)
	}
}

func TestNewCreator(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		repo := NewMockSensitiveDoguConfigRepository(t)

		// when
		result := NewCreator(repo, nil, nil, nil, nil, "")

		// then
		assert.NotNil(t, result)
	})
}

func TestServiceAccountCreator_CreateServiceAccounts(t *testing.T) {
	buf := bytes.NewBufferString("username:user\npassword:password\ndatabase:dbname")

	var postgresCreateExposedCmd core.ExposedCommand
	for _, command := range postgresqlDescriptor.ExposedCommands {
		if command.Name == "service-account-create" {
			postgresCreateExposedCmd = command
			break
		}
	}
	require.NotNil(t, postgresCreateExposedCmd)

	readyPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap-xyz", Labels: postgresqlCr.GetPodLabels()},
		Status:     v1.PodStatus{Conditions: []v1.PodCondition{{Type: v1.ContainersReady, Status: v1.ConditionTrue}}},
	}
	cesControlPod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "k8s-ces-control-2334",
		Labels: map[string]string{"app": "k8s-ces-control"}}}
	cli := fake2.NewClientBuilder().
		WithScheme(getTestScheme()).
		WithObjects(readyPod, cesControlPod).
		Build()

	var cmdParams []string
	cmdParams = append(cmdParams, "service-account-create")
	cmdParams = append(cmdParams, "redmine")

	t.Run("success with dogu account", func(t *testing.T) {
		// given
		sensitiveDoguCfgRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)
		sensitiveDoguCfgRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, nil)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")
		commandExecutorMock := mocks.NewCommandExecutor(t)
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(buf, nil)

		localFetcher := mocks.NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(true, nil)
		serviceAccountCreator := creator{
			client:            cli,
			sensitiveDoguRepo: sensitiveDoguCfgRepoMock,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptorCesSa)

		// then
		require.NoError(t, err)
	})

	t.Run("service account already exists", func(t *testing.T) {
		// given
		sensitiveDoguCfgRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", config.Entries{
			"sa-postgresql/username": "testUser",
			"sa-postgresql/password": "testPassword",
			"sa-postgresql/database": "testDatabase",
		}), nil)

		serviceAccountCreator := creator{sensitiveDoguRepo: sensitiveDoguCfgRepoMock}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptorCesSa)

		// then
		require.NoError(t, err)
	})

	t.Run("failed to check if service account dogu is enabled", func(t *testing.T) {
		// given
		localFetcher := mocks.NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(false, assert.AnError)

		sensitiveDoguCfgRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		serviceAccountCreator := creator{sensitiveDoguRepo: sensitiveDoguCfgRepoMock, doguFetcher: localFetcher}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if dogu postgresql is enabled")
	})

	t.Run("service account is optional", func(t *testing.T) {
		// given
		localFetcher := mocks.NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(false, nil)

		sensitiveDoguCfgRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		serviceAccountCreator := creator{sensitiveDoguRepo: sensitiveDoguCfgRepoMock, doguFetcher: localFetcher}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptorOptional)

		// then
		require.NoError(t, err)
	})

	t.Run("service account is not optional and service account dogu is not enabled", func(t *testing.T) {
		// given
		localFetcher := mocks.NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(false, nil)

		sensitiveDoguCfgRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		serviceAccountCreator := creator{sensitiveDoguRepo: sensitiveDoguCfgRepoMock, doguFetcher: localFetcher}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "service account dogu is not enabled and not optional")
	})

	t.Run("fail to get dogu.json from service account dogu", func(t *testing.T) {
		// given
		localFetcher := mocks.NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(nil, assert.AnError)

		sensitiveDoguCfgRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		serviceAccountCreator := creator{
			sensitiveDoguRepo: sensitiveDoguCfgRepoMock,
			doguFetcher:       localFetcher,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get service account dogu.json")
	})

	t.Run("fail to get service account producer pod", func(t *testing.T) {
		// given
		sensitiveDoguCfgRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		localFetcher := mocks.NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		cliWithoutReadyPod := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()

		serviceAccountCreator := creator{
			client:            cliWithoutReadyPod,
			sensitiveDoguRepo: sensitiveDoguCfgRepoMock,
			doguFetcher:       localFetcher,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "could not find service account producer pod postgresql")
	})

	t.Run("service account dogu does not expose service-account-create command", func(t *testing.T) {
		// given
		sensitiveDoguCfgRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		localFetcher := mocks.NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(invalidPostgresqlDescriptor, nil)
		serviceAccountCreator := creator{
			client:            cli,
			sensitiveDoguRepo: sensitiveDoguCfgRepoMock,
			doguFetcher:       localFetcher,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "service account dogu postgresql does not expose service-account-create command")
	})

	t.Run("fail to exec command", func(t *testing.T) {
		// given
		sensitiveDoguCfgRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")

		commandExecutorMock := mocks.NewCommandExecutor(t)
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(nil, assert.AnError)

		localFetcher := mocks.NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := creator{
			client:            cli,
			sensitiveDoguRepo: sensitiveDoguCfgRepoMock,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to execute command")
	})

	t.Run("fail on invalid executor output", func(t *testing.T) {
		// given
		sensitiveDoguCfgRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")

		commandExecutorMock := mocks.NewCommandExecutor(t)
		invalidBuffer := bytes.NewBufferString("username:user:invalid\npassword:password\ndatabase:dbname")
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(invalidBuffer, nil)

		localFetcher := mocks.NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := creator{
			client:            cli,
			sensitiveDoguRepo: sensitiveDoguCfgRepoMock,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid output from service account command on dogu")
	})

	t.Run("fail to set service account value", func(t *testing.T) {
		// given
		sensitiveDoguCfgRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")

		commandExecutorMock := mocks.NewCommandExecutor(t)
		buf := bytes.NewBufferString("username/username:user\nusername:user\npassword:password\ndatabase:dbname")
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(buf, nil)

		localFetcher := mocks.NewMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, "postgresql").Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := creator{
			client:            cli,
			sensitiveDoguRepo: sensitiveDoguCfgRepoMock,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
	})

	t.Run("fail to create component service account", func(t *testing.T) {
		// given
		fakeClient := fake.NewSimpleClientset()

		sensitiveDoguCfgRepoMock := NewMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		serviceAccountCreator := creator{
			clientSet:         fakeClient,
			sensitiveDoguRepo: sensitiveDoguCfgRepoMock,
		}
		dogu := &core.Dogu{
			Name: "official/grafana",
			ServiceAccounts: []core.ServiceAccount{
				{Kind: "component", Type: "k8s-prometheus"},
			},
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get service")
	})
}

func getTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v1",
		Kind:    "dogu",
	}, &k8sv1.Dogu{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}, &appsv1.Deployment{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}, &v1.Secret{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Service",
	}, &v1.Service{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "PersistentVolumeClaim",
	}, &v1.PersistentVolumeClaim{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	}, &v1.ConfigMap{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Event",
	}, &eventV1.Event{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}, &v1.Pod{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "PodList",
	}, &v1.PodList{})

	return scheme
}

func Test_creator_containsDependency(t *testing.T) {
	t.Run("return false if dependency slice is nil", func(t *testing.T) {
		// given
		saCreator := &creator{}

		// when
		result := saCreator.containsDependency(nil, "test")

		// then
		require.False(t, result)
	})
}

func Test_creator_isOptionalServiceAccount(t *testing.T) {
	t.Run("should return false if sa is not in optional and mandatory dependencies", func(t *testing.T) {
		// given
		saCreator := &creator{}
		dogu := &core.Dogu{}

		// when
		result := saCreator.isOptionalServiceAccount(dogu, "account")

		// then
		require.False(t, result)
	})
}
