package controllers

import (
	"context"
	"errors"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"
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

	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
)

type doguInstallManagerWithMocks struct {
	installManager            *doguInstallManager
	localDoguFetcher          *mocks.LocalDoguFetcher
	resourceDoguFetcher       *mocks.ResourceDoguFetcher
	imageRegistryMock         *mocks.ImageRegistry
	doguRegistratorMock       *mocks.DoguRegistrator
	dependencyValidatorMock   *mocks.DependencyValidator
	serviceAccountCreatorMock *mocks.ServiceAccountCreator
	doguSecretHandlerMock     *mocks.DoguSecretHandler
	applierMock               *mocks.Applier
	fileExtractorMock         *mocks.FileExtractor
	client                    client.WithWatch
	resourceUpserter          *mocks.ResourceUpserter
	recorder                  *extMocks.EventRecorder
	execPodFactory            *mocks.ExecPodFactory
	ecosystemClient           *mocks.EcosystemInterface
	doguInterface             *mocks.DoguInterface
}

func getDoguInstallManagerWithMocks(t *testing.T, scheme *runtime.Scheme) doguInstallManagerWithMocks {
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&k8sv1.Dogu{}).Build()
	ecosystemClientMock := mocks.NewEcosystemInterface(t)
	doguIntrefaceMock := mocks.NewDoguInterface(t)
	upserter := mocks.NewResourceUpserter(t)
	imageRegistry := mocks.NewImageRegistry(t)
	doguRegistrator := mocks.NewDoguRegistrator(t)
	dependencyValidator := mocks.NewDependencyValidator(t)
	serviceAccountCreator := mocks.NewServiceAccountCreator(t)
	doguSecretHandler := mocks.NewDoguSecretHandler(t)
	mockedApplier := mocks.NewApplier(t)
	fileExtract := mocks.NewFileExtractor(t)
	eventRecorderMock := extMocks.NewEventRecorder(t)
	localDoguFetcher := mocks.NewLocalDoguFetcher(t)
	resourceDoguFetcher := mocks.NewResourceDoguFetcher(t)
	collectApplier := resource.NewCollectApplier(mockedApplier)
	podFactory := mocks.NewExecPodFactory(t)

	doguInstallManager := &doguInstallManager{
		client:                k8sClient,
		ecosystemClient:       ecosystemClientMock,
		recorder:              eventRecorderMock,
		imageRegistry:         imageRegistry,
		doguRegistrator:       doguRegistrator,
		localDoguFetcher:      localDoguFetcher,
		resourceDoguFetcher:   resourceDoguFetcher,
		dependencyValidator:   dependencyValidator,
		serviceAccountCreator: serviceAccountCreator,
		doguSecretHandler:     doguSecretHandler,
		fileExtractor:         fileExtract,
		collectApplier:        collectApplier,
		resourceUpserter:      upserter,
		execPodFactory:        podFactory,
	}

	return doguInstallManagerWithMocks{
		installManager:            doguInstallManager,
		client:                    k8sClient,
		recorder:                  eventRecorderMock,
		localDoguFetcher:          localDoguFetcher,
		resourceDoguFetcher:       resourceDoguFetcher,
		imageRegistryMock:         imageRegistry,
		doguRegistratorMock:       doguRegistrator,
		dependencyValidatorMock:   dependencyValidator,
		serviceAccountCreatorMock: serviceAccountCreator,
		doguSecretHandlerMock:     doguSecretHandler,
		fileExtractorMock:         fileExtract,
		applierMock:               mockedApplier,
		resourceUpserter:          upserter,
		execPodFactory:            podFactory,
		ecosystemClient:           ecosystemClientMock,
		doguInterface:             doguIntrefaceMock,
	}
}

func getDoguInstallManagerTestData(t *testing.T) (*k8sv1.Dogu, *core.Dogu, *corev1.ConfigMap, *imagev1.ConfigFile) {
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
		cesRegistry := cesmocks.NewRegistry(t)
		mgrSet := &util.ManagerSet{}
		eventRecorder := extMocks.NewEventRecorder(t)

		// when
		doguManager := NewDoguInstallManager(myClient, cesRegistry, mgrSet, eventRecorder)

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
		managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.EXPECT().WriteDoguSecretsToRegistry(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.ecosystemClient.EXPECT().Dogus(mock.Anything).Return(managerWithMocks.doguInterface)
		managerWithMocks.doguInterface.EXPECT().UpdateStatusWithRetry(testCtx, ldapCr, mock.Anything, mock.Anything).
			RunAndReturn(func(ctx context.Context, dogu *k8sv1.Dogu, f func(k8sv1.DoguStatus) k8sv1.DoguStatus, options metav1.UpdateOptions) (*k8sv1.Dogu, error) {
				dogu.Status = f(dogu.Status)
				return dogu, nil
			})

		yamlResult := map[string]string{"my-custom-resource.yml": "kind: Namespace"}
		managerWithMocks.fileExtractorMock.EXPECT().ExtractK8sResourcesFromContainer(mock.Anything, mock.Anything).Return(yamlResult, nil)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		managerWithMocks.applierMock.EXPECT().ApplyWithOwner(mock.Anything, "", ldapCr).Return(nil)
		upserterExpecter := managerWithMocks.resourceUpserter.EXPECT()
		upserterExpecter.UpsertDoguDeployment(testCtx, ldapCr, ldapDogu, mock.Anything).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguService(testCtx, ldapCr, imageConfig).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguExposedService(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguPVCs(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)

		recorderExpecter := managerWithMocks.recorder.EXPECT()
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating custom dogu resources to the cluster: [%s]", "my-custom-resource.yml")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
		execPod := mocks.NewExecPod(t)
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
		developmentDoguMap := k8sv1.DevelopmentDoguMap(*ldapDevelopmentDoguMap)

		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, &developmentDoguMap, nil)
		managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.EXPECT().WriteDoguSecretsToRegistry(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.ecosystemClient.EXPECT().Dogus(mock.Anything).Return(managerWithMocks.doguInterface)
		managerWithMocks.doguInterface.EXPECT().UpdateStatusWithRetry(testCtx, ldapCr, mock.Anything, mock.Anything).Return(ldapCr, nil)

		yamlResult := make(map[string]string, 0)
		managerWithMocks.fileExtractorMock.EXPECT().ExtractK8sResourcesFromContainer(mock.Anything, mock.Anything).Return(yamlResult, nil)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapDevelopmentDoguMap)

		recorderExpect := managerWithMocks.recorder.EXPECT()
		recorderExpect.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		recorderExpect.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		recorderExpect.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		recorderExpect.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
		recorderExpect.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
		recorderExpect.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
		upserterExpect := managerWithMocks.resourceUpserter.EXPECT()
		upserterExpect.UpsertDoguDeployment(testCtx, ldapCr, ldapDogu, mock.Anything).Once().Return(nil, nil)
		upserterExpect.UpsertDoguService(testCtx, ldapCr, imageConfig).Once().Return(nil, nil)
		upserterExpect.UpsertDoguExposedService(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)
		upserterExpect.UpsertDoguPVCs(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)

		execPod := mocks.NewExecPod(t)
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

	t.Run("failed to register dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("failed to handle dogu secrets from setup", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.EXPECT().WriteDoguSecretsToRegistry(mock.Anything, mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		managerWithMocks.recorder.EXPECT().Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to write dogu secrets from setup")
	})

	t.Run("failed to handle update installed version", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.EXPECT().WriteDoguSecretsToRegistry(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.ecosystemClient.EXPECT().Dogus(mock.Anything).Return(managerWithMocks.doguInterface)
		managerWithMocks.doguInterface.EXPECT().UpdateStatusWithRetry(testCtx, ldapCr, mock.Anything, mock.Anything).Return(nil, assert.AnError)

		yamlResult := make(map[string]string, 0)
		managerWithMocks.fileExtractorMock.EXPECT().ExtractK8sResourcesFromContainer(mock.Anything, mock.Anything).Return(yamlResult, nil)
		ldapCr.ResourceVersion = ""
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		upserterExpecter := managerWithMocks.resourceUpserter.EXPECT()
		upserterExpecter.UpsertDoguDeployment(testCtx, ldapCr, ldapDogu, mock.Anything).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguService(testCtx, ldapCr, imageConfig).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguExposedService(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)
		upserterExpecter.UpsertDoguPVCs(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)

		recorderExpecter := managerWithMocks.recorder.EXPECT()
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
		execPod := mocks.NewExecPod(t)
		execPod.EXPECT().Create(testCtx).Return(nil)
		execPod.EXPECT().Delete(testCtx).Return(nil)
		managerWithMocks.execPodFactory.EXPECT().NewExecPod(ldapCr, ldapDogu).Return(execPod, nil)

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
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.EXPECT().WriteDoguSecretsToRegistry(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		recorderExpecter := managerWithMocks.recorder.EXPECT()
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")

		// when
		err := managerWithMocks.installManager.Install(testCtx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to create service accounts")
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
		managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(nil, assert.AnError)
		managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.EXPECT().WriteDoguSecretsToRegistry(mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
		_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

		recorderExpecter := managerWithMocks.recorder.EXPECT()
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")

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
			managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
			managerWithMocks.doguSecretHandlerMock.EXPECT().WriteDoguSecretsToRegistry(mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ecosystemClient.EXPECT().Dogus(mock.Anything).Return(managerWithMocks.doguInterface)
			managerWithMocks.doguInterface.EXPECT().UpdateStatusWithRetry(testCtx, ldapCr, mock.Anything, mock.Anything).Return(ldapCr, nil)

			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.EXPECT().ExtractK8sResourcesFromContainer(mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

			upserterExpecter := managerWithMocks.resourceUpserter.EXPECT()
			upserterExpecter.UpsertDoguDeployment(testCtx, ldapCr, ldapDogu, mock.Anything).Once().Return(nil, nil)
			upserterExpecter.UpsertDoguService(testCtx, ldapCr, imageConfig).Once().Return(nil, nil)
			upserterExpecter.UpsertDoguExposedService(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)
			upserterExpecter.UpsertDoguPVCs(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)

			recorderExpecter := managerWithMocks.recorder.EXPECT()
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
			execPod := mocks.NewExecPod(t)
			execPod.EXPECT().Create(testCtx).Return(nil)
			execPod.EXPECT().Delete(testCtx).Return(nil)
			managerWithMocks.execPodFactory.EXPECT().NewExecPod(ldapCr, ldapDogu).Return(execPod, nil)

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
			managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(testCtx, mock.Anything).Return(nil)
			managerWithMocks.doguSecretHandlerMock.EXPECT().WriteDoguSecretsToRegistry(mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.EXPECT().ExtractK8sResourcesFromContainer(mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

			recorderExpecter := managerWithMocks.recorder.EXPECT()
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
			execPod := mocks.NewExecPod(t)
			execPod.EXPECT().Create(testCtx).Return(nil)
			execPod.EXPECT().Delete(testCtx).Return(nil)
			managerWithMocks.execPodFactory.EXPECT().NewExecPod(ldapCr, ldapDogu).Return(execPod, nil)

			upserterExpecter := managerWithMocks.resourceUpserter.EXPECT()
			upserterExpecter.UpsertDoguService(testCtx, ldapCr, imageConfig).Once().Return(nil, nil)
			upserterExpecter.UpsertDoguExposedService(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)
			upserterExpecter.UpsertDoguPVCs(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)
			upserterExpecter.UpsertDoguDeployment(testCtx, ldapCr, ldapDogu, mock.Anything).Once().Return(nil, assert.AnError)

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
			managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(mock.Anything, ldapDogu).Return(nil)
			managerWithMocks.doguSecretHandlerMock.EXPECT().WriteDoguSecretsToRegistry(mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

			recorderExpecter := managerWithMocks.recorder.EXPECT()
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
			managerWithMocks.resourceUpserter.EXPECT().UpsertDoguService(testCtx, ldapCr, imageConfig).Once().Return(nil, assert.AnError)

			// when
			err := managerWithMocks.installManager.Install(testCtx, ldapCr)

			// then
			require.Error(t, err)
			assert.ErrorContains(t, err, "failed to create dogu resources")
			assert.ErrorIs(t, err, assert.AnError)
		})
		t.Run("fails when upserting exposed services", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.resourceDoguFetcher.EXPECT().FetchWithResource(testCtx, ldapCr).Return(ldapDogu, nil, nil)
			managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(mock.Anything, ldapDogu).Return(nil)
			managerWithMocks.doguSecretHandlerMock.EXPECT().WriteDoguSecretsToRegistry(mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

			recorderExpecter := managerWithMocks.recorder.EXPECT()
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
			managerWithMocks.resourceUpserter.EXPECT().UpsertDoguService(testCtx, ldapCr, imageConfig).Once().Return(nil, nil)
			managerWithMocks.resourceUpserter.EXPECT().UpsertDoguExposedService(testCtx, ldapCr, ldapDogu).Once().Return(nil, assert.AnError)

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
			managerWithMocks.imageRegistryMock.EXPECT().PullImageConfig(mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.EXPECT().RegisterNewDogu(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.EXPECT().ValidateDependencies(mock.Anything, ldapDogu).Return(nil)
			managerWithMocks.doguSecretHandlerMock.EXPECT().WriteDoguSecretsToRegistry(mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.EXPECT().CreateAll(mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.EXPECT().ExtractK8sResourcesFromContainer(mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(testCtx, ldapCr)

			recorderExpecter := managerWithMocks.recorder.EXPECT()
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			recorderExpecter.Eventf(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...")
			recorderExpecter.Event(mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
			execPod := mocks.NewExecPod(t)
			execPod.EXPECT().Create(testCtx).Return(nil)
			execPod.EXPECT().Delete(testCtx).Return(nil)
			managerWithMocks.execPodFactory.EXPECT().NewExecPod(ldapCr, ldapDogu).Return(execPod, nil)

			upserterExpecter := managerWithMocks.resourceUpserter.EXPECT()
			upserterExpecter.UpsertDoguService(testCtx, ldapCr, imageConfig).Once().Return(nil, nil)
			upserterExpecter.UpsertDoguExposedService(testCtx, ldapCr, ldapDogu).Once().Return(nil, nil)
			upserterExpecter.UpsertDoguPVCs(testCtx, ldapCr, ldapDogu).Once().Return(nil, assert.AnError)

			// when
			err := managerWithMocks.installManager.Install(testCtx, ldapCr)

			// then
			require.Error(t, err)
			assert.ErrorContains(t, err, "failed to create dogu resources")
			assert.ErrorIs(t, err, assert.AnError)
		})
	})
}
