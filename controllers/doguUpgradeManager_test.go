package controllers

import (
	"context"
	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	"github.com/cloudogu/k8s-dogu-operator/controllers/util"
	"github.com/stretchr/testify/mock"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
)

const defaultNamespace = ""

var deploymentTypeMeta = metav1.TypeMeta{
	APIVersion: "apps/v1",
	Kind:       "Deployment",
}

func createTestRestConfig() (*rest.Config, error) {
	return &rest.Config{}, nil
}

func createReadyDeployment(doguName string) *appsv1.Deployment {
	return createDeployment(doguName, 1, 1)
}

func createDeployment(doguName string, replicas, replicasReady int32) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: deploymentTypeMeta,
		ObjectMeta: metav1.ObjectMeta{
			Name:      doguName,
			Namespace: defaultNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{ServiceAccountName: "somethingNonEmptyToo"}},
		},
		Status: appsv1.DeploymentStatus{Replicas: replicas, ReadyReplicas: replicasReady},
	}
}

func TestNewDoguUpgradeManager(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigDelegate := ctrl.GetConfig
	defer func() { ctrl.GetConfig = oldGetConfigDelegate }()
	ctrl.GetConfig = createTestRestConfig

	t.Run("should implement upgradeManager", func(t *testing.T) {
		myClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := cesmocks.NewRegistry(t)
		doguRegistry := cesmocks.NewDoguRegistry(t)
		cesRegistry.On("DoguRegistry").Return(doguRegistry)
		mgrSet := &util.ManagerSet{}

		// when
		actual := NewDoguUpgradeManager(myClient, nil, operatorConfig, cesRegistry, mgrSet, nil)

		// then
		require.NotNil(t, actual)
		assert.Implements(t, (*cloudogu.UpgradeManager)(nil), actual)
	})
}

func newTestDoguUpgradeManager(
	client client.Client,
	ecosystemClient ecoSystem.EcoSystemV1Alpha1Interface,
	recorder record.EventRecorder,
	ldf cloudogu.LocalDoguFetcher,
	rdf cloudogu.ResourceDoguFetcher,
	pc cloudogu.PremisesChecker,
	ue cloudogu.UpgradeExecutor,
) *doguUpgradeManager {
	return &doguUpgradeManager{
		client:              client,
		ecosystemClient:     ecosystemClient,
		eventRecorder:       recorder,
		localDoguFetcher:    ldf,
		resourceDoguFetcher: rdf,
		premisesChecker:     pc,
		upgradeExecutor:     ue,
	}
}

const testNamespace = "test-namespace"

func Test_doguUpgradeManager_Upgrade(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfig = createTestRestConfig

	operatorConfig := &config.OperatorConfig{}
	operatorConfig.Namespace = testNamespace

	t.Run("should succeed on regular upgrade from the remote registry", func(t *testing.T) {
		// given
		redmineCr := readDoguCr(t, redmineCrBytes)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion
		redmineCr.Spec.UpgradeConfig.AllowNamespaceSwitch = true
		redmineCr.Status.InstalledVersion = "4.2.3-9"

		redmineDoguInstalled := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade.Version = upgradeVersion

		recorderMock := extMocks.NewEventRecorder(t)
		recorderMock.On("Event", redmineCr, corev1.EventTypeNormal, upgrade.EventReason, "Checking premises...")
		recorderMock.On("Eventf", redmineCr, corev1.EventTypeNormal, upgrade.EventReason, "Executing upgrade from %s to %s...", "4.2.3-10", upgradeVersion)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.On("FetchInstalled", "redmine").Return(redmineDoguInstalled, nil)

		resourceFetcher := mocks.NewResourceDoguFetcher(t)
		resourceFetcher.On("FetchWithResource", testCtx, redmineCr).Return(redmineDoguUpgrade, nil, nil)

		premChecker := mocks.NewPremisesChecker(t)
		premChecker.On("Check", testCtx, redmineCr, redmineDoguInstalled, redmineDoguUpgrade).Return(nil)

		upgradeExec := mocks.NewUpgradeExecutor(t)
		upgradeExec.On("Upgrade", testCtx, redmineCr, redmineDoguInstalled, redmineDoguUpgrade).Return(nil)

		deplRedmine := createReadyDeployment("redmine")
		deplPostgres := createReadyDeployment("postgresql")
		deplCas := createReadyDeployment("cas")
		deplNginx1 := createReadyDeployment("nginx-ingress")
		deplNginx2 := createReadyDeployment("nginx-static")
		deplPostfix := createReadyDeployment("postfix")

		clientMock := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithStatusSubresource(&v1.Dogu{}).
			WithObjects(redmineCr, deplRedmine, deplPostgres, deplCas, deplNginx1, deplNginx2, deplPostfix).
			Build()

		ecosystemClientMock := mocks.NewEcosystemInterface(t)
		doguClientMock := mocks.NewDoguInterface(t)
		ecosystemClientMock.EXPECT().Dogus("").Return(doguClientMock)
		doguClientMock.EXPECT().UpdateStatusWithRetry(testCtx, redmineCr, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, dogu *v1.Dogu, f func(v1.DoguStatus) v1.DoguStatus, options metav1.UpdateOptions) (*v1.Dogu, error) {
			redmineCr.Status = f(redmineCr.Status)
			return redmineCr, nil
		})
		sut := newTestDoguUpgradeManager(clientMock, ecosystemClientMock, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)
		sut.resourceDoguFetcher = resourceFetcher

		// when
		err := sut.Upgrade(testCtx, redmineCr)

		// then
		require.NoError(t, err)
		// any other mocks assert their expectations during t.CleanUp()
		assert.Equal(t, redmineCr.Spec.Version, redmineCr.Status.InstalledVersion)
	})
	t.Run("should succeed on upgrade from a self-developed dogu", func(t *testing.T) {
		// given
		redmineCr := readDoguCr(t, redmineCrBytes)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion
		redmineCr.Spec.UpgradeConfig.AllowNamespaceSwitch = true

		redmineDoguInstalled := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade.Version = upgradeVersion

		recorderMock := extMocks.NewEventRecorder(t)
		recorderMock.On("Event", redmineCr, corev1.EventTypeNormal, upgrade.EventReason, "Checking premises...")
		recorderMock.On("Eventf", redmineCr, corev1.EventTypeNormal, upgrade.EventReason, "Executing upgrade from %s to %s...", "4.2.3-10", upgradeVersion)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.On("FetchInstalled", "redmine").Return(redmineDoguInstalled, nil)

		devDoguMap := &v1.DevelopmentDoguMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      redmineCr.GetObjectKey().Name,
				Namespace: redmineCr.GetObjectKey().Namespace,
			},
		}
		resourceFetcher := mocks.NewResourceDoguFetcher(t)
		resourceFetcher.On("FetchWithResource", testCtx, redmineCr).Return(redmineDoguUpgrade, devDoguMap, nil)

		premChecker := mocks.NewPremisesChecker(t)
		premChecker.On("Check", testCtx, redmineCr, redmineDoguInstalled, redmineDoguUpgrade).Return(nil)

		upgradeExec := mocks.NewUpgradeExecutor(t)
		upgradeExec.On("Upgrade", testCtx, redmineCr, redmineDoguInstalled, redmineDoguUpgrade).Return(nil)

		deplRedmine := createReadyDeployment("redmine")
		deplPostgres := createReadyDeployment("postgresql")
		deplCas := createReadyDeployment("cas")
		deplNginx1 := createReadyDeployment("nginx-ingress")
		deplNginx2 := createReadyDeployment("nginx-static")
		deplPostfix := createReadyDeployment("postfix")

		clientMock := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithStatusSubresource(&v1.Dogu{}).
			WithObjects(devDoguMap.ToConfigMap(), redmineCr, deplRedmine, deplPostgres, deplCas, deplNginx1, deplNginx2, deplPostfix).
			Build()
		preErr := clientMock.Get(testCtx, redmineCr.GetObjectKey(), devDoguMap.ToConfigMap())
		assert.False(t, errors.IsNotFound(preErr))

		ecosystemClientMock := mocks.NewEcosystemInterface(t)
		doguClientMock := mocks.NewDoguInterface(t)
		ecosystemClientMock.EXPECT().Dogus("").Return(doguClientMock)
		doguClientMock.EXPECT().UpdateStatusWithRetry(testCtx, redmineCr, mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, dogu *v1.Dogu, f func(v1.DoguStatus) v1.DoguStatus, options metav1.UpdateOptions) (*v1.Dogu, error) {
			redmineCr.Status = f(redmineCr.Status)
			return redmineCr, nil
		})

		sut := newTestDoguUpgradeManager(clientMock, ecosystemClientMock, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)
		sut.resourceDoguFetcher = resourceFetcher

		// when
		err := sut.Upgrade(testCtx, redmineCr)

		// then
		require.NoError(t, err)
		expectedToBeDeleted := devDoguMap.ToConfigMap()
		postErr := clientMock.Get(testCtx, redmineCr.GetObjectKey(), expectedToBeDeleted)
		assert.True(t, errors.IsNotFound(postErr))
		// any other mocks assert their expectations during t.CleanUp()
	})
	t.Run("should fail during upgrading redmine and record the error event", func(t *testing.T) {
		// given
		redmineCr := readDoguCr(t, redmineCrBytes)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion
		redmineCr.Spec.UpgradeConfig.AllowNamespaceSwitch = true

		redmineDoguInstalled := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade.Version = upgradeVersion

		recorderMock := extMocks.NewEventRecorder(t)
		recorderMock.On("Event", redmineCr, corev1.EventTypeNormal, upgrade.EventReason, "Checking premises...")
		recorderMock.On("Eventf", redmineCr, corev1.EventTypeNormal, upgrade.EventReason, "Executing upgrade from %s to %s...", "4.2.3-10", "4.2.3-11")

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.On("FetchInstalled", "redmine").Return(redmineDoguInstalled, nil)

		resourceFetcher := mocks.NewResourceDoguFetcher(t)
		resourceFetcher.On("FetchWithResource", testCtx, redmineCr).Return(redmineDoguUpgrade, nil, nil)

		premChecker := mocks.NewPremisesChecker(t)
		premChecker.On("Check", testCtx, redmineCr, redmineDoguInstalled, redmineDoguUpgrade).Return(nil)

		upgradeExec := mocks.NewUpgradeExecutor(t)
		upgradeExec.On("Upgrade", testCtx, redmineCr, redmineDoguInstalled, redmineDoguUpgrade).Return(assert.AnError)

		deplRedmine := createReadyDeployment("redmine")
		deplPostgres := createReadyDeployment("postgresql")
		deplCas := createReadyDeployment("cas")
		deplNginx1 := createReadyDeployment("nginx-ingress")
		deplNginx2 := createReadyDeployment("nginx-static")
		deplPostfix := createReadyDeployment("postfix")

		clientMock := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithStatusSubresource(&v1.Dogu{}).
			WithObjects(redmineCr, deplRedmine, deplPostgres, deplCas, deplNginx1, deplNginx2, deplPostfix).
			Build()

		sut := newTestDoguUpgradeManager(clientMock, nil, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)
		sut.resourceDoguFetcher = resourceFetcher

		// when
		err := sut.Upgrade(testCtx, redmineCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		// any other mocks assert their expectations during t.CleanUp()
	})
	t.Run("should fail during premises check and record the error event", func(t *testing.T) {
		// given
		redmineCr := readDoguCr(t, redmineCrBytes)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion
		redmineCr.Spec.UpgradeConfig.AllowNamespaceSwitch = true

		redmineDoguInstalled := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade.Version = upgradeVersion

		recorderMock := extMocks.NewEventRecorder(t)
		recorderMock.On("Event", redmineCr, corev1.EventTypeNormal, upgrade.EventReason, "Checking premises...")

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.On("FetchInstalled", "redmine").Return(redmineDoguInstalled, nil)

		resourceFetcher := mocks.NewResourceDoguFetcher(t)
		resourceFetcher.On("FetchWithResource", testCtx, redmineCr).Return(redmineDoguUpgrade, nil, nil)

		premChecker := mocks.NewPremisesChecker(t)
		premChecker.On("Check", testCtx, redmineCr, redmineDoguInstalled, redmineDoguUpgrade).Return(assert.AnError)

		upgradeExec := mocks.NewUpgradeExecutor(t)

		deplRedmine := createReadyDeployment("redmine")
		deplPostgres := createReadyDeployment("postgresql")
		deplCas := createReadyDeployment("cas")
		deplNginx1 := createReadyDeployment("nginx-ingress")
		deplNginx2 := createReadyDeployment("nginx-static")
		deplPostfix := createReadyDeployment("postfix")

		clientMock := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithStatusSubresource(&v1.Dogu{}).
			WithObjects(redmineCr, deplRedmine, deplPostgres, deplCas, deplNginx1, deplNginx2, deplPostfix).
			Build()

		sut := newTestDoguUpgradeManager(clientMock, nil, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)
		sut.resourceDoguFetcher = resourceFetcher

		// when
		err := sut.Upgrade(testCtx, redmineCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		// any other mocks assert their expectations during t.CleanUp()
	})
	t.Run("should fail during fetching remote redmine dogu and record the error event", func(t *testing.T) {
		// given
		redmineCr := readDoguCr(t, redmineCrBytes)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion
		redmineCr.Spec.UpgradeConfig.AllowNamespaceSwitch = true

		redmineDoguInstalled := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade.Version = upgradeVersion

		recorderMock := extMocks.NewEventRecorder(t)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.On("FetchInstalled", "redmine").Return(redmineDoguInstalled, nil)

		resourceFetcher := mocks.NewResourceDoguFetcher(t)
		resourceFetcher.On("FetchWithResource", testCtx, redmineCr).Return(nil, nil, assert.AnError)

		premChecker := mocks.NewPremisesChecker(t)
		upgradeExec := mocks.NewUpgradeExecutor(t)

		deplRedmine := createReadyDeployment("redmine")
		deplPostgres := createReadyDeployment("postgresql")
		deplCas := createReadyDeployment("cas")
		deplNginx1 := createReadyDeployment("nginx-ingress")
		deplNginx2 := createReadyDeployment("nginx-static")
		deplPostfix := createReadyDeployment("postfix")

		clientMock := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithStatusSubresource(&v1.Dogu{}).
			WithObjects(redmineCr, deplRedmine, deplPostgres, deplCas, deplNginx1, deplNginx2, deplPostfix).
			Build()

		sut := newTestDoguUpgradeManager(clientMock, nil, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)
		sut.resourceDoguFetcher = resourceFetcher

		// when
		err := sut.Upgrade(testCtx, redmineCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		// any other mocks assert their expectations during t.CleanUp()
	})
	t.Run("should fail during fetching installed redmine and record the error event", func(t *testing.T) {
		// given
		redmineCr := readDoguCr(t, redmineCrBytes)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion
		redmineCr.Spec.UpgradeConfig.AllowNamespaceSwitch = true

		recorderMock := extMocks.NewEventRecorder(t)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.On("FetchInstalled", "redmine").Return(nil, assert.AnError)

		resourceFetcher := mocks.NewResourceDoguFetcher(t)

		premChecker := mocks.NewPremisesChecker(t)
		upgradeExec := mocks.NewUpgradeExecutor(t)

		deplRedmine := createReadyDeployment("redmine")
		deplPostgres := createReadyDeployment("postgresql")
		deplCas := createReadyDeployment("cas")
		deplNginx1 := createReadyDeployment("nginx-ingress")
		deplNginx2 := createReadyDeployment("nginx-static")
		deplPostfix := createReadyDeployment("postfix")

		clientMock := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithStatusSubresource(&v1.Dogu{}).
			WithObjects(redmineCr, deplRedmine, deplPostgres, deplCas, deplNginx1, deplNginx2, deplPostfix).
			Build()

		sut := newTestDoguUpgradeManager(clientMock, nil, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)
		sut.resourceDoguFetcher = resourceFetcher

		// when
		err := sut.Upgrade(testCtx, redmineCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		// any other mocks assert their expectations during t.CleanUp()
	})
	t.Run("should fail on first change state error", func(t *testing.T) {
		// given
		redmineCr := readDoguCr(t, redmineCrBytes)
		recorderMock := extMocks.NewEventRecorder(t)
		localFetcher := mocks.NewLocalDoguFetcher(t)
		resourceFetcher := mocks.NewResourceDoguFetcher(t)
		premChecker := mocks.NewPremisesChecker(t)
		upgradeExec := mocks.NewUpgradeExecutor(t)

		clientMock := extMocks.NewK8sClient(t)
		clientMock.EXPECT().Get(testCtx, mock.Anything, mock.Anything).Return(assert.AnError)

		sut := newTestDoguUpgradeManager(clientMock, nil, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)
		sut.resourceDoguFetcher = resourceFetcher

		// when
		err := sut.Upgrade(testCtx, redmineCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should fail on second change state error", func(t *testing.T) {
		// given
		redmineCr := readDoguCr(t, redmineCrBytes)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion

		redmineDoguInstalled := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade.Version = upgradeVersion

		recorderMock := extMocks.NewEventRecorder(t)
		recorderMock.On("Event", redmineCr, corev1.EventTypeNormal, upgrade.EventReason, "Checking premises...")
		recorderMock.On("Eventf", redmineCr, corev1.EventTypeNormal, upgrade.EventReason, "Executing upgrade from %s to %s...", "4.2.3-10", upgradeVersion)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.On("FetchInstalled", "").Return(redmineDoguInstalled, nil)

		resourceFetcher := mocks.NewResourceDoguFetcher(t)
		resourceFetcher.On("FetchWithResource", testCtx, redmineCr).Return(redmineDoguUpgrade, nil, nil)

		premChecker := mocks.NewPremisesChecker(t)
		premChecker.On("Check", testCtx, redmineCr, redmineDoguInstalled, redmineDoguUpgrade).Return(nil)

		upgradeExec := mocks.NewUpgradeExecutor(t)
		upgradeExec.On("Upgrade", testCtx, redmineCr, redmineDoguInstalled, redmineDoguUpgrade).Return(nil)

		clientMock := extMocks.NewK8sClient(t)
		statusMock := extMocks.NewK8sSubResourceWriter(t)
		clientMock.EXPECT().Get(testCtx, mock.Anything, mock.Anything).Return(nil).Once()
		clientMock.EXPECT().Get(testCtx, mock.Anything, mock.Anything).Return(assert.AnError)
		clientMock.EXPECT().Status().Return(statusMock)
		statusMock.On("Update", testCtx, redmineCr).Return(nil)

		sut := newTestDoguUpgradeManager(clientMock, nil, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)

		// when
		err := sut.Upgrade(testCtx, redmineCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should fail on update installed version error", func(t *testing.T) {
		// given
		redmineCr := readDoguCr(t, redmineCrBytes)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion

		redmineDoguInstalled := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade.Version = upgradeVersion

		recorderMock := extMocks.NewEventRecorder(t)
		recorderMock.On("Event", redmineCr, corev1.EventTypeNormal, upgrade.EventReason, "Checking premises...")
		recorderMock.On("Eventf", redmineCr, corev1.EventTypeNormal, upgrade.EventReason, "Executing upgrade from %s to %s...", "4.2.3-10", upgradeVersion)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.On("FetchInstalled", "").Return(redmineDoguInstalled, nil)

		resourceFetcher := mocks.NewResourceDoguFetcher(t)
		resourceFetcher.On("FetchWithResource", testCtx, redmineCr).Return(redmineDoguUpgrade, nil, nil)

		premChecker := mocks.NewPremisesChecker(t)
		premChecker.On("Check", testCtx, redmineCr, redmineDoguInstalled, redmineDoguUpgrade).Return(nil)

		upgradeExec := mocks.NewUpgradeExecutor(t)
		upgradeExec.On("Upgrade", testCtx, redmineCr, redmineDoguInstalled, redmineDoguUpgrade).Return(nil)

		clientMock := extMocks.NewK8sClient(t)
		statusMock := extMocks.NewK8sSubResourceWriter(t)
		clientMock.EXPECT().Get(testCtx, mock.Anything, mock.Anything).Return(nil).Twice()
		clientMock.EXPECT().Status().Return(statusMock).Twice()
		statusMock.On("Update", testCtx, redmineCr).Return(nil).Twice()

		ecosystemClientMock := mocks.NewEcosystemInterface(t)
		doguClientMock := mocks.NewDoguInterface(t)
		ecosystemClientMock.EXPECT().Dogus("").Return(doguClientMock)
		doguClientMock.EXPECT().UpdateStatusWithRetry(testCtx, redmineCr, mock.Anything, mock.Anything).Return(redmineCr, assert.AnError)

		sut := newTestDoguUpgradeManager(clientMock, ecosystemClientMock, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)

		// when
		err := sut.Upgrade(testCtx, redmineCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}
