package controllers

import (
	"context"
	"errors"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	resourceMocks "github.com/cloudogu/k8s-dogu-operator/controllers/resource/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"
	utilmocks "github.com/cloudogu/k8s-dogu-operator/controllers/util/mocks"
)

type doguInstallManagerWithMocks struct {
	installManager            *doguInstallManager
	localDoguFetcher          *mocks.LocalDoguFetcher
	resourceDoguFetcher       *mocks.ResourceDoguFetcher
	imageRegistryMock         *mocks.ImageRegistry
	doguRegistratorMock       *mocks.DoguRegistrator
	dependencyValidatorMock   *mocks.DependencyValidator
	serviceAccountCreatorMock *mocks.ServiceAccountCreator
	doguSecretHandlerMock     *mocks.DoguSecretsHandler
	applierMock               *mocks.Applier
	fileExtractorMock         *mocks.FileExtractor
	client                    client.WithWatch
	resourceUpserter          *mocks.ResourceUpserter
	recorder                  *mocks.EventRecorder
	execPodFactory            *mocks.ExecPodFactory
}

func (d *doguInstallManagerWithMocks) AssertMocks(t *testing.T) {
	t.Helper()
	mock.AssertExpectationsForObjects(t,
		d.imageRegistryMock,
		d.doguRegistratorMock,
		d.dependencyValidatorMock,
		d.serviceAccountCreatorMock,
		d.doguSecretHandlerMock,
		d.applierMock,
		d.fileExtractorMock,
		d.localDoguFetcher,
		d.resourceDoguFetcher,
		d.recorder,
		d.resourceUpserter,
		d.execPodFactory,
	)
}

func getDoguInstallManagerWithMocks(t *testing.T, scheme *runtime.Scheme) doguInstallManagerWithMocks {
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	limitPatcher := &resourceMocks.LimitPatcher{}
	limitPatcher.On("RetrievePodLimits", mock.Anything).Return(limit.DoguLimits{}, nil)
	limitPatcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(nil)
	upserter := &mocks.ResourceUpserter{}
	imageRegistry := &mocks.ImageRegistry{}
	doguRegistrator := &mocks.DoguRegistrator{}
	dependencyValidator := &mocks.DependencyValidator{}
	serviceAccountCreator := &mocks.ServiceAccountCreator{}
	doguSecretHandler := &mocks.DoguSecretsHandler{}
	mockedApplier := &mocks.Applier{}
	fileExtract := mocks.NewFileExtractor(t)
	eventRecorderMock := mocks.NewEventRecorder(t)
	localDoguFetcher := mocks.NewLocalDoguFetcher(t)
	resourceDoguFetcher := mocks.NewResourceDoguFetcher(t)
	collectApplier := resource.NewCollectApplier(mockedApplier)
	podFactory := mocks.NewExecPodFactory(t)

	doguInstallManager := &doguInstallManager{
		client:                k8sClient,
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
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = func() *rest.Config {
		return &rest.Config{}
	}

	t.Run("success", func(t *testing.T) {
		// given
		myClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := &cesmocks.Registry{}
		doguRegistry := &cesmocks.DoguRegistry{}
		eventRecorder := &mocks.EventRecorder{}
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		// when
		doguManager, err := NewDoguInstallManager(myClient, operatorConfig, cesRegistry, eventRecorder)

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

		myClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := &cesmocks.Registry{}
		eventRecorder := &mocks.EventRecorder{}

		// when
		doguManager, err := NewDoguInstallManager(myClient, operatorConfig, cesRegistry, eventRecorder)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
	})
}

func Test_doguInstallManager_Install(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully install a dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)

		managerWithMocks.resourceDoguFetcher.On("FetchWithResource", ctx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)

		yamlResult := map[string]string{"my-custom-resource.yml": "kind: Namespace"}
		managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

		managerWithMocks.applierMock.On("ApplyWithOwner", mock.Anything, "", ldapCr).Return(nil)
		managerWithMocks.resourceUpserter.On("ApplyDoguResource", ctx, ldapCr, ldapDogu, imageConfig, mock.Anything).Once().Return(nil)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...").
			On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...").
			On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...").
			On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4").
			On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...").
			On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating custom dogu resources to the cluster: [%s]", "my-custom-resource.yml").
			On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
		execPod := utilmocks.NewExecPod(t)
		execPod.On("Create", testCtx).Return(nil)
		execPod.On("Delete", testCtx).Return(nil)
		managerWithMocks.execPodFactory.On("NewExecPod", util.ExecPodVolumeModeInstall, ldapCr, ldapDogu, mock.Anything).Return(execPod, nil)

		// when
		err := managerWithMocks.installManager.Install(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("successfully install dogu with custom descriptor", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, ldapDevelopmentDoguMap, imageConfig := getDoguInstallManagerTestData(t)
		developmentDoguMap := k8sv1.DevelopmentDoguMap(*ldapDevelopmentDoguMap)

		managerWithMocks.resourceDoguFetcher.On("FetchWithResource", ctx, ldapCr).Return(ldapDogu, &developmentDoguMap, nil)
		managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		yamlResult := make(map[string]string, 0)
		managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapDevelopmentDoguMap)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...").
			On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...").
			On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...").
			On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4").
			On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...").
			On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
		managerWithMocks.resourceUpserter.On("ApplyDoguResource", ctx, ldapCr, ldapDogu, imageConfig, mock.Anything).Once().Return(nil)

		execPod := utilmocks.NewExecPod(t)
		execPod.On("Create", testCtx).Return(nil)
		execPod.On("Delete", testCtx).Return(nil)
		managerWithMocks.execPodFactory.On("NewExecPod", util.ExecPodVolumeModeInstall, ldapCr, ldapDogu, mock.Anything).Return(execPod, nil)

		// when
		err := managerWithMocks.installManager.Install(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)

		actualDevelopmentDoguMap := new(corev1.ConfigMap)
		err = managerWithMocks.installManager.client.Get(ctx, ldapCr.GetDevelopmentDoguMapKey(), actualDevelopmentDoguMap)
		require.True(t, apierrors.IsNotFound(err))

	})

	t.Run("failed to validate dependencies", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.On("FetchWithResource", ctx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")

		// when
		err := managerWithMocks.installManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.True(t, errors.Is(err, assert.AnError))
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to register dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.On("FetchWithResource", ctx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")

		// when
		err := managerWithMocks.installManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to handle dogu secrets from setup", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.On("FetchWithResource", ctx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")

		// when
		err := managerWithMocks.installManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to write dogu secrets from setup")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to create service accounts", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.On("FetchWithResource", ctx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")

		// when
		err := managerWithMocks.installManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to create service accounts")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("dogu resource not found", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)

		// when
		err := managerWithMocks.installManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("error get dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.On("FetchWithResource", ctx, ldapCr).Return(nil, nil, assert.AnError)

		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.installManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("error on pull image", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.resourceDoguFetcher.On("FetchWithResource", ctx, ldapCr).Return(ldapDogu, nil, nil)
		managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(nil, assert.AnError)
		managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		managerWithMocks.recorder.On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")

		// when
		err := managerWithMocks.installManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("error on upsert", func(t *testing.T) {
		t.Run("succeeds", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.resourceDoguFetcher.On("FetchWithResource", ctx, ldapCr).Return(ldapDogu, nil, nil)
			managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

			managerWithMocks.resourceUpserter.On("ApplyDoguResource", ctx, ldapCr, ldapDogu, imageConfig, mock.Anything).Once().Return(nil)

			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...").
				On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...").
				On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...").
				On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4").
				On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...").
				On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
			execPod := utilmocks.NewExecPod(t)
			execPod.On("Create", testCtx).Return(nil)
			execPod.On("Delete", testCtx).Return(nil)
			managerWithMocks.execPodFactory.On("NewExecPod", util.ExecPodVolumeModeInstall, ldapCr, ldapDogu, mock.Anything).Return(execPod, nil)

			// when
			err := managerWithMocks.installManager.Install(ctx, ldapCr)

			// then
			require.NoError(t, err)
			managerWithMocks.AssertMocks(t)
		})
		t.Run("fails", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(t, getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.resourceDoguFetcher.On("FetchWithResource", ctx, ldapCr).Return(ldapDogu, nil, nil)
			managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...").
				On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...").
				On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...").
				On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4").
				On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Starting execPod...").
				On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
			execPod := utilmocks.NewExecPod(t)
			execPod.On("Create", testCtx).Return(nil)
			execPod.On("Delete", testCtx).Return(nil)
			managerWithMocks.execPodFactory.On("NewExecPod", util.ExecPodVolumeModeInstall, ldapCr, ldapDogu, mock.Anything).Return(execPod, nil)

			managerWithMocks.resourceUpserter.On("ApplyDoguResource", ctx, ldapCr, ldapDogu, imageConfig, mock.Anything).Once().Return(assert.AnError) // boom

			// when
			err := managerWithMocks.installManager.Install(ctx, ldapCr)

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create dogu resources: failed to create resource(s) for dogu official/ldap")
			assert.ErrorIs(t, err, assert.AnError)
			managerWithMocks.AssertMocks(t)
		})
	})
}
