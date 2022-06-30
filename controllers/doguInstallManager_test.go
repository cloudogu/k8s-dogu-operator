package controllers

import (
	"context"
	"errors"
	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	cesremotemocks "github.com/cloudogu/cesapp-lib/remote/mocks"
	"github.com/cloudogu/k8s-apply-lib/apply"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
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
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

type doguInstallManagerWithMocks struct {
	InstallManager        *doguInstallManager
	DoguRemoteRegistry    *cesremotemocks.Registry
	DoguLocalRegistry     *cesmocks.DoguRegistry
	ImageRegistry         *mocks.ImageRegistry
	DoguRegistrator       *mocks.DoguRegistrator
	DependencyValidator   *mocks.DependencyValidator
	ServiceAccountCreator *mocks.ServiceAccountCreator
	DoguSecretHandler     *mocks.DoguSecretsHandler
	Applier               *mocks.Applier
	FileExtractor         *mocks.FileExtractor
}

func (d *doguInstallManagerWithMocks) AssertMocks(t *testing.T) {
	t.Helper()
	mock.AssertExpectationsForObjects(t,
		d.DoguRemoteRegistry,
		d.DoguLocalRegistry,
		d.ImageRegistry,
		d.DoguRegistrator,
		d.DependencyValidator,
		d.ServiceAccountCreator,
		d.DoguSecretHandler,
		d.Applier,
		d.FileExtractor,
	)
}

func getDoguInstallManagerWithMocks() doguInstallManagerWithMocks {
	scheme := getTestScheme()
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	resourceGenerator := resource.NewResourceGenerator(scheme)
	doguRemoteRegistry := &cesremotemocks.Registry{}
	doguLocalRegistry := &cesmocks.DoguRegistry{}
	imageRegistry := &mocks.ImageRegistry{}
	doguRegistrator := &mocks.DoguRegistrator{}
	dependencyValidator := &mocks.DependencyValidator{}
	serviceAccountCreator := &mocks.ServiceAccountCreator{}
	doguSecretHandler := &mocks.DoguSecretsHandler{}
	mockedApplier := &mocks.Applier{}
	fileExtract := &mocks.FileExtractor{}

	doguInstallManager := &doguInstallManager{
		Client:                k8sClient,
		Scheme:                scheme,
		ResourceGenerator:     resourceGenerator,
		DoguRemoteRegistry:    doguRemoteRegistry,
		DoguLocalRegistry:     doguLocalRegistry,
		ImageRegistry:         imageRegistry,
		DoguRegistrator:       doguRegistrator,
		DependencyValidator:   dependencyValidator,
		ServiceAccountCreator: serviceAccountCreator,
		DoguSecretHandler:     doguSecretHandler,
		FileExtractor:         fileExtract,
		Applier:               mockedApplier,
	}

	return doguInstallManagerWithMocks{
		InstallManager:        doguInstallManager,
		DoguRemoteRegistry:    doguRemoteRegistry,
		ImageRegistry:         imageRegistry,
		DoguLocalRegistry:     doguLocalRegistry,
		DoguRegistrator:       doguRegistrator,
		DependencyValidator:   dependencyValidator,
		ServiceAccountCreator: serviceAccountCreator,
		DoguSecretHandler:     doguSecretHandler,
		FileExtractor:         fileExtract,
		Applier:               mockedApplier,
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
		client := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := &cesmocks.Registry{}
		doguRegistry := &cesmocks.DoguRegistry{}
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		// when
		doguManager, err := NewDoguInstallManager(client, operatorConfig, cesRegistry)

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
		doguManager, err := NewDoguInstallManager(client, operatorConfig, cesRegistry)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
	})
}

func Test_doguInstallManager_Install(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully install a dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks()
		ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)

		managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		yamlResult := make(map[string]string, 0)
		managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
		_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.InstallManager.Install(ctx, ldapCr)

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

		managerWithMocks := getDoguInstallManagerWithMocks()
		managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
		managerWithMocks.Applier.On("ApplyWithOwner", apply.YamlDocument(testRoleBytes), testNamespace, ldapCr).Return(nil)
		managerWithMocks.Applier.On("ApplyWithOwner", mock.Anything, testNamespace, ldapCr).Return(nil)
		_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

		// when
		err = managerWithMocks.InstallManager.Install(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)

		deployment := &appsv1.Deployment{}
		err = managerWithMocks.InstallManager.Client.Get(ctx, types.NamespacedName{
			Namespace: testNamespace,
			Name:      "ldap",
		}, deployment)
		require.NoError(t, err)
		assert.Equal(t, "testServiceAccount", deployment.Spec.Template.Spec.ServiceAccountName)
	})

	t.Run("successfully install dogu with custom descriptor", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks()
		ldapCr, _, _, imageConfig := getDoguInstallManagerTestData(t)
		ldapDescriptorCm := readTestDataLdapDescriptor(t)
		managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		yamlResult := make(map[string]string, 0)
		managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
		_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)
		_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapDescriptorCm)

		// when
		err := managerWithMocks.InstallManager.Install(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to install dogu with error query descriptor configmap", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks()
		ldapCr, _, ldapDescriptor, _ := getDoguInstallManagerTestData(t)
		ldapDescriptor.Data["dogu.json"] = "invalid"
		_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)
		_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapDescriptor)

		// when
		err := managerWithMocks.InstallManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal custom dogu descriptor")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to validate dependencies", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks()
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.InstallManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.True(t, errors.Is(err, assert.AnError))
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to register dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks()
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
		_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.InstallManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to handle dogu secrets from setup", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks()
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.InstallManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to write dogu secrets from setup")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to create service accounts", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks()
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.InstallManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to create service accounts")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("dogu resource not found", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks()
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)

		// when
		err := managerWithMocks.InstallManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("error get dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks()
		ldapCr, _, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(nil, assert.AnError)

		_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.InstallManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("error on pull image", func(t *testing.T) {
		// given
		managerWithMocks := getDoguInstallManagerWithMocks()
		ldapCr, ldapDogu, _, _ := getDoguInstallManagerTestData(t)
		managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(nil, assert.AnError)
		managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.InstallManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("error on createDoguResources", func(t *testing.T) {
		t.Run("volumes - fail on resource generation", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks()
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Once().Return(nil, assert.AnError)
			managerWithMocks.InstallManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.InstallManager.Install(ctx, ldapCr)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			managerWithMocks.AssertMocks(t)
		})

		t.Run("volumes - fail on kubernetes update", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks()
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Once().Return(&corev1.PersistentVolumeClaim{}, nil)
			managerWithMocks.InstallManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.InstallManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create dogu resources: failed to create volumes for dogu")
			managerWithMocks.AssertMocks(t)
		})

		t.Run("deployment - fail on resource generation", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks()
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("GetDoguDeployment", mock.Anything, mock.Anything, mock.Anything).Once().Return(nil, assert.AnError)
			managerWithMocks.InstallManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.InstallManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			managerWithMocks.AssertMocks(t)
		})

		t.Run("deployment - fail on kubernetes update", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks()
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("GetDoguDeployment", mock.Anything, mock.Anything, mock.Anything).Once().Return(&appsv1.Deployment{}, nil)
			managerWithMocks.InstallManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.InstallManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create dogu resources: failed to create deployment for dogu")
			managerWithMocks.AssertMocks(t)
		})

		t.Run("service - fail on resource generation", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks()
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("GetDoguDeployment", mock.Anything, mock.Anything, mock.Anything).Return(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "mydeploy"}}, nil)
			resourceGenerator.On("GetDoguService", mock.Anything, mock.Anything).Once().Return(nil, assert.AnError)
			managerWithMocks.InstallManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.InstallManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			managerWithMocks.AssertMocks(t)
		})

		t.Run("service - fail on kubernetes update", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks()
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("GetDoguDeployment", mock.Anything, mock.Anything, mock.Anything).Return(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "mydeploy"}}, nil)
			resourceGenerator.On("GetDoguService", mock.Anything, mock.Anything).Once().Return(&corev1.Service{}, nil)
			managerWithMocks.InstallManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.InstallManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create dogu resources: failed to create service for dogu")
			managerWithMocks.AssertMocks(t)
		})

		t.Run("exposed services - fail on resource generation", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks()
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("GetDoguDeployment", mock.Anything, mock.Anything, mock.Anything).Return(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "mydeploy"}}, nil)
			resourceGenerator.On("GetDoguService", mock.Anything, mock.Anything).Return(&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "myservice"}}, nil)
			resourceGenerator.On("GetDoguExposedServices", mock.Anything, mock.Anything).Once().Return(nil, assert.AnError)
			managerWithMocks.InstallManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.InstallManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			managerWithMocks.AssertMocks(t)
		})

		t.Run("exposed services - fail on kubernetes update", func(t *testing.T) {
			// given
			managerWithMocks := getDoguInstallManagerWithMocks()
			ldapCr, ldapDogu, _, imageConfig := getDoguInstallManagerTestData(t)
			managerWithMocks.DoguRemoteRegistry.On("Get", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.InstallManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("GetDoguDeployment", mock.Anything, mock.Anything, mock.Anything).Return(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "mydeploy"}}, nil)
			resourceGenerator.On("GetDoguService", mock.Anything, mock.Anything).Return(&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "myservice"}}, nil)
			resourceGenerator.On("GetDoguExposedServices", mock.Anything, mock.Anything).Once().Return([]corev1.Service{{}, {}}, nil)
			managerWithMocks.InstallManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.InstallManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create dogu resources: failed to create exposed services for dogu")
			managerWithMocks.AssertMocks(t)
		})
	})
}
