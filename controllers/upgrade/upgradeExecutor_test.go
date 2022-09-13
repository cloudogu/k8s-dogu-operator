package upgrade

import (
	"context"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/go-logr/logr"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var testCtx = context.TODO()

func Test_upgradeExecutor_Upgrade(t *testing.T) {

}

func Test_registerUpgradedDoguVersion(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = "4.2.3-11"

		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = "4.2.3-11"
		doguRegistryMock := new(mocks.DoguRegistry)
		registryMock := new(mocks.Registry)
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
		toDoguCr.Spec.Version = "4.2.3-11"

		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = "4.2.3-11"
		doguRegistryMock := new(mocks.DoguRegistry)
		registryMock := new(mocks.Registry)
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
		toDogu.Version = "4.2.3-11"
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = "4.2.3-11"
		saCreator := new(saCreatorMock)
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
		toDogu.Version = "4.2.3-11"
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = "4.2.3-11"
		saCreator := new(saCreatorMock)
		saCreator.On("CreateAll", testCtx, toDoguCr.Namespace, toDogu).Return(assert.AnError)

		// when
		err := registerNewServiceAccount(testCtx, saCreator, toDoguCr, toDogu)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to register service accounts: assert.AnError")
		saCreator.AssertExpectations(t)
	})
}

type saCreatorMock struct {
	mock.Mock
}

func Test_upgradeExecutor_pullUpgradeImage(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		toDogu := readTestDataDogu(t, redmineBytes)
		toDogu.Version = "4.2.3-11"
		imagePuller := new(imagePullMock)
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
		toDogu.Version = "4.2.3-11"
		imagePuller := new(imagePullMock)
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
		toDogu.Version = "4.2.3-11"
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = "4.2.3-11"
		extractor := new(fileExtractorMock)
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
		toDogu.Version = "4.2.3-11"
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = "4.2.3-11"
		extractor := new(fileExtractorMock)
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
		toDogu.Version = "4.2.3-11"
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = "4.2.3-11"
		extractor := new(fileExtractorMock)
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
		toDogu.Version = "4.2.3-11"
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = "4.2.3-11"
		collectApplier := new(collectApplyMock)
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
		toDogu.Version = "4.2.3-11"
		toDoguCr := readTestDataRedmineCr(t)
		toDoguCr.Spec.Version = "4.2.3-11"
		collectApplier := new(collectApplyMock)
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

func (s *saCreatorMock) CreateAll(ctx context.Context, namespace string, dogu *core.Dogu) error {
	args := s.Called(ctx, namespace, dogu)
	return args.Error(0)
}

type imagePullMock struct {
	mock.Mock
}

func (i *imagePullMock) PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error) {
	args := i.Called(ctx, image)
	return args.Get(0).(*imagev1.ConfigFile), args.Error(1)
}

type fileExtractorMock struct {
	mock.Mock
}

func (f *fileExtractorMock) ExtractK8sResourcesFromContainer(ctx context.Context, resource *k8sv1.Dogu, dogu *core.Dogu) (map[string]string, error) {
	args := f.Called(ctx, resource, dogu)
	return args.Get(0).(map[string]string), args.Error(1)
}

type collectApplyMock struct {
	mock.Mock
}

func (c *collectApplyMock) CollectApply(logger logr.Logger, customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*appsv1.Deployment, error) {
	args := c.Called(logger, customK8sResources, doguResource)
	return args.Get(0).(*appsv1.Deployment), args.Error(1)
}
