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
	mocks2 "github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry/mocks"
)

func TestEtcdDoguRegistrator_RegisterNewDogu(t *testing.T) {
	ctx := context.TODO()
	scheme := getTestScheme()
	// fake k8sClient
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	ldapCr := &corev1.Dogu{
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
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		globalConfigMock := &cesmocks.ConfigurationContext{}
		doguResourceGenerator := &mocks2.SecretResourceGenerator{}
		doguConfigMock.Mock.On("Set", mock.Anything, mock.Anything).Return(nil)
		doguRegistryMock.Mock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.Mock.On("Enable", ldapDogu).Return(nil)
		doguRegistryMock.Mock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.Mock.On("GlobalConfig").Return(globalConfigMock)
		doguResourceGenerator.Mock.On("CreateDoguSecret", mock.Anything, mock.Anything).Return(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ldap-private", Namespace: "clusterns"}}, nil)
		globalConfigMock.Mock.On("Get", "key_provider").Return("", nil)
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock, doguConfigMock, globalConfigMock)
	})

	t.Run("fail to check if dogu is already registered", func(t *testing.T) {
		// given
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguResourceGenerator := &mocks2.SecretResourceGenerator{}
		doguRegistryMock.Mock.On("IsEnabled", "ldap").Return(false, assert.AnError)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if dogu is already installed and enabled")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock)
	})

	t.Run("skip registration because dogu is already registered", func(t *testing.T) {
		// given
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguResourceGenerator := &mocks2.SecretResourceGenerator{}
		doguRegistryMock.Mock.On("IsEnabled", "ldap").Return(true, nil)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock)
	})

	t.Run("fail to register dogu", func(t *testing.T) {
		// given
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguResourceGenerator := &mocks2.SecretResourceGenerator{}
		doguRegistryMock.Mock.On("Register", ldapDogu).Return(assert.AnError)
		doguRegistryMock.Mock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to register dogu")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock)
	})

	t.Run("fail to enable dogu", func(t *testing.T) {
		// given
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguResourceGenerator := &mocks2.SecretResourceGenerator{}
		doguRegistryMock.Mock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.Mock.On("IsEnabled", "ldap").Return(false, nil)
		doguRegistryMock.Mock.On("Enable", ldapDogu).Return(assert.AnError)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to enable dogu")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock)
	})

	t.Run("fail get key_provider", func(t *testing.T) {
		// given
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		globalConfigMock := &cesmocks.ConfigurationContext{}
		doguResourceGenerator := &mocks2.SecretResourceGenerator{}
		doguRegistryMock.Mock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.Mock.On("Enable", ldapDogu).Return(nil)
		doguRegistryMock.Mock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.Mock.On("GlobalConfig").Return(globalConfigMock)
		globalConfigMock.Mock.On("Get", "key_provider").Return("", assert.AnError)
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get key provider")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock, globalConfigMock)
	})

	t.Run("fail to write public key", func(t *testing.T) {
		// given
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		globalConfigMock := &cesmocks.ConfigurationContext{}
		doguResourceGenerator := &mocks2.SecretResourceGenerator{}
		doguConfigMock.Mock.On("Set", mock.Anything, mock.Anything).Return(assert.AnError)
		doguRegistryMock.Mock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.Mock.On("Enable", ldapDogu).Return(nil)
		doguRegistryMock.Mock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.Mock.On("GlobalConfig").Return(globalConfigMock)
		globalConfigMock.Mock.On("Get", "key_provider").Return("", nil)
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to write")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock, doguConfigMock, globalConfigMock)
	})

	t.Run("fail generate secret", func(t *testing.T) {
		// given
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		globalConfigMock := &cesmocks.ConfigurationContext{}
		doguResourceGenerator := &mocks2.SecretResourceGenerator{}
		doguConfigMock.Mock.On("Set", mock.Anything, mock.Anything).Return(nil)
		doguRegistryMock.Mock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.Mock.On("Enable", ldapDogu).Return(nil)
		doguRegistryMock.Mock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.Mock.On("GlobalConfig").Return(globalConfigMock)
		globalConfigMock.Mock.On("Get", "key_provider").Return("", nil)
		doguResourceGenerator.Mock.On("CreateDoguSecret", mock.Anything, mock.Anything).Return(nil, assert.AnError)
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to generate secret")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock, doguConfigMock, globalConfigMock)
	})

	t.Run("fail create secret", func(t *testing.T) {
		// given
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		globalConfigMock := &cesmocks.ConfigurationContext{}
		doguResourceGenerator := &mocks2.SecretResourceGenerator{}
		doguConfigMock.Mock.On("Set", mock.Anything, mock.Anything).Return(nil)
		doguRegistryMock.Mock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.Mock.On("Enable", ldapDogu).Return(nil)
		doguRegistryMock.Mock.On("IsEnabled", "ldap").Return(false, nil)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.Mock.On("GlobalConfig").Return(globalConfigMock)
		globalConfigMock.Mock.On("Get", "key_provider").Return("", nil)
		doguResourceGenerator.Mock.On("CreateDoguSecret", mock.Anything, mock.Anything).Return(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ldap-private", Namespace: "clusterns"}}, nil)
		registrator := NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		// when
		err := registrator.RegisterNewDogu(ctx, ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create secret")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock, doguConfigMock, globalConfigMock)
	})
}

func TestEtcdDoguRegistrator_RegisterDoguVersion(t *testing.T) {
	ldapDogu := &core.Dogu{
		Name:    "official/ldap",
		Version: "1.0.0",
	}

	t.Run("successfully register a new dogu version", func(t *testing.T) {
		// given
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		globalConfigMock := &cesmocks.ConfigurationContext{}
		doguRegistryMock.Mock.On("Register", ldapDogu).Return(nil)
		doguRegistryMock.Mock.On("Enable", ldapDogu).Return(nil)
		doguRegistryMock.Mock.On("IsEnabled", "ldap").Return(true, nil)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registrator := NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registrator.RegisterDoguVersion(ldapDogu)

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock, doguConfigMock, globalConfigMock)
	})

	t.Run("fail to check if dogu is already registered", func(t *testing.T) {
		// given
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguRegistryMock.Mock.On("IsEnabled", "ldap").Return(true, assert.AnError)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registrator := NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registrator.RegisterDoguVersion(ldapDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to check if dogu is already installed and enabled")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock)
	})
}

func TestCESDoguRegistrator_UnregisterDogu(t *testing.T) {
	t.Run("successfully unregister a dogu", func(t *testing.T) {
		// given
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := &cesmocks.Registry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		registryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		doguConfigMock.Mock.On("RemoveAll").Return(nil)
		doguRegistryMock.Mock.On("Unregister", "ldap").Return(nil)
		registrator := NewCESDoguRegistrator(client, registryMock, &mocks2.SecretResourceGenerator{})

		// when
		err := registrator.UnregisterDogu("ldap")

		// then
		require.NoError(t, err)
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock)
	})

	t.Run("failed to remove dogu config", func(t *testing.T) {
		// given
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := &cesmocks.Registry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		registryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		doguConfigMock.Mock.On("RemoveAll").Return(assert.AnError)
		registrator := NewCESDoguRegistrator(client, registryMock, &mocks2.SecretResourceGenerator{})

		// when
		err := registrator.UnregisterDogu("ldap")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to remove dogu config")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock, doguConfigMock)
	})

	t.Run("failed to unregister dogu", func(t *testing.T) {
		// given
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := &cesmocks.Registry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		registryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		doguConfigMock.Mock.On("RemoveAll").Return(nil)
		doguRegistryMock.Mock.On("Unregister", "ldap").Return(assert.AnError)
		registrator := NewCESDoguRegistrator(client, registryMock, &mocks2.SecretResourceGenerator{})

		// when
		err := registrator.UnregisterDogu("ldap")

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to unregister dogu")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock, doguConfigMock)
	})
}
