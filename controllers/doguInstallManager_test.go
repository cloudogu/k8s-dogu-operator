package controllers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	cesremotemocks "github.com/cloudogu/cesapp-lib/remote/mocks"
	"github.com/cloudogu/k8s-apply-lib/apply"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	resourceMocks "github.com/cloudogu/k8s-dogu-operator/controllers/resource/mocks"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

type doguInstallManagerWithMocks struct {
	installManager            *doguInstallManager
	doguRemoteRegistryMock    *cesremotemocks.Registry
	doguLocalRegistryMock     *cesmocks.DoguRegistry
	imageRegistryMock         *mocks.ImageRegistry
	doguRegistratorMock       *mocks.DoguRegistrator
	dependencyValidatorMock   *mocks.DependencyValidator
	serviceAccountCreatorMock *mocks.ServiceAccountCreator
	doguSecretHandlerMock     *mocks.DoguSecretsHandler
	applierMock               *mocks.Applier
	fileExtractorMock         *mocks.FileExtractor
	client                    client.WithWatch
	recorder                  *mocks.EventRecorder
}

func (d *doguInstallManagerWithMocks) AssertMocks(t *testing.T) {
	t.Helper()
	mock.AssertExpectationsForObjects(t,
		d.doguRemoteRegistryMock,
		d.doguLocalRegistryMock,
		d.imageRegistryMock,
		d.doguRegistratorMock,
		d.dependencyValidatorMock,
		d.serviceAccountCreatorMock,
		d.doguSecretHandlerMock,
		d.applierMock,
		d.fileExtractorMock,
		d.recorder,
	)
}

func getDoguInstallManagerWithMocks(scheme *runtime.Scheme) doguInstallManagerWithMocks {
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	limitPatcher := &resourceMocks.LimitPatcher{}
	limitPatcher.On("RetrievePodLimits", mock.Anything).Return(limit.DoguLimits{}, nil)
	limitPatcher.On("PatchDeployment", mock.Anything, mock.Anything).Return(nil)
	resourceGenerator := resource.NewResourceGenerator(scheme, limitPatcher)
	doguRemoteRegistry := &cesremotemocks.Registry{}
	doguLocalRegistry := &cesmocks.DoguRegistry{}
	imageRegistry := &mocks.ImageRegistry{}
	doguRegistrator := &mocks.DoguRegistrator{}
	dependencyValidator := &mocks.DependencyValidator{}
	serviceAccountCreator := &mocks.ServiceAccountCreator{}
	doguSecretHandler := &mocks.DoguSecretsHandler{}
	mockedApplier := &mocks.Applier{}
	fileExtract := &mocks.FileExtractor{}
	eventRecorderMock := &mocks.EventRecorder{}

	doguInstallManager := &doguInstallManager{
		client:                k8sClient,
		scheme:                scheme,
		resourceGenerator:     resourceGenerator,
		doguRemoteRegistry:    doguRemoteRegistry,
		doguLocalRegistry:     doguLocalRegistry,
		imageRegistry:         imageRegistry,
		doguRegistrator:       doguRegistrator,
		dependencyValidator:   dependencyValidator,
		serviceAccountCreator: serviceAccountCreator,
		doguSecretHandler:     doguSecretHandler,
		fileExtractor:         fileExtract,
		applier:               mockedApplier,
		recorder:              eventRecorderMock,
	}

	return doguInstallManagerWithMocks{
		installManager:            doguInstallManager,
		doguRemoteRegistryMock:    doguRemoteRegistry,
		imageRegistryMock:         imageRegistry,
		doguLocalRegistryMock:     doguLocalRegistry,
		doguRegistratorMock:       doguRegistrator,
		dependencyValidatorMock:   dependencyValidator,
		serviceAccountCreatorMock: serviceAccountCreator,
		doguSecretHandlerMock:     doguSecretHandler,
		fileExtractorMock:         fileExtract,
		applierMock:               mockedApplier,
		client:                    k8sClient,
		recorder:                  eventRecorderMock,
	}
}

func getDoguInstallManagerTestData(t *testing.T) (*k8sv1.Dogu, *core.Dogu, *corev1.ConfigMap, *imagev1.ConfigFile) {
	ldapCr := readTestDataLdapCr(t)
	ldapDogu := readTestDataLdapDogu(t)
	ldapDoguDescriptor := readTestDataLdapDescriptor(t)
	imageConfig := readTestDataImageConfig(t)
	return ldapCr, ldapDogu, ldapDoguDescriptor, imageConfig
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
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)

		managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		yamlResult := make(map[string]string, 0)
		managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		managerWithMocks.recorder.On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")

		// when
		err := managerWithMocks.installManager.Install(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("successfully install a dogu with custom resources including service account and deployment", func(t *testing.T) {
		// given
		ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
		yamlResult := make(map[string]string, 2)

		testRole := &rbacv1.Role{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "Role",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "testRole",
			},
			Rules: []rbacv1.PolicyRule{},
		}
		testRoleBytes, err := yaml.Marshal(testRole)
		require.NoError(t, err)
		yamlResult["testRole.yaml"] = string(testRoleBytes)

		testServiceAccount := &corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testServiceAccount",
				Namespace: "{{ .Namespace }}",
			},
		}

		// set namespace only once to test for namespace templating without to influence other tests
		const testNamespace = "test"
		ldapCr.ObjectMeta.Namespace = testNamespace

		testServiceAccountBytes, err := yaml.Marshal(testServiceAccount)
		require.NoError(t, err)
		yamlResult["testServiceAccount.yaml"] = string(testServiceAccountBytes)

		testDeployment := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Deployment",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testDeployment",
				Namespace: "{{ .Namespace }}",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						ServiceAccountName: "testServiceAccount",
					},
				},
			},
		}

		testDeploymentBytes, err := yaml.Marshal(testDeployment)
		require.NoError(t, err)
		yamlResult["testDeployment.yaml"] = string(testDeploymentBytes)

		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
		managerWithMocks.applierMock.On("ApplyWithOwner", apply.YamlDocument(testRoleBytes), testNamespace, ldapCr).Return(nil)
		managerWithMocks.applierMock.On("ApplyWithOwner", mock.Anything, testNamespace, ldapCr).Return(nil)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		managerWithMocks.recorder.On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")

		// when
		err = managerWithMocks.installManager.Install(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)

		deployment := &appsv1.Deployment{}
		err = managerWithMocks.installManager.client.Get(ctx, types.NamespacedName{
			Namespace: testNamespace,
			Name:      "ldap",
		}, deployment)
		require.NoError(t, err)
		assert.Equal(t, "testServiceAccount", deployment.Spec.Template.Spec.ServiceAccountName)
	})

	t.Run("successfully install dogu with custom descriptor", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		ldapCr, _, _, imageConfig := getDoguInstallManagerTestData(t)
		ldapDescriptorCm := readTestDataLdapDescriptor(t)
		managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		yamlResult := make(map[string]string, 0)
		managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapDescriptorCm)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor using custom configmap...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
		managerWithMocks.recorder.On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-3")
		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")

		// when
		err := managerWithMocks.installManager.Install(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to install dogu with error query descriptor configmap", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		ldapCr, _, ldapDescriptor, _ := getDoguInstallManagerTestData(t)
		ldapDescriptor.Data["dogu.json"] = "invalid"
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapDescriptor)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor using custom configmap...")

		// when
		err := managerWithMocks.installManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal custom dogu descriptor")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to validate dependencies", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
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
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
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
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
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
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
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
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
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
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(nil, assert.AnError)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")

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
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(nil, assert.AnError)
		managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

		managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
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

	t.Run("error on createDoguResources", func(t *testing.T) {
		t.Run("volumes - fail on resource generation", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("CreateDoguPVC", mock.Anything).Once().Return(nil, assert.AnError)
			managerWithMocks.installManager.resourceGenerator = resourceGenerator

			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			managerWithMocks.recorder.On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")

			// when
			err := managerWithMocks.installManager.Install(ctx, ldapCr)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			managerWithMocks.AssertMocks(t)
		})

		t.Run("volumes - fail on kubernetes update", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("CreateDoguPVC", mock.Anything).Once().Return(&corev1.PersistentVolumeClaim{}, nil)
			managerWithMocks.installManager.resourceGenerator = resourceGenerator

			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			managerWithMocks.recorder.On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")

			// when
			err := managerWithMocks.installManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create dogu resources: failed to create volumes for dogu")
			managerWithMocks.AssertMocks(t)
		})

		t.Run("deployment - fail on resource generation", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("CreateDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("CreateDoguDeployment", mock.Anything, mock.Anything, mock.Anything).Once().Return(nil, assert.AnError)
			managerWithMocks.installManager.resourceGenerator = resourceGenerator

			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			managerWithMocks.recorder.On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")

			// when
			err := managerWithMocks.installManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			assert.Contains(t, err.Error(), "failed to generate dogu deployment")
			managerWithMocks.AssertMocks(t)
		})

		t.Run("deployment - fail on kubernetes update", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("CreateDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("CreateDoguDeployment", mock.Anything, mock.Anything, mock.Anything).Once().Return(&appsv1.Deployment{}, nil)
			managerWithMocks.installManager.resourceGenerator = resourceGenerator

			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			managerWithMocks.recorder.On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")

			// when
			err := managerWithMocks.installManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create dogu resources: failed to create deployment for dogu")
			managerWithMocks.AssertMocks(t)
		})

		t.Run("service - fail on resource generation", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("CreateDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("CreateDoguDeployment", mock.Anything, mock.Anything, mock.Anything).Return(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "mydeploy"}}, nil)
			resourceGenerator.On("CreateDoguService", mock.Anything, mock.Anything).Once().Return(nil, assert.AnError)
			managerWithMocks.installManager.resourceGenerator = resourceGenerator

			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			managerWithMocks.recorder.On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")

			// when
			err := managerWithMocks.installManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			assert.Contains(t, err.Error(), "failed to generate dogu service")
			managerWithMocks.AssertMocks(t)
		})

		t.Run("service - fail on kubernetes update", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("CreateDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("CreateDoguDeployment", mock.Anything, mock.Anything, mock.Anything).Return(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "mydeploy"}}, nil)
			resourceGenerator.On("CreateDoguService", mock.Anything, mock.Anything).Once().Return(&corev1.Service{}, nil)
			managerWithMocks.installManager.resourceGenerator = resourceGenerator

			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			managerWithMocks.recorder.On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")

			// when
			err := managerWithMocks.installManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create dogu resources: failed to create service for dogu")
			managerWithMocks.AssertMocks(t)
		})

		t.Run("exposed services - fail on resource generation", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("CreateDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("CreateDoguDeployment", mock.Anything, mock.Anything, mock.Anything).Return(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "mydeploy"}}, nil)
			resourceGenerator.On("CreateDoguService", mock.Anything, mock.Anything).Return(&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "myservice"}}, nil)
			resourceGenerator.On("CreateDoguExposedServices", mock.Anything, mock.Anything).Once().Return(nil, assert.AnError)
			managerWithMocks.installManager.resourceGenerator = resourceGenerator

			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			managerWithMocks.recorder.On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")

			// when
			err := managerWithMocks.installManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			assert.Contains(t, err.Error(), "failed to generate exposed services")
			managerWithMocks.AssertMocks(t)
		})

		t.Run("exposed services - fail on kubernetes update", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.doguRemoteRegistryMock.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.imageRegistryMock.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.doguRegistratorMock.On("RegisterNewDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.dependencyValidatorMock.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.doguSecretHandlerMock.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.serviceAccountCreatorMock.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.fileExtractorMock.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.installManager.client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("CreateDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("CreateDoguDeployment", mock.Anything, mock.Anything, mock.Anything).Return(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "mydeploy"}}, nil)
			resourceGenerator.On("CreateDoguService", mock.Anything, mock.Anything).Return(&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "myservice"}}, nil)
			resourceGenerator.On("CreateDoguExposedServices", mock.Anything, mock.Anything).Once().Return([]corev1.Service{{}, {}}, nil)
			managerWithMocks.installManager.resourceGenerator = resourceGenerator

			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
			managerWithMocks.recorder.On("Eventf", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", "registry.cloudogu.com/official/ldap:2.4.48-4")
			managerWithMocks.recorder.On("Event", mock.Anything, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")

			// when
			err := managerWithMocks.installManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create dogu resources: failed to create exposed services for dogu")
			managerWithMocks.AssertMocks(t)
		})
	})
}

func Test_doguInstallManager_createVolumes(t *testing.T) {
	t.Run("error getting pvc should return an error", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(runtime.NewScheme())
		ldapCr := readTestDataLdapCr(t)
		ldapDogu := readTestDataLdapDogu(t)

		// when
		err := managerWithMocks.installManager.createVolumes(context.TODO(), ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), fmt.Sprintf("failed to get prebuilt dogu pvc for dogu %s", ldapDogu.Name))
		managerWithMocks.AssertMocks(t)
	})

	invalidStorageClass := "test"
	validStorageClass := longhornStorageClassName

	t.Run("invalid beta volume driver on prebuilt pvc should return an error", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		prebuiltPvc := getPvc("ldap")
		err := managerWithMocks.client.Create(context.TODO(), prebuiltPvc)
		require.NoError(t, err)
		ldapCr := readTestDataLdapCr(t)
		ldapDogu := readTestDataLdapDogu(t)

		// when
		err = managerWithMocks.installManager.createVolumes(context.TODO(), ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), fmt.Sprintf("pvc for dogu [%s] is not valid as annotation [%s] does not exist or is not [%s]", ldapDogu.GetFullName(), annotationKubernetesBetaVolumeDriver, longhornDiverID))
		managerWithMocks.AssertMocks(t)
	})

	t.Run("invalid kubernetes volume driver on prebuilt pvc should return an error", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		prebuiltPvc := getPvc("ldap")
		prebuiltPvc.Annotations[annotationKubernetesBetaVolumeDriver] = longhornDiverID
		err := managerWithMocks.client.Create(context.TODO(), prebuiltPvc)
		require.NoError(t, err)
		ldapCr := readTestDataLdapCr(t)
		ldapDogu := readTestDataLdapDogu(t)

		// when
		err = managerWithMocks.installManager.createVolumes(context.TODO(), ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), fmt.Sprintf("pvc for dogu [%s] is not valid as annotation [%s] does not exist or is not [%s]", ldapDogu.GetFullName(), annotationKubernetesVolumeDriver, longhornDiverID))
		managerWithMocks.AssertMocks(t)
	})

	t.Run("invalid dogu label on prebuilt pvc should return an error", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		prebuiltPvc := getPvc("ldap")
		prebuiltPvc.Annotations[annotationKubernetesBetaVolumeDriver] = longhornDiverID
		prebuiltPvc.Annotations[annotationKubernetesVolumeDriver] = longhornDiverID
		err := managerWithMocks.client.Create(context.TODO(), prebuiltPvc)
		require.NoError(t, err)
		ldapCr := readTestDataLdapCr(t)
		ldapDogu := readTestDataLdapDogu(t)

		// when
		err = managerWithMocks.installManager.createVolumes(context.TODO(), ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), fmt.Sprintf("pvc for dogu [%s] is not valid as pvc does not contain label [dogu] with value [%s]", ldapDogu.GetFullName(), ldapCr.Name))
		managerWithMocks.AssertMocks(t)
	})

	t.Run("invalid storage class name on prebuilt pvc should return an error", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		prebuiltPvc := getPvc("ldap")
		prebuiltPvc.Annotations[annotationKubernetesBetaVolumeDriver] = longhornDiverID
		prebuiltPvc.Annotations[annotationKubernetesVolumeDriver] = longhornDiverID
		prebuiltPvc.Spec.StorageClassName = &invalidStorageClass
		prebuiltPvc.Labels["dogu"] = "ldap"
		err := managerWithMocks.client.Create(context.TODO(), prebuiltPvc)
		require.NoError(t, err)
		ldapCr := readTestDataLdapCr(t)
		ldapDogu := readTestDataLdapDogu(t)

		// when
		err = managerWithMocks.installManager.createVolumes(context.TODO(), ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), fmt.Sprintf("pvc for dogu [%s] is not valid as pvc has invalid storage class: the storage class must be [%s]", ldapDogu.GetFullName(), longhornStorageClassName))
		managerWithMocks.AssertMocks(t)
	})

	t.Run("error setting controller reference should return an error", func(t *testing.T) {
		// given
		oldMethod := ctrl.SetControllerReference
		ctrl.SetControllerReference = func(owner, controlled metav1.Object, scheme *runtime.Scheme) error {
			return assert.AnError
		}
		defer func() { ctrl.SetControllerReference = oldMethod }()
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		prebuiltPvc := getPvc("ldap")
		prebuiltPvc.Annotations[annotationKubernetesBetaVolumeDriver] = longhornDiverID
		prebuiltPvc.Annotations[annotationKubernetesVolumeDriver] = longhornDiverID
		prebuiltPvc.Spec.StorageClassName = &validStorageClass
		prebuiltPvc.Labels["dogu"] = "ldap"
		err := managerWithMocks.client.Create(context.TODO(), prebuiltPvc)
		require.NoError(t, err)
		ldapCr := readTestDataLdapCr(t)
		ldapDogu := readTestDataLdapDogu(t)

		// when
		err = managerWithMocks.installManager.createVolumes(context.TODO(), ldapCr, ldapDogu)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to set controller reference")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("success with prebuilt pvc", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks(getTestScheme())
		prebuiltPvc := getPvc("ldap")
		prebuiltPvc.Annotations[annotationKubernetesBetaVolumeDriver] = longhornDiverID
		prebuiltPvc.Annotations[annotationKubernetesVolumeDriver] = longhornDiverID
		prebuiltPvc.Labels["dogu"] = "ldap"
		prebuiltPvc.Spec.StorageClassName = &validStorageClass
		err := managerWithMocks.client.Create(context.TODO(), prebuiltPvc)
		require.NoError(t, err)
		ldapCr := readTestDataLdapCr(t)
		ldapDogu := readTestDataLdapDogu(t)

		// when
		err = managerWithMocks.installManager.createVolumes(context.TODO(), ldapCr, ldapDogu)

		// then
		require.NoError(t, err)
		doguPVCClaim := &corev1.PersistentVolumeClaim{}
		doguPVCKey := client.ObjectKey{
			Name: ldapCr.Name,
		}
		err = managerWithMocks.installManager.client.Get(context.TODO(), doguPVCKey, doguPVCClaim)
		require.NoError(t, err)
		assert.Equal(t, ldapCr.Name, doguPVCClaim.OwnerReferences[0].Name)
	})
}

func getPvc(name string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: name, Annotations: make(map[string]string), Labels: make(map[string]string)},
		Spec:       corev1.PersistentVolumeClaimSpec{},
	}
}
