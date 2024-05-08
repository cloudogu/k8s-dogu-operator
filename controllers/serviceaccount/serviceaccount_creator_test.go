package serviceaccount

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
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
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"

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
		registryMock := cesmocks.NewRegistry(t)

		// when
		result := NewCreator(registryMock, nil, nil, nil, nil, "")

		// then
		assert.NotNil(t, result)
	})
}

func TestServiceAccountCreator_CreateServiceAccounts(t *testing.T) {
	validPubKey := "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEApbhnnaIIXCADt0V7UCM7\nZfBEhpEeB5LTlvISkPQ91g+l06/soWFD65ba0PcZbIeKFqr7vkMB0nDNxX1p8PGv\nVJdUmwdB7U/bQlnO6c1DoY10g29O7itDfk92RCKeU5Vks9uRQ5ayZMjxEuahg2BW\nua72wi3GCiwLa9FZxGIP3hcYB21O6PfpxXsQYR8o3HULgL1ppDpuLv4fk/+jD31Z\n9ACoWOg6upyyNUsiA3hS9Kn1p3scVgsIN2jSSpxW42NvMo6KQY1Zo0N4Aw/mqySd\n+zdKytLqFto1t0gCbTCFPNMIObhWYXmAe26+h1b1xUI8ymsrXklwJVn0I77j9MM1\nHQIDAQAB\n-----END PUBLIC KEY-----"
	invalidPubKey := "-----BEGIN PUBLIC KEY-----\nHQIDAQAB\n-----END PUBLIC KEY-----"
	buf := bytes.NewBufferString("username:user\npassword:password\ndatabase:dbname")
	cesControlBuf := bytes.NewBufferString("username:user\npassword:password")
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
	cesControlCreateSAShellCmd := exec.NewShellCommand(fmt.Sprintf("/%s/%s", k8sCesControl, k8sCesControl), cmdParams...)

	t.Run("success with dogu and ces control service account", func(t *testing.T) {
		// given
		globalConfig := cesmocks.NewConfigurationContext(t)
		globalConfig.Mock.On("Get", "key_provider").Return("pkcs1v15", nil)

		hostConfig := cesmocks.NewConfigurationContext(t)
		hostConfig.Mock.On("Exists", "redmine").Return(false, nil)

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		doguConfig.Mock.On("Exists", "sa-cesappd").Return(false, nil)
		doguConfig.Mock.On("Get", "public.pem").Return(validPubKey, nil)
		doguConfig.Mock.On("Set", "/sa-cesappd/username", mock.Anything).Return(nil)
		doguConfig.Mock.On("Set", "/sa-cesappd/password", mock.Anything).Return(nil)
		doguConfig.Mock.On("Set", "/sa-postgresql/username", mock.Anything).Return(nil)
		doguConfig.Mock.On("Set", "/sa-postgresql/password", mock.Anything).Return(nil)
		doguConfig.Mock.On("Set", "/sa-postgresql/database", mock.Anything).Return(nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("GlobalConfig").Return(globalConfig)
		registry.Mock.On("HostConfig", "k8s-ces-control").Return(hostConfig)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")
		commandExecutorMock := mocks.NewCommandExecutor(t)
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(buf, nil)
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, cesControlPod, cesControlCreateSAShellCmd, cloudogu.ContainersStarted).Return(cesControlBuf, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := creator{
			client:            cli,
			registry:          registry,
			doguFetcher:       localFetcher,
			localDoguRegistry: localDoguRegMock,
			executor:          commandExecutorMock,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptorCesSa)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to check if service account exists", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, assert.AnError)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		serviceAccountCreator := creator{registry: registry}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if service account already exists")
	})

	t.Run("fail to check if ces control service account already exists", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "sa-cesappd").Return(false, assert.AnError)
		serviceAccountCreator := creator{}
		sa := core.ServiceAccount{Kind: "ces", Type: "cesappd"}

		// when
		err := serviceAccountCreator.createCesControlServiceAccount(testCtx, redmineDescriptor, doguConfig, sa,
			"sa-cesappd")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to check if service account already exists")
	})

	t.Run("fail to check if ces control service account in hostconfig already exists", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "sa-cesappd").Return(false, nil)
		hostConfig := cesmocks.NewConfigurationContext(t)
		hostConfig.On("Exists", "redmine").Return(false, assert.AnError)
		registry := cesmocks.NewRegistry(t)
		registry.On("HostConfig", "k8s-ces-control").Return(hostConfig)

		serviceAccountCreator := creator{registry: registry}
		sa := core.ServiceAccount{Kind: "ces", Type: "cesappd"}

		// when
		err := serviceAccountCreator.createCesControlServiceAccount(testCtx, redmineDescriptor, doguConfig, sa,
			"sa-cesappd")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to check if service account already exists")
	})

	t.Run("skip if ces control service account in hostconfig already exists", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "sa-cesappd").Return(false, nil)
		hostConfig := cesmocks.NewConfigurationContext(t)
		hostConfig.On("Exists", "redmine").Return(true, nil)
		registry := cesmocks.NewRegistry(t)
		registry.On("HostConfig", "k8s-ces-control").Return(hostConfig)

		serviceAccountCreator := creator{registry: registry}
		sa := core.ServiceAccount{Kind: "ces", Type: "cesappd"}

		// when
		err := serviceAccountCreator.createCesControlServiceAccount(testCtx, redmineDescriptor, doguConfig, sa,
			"sa-cesappd")

		// then
		require.NoError(t, err)
		assert.Nil(t, err)
	})

	t.Run("skip if ces control service pod is not found and the sa is optional", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "sa-k8s-ces-control").Return(false, nil)
		hostConfig := cesmocks.NewConfigurationContext(t)
		hostConfig.On("Exists", "redmine").Return(false, nil)
		registry := cesmocks.NewRegistry(t)
		registry.On("HostConfig", "k8s-ces-control").Return(hostConfig)

		serviceAccountCreator := creator{
			client:   fake2.NewClientBuilder().WithScheme(getTestScheme()).Build(),
			registry: registry,
		}
		sa := core.ServiceAccount{Kind: "ces", Type: "k8s-ces-control"}

		// when
		err := serviceAccountCreator.createCesControlServiceAccount(testCtx, redmineDescriptorCesSa, doguConfig, sa,
			"sa-k8s-ces-control")

		// then
		require.NoError(t, err)
		assert.Nil(t, err)
	})

	t.Run("fail if ces control service pod is not found and the sa is mandatory", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "sa-cesappd").Return(false, nil)
		hostConfig := cesmocks.NewConfigurationContext(t)
		hostConfig.On("Exists", "redmine").Return(false, nil)
		registry := cesmocks.NewRegistry(t)
		registry.On("HostConfig", "k8s-ces-control").Return(hostConfig)

		serviceAccountCreator := creator{
			client:   fake2.NewClientBuilder().WithScheme(getTestScheme()).Build(),
			registry: registry,
		}
		sa := core.ServiceAccount{Kind: "ces", Type: "cesappd"}

		// when
		err := serviceAccountCreator.createCesControlServiceAccount(testCtx, redmineDescriptor, doguConfig, sa,
			"sa-cesappd")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get pod for labels")
	})

	t.Run("failed to exec command for host service account pod", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "sa-cesappd").Return(false, nil)
		hostConfig := cesmocks.NewConfigurationContext(t)
		hostConfig.On("Exists", "redmine").Return(false, nil)
		registry := cesmocks.NewRegistry(t)
		registry.On("HostConfig", "k8s-ces-control").Return(hostConfig)
		executor := mocks.NewCommandExecutor(t)
		executor.Mock.On("ExecCommandForPod", testCtx, cesControlPod, cesControlCreateSAShellCmd,
			cloudogu.ContainersStarted).Return(cesControlBuf, assert.AnError)

		serviceAccountCreator := creator{
			client:   cli,
			registry: registry,
			executor: executor,
		}
		sa := core.ServiceAccount{Kind: "ces", Type: "cesappd"}

		// when
		err := serviceAccountCreator.createCesControlServiceAccount(testCtx, redmineDescriptor, doguConfig, sa,
			"sa-cesappd")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to exec command")
	})

	t.Run("failed to parse command output for host service account", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "sa-cesappd").Return(false, nil)
		hostConfig := cesmocks.NewConfigurationContext(t)
		hostConfig.On("Exists", "redmine").Return(false, nil)
		registry := cesmocks.NewRegistry(t)
		registry.On("HostConfig", "k8s-ces-control").Return(hostConfig)
		executor := mocks.NewCommandExecutor(t)
		executor.Mock.On("ExecCommandForPod", testCtx, cesControlPod, cesControlCreateSAShellCmd,
			cloudogu.ContainersStarted).Return(bytes.NewBufferString("invalid:sa:output"), nil)

		serviceAccountCreator := creator{
			client:   cli,
			registry: registry,
			executor: executor,
		}
		sa := core.ServiceAccount{Kind: "ces", Type: "cesappd"}

		// when
		err := serviceAccountCreator.createCesControlServiceAccount(testCtx, redmineDescriptor, doguConfig, sa,
			"sa-cesappd")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse service account")
	})

	t.Run("failed to save host service account", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.On("Exists", "sa-cesappd").Return(false, nil)
		hostConfig := cesmocks.NewConfigurationContext(t)
		hostConfig.On("Exists", "redmine").Return(false, nil)
		globalConfig := cesmocks.NewConfigurationContext(t)
		globalConfig.On("Get", "key_provider").Return("", assert.AnError)
		registry := cesmocks.NewRegistry(t)
		registry.On("HostConfig", "k8s-ces-control").Return(hostConfig)
		registry.On("GlobalConfig").Return(globalConfig)
		executor := mocks.NewCommandExecutor(t)
		executor.Mock.On("ExecCommandForPod", testCtx, cesControlPod, cesControlCreateSAShellCmd,
			cloudogu.ContainersStarted).Return(cesControlBuf, nil)

		serviceAccountCreator := creator{
			client:   cli,
			registry: registry,
			executor: executor,
		}
		sa := core.ServiceAccount{Kind: "ces", Type: "cesappd"}

		// when
		err := serviceAccountCreator.createCesControlServiceAccount(testCtx, redmineDescriptor, doguConfig, sa,
			"sa-cesappd")

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to save service account")
	})

	t.Run("service account already exists", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)
		doguConfig.Mock.On("Exists", "sa-cesappd").Return(true, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		serviceAccountCreator := creator{registry: registry}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptorCesSa)

		// then
		require.NoError(t, err)
	})
	t.Run("failed to check if service account dogu is enabled", func(t *testing.T) {
		// given
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(false, assert.AnError)

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		serviceAccountCreator := creator{registry: registry, localDoguRegistry: localDoguRegMock}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if dogu postgresql is enabled")
	})
	t.Run("service account is optional", func(t *testing.T) {
		// given
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(false, nil)

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		serviceAccountCreator := creator{registry: registry, localDoguRegistry: localDoguRegMock}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptorOptional)

		// then
		require.NoError(t, err)
	})
	t.Run("service account is not optional and service account dogu is not enabled", func(t *testing.T) {
		// given
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(false, nil)

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		serviceAccountCreator := creator{registry: registry, localDoguRegistry: localDoguRegMock}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "service account dogu is not enabled and not optional")
	})
	t.Run("fail to get dogu.json from service account dogu", func(t *testing.T) {
		// given
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(nil, assert.AnError)
		serviceAccountCreator := creator{
			registry:          registry,
			doguFetcher:       localFetcher,
			localDoguRegistry: localDoguRegMock,
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
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		cliWithoutReadyPod := fake2.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()

		serviceAccountCreator := creator{
			client:            cliWithoutReadyPod,
			registry:          registry,
			doguFetcher:       localFetcher,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "could not find service account producer pod postgresql")
	})

	t.Run("service account dogu does not expose service-account-create command", func(t *testing.T) {
		// given
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(invalidPostgresqlDescriptor, nil)
		serviceAccountCreator := creator{
			client:            cli,
			registry:          registry,
			doguFetcher:       localFetcher,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "service account dogu postgresql does not expose service-account-create command")
	})
	t.Run("fail to exec command", func(t *testing.T) {
		// given
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")

		commandExecutorMock := mocks.NewCommandExecutor(t)
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(nil, assert.AnError)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := creator{
			client:            cli,
			registry:          registry,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
			localDoguRegistry: localDoguRegMock,
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
		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")

		commandExecutorMock := mocks.NewCommandExecutor(t)
		invalidBuffer := bytes.NewBufferString("username:user:invalid\npassword:password\ndatabase:dbname")
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(invalidBuffer, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := creator{
			client:            cli,
			registry:          registry,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "invalid output from service account command on dogu")
	})
	t.Run("fail to get key_provider", func(t *testing.T) {
		// given
		globalConfig := cesmocks.NewConfigurationContext(t)
		globalConfig.Mock.On("Get", "key_provider").Return("", assert.AnError)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("GlobalConfig").Return(globalConfig)
		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")

		commandExecutorMock := mocks.NewCommandExecutor(t)
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(buf, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := creator{
			client:            cli,
			registry:          registry,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get key provider")
	})
	t.Run("fail to create key_provider", func(t *testing.T) {
		// given
		globalConfig := cesmocks.NewConfigurationContext(t)
		globalConfig.Mock.On("Get", "key_provider").Return("invalid", nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("GlobalConfig").Return(globalConfig)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")

		commandExecutorMock := mocks.NewCommandExecutor(t)
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(buf, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := creator{
			client:            cli,
			registry:          registry,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create keyprovider")
	})
	t.Run("fail to get dogu public key", func(t *testing.T) {
		// given
		globalConfig := cesmocks.NewConfigurationContext(t)
		globalConfig.Mock.On("Get", "key_provider").Return("", nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		doguConfig.Mock.On("Get", "public.pem").Return("", assert.AnError)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("GlobalConfig").Return(globalConfig)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")

		commandExecutorMock := mocks.NewCommandExecutor(t)
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(buf, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := creator{
			client:            cli,
			registry:          registry,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get dogu public key")
	})
	t.Run("fail to read public key from string", func(t *testing.T) {
		// given
		globalConfig := cesmocks.NewConfigurationContext(t)
		globalConfig.Mock.On("Get", "key_provider").Return("", nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		doguConfig.Mock.On("Get", "public.pem").Return(invalidPubKey, nil)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("GlobalConfig").Return(globalConfig)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")

		commandExecutorMock := mocks.NewCommandExecutor(t)
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(buf, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := creator{
			client:            cli,
			registry:          registry,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to read public key from string")
	})

	t.Run("fail to set service account value", func(t *testing.T) {
		// given
		globalConfig := cesmocks.NewConfigurationContext(t)
		globalConfig.Mock.On("Get", "key_provider").Return("", nil)

		localDoguRegMock := extMocks.NewLocalDoguRegistry(t)
		localDoguRegMock.EXPECT().IsEnabled(testCtx, "postgresql").Return(true, nil)

		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		doguConfig.Mock.On("Get", "public.pem").Return(validPubKey, nil)
		doguConfig.Mock.On("Set", mock.Anything, mock.Anything).Return(assert.AnError)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("GlobalConfig").Return(globalConfig)

		postgresCreateSAShellCmd := exec.NewShellCommand(postgresCreateExposedCmd.Command, "redmine")

		commandExecutorMock := mocks.NewCommandExecutor(t)
		buf := bytes.NewBufferString("username:user\npassword:password\ndatabase:dbname")
		commandExecutorMock.Mock.On("ExecCommandForPod", testCtx, readyPod, postgresCreateSAShellCmd, cloudogu.PodReady).Return(buf, nil)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.EXPECT().FetchInstalled(testCtx, "postgresql").Return(postgresqlDescriptor, nil)
		serviceAccountCreator := creator{
			client:            cli,
			registry:          registry,
			doguFetcher:       localFetcher,
			executor:          commandExecutorMock,
			localDoguRegistry: localDoguRegMock,
		}

		// when
		err := serviceAccountCreator.CreateAll(testCtx, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set encrypted sa value of key")
	})

	t.Run("fail to create component service account", func(t *testing.T) {
		// given
		doguConfig := cesmocks.NewConfigurationContext(t)
		doguConfig.Mock.On("Exists", "sa-k8s-prometheus").Return(false, assert.AnError)

		registry := cesmocks.NewRegistry(t)
		registry.Mock.On("DoguConfig", "grafana").Return(doguConfig)

		serviceAccountCreator := creator{
			registry: registry,
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
		assert.ErrorContains(t, err, "failed to check if service account already exists")
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
