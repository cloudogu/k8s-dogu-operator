package serviceaccount_test

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	cesmocks "github.com/cloudogu/cesapp/v4/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	testclient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"
	"k8s.io/client-go/tools/remotecommand"
	"net/url"
	"testing"
)

//go:embed testdata/redmine-cr.yaml
var redmineBytes []byte
var redmineCr = &k8sv1.Dogu{}

//go:embed testdata/redmine-dogu.json
var redmineDescriptorBytes []byte
var redmineDescriptor = &core.Dogu{}

//go:embed testdata/postgresql-dogu.json
var postgresqlDescriptorBytes []byte
var postgresqlDescriptor = &core.Dogu{}

//go:embed testdata/invalid-sa-dogu.json
var invalidPostgresqlDescriptorBytes []byte
var invalidPostgresqlDescriptor = &core.Dogu{}

func init() {
	err := yaml.Unmarshal(redmineBytes, redmineCr)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(redmineDescriptorBytes, redmineDescriptor)
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
}

type fakeExecutor struct {
	method string
	url    *url.URL
}

type fakeFailExecutor struct {
	method string
	url    *url.URL
}

type fakeInvalidOutputExecutor struct {
	method string
	url    *url.URL
}

func (f *fakeExecutor) Stream(options remotecommand.StreamOptions) error {
	if options.Stdout != nil {
		buf := new(bytes.Buffer)
		buf.WriteString("username:user\npassword:password\ndatabase:dbname")
		if _, err := options.Stdout.Write(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeFailExecutor) Stream(_ remotecommand.StreamOptions) error {
	return errors.New("test")
}
func (f *fakeInvalidOutputExecutor) Stream(options remotecommand.StreamOptions) error {
	if options.Stdout != nil {
		buf := new(bytes.Buffer)
		buf.WriteString("username:user:invalid\npassword:password\ndatabase:dbname")
		if _, err := options.Stdout.Write(buf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

func TestServiceAccountCreator_CreateServiceAccounts(t *testing.T) {
	testErr := errors.New("test")
	fakeNewSPDYExecutor := func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
		return &fakeExecutor{method: method, url: url}, nil
	}
	fakeErrorInitNewSPDYExecutor := func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
		return nil, testErr
	}
	fakeErrorStreamNewSPDYExecutor := func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
		return &fakeFailExecutor{method: method, url: url}, nil
	}
	fakeErrorInvalidOutputNewSPDYExecutor := func(config *rest.Config, method string, url *url.URL) (remotecommand.Executor, error) {
		return &fakeInvalidOutputExecutor{method: method, url: url}, nil
	}
	validPubKey := "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEApbhnnaIIXCADt0V7UCM7\nZfBEhpEeB5LTlvISkPQ91g+l06/soWFD65ba0PcZbIeKFqr7vkMB0nDNxX1p8PGv\nVJdUmwdB7U/bQlnO6c1DoY10g29O7itDfk92RCKeU5Vks9uRQ5ayZMjxEuahg2BW\nua72wi3GCiwLa9FZxGIP3hcYB21O6PfpxXsQYR8o3HULgL1ppDpuLv4fk/+jD31Z\n9ACoWOg6upyyNUsiA3hS9Kn1p3scVgsIN2jSSpxW42NvMo6KQY1Zo0N4Aw/mqySd\n+zdKytLqFto1t0gCbTCFPNMIObhWYXmAe26+h1b1xUI8ymsrXklwJVn0I77j9MM1\nHQIDAQAB\n-----END PUBLIC KEY-----"
	redmineCr.Namespace = "test"
	t.Run("success", func(t *testing.T) {
		// given
		fmt.Println("Debug1")
		ctx := context.TODO()
		fmt.Println("Debug2")
		globalConfig := &cesmocks.ConfigurationContext{}
		globalConfig.Mock.On("Get", "key_provider").Return("pkcs1v15", nil)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		doguConfig.Mock.On("Get", "public.pem").Return(validPubKey, nil)
		doguConfig.Mock.On("Set", "/sa-postgresql/username", mock.Anything).Return(nil)
		doguConfig.Mock.On("Set", "/sa-postgresql/password", mock.Anything).Return(nil)
		doguConfig.Mock.On("Set", "/sa-postgresql/database", mock.Anything).Return(nil)
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(postgresqlDescriptor, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		registry.Mock.On("GlobalConfig").Return(globalConfig)
		fmt.Println("Debug3")
		labels := map[string]string{}
		labels["dogu"] = "postgresql"
		fmt.Println("Debug4")
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "postgresql", Namespace: "test", Labels: labels}}
		fmt.Println("Debug5")
		client := testclient.NewSimpleClientset(&pod)
		fmt.Println("Debug6")

		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)
		fmt.Println("Debug7")
		serviceAccountCreator.CommandExecutorCreator = fakeNewSPDYExecutor
		fmt.Println("Debug8")

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)
		fmt.Println("Debug9")

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, globalConfig, doguConfig, doguRegistry, registry)
	})

	t.Run("failed to check if service account exists", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, testErr)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		client := testclient.NewSimpleClientset()
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check if service account already exists")
		mock.AssertExpectationsForObjects(t, doguConfig, registry)
	})

	t.Run("service account already exists", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		client := testclient.NewSimpleClientset()
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, doguConfig, registry)
	})

	t.Run("service account dogu is not enabled", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(false, testErr)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		client := testclient.NewSimpleClientset()
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check if dogu postgresql is enabled")
		mock.AssertExpectationsForObjects(t, doguConfig, doguRegistry, registry)
	})

	t.Run("failed to get dogu.json from service account dogu", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(nil, testErr)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		client := testclient.NewSimpleClientset()
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get service account dogu.json")
		mock.AssertExpectationsForObjects(t, doguConfig, doguRegistry, registry)
	})

	t.Run("service account dogu does not expose service-account-create command", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(invalidPostgresqlDescriptor, nil)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		client := testclient.NewSimpleClientset()
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service account dogu does not expose create command")
		mock.AssertExpectationsForObjects(t, doguConfig, doguRegistry, registry)
	})

	t.Run("found no pods for service account dogu", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(postgresqlDescriptor, nil)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		client := testclient.NewSimpleClientset()
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "found no pods for name postgresql")
		mock.AssertExpectationsForObjects(t, doguConfig, doguRegistry, registry)
	})

	t.Run("fail to create command executor", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(postgresqlDescriptor, nil)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		labels := map[string]string{}
		labels["dogu"] = "postgresql"
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "postgresql", Namespace: "test", Labels: labels}}
		client := testclient.NewSimpleClientset(&pod)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)
		serviceAccountCreator.CommandExecutorCreator = fakeErrorInitNewSPDYExecutor

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create new spdy executor")
		mock.AssertExpectationsForObjects(t, doguConfig, doguRegistry, registry)
	})

	t.Run("fail to exec command executor stream", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(postgresqlDescriptor, nil)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		labels := map[string]string{}
		labels["dogu"] = "postgresql"
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "postgresql", Namespace: "test", Labels: labels}}
		client := testclient.NewSimpleClientset(&pod)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)
		serviceAccountCreator.CommandExecutorCreator = fakeErrorStreamNewSPDYExecutor

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to exec stream")
		mock.AssertExpectationsForObjects(t, doguConfig, doguRegistry, registry)
	})

	t.Run("fail on invalid executor output", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(postgresqlDescriptor, nil)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		labels := map[string]string{}
		labels["dogu"] = "postgresql"
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "postgresql", Namespace: "test", Labels: labels}}
		client := testclient.NewSimpleClientset(&pod)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)
		serviceAccountCreator.CommandExecutorCreator = fakeErrorInvalidOutputNewSPDYExecutor

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid output from service account command on dogu")
		mock.AssertExpectationsForObjects(t, doguConfig, doguRegistry, registry)
	})

	t.Run("fail to get key_provider", func(t *testing.T) {
		// given
		ctx := context.TODO()
		globalConfig := &cesmocks.ConfigurationContext{}
		globalConfig.Mock.On("Get", "key_provider").Return("", testErr)
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(postgresqlDescriptor, nil)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("GlobalConfig").Return(globalConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		labels := map[string]string{}
		labels["dogu"] = "postgresql"
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "postgresql", Namespace: "test", Labels: labels}}
		client := testclient.NewSimpleClientset(&pod)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)
		serviceAccountCreator.CommandExecutorCreator = fakeNewSPDYExecutor

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get key provider")
		mock.AssertExpectationsForObjects(t, doguConfig, globalConfig, doguRegistry, registry)
	})

	t.Run("fail to create key_provider", func(t *testing.T) {
		// given
		ctx := context.TODO()
		globalConfig := &cesmocks.ConfigurationContext{}
		globalConfig.Mock.On("Get", "key_provider").Return("invalid", nil)
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(postgresqlDescriptor, nil)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("GlobalConfig").Return(globalConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		labels := map[string]string{}
		labels["dogu"] = "postgresql"
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "postgresql", Namespace: "test", Labels: labels}}
		client := testclient.NewSimpleClientset(&pod)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)
		serviceAccountCreator.CommandExecutorCreator = fakeNewSPDYExecutor

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create keyprovider")
		mock.AssertExpectationsForObjects(t, doguConfig, globalConfig, doguRegistry, registry)
	})

	t.Run("fail to get dogu public key", func(t *testing.T) {
		// given
		ctx := context.TODO()
		globalConfig := &cesmocks.ConfigurationContext{}
		globalConfig.Mock.On("Get", "key_provider").Return("", nil)
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(postgresqlDescriptor, nil)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		doguConfig.Mock.On("Get", "public.pem").Return("", testErr)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("GlobalConfig").Return(globalConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		labels := map[string]string{}
		labels["dogu"] = "postgresql"
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "postgresql", Namespace: "test", Labels: labels}}
		client := testclient.NewSimpleClientset(&pod)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)
		serviceAccountCreator.CommandExecutorCreator = fakeNewSPDYExecutor

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get dogu public key")
		mock.AssertExpectationsForObjects(t, doguConfig, globalConfig, doguRegistry, registry)
	})

	t.Run("fail to read public key from string", func(t *testing.T) {
		// given
		ctx := context.TODO()
		globalConfig := &cesmocks.ConfigurationContext{}
		globalConfig.Mock.On("Get", "key_provider").Return("", nil)
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(postgresqlDescriptor, nil)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		doguConfig.Mock.On("Get", "public.pem").Return("invalid", testErr)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("GlobalConfig").Return(globalConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		labels := map[string]string{}
		labels["dogu"] = "postgresql"
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "postgresql", Namespace: "test", Labels: labels}}
		client := testclient.NewSimpleClientset(&pod)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)
		serviceAccountCreator.CommandExecutorCreator = fakeNewSPDYExecutor

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read public key from string")
		mock.AssertExpectationsForObjects(t, doguConfig, globalConfig, doguRegistry, registry)
	})

	t.Run("fail to set service account value", func(t *testing.T) {
		// given
		ctx := context.TODO()
		globalConfig := &cesmocks.ConfigurationContext{}
		globalConfig.Mock.On("Get", "key_provider").Return("", nil)
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(true, nil)
		doguRegistry.Mock.On("Get", "postgresql").Return(postgresqlDescriptor, nil)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		doguConfig.Mock.On("Get", "public.pem").Return(validPubKey, nil)
		doguConfig.Mock.On("Set", mock.Anything, mock.Anything).Return(testErr)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("GlobalConfig").Return(globalConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		labels := map[string]string{}
		labels["dogu"] = "postgresql"
		pod := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "postgresql", Namespace: "test", Labels: labels}}
		client := testclient.NewSimpleClientset(&pod)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(client, &fake.RESTClient{}, registry)
		serviceAccountCreator.CommandExecutorCreator = fakeNewSPDYExecutor

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set encrypted sa value of key")
		mock.AssertExpectationsForObjects(t, doguConfig, globalConfig, doguRegistry, registry)
	})
}
