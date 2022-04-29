package serviceaccount_test

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"github.com/cloudogu/cesapp/v4/core"
	cesmocks "github.com/cloudogu/cesapp/v4/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	"testing"
)

//go:embed testdata/redmine-cr.yaml
var redmineBytes []byte
var redmineCr = &k8sv1.Dogu{}

//go:embed testdata/redmine-dogu.json
var redmineDescriptorBytes []byte
var redmineDescriptor = &core.Dogu{}

//go:embed testdata/redmine-dogu-optional.json
var redmineDescriptorOptionalBytes []byte
var redmineDescriptorOptional = &core.Dogu{}

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

	err = json.Unmarshal(redmineDescriptorOptionalBytes, redmineDescriptorOptional)
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

func TestServiceAccountCreator_CreateServiceAccounts(t *testing.T) {
	testErr := errors.New("test")
	validPubKey := "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEApbhnnaIIXCADt0V7UCM7\nZfBEhpEeB5LTlvISkPQ91g+l06/soWFD65ba0PcZbIeKFqr7vkMB0nDNxX1p8PGv\nVJdUmwdB7U/bQlnO6c1DoY10g29O7itDfk92RCKeU5Vks9uRQ5ayZMjxEuahg2BW\nua72wi3GCiwLa9FZxGIP3hcYB21O6PfpxXsQYR8o3HULgL1ppDpuLv4fk/+jD31Z\n9ACoWOg6upyyNUsiA3hS9Kn1p3scVgsIN2jSSpxW42NvMo6KQY1Zo0N4Aw/mqySd\n+zdKytLqFto1t0gCbTCFPNMIObhWYXmAe26+h1b1xUI8ymsrXklwJVn0I77j9MM1\nHQIDAQAB\n-----END PUBLIC KEY-----"
	invalidPubKey := "-----BEGIN PUBLIC KEY-----\nHQIDAQAB\n-----END PUBLIC KEY-----"
	redmineCr.Namespace = "test"
	buf := new(bytes.Buffer)
	buf.WriteString("username:user\npassword:password\ndatabase:dbname")

	t.Run("success", func(t *testing.T) {
		// given
		ctx := context.TODO()
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
		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommand", mock.Anything, "postgresql", "test", mock.Anything, mock.Anything).Return(buf, nil)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, commandExecutorMock)

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, globalConfig, doguConfig, doguRegistry, registry)
	})

	t.Run("fail to check if service account exists", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(true, testErr)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, nil)

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
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, nil)

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
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, nil)

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check if dogu postgresql is enabled")
		mock.AssertExpectationsForObjects(t, doguConfig, doguRegistry, registry)
	})

	t.Run("service account is optional", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(false, nil)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, nil)

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptorOptional)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, doguConfig, doguRegistry, registry)
	})

	t.Run("service account is not optional and service account dogu is not enabled", func(t *testing.T) {
		// given
		ctx := context.TODO()
		doguRegistry := &cesmocks.DoguRegistry{}
		doguRegistry.Mock.On("IsEnabled", "postgresql").Return(false, nil)
		doguConfig := &cesmocks.ConfigurationContext{}
		doguConfig.Mock.On("Exists", "sa-postgresql").Return(false, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, nil)

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service account dogu is not enabled and not optional")
		mock.AssertExpectationsForObjects(t, doguConfig, doguRegistry, registry)
	})

	t.Run("fail to get dogu.json from service account dogu", func(t *testing.T) {
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
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, nil)

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
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, nil)

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "service account dogu does not expose create command")
		mock.AssertExpectationsForObjects(t, doguConfig, doguRegistry, registry)
	})

	t.Run("fail to exec command", func(t *testing.T) {
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
		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommand", mock.Anything, "postgresql", "test", mock.Anything, mock.Anything).Return(nil, testErr)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, commandExecutorMock)

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to execute command")
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
		commandExecutorMock := &mocks.CommandExecutor{}
		buf := new(bytes.Buffer)
		buf.WriteString("username:user:invalid\npassword:password\ndatabase:dbname")
		commandExecutorMock.Mock.On("ExecCommand", mock.Anything, "postgresql", "test", mock.Anything, mock.Anything).Return(buf, nil)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, commandExecutorMock)

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
		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommand", mock.Anything, "postgresql", "test", mock.Anything, mock.Anything).Return(buf, nil)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, commandExecutorMock)

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
		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommand", mock.Anything, "postgresql", "test", mock.Anything, mock.Anything).Return(buf, nil)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, commandExecutorMock)

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
		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommand", mock.Anything, "postgresql", "test", mock.Anything, mock.Anything).Return(buf, nil)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, commandExecutorMock)

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
		doguConfig.Mock.On("Get", "public.pem").Return(invalidPubKey, nil)
		registry := &cesmocks.Registry{}
		registry.Mock.On("DoguConfig", "redmine").Return(doguConfig)
		registry.Mock.On("GlobalConfig").Return(globalConfig)
		registry.Mock.On("DoguRegistry").Return(doguRegistry)
		commandExecutorMock := &mocks.CommandExecutor{}
		commandExecutorMock.Mock.On("ExecCommand", mock.Anything, "postgresql", "test", mock.Anything, mock.Anything).Return(buf, nil)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, commandExecutorMock)

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
		commandExecutorMock := &mocks.CommandExecutor{}
		buf := new(bytes.Buffer)
		buf.WriteString("username:user\npassword:password\ndatabase:dbname")
		commandExecutorMock.Mock.On("ExecCommand", mock.Anything, "postgresql", "test", mock.Anything, mock.Anything).Return(buf, nil)
		serviceAccountCreator := serviceaccount.NewServiceAccountCreator(registry, commandExecutorMock)

		// when
		err := serviceAccountCreator.CreateServiceAccounts(ctx, redmineCr, redmineDescriptor)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to set encrypted sa value of key")
		mock.AssertExpectationsForObjects(t, doguConfig, globalConfig, doguRegistry, registry)
	})
}