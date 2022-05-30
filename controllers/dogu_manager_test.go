package controllers_test

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"github.com/cloudogu/cesapp-lib/core"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	sigsclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
	"testing"
)

type doguManagerWithMocks struct {
	DoguManager           controllers.DoguManager
	DoguRemoteRegistry    *mocks.DoguRegistry
	DoguLocalRegistry     *cesmocks.DoguRegistry
	ImageRegistry         *mocks.ImageRegistry
	DoguRegistrator       *mocks.DoguRegistrator
	DependencyValidator   *mocks.DependencyValidator
	ServiceAccountCreator *mocks.ServiceAccountCreator
	ServiceAccountRemover *mocks.ServiceAccountRemover
	DoguSecretHandler     *mocks.DoguSecretsHandler
	Applier               *mockK8sClient
	FileExtractor         *mockFileExtractor
}

func (d *doguManagerWithMocks) AssertMocks(t *testing.T) {
	t.Helper()
	mock.AssertExpectationsForObjects(t,
		d.DoguRemoteRegistry,
		d.DoguLocalRegistry,
		d.ImageRegistry,
		d.DoguRegistrator,
		d.DependencyValidator,
		d.ServiceAccountCreator,
		d.ServiceAccountRemover,
		d.DoguSecretHandler,
		d.Applier,
		d.FileExtractor,
	)
}

func getDoguManagerWithMocks() doguManagerWithMocks {
	// Reset resource version otherwise the resource can't be created
	ldapCr.ResourceVersion = ""

	scheme := getInstallScheme()
	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	resourceGenerator := resource.NewResourceGenerator(scheme)
	doguRemoteRegistry := &mocks.DoguRegistry{}
	doguLocalRegistry := &cesmocks.DoguRegistry{}
	imageRegistry := &mocks.ImageRegistry{}
	doguRegistrator := &mocks.DoguRegistrator{}
	dependencyValidator := &mocks.DependencyValidator{}
	serviceAccountCreator := &mocks.ServiceAccountCreator{}
	serviceAccountRemover := &mocks.ServiceAccountRemover{}
	doguSecretHandler := &mocks.DoguSecretsHandler{}
	mockedK8sClient := &mockK8sClient{}
	fileExtract := &mockFileExtractor{}

	doguManager := controllers.DoguManager{
		Client:                k8sClient,
		Scheme:                scheme,
		ResourceGenerator:     resourceGenerator,
		DoguRemoteRegistry:    doguRemoteRegistry,
		DoguLocalRegistry:     doguLocalRegistry,
		ImageRegistry:         imageRegistry,
		DoguRegistrator:       doguRegistrator,
		DependencyValidator:   dependencyValidator,
		ServiceAccountCreator: serviceAccountCreator,
		ServiceAccountRemover: serviceAccountRemover,
		DoguSecretHandler:     doguSecretHandler,
		FileExtractor:         fileExtract,
		Applier:               mockedK8sClient,
	}

	return doguManagerWithMocks{
		DoguManager:           doguManager,
		DoguRemoteRegistry:    doguRemoteRegistry,
		ImageRegistry:         imageRegistry,
		DoguLocalRegistry:     doguLocalRegistry,
		DoguRegistrator:       doguRegistrator,
		DependencyValidator:   dependencyValidator,
		ServiceAccountCreator: serviceAccountCreator,
		ServiceAccountRemover: serviceAccountRemover,
		DoguSecretHandler:     doguSecretHandler,
		FileExtractor:         fileExtract,
		Applier:               mockedK8sClient,
	}
}

//go:embed testdata/redmine-cr.yaml
var redmineCrBytes []byte
var redmineCr = &k8sv1.Dogu{}

//go:embed testdata/redmine-dogu.json
var redmineBytes []byte
var redmineDogu = &core.Dogu{}

//go:embed testdata/ldap-cr.yaml
var ldapCrBytes []byte
var ldapCr = &k8sv1.Dogu{}

//go:embed testdata/image-config.json
var imageConfigBytes []byte
var image = &mocks.Image{}
var imageConfig = &imagev1.ConfigFile{}

//go:embed testdata/ldap-descriptor-cm.yaml
var ldapDescriptorCmBytes []byte
var ldapDescriptorCm = &corev1.ConfigMap{}

//go:embed testdata/ldap-dogu.json
var ldapBytes []byte
var ldapDogu = &core.Dogu{}

func init() {
	err := json.Unmarshal(ldapBytes, ldapDogu)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(ldapCrBytes, ldapCr)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(imageConfigBytes, imageConfig)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(ldapDescriptorCmBytes, ldapDescriptorCm)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(redmineCrBytes, redmineCr)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(redmineBytes, redmineDogu)
	if err != nil {
		panic(err)
	}
}

func TestDoguManager_Install(t *testing.T) {
	ctx := context.TODO()

	t.Run("successfully install a dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguManagerWithMocks()
		image = &mocks.Image{}
		image.On("ConfigFile").Return(imageConfig, nil)
		managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		yamlResult := make(map[string]string, 0)
		managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
		_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.DoguManager.Install(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("successfully install dogu with custom descriptor", func(t *testing.T) {
		// given
		managerWithMocks := getDoguManagerWithMocks()
		image = &mocks.Image{}
		image.On("ConfigFile").Return(imageConfig, nil)
		managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
		managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		yamlResult := make(map[string]string, 0)
		managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
		_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)
		_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapDescriptorCm)

		// when
		err := managerWithMocks.DoguManager.Install(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to install dogu with invalid custom descriptor", func(t *testing.T) {
		// given
		managerWithMocks := getDoguManagerWithMocks()
		managerWithMocks.DoguManager.Client = fake.NewClientBuilder().WithScheme(getDoguOnlyScheme()).WithObjects(ldapCr).Build()

		// when
		err := managerWithMocks.DoguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get custom dogu descriptor")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to install dogu with error query descriptor configmap", func(t *testing.T) {
		// given
		managerWithMocks := getDoguManagerWithMocks()
		_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)
		_ = managerWithMocks.DoguManager.Client.Create(ctx, getCustomDoguDescriptorCm("invalid"))

		// when
		err := managerWithMocks.DoguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal custom dogu descriptor")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to validate dependencies", func(t *testing.T) {
		// given
		managerWithMocks := getDoguManagerWithMocks()
		managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.DoguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.True(t, errors.Is(err, assert.AnError))
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to register dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguManagerWithMocks()
		managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
		_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.DoguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to handle dogu secrets from setup", func(t *testing.T) {
		// given
		managerWithMocks := getDoguManagerWithMocks()
		managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.DoguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to write dogu secrets from setup")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to create service accounts", func(t *testing.T) {
		// given
		managerWithMocks := getDoguManagerWithMocks()
		managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
		_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.DoguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to create service accounts")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("dogu resource not found", func(t *testing.T) {
		// given
		managerWithMocks := getDoguManagerWithMocks()

		// when
		err := managerWithMocks.DoguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("error get dogu", func(t *testing.T) {
		// given
		managerWithMocks := getDoguManagerWithMocks()
		managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(nil, assert.AnError)

		_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.DoguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("error on pull image", func(t *testing.T) {
		// given
		managerWithMocks := getDoguManagerWithMocks()
		managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
		managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(nil, assert.AnError)
		managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
		managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.DoguManager.Install(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		managerWithMocks.AssertMocks(t)
	})

	t.Run("error on createDoguResources", func(t *testing.T) {
		t.Run("volumes - fail on resource generation", func(t *testing.T) {
			// given
			managerWithMocks := getDoguManagerWithMocks()
			image = &mocks.Image{}
			image.On("ConfigFile").Return(imageConfig, nil)
			managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Once().Return(nil, assert.AnError)
			managerWithMocks.DoguManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.DoguManager.Install(ctx, ldapCr)

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			managerWithMocks.AssertMocks(t)
		})

		t.Run("volumes - fail on kubernetes update", func(t *testing.T) {
			// given
			managerWithMocks := getDoguManagerWithMocks()
			image = &mocks.Image{}
			image.On("ConfigFile").Return(imageConfig, nil)
			managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Once().Return(&corev1.PersistentVolumeClaim{}, nil)
			managerWithMocks.DoguManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.DoguManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create dogu resources: failed to create volumes for dogu")
			managerWithMocks.AssertMocks(t)
		})

		t.Run("deployment - fail on resource generation", func(t *testing.T) {
			// given
			managerWithMocks := getDoguManagerWithMocks()
			image = &mocks.Image{}
			image.On("ConfigFile").Return(imageConfig, nil)
			managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("GetDoguDeployment", mock.Anything, mock.Anything).Once().Return(nil, assert.AnError)
			managerWithMocks.DoguManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.DoguManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			managerWithMocks.AssertMocks(t)
		})

		t.Run("deployment - fail on kubernetes update", func(t *testing.T) {
			// given
			managerWithMocks := getDoguManagerWithMocks()
			image = &mocks.Image{}
			image.On("ConfigFile").Return(imageConfig, nil)
			managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("GetDoguDeployment", mock.Anything, mock.Anything).Once().Return(&v1.Deployment{}, nil)
			managerWithMocks.DoguManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.DoguManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create dogu resources: failed to create deployment for dogu")
			managerWithMocks.AssertMocks(t)
		})

		t.Run("service - fail on resource generation", func(t *testing.T) {
			// given
			managerWithMocks := getDoguManagerWithMocks()
			image = &mocks.Image{}
			image.On("ConfigFile").Return(imageConfig, nil)
			managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("GetDoguDeployment", mock.Anything, mock.Anything).Return(&v1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "mydeploy"}}, nil)
			resourceGenerator.On("GetDoguService", mock.Anything, mock.Anything).Once().Return(nil, assert.AnError)
			managerWithMocks.DoguManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.DoguManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			managerWithMocks.AssertMocks(t)
		})

		t.Run("service - fail on kubernetes update", func(t *testing.T) {
			// given
			managerWithMocks := getDoguManagerWithMocks()
			image = &mocks.Image{}
			image.On("ConfigFile").Return(imageConfig, nil)
			managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("GetDoguDeployment", mock.Anything, mock.Anything).Return(&v1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "mydeploy"}}, nil)
			resourceGenerator.On("GetDoguService", mock.Anything, mock.Anything).Once().Return(&corev1.Service{}, nil)
			managerWithMocks.DoguManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.DoguManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create dogu resources: failed to create service for dogu")
			managerWithMocks.AssertMocks(t)
		})

		t.Run("exposed services - fail on resource generation", func(t *testing.T) {
			// given
			managerWithMocks := getDoguManagerWithMocks()
			image = &mocks.Image{}
			image.On("ConfigFile").Return(imageConfig, nil)
			managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("GetDoguDeployment", mock.Anything, mock.Anything).Return(&v1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "mydeploy"}}, nil)
			resourceGenerator.On("GetDoguService", mock.Anything, mock.Anything).Return(&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "myservice"}}, nil)
			resourceGenerator.On("GetDoguExposedServices", mock.Anything, mock.Anything).Once().Return(nil, assert.AnError)
			managerWithMocks.DoguManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.DoguManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
			managerWithMocks.AssertMocks(t)
		})

		t.Run("exposed services - fail on kubernetes update", func(t *testing.T) {
			// given
			managerWithMocks := getDoguManagerWithMocks()
			image = &mocks.Image{}
			image.On("ConfigFile").Return(imageConfig, nil)
			managerWithMocks.DoguRemoteRegistry.On("GetDogu", mock.Anything).Return(ldapDogu, nil)
			managerWithMocks.ImageRegistry.On("PullImageConfig", mock.Anything, mock.Anything).Return(imageConfig, nil)
			managerWithMocks.DoguRegistrator.On("RegisterDogu", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.DependencyValidator.On("ValidateDependencies", mock.Anything).Return(nil)
			managerWithMocks.DoguSecretHandler.On("WriteDoguSecretsToRegistry", mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			managerWithMocks.ServiceAccountCreator.On("CreateAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			yamlResult := make(map[string]string, 0)
			managerWithMocks.FileExtractor.On("ExtractK8sResourcesFromContainer", mock.Anything, mock.Anything, mock.Anything).Return(yamlResult, nil)
			ldapCr.ResourceVersion = ""
			_ = managerWithMocks.DoguManager.Client.Create(ctx, ldapCr)

			resourceGenerator := &mocks.DoguResourceGenerator{}
			resourceGenerator.On("GetDoguPVC", mock.Anything).Return(&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "myclaim"}}, nil)
			resourceGenerator.On("GetDoguDeployment", mock.Anything, mock.Anything).Return(&v1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "mydeploy"}}, nil)
			resourceGenerator.On("GetDoguService", mock.Anything, mock.Anything).Return(&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "myservice"}}, nil)
			resourceGenerator.On("GetDoguExposedServices", mock.Anything, mock.Anything).Once().Return([]corev1.Service{{}, {}}, nil)
			managerWithMocks.DoguManager.ResourceGenerator = resourceGenerator

			// when
			err := managerWithMocks.DoguManager.Install(ctx, ldapCr)
			ldapCr.ResourceVersion = ""

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to create dogu resources: failed to create exposed services for dogu")
			managerWithMocks.AssertMocks(t)
		})
	})
}

func TestDoguManager_Delete(t *testing.T) {
	scheme := getDoguOnlyScheme()
	ctx := context.TODO()
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	}, &corev1.ConfigMap{})

	t.Run("successfully delete a dogu", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		managerWithMocks := getDoguManagerWithMocks()
		managerWithMocks.DoguRegistrator.On("UnregisterDogu", mock.Anything).Return(nil)
		managerWithMocks.ServiceAccountRemover.On("RemoveAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DoguLocalRegistry.On("Get", "ldap").Return(ldapDogu, nil)
		managerWithMocks.DoguManager.Client = client
		_ = client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.DoguManager.Delete(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)
		deletedDogu := k8sv1.Dogu{}
		err = client.Get(ctx, sigsclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failed to update dogu status", func(t *testing.T) {
		// given
		managerWithMocks := getDoguManagerWithMocks()

		// when
		err := managerWithMocks.DoguManager.Delete(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update dogu status")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failed to get dogu descriptor", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		managerWithMocks := getDoguManagerWithMocks()
		managerWithMocks.DoguManager.Client = client
		_ = client.Create(ctx, ldapCr)
		managerWithMocks.DoguLocalRegistry.On("Get", "ldap").Return(nil, assert.AnError)

		// when
		err := managerWithMocks.DoguManager.Delete(ctx, ldapCr)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get dogu")
		managerWithMocks.AssertMocks(t)
	})

	t.Run("failure during service account removal should not interrupt the delete routine", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		managerWithMocks := getDoguManagerWithMocks()
		managerWithMocks.DoguLocalRegistry.On("Get", "ldap").Return(ldapDogu, nil)
		managerWithMocks.ServiceAccountRemover.On("RemoveAll", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)
		managerWithMocks.DoguRegistrator.On("UnregisterDogu", mock.Anything).Return(nil)
		managerWithMocks.DoguManager.Client = client
		_ = client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.DoguManager.Delete(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)
		deletedDogu := k8sv1.Dogu{}
		err = client.Get(ctx, sigsclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})

	t.Run("failure during unregister should not interrupt the delete routine", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(scheme).Build()
		managerWithMocks := getDoguManagerWithMocks()
		managerWithMocks.DoguLocalRegistry.On("Get", "ldap").Return(ldapDogu, nil)
		managerWithMocks.ServiceAccountRemover.On("RemoveAll", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		managerWithMocks.DoguRegistrator.On("UnregisterDogu", mock.Anything).Return(assert.AnError)
		managerWithMocks.DoguManager.Client = client
		_ = client.Create(ctx, ldapCr)

		// when
		err := managerWithMocks.DoguManager.Delete(ctx, ldapCr)

		// then
		require.NoError(t, err)
		managerWithMocks.AssertMocks(t)
		deletedDogu := k8sv1.Dogu{}
		err = client.Get(ctx, sigsclient.ObjectKey{Name: ldapCr.Name, Namespace: ldapCr.Namespace}, &deletedDogu)
		require.NoError(t, err)
		assert.Empty(t, deletedDogu.Finalizers)
	})
}

func getDoguOnlyScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "dogu.cloudogu.com",
		Version: "v1",
		Kind:    "dogu",
	}, ldapCr)

	return scheme
}

func getInstallScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "dogu.cloudogu.com",
		Version: "v1",
		Kind:    "dogu",
	}, ldapCr)
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}, &v1.Deployment{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}, &corev1.Secret{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Service",
	}, &corev1.Service{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "PersistentVolumeClaim",
	}, &corev1.PersistentVolumeClaim{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	}, &corev1.ConfigMap{})

	return scheme
}

func getCustomDoguDescriptorCm(value string) *corev1.ConfigMap {
	data := make(map[string]string)
	data["dogu.json"] = value
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: ldapDogu.GetSimpleName() + "-descriptor", Namespace: ldapCr.Namespace},
		Data:       data,
	}
}

func TestNewDoguManager(t *testing.T) {
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
		globalConfig := &cesmocks.ConfigurationContext{}
		doguRegistry := &cesmocks.DoguRegistry{}
		globalConfig.On("Exists", "key_provider").Return(true, nil)
		cesRegistry.On("GlobalConfig").Return(globalConfig)
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		// when
		doguManager, err := controllers.NewDoguManager(client, operatorConfig, cesRegistry)

		// then
		require.NoError(t, err)
		require.NotNil(t, doguManager)
		mock.AssertExpectationsForObjects(t, cesRegistry, globalConfig)
	})

	t.Run("failed to query existing key provider", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := &cesmocks.Registry{}
		globalConfig := &cesmocks.ConfigurationContext{}
		globalConfig.On("Exists", "key_provider").Return(true, assert.AnError)
		cesRegistry.On("GlobalConfig").Return(globalConfig)

		// when
		doguManager, err := controllers.NewDoguManager(client, operatorConfig, cesRegistry)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
		assert.ErrorIs(t, err, assert.AnError)
		mock.AssertExpectationsForObjects(t, cesRegistry, globalConfig)
	})

	t.Run("failed to query existing key provider", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := &cesmocks.Registry{}
		globalConfig := &cesmocks.ConfigurationContext{}
		globalConfig.On("Exists", "key_provider").Return(true, assert.AnError)
		cesRegistry.On("GlobalConfig").Return(globalConfig)

		// when
		doguManager, err := controllers.NewDoguManager(client, operatorConfig, cesRegistry)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to query key provider")
		mock.AssertExpectationsForObjects(t, cesRegistry, globalConfig)
	})

	t.Run("failed to set default key provider", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := &cesmocks.Registry{}
		globalConfig := &cesmocks.ConfigurationContext{}
		globalConfig.On("Exists", "key_provider").Return(false, nil)
		globalConfig.On("Set", "key_provider", "pkcs1v15").Return(assert.AnError)
		cesRegistry.On("GlobalConfig").Return(globalConfig)

		// when
		doguManager, err := controllers.NewDoguManager(client, operatorConfig, cesRegistry)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to set default key provider")
		mock.AssertExpectationsForObjects(t, cesRegistry, globalConfig)
	})
}

type mockK8sClient struct {
	mock.Mock
}

func (m *mockK8sClient) ApplyWithOwner(yamlResources []byte, namespace string, owner metav1.Object) error {
	args := m.Called(yamlResources, namespace, owner)
	return args.Error(0)
}

type mockFileExtractor struct {
	mock.Mock
}

func (m *mockFileExtractor) ExtractK8sResourcesFromContainer(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) (map[string]string, error) {
	args := m.Called(ctx, doguResource, dogu)
	return args.Get(0).(map[string]string), args.Error(1)
}
