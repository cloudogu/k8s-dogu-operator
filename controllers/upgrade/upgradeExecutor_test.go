package upgrade

import (
	"context"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	regmock "github.com/cloudogu/cesapp-lib/registry/mocks"
	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/upgrade/mocks"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const redmineUpgradeVersion = "4.2.3-11"

var testCtx = context.TODO()

func TestNewUpgradeExecutor(t *testing.T) {
	t.Run("should return a valid object", func(t *testing.T) {
		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			Build()
		imageRegMock := mocks.NewImageRegistry(t)
		saCreator := mocks.NewServiceAccountCreator(t)
		fileEx := mocks.NewFileExtractor(t)
		applier := mocks.NewCollectApplier(t)
		doguRegistry := new(regmock.DoguRegistry)
		mockRegistry := new(regmock.Registry)
		mockRegistry.On("DoguRegistry").Return(doguRegistry, nil)

		// when
		actual := NewUpgradeExecutor(myClient, imageRegMock, applier, fileEx, saCreator, mockRegistry)

		// then
		require.NotNil(t, actual)
		assert.IsType(t, &upgradeExecutor{}, actual)
	})
}

func Test_upgradeExecutor_Upgrade(t *testing.T) {

	t.Run("should succeed", func(t *testing.T) {
		// given
		// fromDogu := readTestDataDogu(t, redmineBytes) // v4.2.3-10
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion

		dependentDeployment := createTestDeployment("redmine")
		dependencyDeployment := createTestDeployment("dependency-dogu")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDoguResource.Namespace, toDogu).Return(nil)
		imageRegMock := mocks.NewImageRegistry(t)
		image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
		imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)
		fileEx := mocks.NewFileExtractor(t)
		fileEx.On("ExtractK8sResourcesFromContainer", testCtx, toDoguResource, toDogu).Return(nil, nil)
		applier := mocks.NewCollectApplier(t)
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "my-deployment"}}
		var emptyCustomK8sResource map[string]string
		applier.On("CollectApply", testCtx, emptyCustomK8sResource, toDoguResource).Return(deployment, nil)
		upserter := mocks.NewResourceUpserter(t)
		upserter.On("ApplyDoguResource", testCtx, toDoguResource, toDogu, image, deployment).Return(nil)

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			fileExtractor:         fileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, toDogu)

		// then
		require.NoError(t, err)
		update := getUpdateDoguResource(t, myClient, toDoguResource.GetObjectKey())
		assert.Equal(t, "installed", update.Status.Status)
		// mocks will be asserted during t.CleanUp
	})
	t.Run("should fail during resource upgrade", func(t *testing.T) {
		// given
		// fromDogu := readTestDataDogu(t, redmineBytes) // v4.2.3-10
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion

		dependentDeployment := createTestDeployment("redmine")
		dependencyDeployment := createTestDeployment("dependency-dogu")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDoguResource.Namespace, toDogu).Return(nil)
		imageRegMock := mocks.NewImageRegistry(t)
		image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
		imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)
		fileEx := mocks.NewFileExtractor(t)
		fileEx.On("ExtractK8sResourcesFromContainer", testCtx, toDoguResource, toDogu).Return(nil, nil)
		applier := mocks.NewCollectApplier(t)
		deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "my-deployment"}}
		var emptyCustomK8sResource map[string]string
		applier.On("CollectApply", testCtx, emptyCustomK8sResource, toDoguResource).Return(deployment, nil)
		upserter := mocks.NewResourceUpserter(t)
		upserter.On("ApplyDoguResource", testCtx, toDoguResource, toDogu, image, deployment).Return(assert.AnError)

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			fileExtractor:         fileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		update := getUpdateDoguResource(t, myClient, toDoguResource.GetObjectKey())
		assert.Equal(t, "upgrading", update.Status.Status)
		// mocks will be asserted during t.CleanUp
	})
	t.Run("should fail during resource application", func(t *testing.T) {
		// given
		// fromDogu := readTestDataDogu(t, redmineBytes) // v4.2.3-10
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion

		dependentDeployment := createTestDeployment("redmine")
		dependencyDeployment := createTestDeployment("dependency-dogu")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDoguResource.Namespace, toDogu).Return(nil)
		imageRegMock := mocks.NewImageRegistry(t)
		image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
		imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)
		fileEx := mocks.NewFileExtractor(t)
		fileEx.On("ExtractK8sResourcesFromContainer", testCtx, toDoguResource, toDogu).Return(nil, nil)
		applier := mocks.NewCollectApplier(t)
		var emptyCustomK8sResource map[string]string
		applier.On("CollectApply", testCtx, emptyCustomK8sResource, toDoguResource).Return(nil, assert.AnError)
		upserter := mocks.NewResourceUpserter(t)

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			fileExtractor:         fileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		update := getUpdateDoguResource(t, myClient, toDoguResource.GetObjectKey())
		assert.Equal(t, "upgrading", update.Status.Status)
		// mocks will be asserted during t.CleanUp
	})
	t.Run("should fail during K8s resource extraction", func(t *testing.T) {
		// given
		// fromDogu := readTestDataDogu(t, redmineBytes) // v4.2.3-10
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion

		dependentDeployment := createTestDeployment("redmine")
		dependencyDeployment := createTestDeployment("dependency-dogu")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDoguResource.Namespace, toDogu).Return(nil)
		imageRegMock := mocks.NewImageRegistry(t)
		image := &imagev1.ConfigFile{Author: "Gerard du Testeaux"}
		imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(image, nil)
		fileEx := mocks.NewFileExtractor(t)
		fileEx.On("ExtractK8sResourcesFromContainer", testCtx, toDoguResource, toDogu).Return(nil, assert.AnError)
		applier := mocks.NewCollectApplier(t)
		upserter := mocks.NewResourceUpserter(t)

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			fileExtractor:         fileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		update := getUpdateDoguResource(t, myClient, toDoguResource.GetObjectKey())
		assert.Equal(t, "upgrading", update.Status.Status)
		// mocks will be asserted during t.CleanUp
	})
	t.Run("should fail during image pull", func(t *testing.T) {
		// given
		// fromDogu := readTestDataDogu(t, redmineBytes) // v4.2.3-10
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion

		dependentDeployment := createTestDeployment("redmine")
		dependencyDeployment := createTestDeployment("dependency-dogu")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDoguResource.Namespace, toDogu).Return(nil)
		imageRegMock := mocks.NewImageRegistry(t)
		imageRegMock.On("PullImageConfig", testCtx, toDogu.Image+":"+toDogu.Version).Return(nil, assert.AnError)
		fileEx := mocks.NewFileExtractor(t)
		applier := mocks.NewCollectApplier(t)
		upserter := mocks.NewResourceUpserter(t)

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			fileExtractor:         fileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		update := getUpdateDoguResource(t, myClient, toDoguResource.GetObjectKey())
		assert.Equal(t, "upgrading", update.Status.Status)
		// mocks will be asserted during t.CleanUp
	})
	t.Run("should fail during SA creation", func(t *testing.T) {
		// given
		// fromDogu := readTestDataDogu(t, redmineBytes) // v4.2.3-10
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion

		dependentDeployment := createTestDeployment("redmine")
		dependencyDeployment := createTestDeployment("dependency-dogu")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(nil)
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDoguResource.Namespace, toDogu).Return(assert.AnError)
		imageRegMock := mocks.NewImageRegistry(t)
		fileEx := mocks.NewFileExtractor(t)
		applier := mocks.NewCollectApplier(t)
		upserter := mocks.NewResourceUpserter(t)

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			fileExtractor:         fileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		update := getUpdateDoguResource(t, myClient, toDoguResource.GetObjectKey())
		assert.Equal(t, "upgrading", update.Status.Status)
		// mocks will be asserted during t.CleanUp
	})
	t.Run("should fail for etcd error", func(t *testing.T) {
		// given
		// fromDogu := readTestDataDogu(t, redmineBytes) // v4.2.3-10
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDogu.Dependencies = []core.Dependency{{
			Type: core.DependencyTypeDogu,
			Name: "dependencyDogu",
		}}

		toDoguResource := readTestDataRedmineCr(t)
		toDoguResource.Spec.Version = redmineUpgradeVersion

		dependentDeployment := createTestDeployment("redmine")
		dependencyDeployment := createTestDeployment("dependency-dogu")

		myClient := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(toDoguResource, dependentDeployment, dependencyDeployment).
			Build()

		registrator := mocks.NewDoguRegistrator(t)
		registrator.On("RegisterDoguVersion", toDogu).Return(assert.AnError)
		saCreator := mocks.NewServiceAccountCreator(t)
		imageRegMock := mocks.NewImageRegistry(t)
		fileEx := mocks.NewFileExtractor(t)
		applier := mocks.NewCollectApplier(t)
		upserter := mocks.NewResourceUpserter(t)

		sut := &upgradeExecutor{
			client:                myClient,
			imageRegistry:         imageRegMock,
			collectApplier:        applier,
			fileExtractor:         fileEx,
			serviceAccountCreator: saCreator,
			doguRegistrator:       registrator,
			resourceUpserter:      upserter,
		}

		// when
		err := sut.Upgrade(testCtx, toDoguResource, toDogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		update := getUpdateDoguResource(t, myClient, toDoguResource.GetObjectKey())
		assert.Equal(t, "upgrading", update.Status.Status)
		// mocks will be asserted during t.CleanUp
	})
}

func getUpdateDoguResource(t *testing.T, myClient client.Client, doguObjKey client.ObjectKey) *v1.Dogu {
	t.Helper()

	updatedDoguResource := &v1.Dogu{}
	err := myClient.Get(testCtx, doguObjKey, updatedDoguResource)
	require.NoError(t, err)

	return updatedDoguResource
}

func Test_registerUpgradedDoguVersion(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion

		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		doguRegistryMock := new(regmock.DoguRegistry)
		registryMock := new(regmock.Registry)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		doguRegistryMock.On("IsEnabled", toDogu.GetSimpleName()).Return(true, nil)
		doguRegistryMock.On("Register", toDogu).Return(nil)
		doguRegistryMock.On("Enable", toDogu).Return(nil)

		cesreg := cesregistry.NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registerUpgradedDoguVersion(cesreg, toDogu)

		// then
		require.NoError(t, err)
		registryMock.AssertExpectations(t)
		doguRegistryMock.AssertExpectations(t)
	})
	t.Run("should fail", func(t *testing.T) {
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion

		doguRegistryMock := new(regmock.DoguRegistry)
		registryMock := new(regmock.Registry)
		registryMock.On("DoguRegistry").Return(doguRegistryMock)
		doguRegistryMock.On("IsEnabled", toDogu.GetSimpleName()).Return(false, nil)

		cesreg := cesregistry.NewCESDoguRegistrator(nil, registryMock, nil)

		// when
		err := registerUpgradedDoguVersion(cesreg, toDogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to register upgrade: could not register dogu version: previous version not found")
		registryMock.AssertExpectations(t)
		doguRegistryMock.AssertExpectations(t)
	})
}

func Test_registerNewServiceAccount(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDoguCr.Namespace, toDogu).Return(nil)

		// when
		err := registerNewServiceAccount(testCtx, saCreator, toDoguCr, toDogu)

		// then
		require.NoError(t, err)
		saCreator.AssertExpectations(t)
	})
	t.Run("should fail", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		saCreator := mocks.NewServiceAccountCreator(t)
		saCreator.On("CreateAll", testCtx, toDoguCr.Namespace, toDogu).Return(assert.AnError)

		// when
		err := registerNewServiceAccount(testCtx, saCreator, toDoguCr, toDogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to register service accounts: assert.AnError")
		saCreator.AssertExpectations(t)
	})
}

func Test_upgradeExecutor_pullUpgradeImage(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		imagePuller := mocks.NewImageRegistry(t)
		doguImage := toDogu.Image + ":" + toDogu.Version

		imagePuller.On("PullImageConfig", testCtx, doguImage).Return(&imagev1.ConfigFile{}, nil)

		// when
		image, err := pullUpgradeImage(testCtx, imagePuller, toDogu)

		// then
		require.NoError(t, err)
		require.NotNil(t, image)
	})
	t.Run("should fail", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		imagePuller := mocks.NewImageRegistry(t)
		doguImage := toDogu.Image + ":" + toDogu.Version
		var noConfigFile *imagev1.ConfigFile

		imagePuller.On("PullImageConfig", testCtx, doguImage).Return(noConfigFile, assert.AnError)

		// when
		_, err := pullUpgradeImage(testCtx, imagePuller, toDogu)

		// then
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to pull upgrade image: assert.AnError")
	})
}

func Test_extractCustomK8sResources(t *testing.T) {
	t.Run("should return custom K8s resources", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		extractor := mocks.NewFileExtractor(t)
		fakeResources := make(map[string]string, 0)
		fakeResources["lefile.yaml"] = "levalue"
		extractor.On("ExtractK8sResourcesFromContainer", testCtx, toDoguCr, toDogu).Return(fakeResources, nil)

		// when
		resources, err := extractCustomK8sResources(testCtx, extractor, toDoguCr, toDogu)

		// then
		require.NoError(t, err)
		assert.Equal(t, fakeResources, resources)
	})
	t.Run("should return no custom K8s resources", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		extractor := mocks.NewFileExtractor(t)
		var emptyResourcesAreValidToo map[string]string
		extractor.On("ExtractK8sResourcesFromContainer", testCtx, toDoguCr, toDogu).Return(emptyResourcesAreValidToo, nil)

		// when
		resources, err := extractCustomK8sResources(testCtx, extractor, toDoguCr, toDogu)

		// then
		require.NoError(t, err)
		assert.Nil(t, resources)
	})
	t.Run("should fail", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		extractor := mocks.NewFileExtractor(t)
		var nilMap map[string]string
		extractor.On("ExtractK8sResourcesFromContainer", testCtx, toDoguCr, toDogu).Return(nilMap, assert.AnError)

		// when
		_, err := extractCustomK8sResources(testCtx, extractor, toDoguCr, toDogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to extract custom K8s resources: assert.AnError")
	})
}

func Test_applyCustomK8sResources(t *testing.T) {
	t.Run("should apply K8s resources and return deployment", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		collectApplier := mocks.NewCollectApplier(t)
		fakeResources := make(map[string]string, 0)
		fakeResources["lefile.yaml"] = "levalue"
		fakeDeployment := createTestDeployment("redmine")
		collectApplier.On("CollectApply", mock.Anything, fakeResources, toDoguCr).Return(fakeDeployment, nil)

		// when
		deployment, err := applyCustomK8sResources(testCtx, collectApplier, toDoguCr, fakeResources)

		// then
		require.NoError(t, err)
		assert.Equal(t, fakeDeployment, deployment)
	})
	t.Run("should fail", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = redmineUpgradeVersion
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = redmineUpgradeVersion
		collectApplier := mocks.NewCollectApplier(t)
		fakeResources := make(map[string]string, 0)
		fakeResources["lefile.yaml"] = "levalue"
		var noDeployment *appsv1.Deployment
		collectApplier.On("CollectApply", mock.Anything, fakeResources, toDoguCr).Return(noDeployment, assert.AnError)

		// when
		_, err := applyCustomK8sResources(testCtx, collectApplier, toDoguCr, fakeResources)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to apply custom K8s resources: assert.AnError")
	})
}

func createTestDeployment(doguName string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: deploymentTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      doguName,
			Namespace: testNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{ServiceAccountName: "somethingNonEmptyToo"}},
		},
		Status: appsv1.DeploymentStatus{Replicas: 1, ReadyReplicas: 1},
	}
}
