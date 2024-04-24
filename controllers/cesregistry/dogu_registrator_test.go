package cesregistry

import (
	"context"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	corev1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
)

//go:embed testdata/examplePrivateKey
var privateKeyBytes []byte

func TestEtcdDoguRegistrator_RegisterNewDogu(t *testing.T) {
	ctx := context.TODO()
	scheme := getTestScheme()

	ldapCr := &corev1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: "clusterns"},
		Spec: corev1.DoguSpec{
			Name:    "official/ldap",
			Version: "1.0.0",
		},
	}
	ldapDogu := &core.Dogu{
		Name:    "official/ldap",
		Version: "1.0.0",
	}

	t.Run("successfully register a dogu", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		doguConfigMock.On("Set", "public.pem", mock.Anything).Return(nil)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.On("Enable", ldapDogu).Return(nil)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		doguResourceGenerator.On("CreateDoguSecret", mock.Anything, mock.Anything).Return(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ldap-private", Namespace: "clusterns"}}, nil)
		globalConfigMock.On("Get", "key_provider").Return("", nil)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to check if dogu is already registered", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, assert.AnError)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if dogu is already installed and enabled")
	})

	t.Run("skip registration because dogu is already registered", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(true, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to register dogu", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		doguRegistryMock.On("Register", ldapDogu).Return(assert.AnError)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to register dogu")
	})

	t.Run("fail get key_provider", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		globalConfigMock.On("Get", "key_provider").Return("", assert.AnError)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get key provider")
	})

	t.Run("fail to write public key", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		doguConfigMock.On("Set", "public.pem", mock.Anything).Return(assert.AnError)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		globalConfigMock.On("Get", "key_provider").Return("", nil)
		doguResourceGenerator.On("CreateDoguSecret", mock.Anything, mock.Anything).Return(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ldap-private", Namespace: "clusterns"}}, nil)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to write")
	})

	t.Run("fail generate secret", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		globalConfigMock.On("Get", "key_provider").Return("", nil)
		doguResourceGenerator.On("CreateDoguSecret", mock.Anything, mock.Anything).Return(nil, assert.AnError)
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to generate secret")
	})

	t.Run("fail to enable dogu", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		doguResourceGenerator := mocks.NewSecretResourceGenerator(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguConfigMock.On("Set", "public.pem", mock.Anything).Return(nil)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		doguRegistryMock.On("Enable", ldapDogu).Return(assert.AnError)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		globalConfigMock.On("Get", "key_provider").Return("", nil)
		doguResourceGenerator.On("CreateDoguSecret", mock.Anything, mock.Anything).Return(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ldap-private", Namespace: "clusterns"}}, nil)
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to enable dogu")
	})

	t.Run("success with existing private key", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		globalConfigMock.On("Get", "key_provider").Return("", nil)
		doguConfigMock.On("Set", "public.pem", mock.Anything).Return(nil)
		doguRegistryMock.On("Enable", ldapDogu).Return(nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)

		existingSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "ldap-private", Namespace: "clusterns"},
			Data:       map[string][]byte{"private.pem": privateKeyBytes},
		}

		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingSecret).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, nil)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to get existing private key secret", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		globalConfigMock.On("Get", "key_provider").Return("", nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)

		client := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).WithObjects().Build()
		registrator := NewCESDoguRegistrator(client, registryMock, nil)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get private key secret for dogu ldap")
	})

	t.Run("failed to write public key from existing private key", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		globalConfigMock := cesmocks.NewConfigurationContext(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		globalConfigMock.On("Get", "key_provider").Return("", nil)
		doguConfigMock.On("Set", "public.pem", mock.Anything).Return(assert.AnError)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.On("GlobalConfig").Return(globalConfigMock)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)

		existingSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "ldap-private", Namespace: "clusterns"},
			Data:       map[string][]byte{"private.pem": privateKeyBytes},
		}

		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existingSecret).Build()
		registrator := NewCESDoguRegistrator(client, registryMock, nil)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to write public key from existing private key")
	})
}

func TestEtcdDoguRegistrator_RegisterDoguVersion(t *testing.T) {
	ldapDogu := &core.Dogu{
		Name:    "official/ldap",
		Version: "1.0.0",
	}

	t.Run("successfully register a new dogu version", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguRegistryMock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.On("Enable", ldapDogu).Return(nil)
		doguRegistryMock.On("IsEnabled", "ldap").Return(true, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registrator := NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registrator.RegisterDoguVersion(nil, ldapDogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to check if dogu is already registered", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(true, assert.AnError)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registrator := NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registrator.RegisterDoguVersion(nil, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if dogu is already installed and enabled")
	})

	t.Run("fail because the dogu is not enabled an no current key exists in upgrade process", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registrator := NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registrator.RegisterDoguVersion(nil, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "could not register dogu version: previous version not found")
	})

	t.Run("fail because the dogu cant be registered", func(t *testing.T) {
		// given
		registryMock := cesmocks.NewRegistry(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		doguRegistryMock.On("IsEnabled", "ldap").Return(true, nil)
		doguRegistryMock.On("Register", ldapDogu).Return(assert.AnError)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		registrator := NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registrator.RegisterDoguVersion(nil, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to register dogu ldap")
	})
}

func TestCESDoguRegistrator_UnregisterDogu(t *testing.T) {
	t.Run("successfully unregister a dogu", func(t *testing.T) {
		// given
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := cesmocks.NewRegistry(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		doguConfigMock.On("RemoveAll").Return(nil)
		doguRegistryMock.On("Unregister", "ldap").Return(nil)
		registrator := NewCESDoguRegistrator(client, registryMock, &mocks.SecretResourceGenerator{})

		// when
		err := registrator.UnregisterDogu(nil, "ldap")

		// then
		require.NoError(t, err)
	})

	t.Run("failed to remove dogu config", func(t *testing.T) {
		// given
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := cesmocks.NewRegistry(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		doguConfigMock.On("RemoveAll").Return(assert.AnError)
		registrator := NewCESDoguRegistrator(client, registryMock, &mocks.SecretResourceGenerator{})

		// when
		err := registrator.UnregisterDogu(nil, "ldap")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to remove dogu config")
	})

	t.Run("failed to unregister dogu", func(t *testing.T) {
		// given
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := cesmocks.NewRegistry(t)
		doguConfigMock := cesmocks.NewConfigurationContext(t)
		doguRegistryMock := cesmocks.NewDoguRegistry(t)
		registryMock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		doguConfigMock.On("RemoveAll").Return(nil)
		doguRegistryMock.On("Unregister", "ldap").Return(assert.AnError)
		registrator := NewCESDoguRegistrator(client, registryMock, &mocks.SecretResourceGenerator{})

		// when
		err := registrator.UnregisterDogu(nil, "ldap")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to unregister dogu")
	})
}
