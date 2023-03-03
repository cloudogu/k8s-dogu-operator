package controllers

import (
	"context"
	"testing"

	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	client3 "go.etcd.io/etcd/client/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type doguDeleteManagerWithMocks struct {
	deleteManager             *doguDeleteManager
	imageRegistryMock         *mocks.ImageRegistry
	doguRegistratorMock       *mocks.DoguRegistrator
	localDoguFetcherMock      *mocks.LocalDoguFetcher
	serviceAccountRemoverMock *mocks.ServiceAccountRemover
}

func (d *doguDeleteManagerWithMocks) AssertMocks(t *testing.T) {
	t.Helper()
	mock.AssertExpectationsForObjects(t,
		d.imageRegistryMock,
		d.doguRegistratorMock,
		d.serviceAccountRemoverMock,
		d.localDoguFetcherMock,
	)
}

func getDoguDeleteManagerWithMocks() doguDeleteManagerWithMocks {
	k8sClient := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
	imageRegistry := &mocks.ImageRegistry{}
	doguRegistrator := &mocks.DoguRegistrator{}
	serviceAccountRemover := &mocks.ServiceAccountRemover{}
	doguFetcher := &mocks.LocalDoguFetcher{}

	doguDeleteManager := &doguDeleteManager{
		client:                k8sClient,
		localDoguFetcher:      doguFetcher,
		imageRegistry:         imageRegistry,
		doguRegistrator:       doguRegistrator,
		serviceAccountRemover: serviceAccountRemover,
	}

	return doguDeleteManagerWithMocks{
		deleteManager:             doguDeleteManager,
		localDoguFetcherMock:      doguFetcher,
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
		doguManager, err := NewDoguDeleteManager(client, cesRegistry)

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
		doguManager, err := NewDoguDeleteManager(client, cesRegistry)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
	})
}

func Test_doguDeleteManager_Delete(t *testing.T) {
	scheme := getTestScheme()
	ctx := context.Background()
	ldapCr := readDoguCr(t, ldapCrBytes)
	ldapDogu := readDoguDescriptor(t, ldapDoguDescriptorBytes)

	t.Run("successfully delete a dogu", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ldapCr).Build()
		managerWithMocks := getDoguDeleteManagerWithMocks()
		managerWithMocks.localDoguFetcherMock.On("FetchInstalled", "ldap").Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.On("RemoveAll", ctx, ldapDogu).Return(nil)
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
		assert.ErrorContains(t, err, "failed to update dogu status")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failure during fetching local dogu should not interrupt the delete routine", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ldapCr).Build()
		managerWithMocks := getDoguDeleteManagerWithMocks()

		keyNotFoundErr := client3.Error{Code: client3.ErrorCodeKeyNotFound}
		managerWithMocks.localDoguFetcherMock.On("FetchInstalled", "ldap").Return(nil, keyNotFoundErr)
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

	t.Run("failure during service account removal should not interrupt the delete routine", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(ldapCr).Build()
		managerWithMocks := getDoguDeleteManagerWithMocks()
		managerWithMocks.localDoguFetcherMock.On("FetchInstalled", "ldap").Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.On("RemoveAll", ctx, ldapDogu).Return(assert.AnError)
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
		managerWithMocks.localDoguFetcherMock.On("FetchInstalled", "ldap").Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.On("RemoveAll", ctx, ldapDogu).Return(nil)
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
