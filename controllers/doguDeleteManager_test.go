package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	ctrl "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/k8s-registry-lib/repository"
)

type doguDeleteManagerWithMocks struct {
	deleteManager             *doguDeleteManager
	imageRegistryMock         *mockImageRegistry
	doguRegistratorMock       *mockDoguRegistrator
	localDoguFetcherMock      *mockLocalDoguFetcher
	serviceAccountRemoverMock *mockServiceAccountRemover
	doguConfigRepo            *mockDoguConfigRepository
	sensitiveConfigRepo       *mockDoguConfigRepository
}

func getDoguDeleteManagerWithMocks(t *testing.T) doguDeleteManagerWithMocks {
	k8sClient := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
	imageRegistry := newMockImageRegistry(t)
	doguRegistrator := newMockDoguRegistrator(t)
	serviceAccountRemover := newMockServiceAccountRemover(t)
	doguFetcher := newMockLocalDoguFetcher(t)
	doguConfigRepo := newMockDoguConfigRepository(t)
	sensitiveConfigRepo := newMockDoguConfigRepository(t)

	doguDeleteManager := &doguDeleteManager{
		client:                  k8sClient,
		localDoguFetcher:        doguFetcher,
		doguRegistrator:         doguRegistrator,
		serviceAccountRemover:   serviceAccountRemover,
		doguConfigRepository:    doguConfigRepo,
		sensitiveDoguRepository: sensitiveConfigRepo,
	}

	return doguDeleteManagerWithMocks{
		deleteManager:             doguDeleteManager,
		localDoguFetcherMock:      doguFetcher,
		imageRegistryMock:         imageRegistry,
		doguRegistratorMock:       doguRegistrator,
		serviceAccountRemoverMock: serviceAccountRemover,
		doguConfigRepo:            doguConfigRepo,
		sensitiveConfigRepo:       sensitiveConfigRepo,
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
		mgrSet := &util.ManagerSet{}

		configRepos := util.ConfigRepositories{
			GlobalConfigRepository:  &repository.GlobalConfigRepository{},
			DoguConfigRepository:    &repository.DoguConfigRepository{},
			SensitiveDoguRepository: &repository.DoguConfigRepository{},
		}

		// when
		doguManager := NewDoguDeleteManager(client, operatorConfig, mgrSet, nil, configRepos)

		// then
		require.NotNil(t, doguManager)
	})
}

func Test_doguDeleteManager_Delete(t *testing.T) {
	scheme := getTestScheme()
	ldapCr := readDoguCr(t, ldapCrBytes)
	ldapDogu := readDoguDescriptor(t, ldapDoguDescriptorBytes)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&k8sv2.Dogu{}).WithObjects(ldapCr).Build()
	t.Run("successfully delete a dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.EXPECT().RemoveAll(testCtx, ldapDogu).Return(nil)
		managerWithMocks.doguRegistratorMock.EXPECT().UnregisterDogu(testCtx, "ldap").Return(nil)
		managerWithMocks.doguConfigRepo.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.sensitiveConfigRepo.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv2.Dogu{}
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

		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(nil, assert.AnError)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv2.Dogu{}
		err = fakeClient.Get(testCtx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failure during service account removal should not interrupt the delete routine", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.EXPECT().RemoveAll(testCtx, ldapDogu).Return(assert.AnError)
		managerWithMocks.doguRegistratorMock.EXPECT().UnregisterDogu(testCtx, "ldap").Return(nil)
		managerWithMocks.doguConfigRepo.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.sensitiveConfigRepo.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv2.Dogu{}
		err = fakeClient.Get(testCtx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failure during unregister should not interrupt the delete routine", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.EXPECT().RemoveAll(testCtx, ldapDogu).Return(nil)
		managerWithMocks.doguRegistratorMock.EXPECT().UnregisterDogu(testCtx, "ldap").Return(assert.AnError)
		managerWithMocks.doguConfigRepo.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.sensitiveConfigRepo.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv2.Dogu{}
		err = fakeClient.Get(testCtx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failure during exposed port removal should not interrupt the delete routine", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.EXPECT().RemoveAll(testCtx, ldapDogu).Return(nil)
		managerWithMocks.doguRegistratorMock.EXPECT().UnregisterDogu(testCtx, "ldap").Return(nil)
		managerWithMocks.doguConfigRepo.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.sensitiveConfigRepo.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv2.Dogu{}
		err = fakeClient.Get(testCtx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failure during config removal should not interrupt the delete routine", func(t *testing.T) {
		// given
		managerWithMocks := getDoguDeleteManagerWithMocks(t)
		managerWithMocks.localDoguFetcherMock.EXPECT().FetchInstalled(testCtx, cescommons.SimpleName("ldap")).Return(ldapDogu, nil)
		managerWithMocks.serviceAccountRemoverMock.EXPECT().RemoveAll(testCtx, ldapDogu).Return(nil)
		managerWithMocks.doguRegistratorMock.EXPECT().UnregisterDogu(testCtx, "ldap").Return(nil)
		managerWithMocks.doguConfigRepo.EXPECT().Delete(mock.Anything, mock.Anything).Return(assert.AnError)
		managerWithMocks.sensitiveConfigRepo.EXPECT().Delete(mock.Anything, mock.Anything).Return(assert.AnError)
		managerWithMocks.deleteManager.client = fakeClient

		// when
		err := managerWithMocks.deleteManager.Delete(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		deletedDogu := k8sv2.Dogu{}
		err = fakeClient.Get(testCtx, runtimeclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})
}
