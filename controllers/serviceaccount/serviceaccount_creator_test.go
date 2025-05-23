package serviceaccount

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	eventV1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/fake"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cloudoguerrors "github.com/cloudogu/ces-commons-lib/errors"
	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-registry-lib/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var testCtx = context.TODO()

//go:embed testdata/redmine-cr.yaml
var redmineBytes []byte
var redmineCr = &k8sv2.Dogu{}

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
var postgresqlCr = &k8sv2.Dogu{}

//go:embed testdata/cas-cr.yaml
var casCrBytes []byte
var casCr = &k8sv2.Dogu{}

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
		repo := newMockSensitiveDoguConfigRepository(t)

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

	cesControlPod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "k8s-ces-control-2334",
		Labels: map[string]string{"app": "k8s-ces-control"}}}
	availablePostgresqlDoguResource := &k8sv2.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "postgresql"},
		Spec:       k8sv2.DoguSpec{Resources: k8sv2.DoguResources{MinDataVolumeSize: resource.MustParse("0")}},
		Status:     k8sv2.DoguStatus{Health: k8sv2.AvailableHealthStatus},
	}
	cli := fake2.NewClientBuilder().
		WithScheme(getTestScheme()).
		WithObjects(cesControlPod, availablePostgresqlDoguResource).
		Build()

	var cmdParams []string
	cmdParams = append(cmdParams, "service-account-create")
	cmdParams = append(cmdParams, "redmine")

	t.Run("success with dogu account", func(t *testing.T) {
		// given
		sensitiveDoguCfgRepoMock := newMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)
		sensitiveDoguCfgRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, nil)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")
		commandExecutorMock := newMockCommandExecutor(t)
		commandExecutorMock.Mock.On("ExecCommandForDogu", testCtx, availablePostgresqlDoguResource, postgresCreateSAShellCmd, exec.PodReady).Return(buf, nil)

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(postgresqlDescriptor, nil)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
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
	t.Run("success with dogu account with merge", func(t *testing.T) {
		// given
		sensitiveDoguCfgRepoMock := newMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)
		sensitiveDoguCfgRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, cloudoguerrors.NewConflictError(errors.New("")))
		sensitiveDoguCfgRepoMock.EXPECT().SaveOrMerge(mock.Anything, mock.Anything).Return(config.DoguConfig{}, nil)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")
		commandExecutorMock := newMockCommandExecutor(t)
		commandExecutorMock.Mock.On("ExecCommandForDogu", testCtx, availablePostgresqlDoguResource, postgresCreateSAShellCmd, exec.PodReady).Return(buf, nil)

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(postgresqlDescriptor, nil)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
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
		sensitiveDoguCfgRepoMock := newMockSensitiveDoguConfigRepository(t)
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
		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(false, assert.AnError)

		sensitiveDoguCfgRepoMock := newMockSensitiveDoguConfigRepository(t)
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
		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(false, nil)

		sensitiveDoguCfgRepoMock := newMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		serviceAccountCreator := creator{sensitiveDoguRepo: sensitiveDoguCfgRepoMock, doguFetcher: localFetcher}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptorOptional)

		// then
		require.NoError(t, err)
	})

	t.Run("service account is not optional and service account dogu is not enabled", func(t *testing.T) {
		// given
		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(false, nil)

		sensitiveDoguCfgRepoMock := newMockSensitiveDoguConfigRepository(t)
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
		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(nil, assert.AnError)

		sensitiveDoguCfgRepoMock := newMockSensitiveDoguConfigRepository(t)
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

	t.Run("service account dogu does not expose service-account-create command", func(t *testing.T) {
		// given
		sensitiveDoguCfgRepoMock := newMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(invalidPostgresqlDescriptor, nil)
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

	t.Run("failed to get dogu resource of service account dogu", func(t *testing.T) {
		// given
		sensitiveDoguCfgRepoMock := newMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(postgresqlDescriptor, nil)

		cli := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()

		serviceAccountCreator := creator{
			client:            cli,
			sensitiveDoguRepo: sensitiveDoguCfgRepoMock,
			doguFetcher:       localFetcher,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to fetch dogu resource")
	})

	t.Run("fail to exec command", func(t *testing.T) {
		// given
		sensitiveDoguCfgRepoMock := newMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")

		commandExecutorMock := newMockCommandExecutor(t)
		commandExecutorMock.Mock.On("ExecCommandForDogu", testCtx, availablePostgresqlDoguResource, postgresCreateSAShellCmd, exec.PodReady).Return(nil, assert.AnError)

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(postgresqlDescriptor, nil)
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
		sensitiveDoguCfgRepoMock := newMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")

		commandExecutorMock := newMockCommandExecutor(t)
		invalidBuffer := bytes.NewBufferString("username:user:invalid\npassword:password\ndatabase:dbname")
		commandExecutorMock.Mock.On("ExecCommandForDogu", testCtx, availablePostgresqlDoguResource, postgresCreateSAShellCmd, exec.PodReady).Return(invalidBuffer, nil)

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(postgresqlDescriptor, nil)
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
		sensitiveDoguCfgRepoMock := newMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")

		commandExecutorMock := newMockCommandExecutor(t)
		buf := bytes.NewBufferString("username/username:user\nusername:user\npassword:password\ndatabase:dbname")
		commandExecutorMock.Mock.On("ExecCommandForDogu", testCtx, availablePostgresqlDoguResource, postgresCreateSAShellCmd, exec.PodReady).Return(buf, nil)

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(postgresqlDescriptor, nil)
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
		fakeClient := fake.NewClientset()

		sensitiveDoguCfgRepoMock := newMockSensitiveDoguConfigRepository(t)
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
	t.Run("fails to write Dogu config", func(t *testing.T) {
		// given
		sensitiveDoguCfgRepoMock := newMockSensitiveDoguConfigRepository(t)
		sensitiveDoguCfgRepoMock.EXPECT().Get(mock.Anything, mock.Anything).Return(config.CreateDoguConfig("test", make(config.Entries)), nil)
		sensitiveDoguCfgRepoMock.EXPECT().Update(mock.Anything, mock.Anything).Return(config.DoguConfig{}, cloudoguerrors.NewConflictError(errors.New("")))
		sensitiveDoguCfgRepoMock.EXPECT().SaveOrMerge(mock.Anything, mock.Anything).Return(config.DoguConfig{}, assert.AnError)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")
		commandExecutorMock := newMockCommandExecutor(t)
		commandExecutorMock.Mock.On("ExecCommandForDogu", testCtx, availablePostgresqlDoguResource, postgresCreateSAShellCmd, exec.PodReady).Return(buf, nil)

		localFetcher := newMockLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("postgresql")).Return(postgresqlDescriptor, nil)
		localFetcher.EXPECT().Enabled(testCtx, cescommons.SimpleName("postgresql")).Return(true, nil)
		serviceAccountCreator := creator{
			client:            cli,
			sensitiveDoguRepo: sensitiveDoguCfgRepoMock,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptorCesSa)

		// then
		require.Error(t, err)
	})
}

func getTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v2",
		Kind:    "dogu",
	}, &k8sv2.Dogu{})
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

func Test_readGetServiceAccountPodMaxRetriesEnv(t *testing.T) {
	tests := []struct {
		name           string
		want           int
		setEnv         bool
		actualEnvValue string
	}{
		{
			name:   "Environment variable not found",
			setEnv: false,
			want:   defaultMaxTries,
		},
		{
			name:           "failed to parse environment variable (string). Use default value",
			setEnv:         true,
			actualEnvValue: "NoIntValue",
			want:           defaultMaxTries,
		},
		{
			name:           "failed to parse environment variable (float). Use default value",
			setEnv:         true,
			actualEnvValue: "5.5",
			want:           defaultMaxTries,
		},
		{
			name:           "parsed environment variable is smaller than 0. Use default value",
			setEnv:         true,
			actualEnvValue: "-5",
			want:           defaultMaxTries,
		},
		{
			name:           "Successfully read environment variable",
			setEnv:         true,
			actualEnvValue: "10",
			want:           10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv("GET_SERVICE_ACCOUNT_POD_MAX_RETRIES", tt.actualEnvValue)
			}
			assert.Equalf(t, tt.want, readGetServiceAccountPodMaxRetriesEnv(), "readGetServiceAccountPodMaxRetriesEnv()")
		})
	}
}
