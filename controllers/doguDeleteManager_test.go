package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	regclient "go.etcd.io/etcd/client/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
)

type doguDeleteManagerWithMocks struct {
	deleteManager             *doguDeleteManager
	imageRegistryMock         *mocks.ImageRegistry
	doguRegistratorMock       *mocks.DoguRegistrator
	localDoguFetcherMock      *mocks.LocalDoguFetcher
	serviceAccountRemoverMock *mocks.ServiceAccountRemover
	exposedPortRemover        *mocks.ExposePortRemover
}

func getDoguDeleteManagerWithMocks(t *testing.T) doguDeleteManagerWithMocks {
	k8sClient := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
	imageRegistry := mocks.NewImageRegistry(t)
	doguRegistrator := mocks.NewDoguRegistrator(t)
	serviceAccountRemover := mocks.NewServiceAccountRemover(t)
	doguFetcher := mocks.NewLocalDoguFetcher(t)
	exposedPortRemover := mocks.NewExposePortRemover(t)

	doguDeleteManager := &doguDeleteManager{
		client:                k8sClient,
		localDoguFetcher:      doguFetcher,
		imageRegistry:         imageRegistry,
		doguRegistrator:       doguRegistrator,
		serviceAccountRemover: serviceAccountRemover,
		exposedPortRemover:    exposedPortRemover,
	}

	return doguDeleteManagerWithMocks{
		deleteManager:             doguDeleteManager,
		localDoguFetcherMock:      doguFetcher,
		imageRegistryMock:         imageRegistry,
		doguRegistratorMock:       doguRegistrator,
		serviceAccountRemoverMock: serviceAccountRemover,
		exposedPortRemover:        exposedPortRemover,
	}
}

func TestNewDoguDeleteManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		// override default controller method to retrieve a kube config
		oldGetConfigDelegate := ctrl.GetConfig
		defer func() { ctrl.GetConfig = oldGetConfigDelegate }()
		ctrl.GetConfig = createTestRestConfig

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects().Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := cesmocks.NewRegistry(t)
		doguRegistry := cesmocks.NewDoguRegistry(t)
		cesRegistry.On("DoguRegistry").Return(doguRegistry)
		globalConfig := cesmocks.NewConfigurationContext(t)
		cesRegistry.On("GlobalConfig").Return(globalConfig)

		// when
		doguManager, err := NewDoguDeleteManager(client, operatorConfig, cesRegistry, nil)

		// then
		require.NoError(t, err)
		require.NotNil(t, doguManager)
	})

	t.Run("fail when creating client", func(t *testing.T) {
		// given

		// override default controller method to return a config that fail the client creation
		oldGetConfigDelegate := ctrl.GetConfig
		defer func() { ctrl.GetConfig = oldGetConfigDelegate }()
		ctrl.GetConfig = func() (*rest.Config, error) {
			return nil, assert.AnError
		}

		client := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := cesmocks.NewRegistry(t)

		// when
		doguManager, err := NewDoguDeleteManager(client, operatorConfig, cesRegistry, nil)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		require.Nil(t, doguManager)
	})
}

func Test_doguDeleteManager_Delete(t *testing.T) {
	scheme := getTestScheme()
	ctx := context.Background()
	ldapCr := readDoguCr(t, ldapCrBytes)
	ldapDogu := readDoguDescriptor(t, ldapDoguDescriptorBytes)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&k8sv1.Dogu{}).WithObjects(ldapCr).Build()
	t.Run("successfully delete a dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled("ldap").Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.EXPECT().RemoveAll(ctx, ldapDogu).Return(nil)
		managerWithMocks.doguRegistratorMock.EXPECT().UnregisterDogu("ldap").Return(nil)
		managerWithMocks.exposedPortRemover.EXPECT().RemoveExposedPorts(ctx, ldapCr, ldapDogu).Return(nil)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(ctx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv1.Dogu{}
		err = fakeClient.Get(ctx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failed to update dogu status", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)

		// when
		err := managerWithMocks.deleteManager.Delete(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update dogu status")
	})

	t.Run("failure during fetching local dogu should not interrupt the delete routine", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)

		keyNotFoundErr := regclient.Error{Code: regclient.ErrorCodeKeyNotFound}
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled("ldap").Return(nil, keyNotFoundErr)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(ctx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv1.Dogu{}
		err = fakeClient.Get(ctx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failure during service account removal should not interrupt the delete routine", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled("ldap").Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.EXPECT().RemoveAll(ctx, ldapDogu).Return(assert.AnError)
		managerWithMocks.doguRegistratorMock.EXPECT().UnregisterDogu("ldap").Return(nil)
		managerWithMocks.exposedPortRemover.EXPECT().RemoveExposedPorts(ctx, ldapCr, ldapDogu).Return(nil)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(ctx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv1.Dogu{}
		err = fakeClient.Get(ctx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failure during unregister should not interrupt the delete routine", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled("ldap").Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.EXPECT().RemoveAll(ctx, ldapDogu).Return(nil)
		managerWithMocks.doguRegistratorMock.EXPECT().UnregisterDogu("ldap").Return(assert.AnError)
		managerWithMocks.exposedPortRemover.EXPECT().RemoveExposedPorts(ctx, ldapCr, ldapDogu).Return(nil)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(ctx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv1.Dogu{}
		err = fakeClient.Get(ctx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failure during exposed port removal should not interrupt the delete routine", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled("ldap").Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.EXPECT().RemoveAll(ctx, ldapDogu).Return(nil)
		managerWithMocks.doguRegistratorMock.EXPECT().UnregisterDogu("ldap").Return(nil)
		managerWithMocks.exposedPortRemover.EXPECT().RemoveExposedPorts(ctx, ldapCr, ldapDogu).Return(assert.AnError)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(ctx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv1.Dogu{}
		err = fakeClient.Get(ctx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})
}
