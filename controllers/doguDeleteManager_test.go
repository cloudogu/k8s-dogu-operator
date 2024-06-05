package controllers

import (
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"
	fake2 "k8s.io/client-go/kubernetes/fake"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	regclient "go.etcd.io/etcd/client/v2"
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
		mgrSet := &util.ManagerSet{}
		clientset := fake2.NewSimpleClientset()

		// when
		doguManager := NewDoguDeleteManager(client, operatorConfig, cesRegistry, mgrSet, nil, clientset)

		// then
		require.NotNil(t, doguManager)
	})
}

func Test_doguDeleteManager_Delete(t *testing.T) {
	scheme := getTestScheme()
	ldapCr := readDoguCr(t, ldapCrBytes)
	ldapDogu := readDoguDescriptor(t, ldapDoguDescriptorBytes)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&k8sv1.Dogu{}).WithObjects(ldapCr).Build()
	t.Run("successfully delete a dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, "ldap").Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.EXPECT().RemoveAll(testCtx, ldapDogu).Return(nil)
		managerWithMocks.doguRegistratorMock.EXPECT().UnregisterDogu(testCtx, "ldap").Return(nil)
		managerWithMocks.exposedPortRemover.EXPECT().RemoveExposedPorts(testCtx, ldapCr, ldapDogu).Return(nil)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv1.Dogu{}
		err = fakeClient.Get(testCtx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failed to update dogu status because the dogu cr is not found", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)

		// when
		err := managerWithMocks.deleteManager.Delete(testCtx, ldapCr)

		// then
		require.Error(t, err)
	})

	t.Run("failure during fetching local dogu should not interrupt the delete routine", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)

		keyNotFoundErr := regclient.Error{Code: regclient.ErrorCodeKeyNotFound}
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, "ldap").Return(nil, keyNotFoundErr)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv1.Dogu{}
		err = fakeClient.Get(testCtx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failure during service account removal should not interrupt the delete routine", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, "ldap").Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.EXPECT().RemoveAll(testCtx, ldapDogu).Return(assert.AnError)
		managerWithMocks.doguRegistratorMock.EXPECT().UnregisterDogu(testCtx, "ldap").Return(nil)
		managerWithMocks.exposedPortRemover.EXPECT().RemoveExposedPorts(testCtx, ldapCr, ldapDogu).Return(nil)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv1.Dogu{}
		err = fakeClient.Get(testCtx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failure during unregister should not interrupt the delete routine", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, "ldap").Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.EXPECT().RemoveAll(testCtx, ldapDogu).Return(nil)
		managerWithMocks.doguRegistratorMock.EXPECT().UnregisterDogu(testCtx, "ldap").Return(assert.AnError)
		managerWithMocks.exposedPortRemover.EXPECT().RemoveExposedPorts(testCtx, ldapCr, ldapDogu).Return(nil)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv1.Dogu{}
		err = fakeClient.Get(testCtx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failure during exposed port removal should not interrupt the delete routine", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, "ldap").Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.EXPECT().RemoveAll(testCtx, ldapDogu).Return(nil)
		managerWithMocks.doguRegistratorMock.EXPECT().UnregisterDogu(testCtx, "ldap").Return(nil)
		managerWithMocks.exposedPortRemover.EXPECT().RemoveExposedPorts(testCtx, ldapCr, ldapDogu).Return(assert.AnError)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv1.Dogu{}
		err = fakeClient.Get(testCtx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})
}
