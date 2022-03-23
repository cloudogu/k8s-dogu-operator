package controllers_test

import (
	"context"
	"errors"
	cesmocks "github.com/cloudogu/cesapp/v4/registry/mocks"
	corev1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestEtcdDoguRegistrator_RegisterDogu(t *testing.T) {

	ctx := context.TODO()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v1",
		Kind:    "Dogu",
	}, &corev1.Dogu{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "app",
		Version: "v1",
		Kind:    "Secret",
	}, &v1.Secret{})
	// fake k8sClient
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	testErr := errors.New("test")

	t.Run("success", func(t *testing.T) {
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		doguResourceGenerator := &mocks.DoguResourceGenerator{}
		doguConfigMock.Mock.On("Set", mock.Anything, mock.Anything).Return(nil)
		doguRegistryMock.Mock.On("Register", mock.Anything).Return(nil)
		doguRegistryMock.Mock.On("Enable", mock.Anything).Return(nil)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		doguResourceGenerator.Mock.On("GetDoguSecret", mock.Anything, mock.Anything).Return(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ldap-private", Namespace: "clusterns"}}, nil)

		registrator := controllers.NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		err := registrator.RegisterDogu(ctx, ldapCr, ldapDogu)
		require.NoError(t, err)

		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock, doguConfigMock)
	})

	t.Run("fail to register dogu", func(t *testing.T) {
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguResourceGenerator := &mocks.DoguResourceGenerator{}
		doguRegistryMock.Mock.On("Register", mock.Anything).Return(testErr)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)

		registrator := controllers.NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		err := registrator.RegisterDogu(ctx, ldapCr, ldapDogu)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "failed to register dogu")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock)

	})

	t.Run("fail to enable dogu", func(t *testing.T) {
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguResourceGenerator := &mocks.DoguResourceGenerator{}
		doguRegistryMock.Mock.On("Register", mock.Anything).Return(nil)
		doguRegistryMock.Mock.On("Enable", mock.Anything).Return(testErr)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)

		registrator := controllers.NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		err := registrator.RegisterDogu(ctx, ldapCr, ldapDogu)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "failed to enable dogu")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock)

	})

	t.Run("fail to write public key", func(t *testing.T) {
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		doguResourceGenerator := &mocks.DoguResourceGenerator{}
		doguConfigMock.Mock.On("Set", mock.Anything, mock.Anything).Return(testErr)
		doguRegistryMock.Mock.On("Register", mock.Anything).Return(nil)
		doguRegistryMock.Mock.On("Enable", mock.Anything).Return(nil)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigMock)

		registrator := controllers.NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		err := registrator.RegisterDogu(ctx, ldapCr, ldapDogu)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "failed to write")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock, doguConfigMock)
	})
	t.Run("fail generate secret", func(t *testing.T) {
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		doguResourceGenerator := &mocks.DoguResourceGenerator{}
		doguConfigMock.Mock.On("Set", mock.Anything, mock.Anything).Return(nil)
		doguRegistryMock.Mock.On("Register", mock.Anything).Return(nil)
		doguRegistryMock.Mock.On("Enable", mock.Anything).Return(nil)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		doguResourceGenerator.Mock.On("GetDoguSecret", mock.Anything, mock.Anything).Return(nil, testErr)

		registrator := controllers.NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		err := registrator.RegisterDogu(ctx, ldapCr, ldapDogu)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "failed to generate secret")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock, doguConfigMock)
	})

	t.Run("fail create secret", func(t *testing.T) {
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := &cesmocks.Registry{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		doguResourceGenerator := &mocks.DoguResourceGenerator{}
		doguConfigMock.Mock.On("Set", mock.Anything, mock.Anything).Return(nil)
		doguRegistryMock.Mock.On("Register", mock.Anything).Return(nil)
		doguRegistryMock.Mock.On("Enable", mock.Anything).Return(nil)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		registryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		doguResourceGenerator.Mock.On("GetDoguSecret", mock.Anything, mock.Anything).Return(&v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ldap-private", Namespace: "clusterns"}}, nil)

		registrator := controllers.NewCESDoguRegistrator(client, registryMock, doguResourceGenerator)

		err := registrator.RegisterDogu(ctx, ldapCr, ldapDogu)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "failed to create secret")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock, doguConfigMock)
	})
}

func TestCESDoguRegistrator_UnregisterDogu(t *testing.T) {
	testErr := errors.New("test")

	t.Run("success", func(t *testing.T) {
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := &cesmocks.Registry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		registryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		doguConfigMock.Mock.On("RemoveAll").Return(nil)
		doguRegistryMock.Mock.On("Unregister", mock.Anything).Return(nil)

		registrator := NewCESDoguRegistrator(client, registryMock, &ResourceGenerator{})

		err := registrator.UnregisterDogu("ldap")
		require.NoError(t, err)

		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock)
	})

	t.Run("failed to remove dogu config", func(t *testing.T) {
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := &cesmocks.Registry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		registryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		doguConfigMock.Mock.On("RemoveAll").Return(testErr)

		registrator := NewCESDoguRegistrator(client, registryMock, &ResourceGenerator{})

		err := registrator.UnregisterDogu("ldap")
		require.Error(t, err)

		assert.Contains(t, err.Error(), "failed to remove dogu config")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock, doguConfigMock)
	})

	t.Run("failed to unregister dogu", func(t *testing.T) {
		scheme := runtime.NewScheme()
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		registryMock := &cesmocks.Registry{}
		doguConfigMock := &cesmocks.ConfigurationContext{}
		doguRegistryMock := &cesmocks.DoguRegistry{}
		registryMock.Mock.On("DoguConfig", mock.Anything).Return(doguConfigMock)
		registryMock.Mock.On("DoguRegistry").Return(doguRegistryMock)
		doguConfigMock.Mock.On("RemoveAll").Return(nil)
		doguRegistryMock.Mock.On("Unregister", mock.Anything).Return(testErr)

		registrator := NewCESDoguRegistrator(client, registryMock, &ResourceGenerator{})

		err := registrator.UnregisterDogu("ldap")
		require.Error(t, err)

		assert.Contains(t, err.Error(), "failed to unregister dogu")
		mock.AssertExpectationsForObjects(t, registryMock, doguRegistryMock, doguConfigMock)
	})
}
