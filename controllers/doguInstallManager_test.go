package controllers

import (
	"context"
	"errors"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/k8s-registry-lib/repository"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	resErrors "github.com/cloudogu/ces-commons-lib/errors"
	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	resConfig "github.com/cloudogu/k8s-registry-lib/config"
)

type doguInstallManagerWithMocks struct {
	installManager                *doguInstallManager
	localDoguFetcher              *mockLocalDoguFetcher
	resourceDoguFetcher           *mockResourceDoguFetcher
	imageRegistryMock             *mockImageRegistry
	doguRegistratorMock           *mockDoguRegistrator
	dependencyValidatorMock       *mockDependencyValidator
	serviceAccountCreatorMock     *mockServiceAccountCreator
	applierMock                   *mockApplier
	fileExtractorMock             *mockFileExtractor
	client                        client.WithWatch
	resourceUpserter              *mockResourceUpserter
	recorder                      *mockEventRecorder
	execPodFactory                *mockExecPodFactory
	ecosystemClient               *mockEcosystemInterface
	doguInterface                 *mockDoguInterface
	doguConfigRepository          *mockDoguConfigRepository
	sensitiveDoguRepository       *mockDoguConfigRepository
	securityValidator             *mockSecurityValidator
	doguAdditionalMountsValidator *mockDoguAdditionalMountsValidator
}

func getDoguInstallManagerWithMocks(t *testing.T, scheme *runtime.Scheme) doguInstallManagerWithMocks {
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&doguv2.Dogu{}).Build()
	ecosystemClientMock := newMockEcosystemInterface(t)
	doguIntrefaceMock := newMockDoguInterface(t)
	upserter := newMockResourceUpserter(t)
	imageRegistry := newMockImageRegistry(t)
	doguRegistrator := newMockDoguRegistrator(t)
	dependencyValidator := newMockDependencyValidator(t)
	serviceAccountCreator := newMockServiceAccountCreator(t)
	mockedApplier := newMockApplier(t)
	fileExtract := newMockFileExtractor(t)
	eventRecorderMock := newMockEventRecorder(t)
	localDoguFetcher := newMockLocalDoguFetcher(t)
	resourceDoguFetcher := newMockResourceDoguFetcher(t)
	collectApplier := resource.NewCollectApplier(mockedApplier)
	podFactory := newMockExecPodFactory(t)
	doguConfigRepoMock := newMockDoguConfigRepository(t)
	sensitiveConfigRepoMock := newMockDoguConfigRepository(t)
	securityValidatorMock := newMockSecurityValidator(t)
	additionalMountValidatorMock := newMockDoguAdditionalMountsValidator(t)

	doguInstallManager := &doguInstallManager{
		client:                        k8sClient,
		ecosystemClient:               ecosystemClientMock,
		recorder:                      eventRecorderMock,
		imageRegistry:                 imageRegistry,
		doguRegistrator:               doguRegistrator,
		localDoguFetcher:              localDoguFetcher,
		resourceDoguFetcher:           resourceDoguFetcher,
		dependencyValidator:           dependencyValidator,
		serviceAccountCreator:         serviceAccountCreator,
		fileExtractor:                 fileExtract,
		collectApplier:                collectApplier,
		resourceUpserter:              upserter,
		execPodFactory:                podFactory,
		doguConfigRepository:          doguConfigRepoMock,
		sensitiveDoguRepository:       sensitiveConfigRepoMock,
		securityValidator:             securityValidatorMock,
		doguAdditionalMountsValidator: additionalMountValidatorMock,
	}

	return doguInstallManagerWithMocks{
		installManager:                doguInstallManager,
		client:                        k8sClient,
		recorder:                      eventRecorderMock,
		localDoguFetcher:              localDoguFetcher,
		resourceDoguFetcher:           resourceDoguFetcher,
		imageRegistryMock:             imageRegistry,
		doguRegistratorMock:           doguRegistrator,
		dependencyValidatorMock:       dependencyValidator,
		serviceAccountCreatorMock:     serviceAccountCreator,
		fileExtractorMock:             fileExtract,
		applierMock:                   mockedApplier,
		resourceUpserter:              upserter,
		execPodFactory:                podFactory,
		ecosystemClient:               ecosystemClientMock,
		doguInterface:                 doguIntrefaceMock,
		doguConfigRepository:          doguConfigRepoMock,
		sensitiveDoguRepository:       sensitiveConfigRepoMock,
		securityValidator:             securityValidatorMock,
		doguAdditionalMountsValidator: additionalMountValidatorMock,
	}
}

func getDoguInstallManagerTestData(t *testing.T) (*doguv2.Dogu, *core.Dogu, *corev1.ConfigMap, *imagev1.ConfigFile) {
	ldapCr := readDoguCr(t, ldapCrBytes)
	ldapDogu := readDoguDescriptor(t, ldapDoguDescriptorBytes)
	ldapDoguDescriptor := readDoguDevelopmentMap(t, ldapDoguDevelopmentMapBytes)
	imageConfig := readImageConfig(t, imageConfigBytes)
	return ldapCr, ldapDogu, ldapDoguDescriptor.ToConfigMap(), imageConfig
}

func TestNewDoguInstallManager(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigDelegate := ctrl.GetConfig
	defer func() { ctrl.GetConfig = oldGetConfigDelegate }()
	ctrl.GetConfig = createTestRestConfig

	t.Run("success", func(t *testing.T) {
		// given
		myClient := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		mgrSet := &util.ManagerSet{}
		eventRecorder := newMockEventRecorder(t)

		configRepos := util.ConfigRepositories{
			GlobalConfigRepository:  &repository.GlobalConfigRepository{},
			DoguConfigRepository:    &repository.DoguConfigRepository{},
			SensitiveDoguRepository: &repository.DoguConfigRepository{},
		}

		// when
		doguManager := NewDoguInstallManager(myClient, mgrSet, eventRecorder, configRepos)

		// then
		require.NotNil(t, doguManager)
	})
}

func Test_doguInstallManager_Install(t *testing.T) {
	t.Run("successfully install a dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
		assert.Empty(t, ldapCr.Status.InstalledVersion)

		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
		managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.ecosystemClient.EXPECT().Dogus(mock.Anything).Return(managerWithMocks.doguInterface)
		managerWithMocks.doguInterface.EXPECT().UpdateStatusWithRetry(testCtx, ldapCr, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, dogu *doguv2.Dogu, f func(doguv2.DoguStatus) doguv2.DoguStatus, options metav1.UpdateOptions) (*doguv2.Dogu, error) {
				dogu.Status = f(dogu.Status)
				return dogu, nil
			})

		yamlResult := map[string]string{"my-custom-resource.yml": "kind: Namespace"}
		managerWithMocks.fileExtractorMock.EXPECT().ExtractK8sResourcesFromContainer(mock.Anything, mock.Anything).Return(yamlResult, nil)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		managerWithMocks.applierMock.EXPECT().ApplyWithOwner(mock.Anything, "", ldapCr).Return(nil)
		upserterExpecter := managerWithMocks.resourceUpserter.EXPECT()
		upserterExpecter.UpsertDoguDeployment(testCtx, ldapCr, ldapDogu, mock.Anything).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguService(testCtx, ldapCr, ldapDogu, imageConfig).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguPVCs(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguNetworkPolicies(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)

		recorderExpecter := managerWithMocks.recorder.EXPECT()
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating custom dogu resources to the cluster: [%s]", "my-custom-resource.yml")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")
		execPod := newMockExecPod(t)
		execPod.EXPECT().Create(testCtx).Return(nil)
		execPod.EXPECT().Delete(testCtx).Return(nil)
		managerWithMocks.execPodFactory.EXPECT().NewExecPod(ldapCr, ldapDogu).Return(execPod, nil)

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, "2.4.48-4", ldapCr.Status.InstalledVersion)
	})

	t.Run("successfully install dogu with custom descriptor", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, ldapDevelopmentDoguMap, imageConfig := getDoguInstallManagerTestData(t)
		developmentDoguMap := doguv2.DevelopmentDoguMap(*ldapDevelopmentDoguMap)

		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, &developmentDoguMap, nil)
		managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
		managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.ecosystemClient.EXPECT().Dogus(mock.Anything).Return(managerWithMocks.doguInterface)
		managerWithMocks.doguInterface.EXPECT().UpdateStatusWithRetry(testCtx, ldapCr, mock.Anything, mock.Anything).Return(ldapCr, nil)

		yamlResult := make(map[string]string, 0)
		managerWithMocks.fileExtractorMock.EXPECT().ExtractK8sResourcesFromContainer(mock.Anything, mock.Anything).Return(yamlResult, nil)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapDevelopmentDoguMap)

		recorderExpect := managerWithMocks.recorder.EXPECT()
		recorderExpect.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		recorderExpect.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
		recorderExpect.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")
		recorderExpect.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		recorderExpect.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		recorderExpect.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
		recorderExpect.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
		recorderExpect.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
		recorderExpect.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")
		upserterExpect := managerWithMocks.resourceUpserter.EXPECT()
		upserterExpect.UpsertDoguDeployment(testCtx, ldapCr, ldapDogu, mock.Anything).Once().Return(nil, nil)
		upserterExpect.UpsertDoguService(testCtx, ldapCr, ldapDogu, imageConfig).Once().Return(nil, nil)
		upserterExpect.UpsertDoguPVCs(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)
		upserterExpect.UpsertDoguNetworkPolicies(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)

		execPod := newMockExecPod(t)
		execPod.EXPECT().Create(testCtx).Return(nil)
		execPod.EXPECT().Delete(testCtx).Return(nil)
		managerWithMocks.execPodFactory.EXPECT().NewExecPod(ldapCr, ldapDogu).Return(execPod, nil)

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.NoError(t, err)

		actualDevelopmentDoguMap := new(corev1.ConfigMap)
		err = managerWithMocks.installManager.client.Get(testCtx, ldapCr.GetDevelopmentDoguMapKey(), actualDevelopmentDoguMap)
		require.True(t, apierrors.IsNotFound(err))

	})

	t.Run("successfully install a dogu when configs already exists", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
		assert.Empty(t, ldapCr.Status.InstalledVersion)

		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, resErrors.NewAlreadyExistsError(assert.AnError))
		managerWithMocks.sensitiveDoguRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, resErrors.NewAlreadyExistsError(assert.AnError))
		managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
		managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.ecosystemClient.EXPECT().Dogus(mock.Anything).Return(managerWithMocks.doguInterface)
		managerWithMocks.doguInterface.EXPECT().UpdateStatusWithRetry(testCtx, ldapCr, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, dogu *doguv2.Dogu, f func(doguv2.DoguStatus) doguv2.DoguStatus, options metav1.UpdateOptions) (*doguv2.Dogu, error) {
				dogu.Status = f(dogu.Status)
				return dogu, nil
			})

		yamlResult := map[string]string{"my-custom-resource.yml": "kind: Namespace"}
		managerWithMocks.fileExtractorMock.EXPECT().ExtractK8sResourcesFromContainer(mock.Anything, mock.Anything).Return(yamlResult, nil)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		managerWithMocks.applierMock.EXPECT().ApplyWithOwner(mock.Anything, "", ldapCr).Return(nil)
		upserterExpecter := managerWithMocks.resourceUpserter.EXPECT()
		upserterExpecter.UpsertDoguDeployment(testCtx, ldapCr, ldapDogu, mock.Anything).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguService(testCtx, ldapCr, ldapDogu, imageConfig).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguPVCs(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguNetworkPolicies(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)

		recorderExpecter := managerWithMocks.recorder.EXPECT()
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating custom dogu resources to the cluster: [%s]", "my-custom-resource.yml")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")
		execPod := newMockExecPod(t)
		execPod.EXPECT().Create(testCtx).Return(nil)
		execPod.EXPECT().Delete(testCtx).Return(nil)
		managerWithMocks.execPodFactory.EXPECT().NewExecPod(ldapCr, ldapDogu).Return(execPod, nil)

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.NoError(t, err)
		assert.Equal(t, "2.4.48-4", ldapCr.Status.InstalledVersion)
	})

	t.Run("failed to validate dependencies", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.True(t, errors.Is(err, assert.AnError))
	})

	t.Run("failed to validate security", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(assert.AnError)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.True(t, errors.Is(err, assert.AnError))
	})

	t.Run("failed to validate dogu data additional mounts", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
		managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(assert.AnError)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.True(t, errors.Is(err, assert.AnError))
	})

	t.Run("failed to create dogu config", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		assert.Empty(t, ldapCr.Status.InstalledVersion)

		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
		managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
		managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, assert.AnError)
		managerWithMocks.doguConfigRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)

		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		recorderExpecter := managerWithMocks.recorder.EXPECT()
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("failed to create sensitive dogu config", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		assert.Empty(t, ldapCr.Status.InstalledVersion)

		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
		managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
		managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, assert.AnError)
		managerWithMocks.doguConfigRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)

		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		recorderExpecter := managerWithMocks.recorder.EXPECT()
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("dont delete configs when they have already existed", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, resErrors.NewAlreadyExistsError(assert.AnError))
		managerWithMocks.sensitiveDoguRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, resErrors.NewAlreadyExistsError(assert.AnError))
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
		managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")
		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("failed to register dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
		managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")
		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")

		managerWithMocks.doguConfigRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("failed to handle update installed version", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
		managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.ecosystemClient.EXPECT().Dogus(mock.Anything).Return(managerWithMocks.doguInterface)
		managerWithMocks.doguInterface.EXPECT().UpdateStatusWithRetry(testCtx, ldapCr, mock.Anything, mock.Anything).Return(nil, assert.AnError)

		yamlResult := make(map[string]string, 0)
		managerWithMocks.fileExtractorMock.EXPECT().ExtractK8sResourcesFromContainer(mock.Anything, mock.Anything).Return(yamlResult, nil)
		ldapCr.ResourceVersion = ""
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		upserterExpecter := managerWithMocks.resourceUpserter.EXPECT()
		upserterExpecter.UpsertDoguDeployment(testCtx, ldapCr, ldapDogu, mock.Anything).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguService(testCtx, ldapCr, ldapDogu, imageConfig).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguPVCs(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguNetworkPolicies(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)

		recorderExpecter := managerWithMocks.recorder.EXPECT()
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")
		execPod := newMockExecPod(t)
		execPod.EXPECT().Create(testCtx).Return(nil)
		execPod.EXPECT().Delete(testCtx).Return(nil)
		managerWithMocks.execPodFactory.EXPECT().NewExecPod(ldapCr, ldapDogu).Return(execPod, nil)

		managerWithMocks.doguConfigRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update dogu installed version")
	})

	t.Run("failed to create service accounts", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		recorderExpecter := managerWithMocks.recorder.EXPECT()
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")

		managerWithMocks.doguConfigRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)

		managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create service accounts")
	})

	t.Run("fail to create network policies", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
		assert.Empty(t, ldapCr.Status.InstalledVersion)

		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)

		yamlResult := map[string]string{"my-custom-resource.yml": "kind: Namespace"}
		managerWithMocks.fileExtractorMock.EXPECT().ExtractK8sResourcesFromContainer(mock.Anything, mock.Anything).Return(yamlResult, nil)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		managerWithMocks.applierMock.EXPECT().ApplyWithOwner(mock.Anything, "", ldapCr).Return(nil)
		upserterExpecter := managerWithMocks.resourceUpserter.EXPECT()
		upserterExpecter.UpsertDoguDeployment(testCtx, ldapCr, ldapDogu, mock.Anything).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguService(testCtx, ldapCr, ldapDogu, imageConfig).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguPVCs(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguNetworkPolicies(testCtx, ldapCr, ldapDogu).Once().Return(assert.AnError)

		recorderExpecter := managerWithMocks.recorder.EXPECT()
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating custom dogu resources to the cluster: [%s]", "my-custom-resource.yml")
		execPod := newMockExecPod(t)
		execPod.EXPECT().Create(testCtx).Return(nil)
		execPod.EXPECT().Delete(testCtx).Return(nil)
		managerWithMocks.execPodFactory.EXPECT().NewExecPod(ldapCr, ldapDogu).Return(execPod, nil)

		managerWithMocks.doguConfigRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)

		managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create dogu resources: assert.AnError general error for testing")
	})

	t.Run("dogu resource not found", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("error get dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(nil, nil, assert.AnError)

		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("error on pull image", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
		managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(nil, assert.AnError)
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		recorderExpecter := managerWithMocks.recorder.EXPECT()
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")

		managerWithMocks.doguConfigRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.sensitiveDoguRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)

		managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("error on upsert", func(t *testing.T) {
		t.Run("succeeds", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
			managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
			managerWithMocks.sensitiveDoguRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
			managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
			managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ecosystemClient.EXPECT().Dogus(mock.Anything).Return(managerWithMocks.doguInterface)
			managerWithMocks.doguInterface.EXPECT().UpdateStatusWithRetry(testCtx, ldapCr, mock.Anything, mock.Anything).Return(ldapCr, nil)

			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.EXPECT().ExtractK8sResourcesFromContainer(mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

			upserterExpecter := managerWithMocks.resourceUpserter.EXPECT()
			upserterExpecter.UpsertDoguDeployment(testCtx, ldapCr, ldapDogu, mock.Anything).Once().Return(nil, nil)
			upserterExpecter.UpsertDoguService(testCtx, ldapCr, ldapDogu, imageConfig).Once().Return(nil, nil)
			upserterExpecter.UpsertDoguPVCs(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)
			upserterExpecter.UpsertDoguNetworkPolicies(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)

			recorderExpecter := managerWithMocks.recorder.EXPECT()
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")
			execPod := newMockExecPod(t)
			execPod.EXPECT().Create(testCtx).Return(nil)
			execPod.EXPECT().Delete(testCtx).Return(nil)
			managerWithMocks.execPodFactory.EXPECT().NewExecPod(ldapCr, ldapDogu).Return(execPod, nil)

			managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")

			// when
			err := managerWithMocks.installManager.Install(testCtx, ldapCr)

			// then
			require.NoError(t, err)
		})
		t.Run("fails when upserting deployment", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
			managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
			managerWithMocks.sensitiveDoguRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
			managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
			managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.EXPECT().ExtractK8sResourcesFromContainer(mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

			recorderExpecter := managerWithMocks.recorder.EXPECT()
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")
			execPod := newMockExecPod(t)
			execPod.EXPECT().Create(testCtx).Return(nil)
			execPod.EXPECT().Delete(testCtx).Return(nil)
			managerWithMocks.execPodFactory.EXPECT().NewExecPod(ldapCr, ldapDogu).Return(execPod, nil)

			upserterExpecter := managerWithMocks.resourceUpserter.EXPECT()
			upserterExpecter.UpsertDoguService(testCtx, ldapCr, ldapDogu, imageConfig).Once().Return(nil, nil)
			upserterExpecter.UpsertDoguPVCs(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)
			upserterExpecter.UpsertDoguDeployment(testCtx, ldapCr, ldapDogu, mock.Anything).Once().Return(nil, assert.AnError)

			managerWithMocks.doguConfigRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.sensitiveDoguRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)

			managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")

			// when
			err := managerWithMocks.installManager.Install(testCtx, ldapCr)

			// then
			require.Error(t, err)
			assert.ErrorContains(t, err, "failed to create dogu resources")
			assert.ErrorIs(t, err, assert.AnError)
		})
		t.Run("fails when upserting service", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
			managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
			managerWithMocks.sensitiveDoguRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
			managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(mock.Anything, ldapDogu).Return(nil)
			managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

			recorderExpecter := managerWithMocks.recorder.EXPECT()
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")
			managerWithMocks.resourceUpserter.EXPECT().UpsertDoguService(testCtx, ldapCr, ldapDogu, imageConfig).Once().Return(nil, assert.AnError)

			managerWithMocks.doguConfigRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.sensitiveDoguRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)

			managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")

			// when
			err := managerWithMocks.installManager.Install(testCtx, ldapCr)

			// then
			require.Error(t, err)
			assert.ErrorContains(t, err, "failed to create dogu resources")
			assert.ErrorIs(t, err, assert.AnError)
		})
		t.Run("fails when upserting pvcs", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
			managerWithMocks.doguConfigRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
			managerWithMocks.sensitiveDoguRepository.EXPECT().Create(mock.Anything, mock.Anything).Return(resConfig.DoguConfig{}, nil)
			managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(mock.Anything, ldapDogu).Return(nil)
			managerWithMocks.securityValidator.EXPECT().ValidateSecurity(ldapDogu, ldapCr).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.EXPECT().ExtractK8sResourcesFromContainer(mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

			recorderExpecter := managerWithMocks.recorder.EXPECT()
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu security...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Create dogu and sensitive config...")
			execPod := newMockExecPod(t)
			execPod.EXPECT().Create(testCtx).Return(nil)
			execPod.EXPECT().Delete(testCtx).Return(nil)
			managerWithMocks.execPodFactory.EXPECT().NewExecPod(ldapCr, ldapDogu).Return(execPod, nil)

			upserterExpecter := managerWithMocks.resourceUpserter.EXPECT()
			upserterExpecter.UpsertDoguService(testCtx, ldapCr, ldapDogu, imageConfig).Once().Return(nil, nil)
			upserterExpecter.UpsertDoguPVCs(testCtx, ldapCr, ldapDogu).Once().Return(nil, assert.AnError)

			managerWithMocks.doguConfigRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.sensitiveDoguRepository.EXPECT().Delete(mock.Anything, mock.Anything).Return(nil)

			managerWithMocks.doguAdditionalMountsValidator.EXPECT().ValidateAdditionalMounts(testCtx, ldapDogu, ldapCr).Return(nil)
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Validating dogu additional mounts...")

			// when
			err := managerWithMocks.installManager.Install(testCtx, ldapCr)

			// then
			require.Error(t, err)
			assert.ErrorContains(t, err, "failed to create dogu resources")
			assert.ErrorIs(t, err, assert.AnError)
		})
	})
}
