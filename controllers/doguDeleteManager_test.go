package controllers

import (
	"context"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	cesremotemocks "github.com/cloudogu/cesapp-lib/remote/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

type doguDeleteManagerWithMocks struct {
	deleteManager             *doguDeleteManager
	doguRemoteRegistryMock    *cesremotemocks.Registry
	doguLocalRegistryMock     *cesmocks.DoguRegistry
	imageRegistryMock         *mocks.ImageRegistry
	doguRegistratorMock       *mocks.DoguRegistrator
	serviceAccountRemoverMock *mocks.ServiceAccountRemover
}

func (d *doguDeleteManagerWithMocks) AssertMocks(t *testing.T) {
	t.Helper()
	mock.AssertExpectationsForObjects(t,
		d.doguRemoteRegistryMock,
		d.doguLocalRegistryMock,
		d.imageRegistryMock,
		d.doguRegistratorMock,
		d.serviceAccountRemoverMock,
	)
}

func getDoguDeleteManagerWithMocks() doguDeleteManagerWithMocks {
	scheme := getTestScheme()
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	doguRemoteRegistry := &cesremotemocks.Registry{}
	doguLocalRegistry := &cesmocks.DoguRegistry{}
	imageRegistry := &mocks.ImageRegistry{}
	doguRegistrator := &mocks.DoguRegistrator{}
	serviceAccountRemover := &mocks.ServiceAccountRemover{}

	doguDeleteManager := &doguDeleteManager{
		client:                k8sClient,
		scheme:                scheme,
		doguRemoteRegistry:    doguRemoteRegistry,
		doguLocalRegistry:     doguLocalRegistry,
		imageRegistry:         imageRegistry,
		doguRegistrator:       doguRegistrator,
		serviceAccountRemover: serviceAccountRemover,
	}

	return doguDeleteManagerWithMocks{
		deleteManager:             doguDeleteManager,
		doguRemoteRegistryMock:    doguRemoteRegistry,
		doguLocalRegistryMock:     doguLocalRegistry,
		imageRegistryMock:         imageRegistry,
		doguRegistratorMock:       doguRegistrator,
		serviceAccountRemoverMock: serviceAccountRemover,
	}
}

func TestNewDoguDeleteManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given

		// override default controller method to retrieve a kube config
		oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
		defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
		ctrl.GetConfigOrDie = func() *rest.Config {
			return &rest.Config{}
		}

		client := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := &cesmocks.Registry{}
		doguRegistry := &cesmocks.DoguRegistry{}
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		// when
		doguManager, err := NewDoguDeleteManager(client, operatorConfig, cesRegistry)

		// then
		require.NoError(t, err)
		require.NotNil(t, doguManager)
		mock.AssertExpectationsForObjects(t, cesRegistry, doguRegistry)
	})

	t.Run("fail when creating client", func(t *testing.T) {
		// given

		// override default controller method to return a config that fail the client creation
		oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
		defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
		ctrl.GetConfigOrDie = func() *rest.Config {
			return &rest.Config{ExecProvider: &api.ExecConfig{}, AuthProvider: &api.AuthProviderConfig{}}
		}

		client := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := &cesmocks.Registry{}

		// when
		doguManager, err := NewDoguDeleteManager(client, operatorConfig, cesRegistry)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
	})
}

func Test_doguDeleteManager_Delete(t *testing.T) {
	scheme := getTestScheme()
	ctx := context.Background()
	ldapCr := readTestDataLdapCr(t)
	ldapDogu := readTestDataLdapDogu(t)

	t.Run("successfully delete a dogu", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ldapCr).Build()
		managerWithMocks := getDoguDeleteManagerWithMocks()
		managerWithMocks.doguLocalRegistryMock.On("Get", "ldap").Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.On("RemoveAll", mock.Anything, ldapCr.ObjectMeta.Namespace, ldapDogu).Return(nil)
		managerWithMocks.doguRegistratorMock.On("UnregisterDogu", "ldap").Return(nil)
		managerWithMocks.deleteManager.client = client

		// when
		err := managerWithMocks.deleteManager.Delete(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)
		deletedDogu := k8sv1.Dogu{}
		err = client.Get(ctx, client2.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failed to update dogu status", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks()

		// when
		err := managerWithMocks.deleteManager.Delete(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update dogu status")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to get dogu descriptor", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks()
		managerWithMocks.deleteManager.client = fake.NewClientBuilder().WithScheme(scheme).WithObjects(ldapCr).Build()
		managerWithMocks.doguLocalRegistryMock.On("Get", "ldap").Return(nil, assert.AnError)

		// when
		err := managerWithMocks.deleteManager.Delete(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to get dogu")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failure during service account removal should not interrupt the delete routine", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ldapCr).Build()
		managerWithMocks := getDoguDeleteManagerWithMocks()
		managerWithMocks.doguLocalRegistryMock.On("Get", "ldap").Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.On("RemoveAll", mock.Anything, ldapCr.ObjectMeta.Namespace, ldapDogu).Return(assert.AnError)
		managerWithMocks.doguRegistratorMock.On("UnregisterDogu", "ldap").Return(nil)
		managerWithMocks.deleteManager.client = client

		// when
		err := managerWithMocks.deleteManager.Delete(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)
		deletedDogu := k8sv1.Dogu{}
		err = client.Get(ctx, client2.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failure during unregister should not interrupt the delete routine", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ldapCr).Build()
		managerWithMocks := getDoguDeleteManagerWithMocks()
		managerWithMocks.doguLocalRegistryMock.On("Get", "ldap").Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.On("RemoveAll", mock.Anything, ldapCr.ObjectMeta.Namespace, ldapDogu).Return(nil)
		managerWithMocks.doguRegistratorMock.On("UnregisterDogu", "ldap").Return(assert.AnError)
		managerWithMocks.deleteManager.client = client

		// when
		err := managerWithMocks.deleteManager.Delete(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)
		deletedDogu := k8sv1.Dogu{}
		err = client.Get(ctx, client2.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})
}
