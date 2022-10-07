package controllers

import (
	"testing"

	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/cloudogu/k8s-dogu-operator/controllers/upgrade"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stretchr/testify/mock"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const defaultNamespace = ""

var deploymentTypeMeta = metav1.TypeMeta{
	APIVersion: "apps/v1",
	Kind:       "Deployment",
}

func createTestRestConfig() *rest.Config {
	return &rest.Config{}
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
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = createTestRestConfig

	t.Run("fail when no valid kube config was found", func(t *testing.T) {
		// given

		// override default controller method to return a config that fail the client creation
		oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
		defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
		ctrl.GetConfigOrDie = func() *rest.Config {
			return &rest.Config{ExecProvider: &api.ExecConfig{}, AuthProvider: &api.AuthProviderConfig{}}
		}

		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"

		// when
		doguManager, err := NewDoguUpgradeManager(nil, operatorConfig, nil, nil)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
	})

	t.Run("should implement upgradeManager", func(t *testing.T) {
		myClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := &cesmocks.Registry{}
		doguRegistry := &cesmocks.DoguRegistry{}
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		// when
		actual, err := NewDoguUpgradeManager(myClient, operatorConfig, cesRegistry, nil)

		// then
		require.NoError(t, err)
		require.NotNil(t, actual)
		assert.Implements(t, (*upgradeManager)(nil), actual)
		mock.AssertExpectationsForObjects(t, doguRegistry, cesRegistry)
	})
}

func newTestDoguUpgradeManager(client client.Client, recorder record.EventRecorder, ldf localDoguFetcher, rdf resourceDoguFetcher, pc premisesChecker, ue upgradeExecutor) *doguUpgradeManager {
	return &doguUpgradeManager{
		client:              client,
		eventRecorder:       recorder,
		localDoguFetcher:    ldf,
		resourceDoguFetcher: rdf,
		premisesChecker:     pc,
		upgradeExecutor:     ue,
	}
}

func Test_doguUpgradeManager_Upgrade(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigOrDieDelegate := ctrl.GetConfigOrDie
	defer func() { ctrl.GetConfigOrDie = oldGetConfigOrDieDelegate }()
	ctrl.GetConfigOrDie = createTestRestConfig

	operatorConfig := &config.OperatorConfig{}
	operatorConfig.Namespace = testNamespace

	t.Run("should succeed on regular upgrade from the remote registry", func(t *testing.T) {
		// given
		redmineCr := readDoguCr(t, redmineCrBytes)
		upgradeVersion := "4.2.3-11"
		redmineCr.Spec.Version = upgradeVersion
		redmineCr.Spec.UpgradeConfig.AllowNamespaceSwitch = true

		redmineDoguInstalled := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade := readDoguDescriptor(t, redmineDoguDescriptorBytes)
		redmineDoguUpgrade.Version = upgradeVersion

		recorderMock := mocks.NewEventRecorder(t)
		recorderMock.On("Event", redmineCr, corev1.EventTypeNormal, upgrade.UpgradeEventReason, "Checking premises...")
		recorderMock.On("Eventf", redmineCr, corev1.EventTypeNormal, upgrade.UpgradeEventReason, "Executing upgrade from %s to %s...", "4.2.3-10", upgradeVersion)

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.On("FetchInstalled", "redmine").Return(redmineDoguInstalled, nil)

		resourceFetcher := mocks.NewResourceDoguFetcher(t)
		resourceFetcher.On("FetchWithResource", testCtx, redmineCr).Return(redmineDoguUpgrade, nil, nil)

		premChecker := mocks.NewPremisesChecker(t)
		premChecker.On("Check", testCtx, redmineCr, redmineDoguInstalled, redmineDoguUpgrade).Return(nil)

		upgradeExec := mocks.NewUpgradeExecutor(t)
		upgradeExec.On("Upgrade", testCtx, redmineCr, redmineDoguUpgrade).Return(nil)

		deplRedmine := createReadyDeployment("redmine")
		deplPostgres := createReadyDeployment("postgresql")
		deplCas := createReadyDeployment("cas")
		deplNginx1 := createReadyDeployment("nginx-ingress")
		deplNginx2 := createReadyDeployment("nginx-static")
		deplPostfix := createReadyDeployment("postfix")

		clientMock := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(redmineCr, deplRedmine, deplPostgres, deplCas, deplNginx1, deplNginx2, deplPostfix).
			Build()

		sut := newTestDoguUpgradeManager(clientMock, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)
		sut.resourceDoguFetcher = resourceFetcher

		// when
		err := sut.Upgrade(testCtx, redmineCr)

		// then
		require.NoError(t, err)
		// any other mocks assert their expectations during t.CleanUp()
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

		recorderMock := mocks.NewEventRecorder(t)
		recorderMock.On("Event", redmineCr, corev1.EventTypeNormal, upgrade.UpgradeEventReason, "Checking premises...")
		recorderMock.On("Eventf", redmineCr, corev1.EventTypeNormal, upgrade.UpgradeEventReason, "Executing upgrade from %s to %s...", "4.2.3-10", upgradeVersion)

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
		upgradeExec.On("Upgrade", testCtx, redmineCr, redmineDoguUpgrade).Return(nil)

		deplRedmine := createReadyDeployment("redmine")
		deplPostgres := createReadyDeployment("postgresql")
		deplCas := createReadyDeployment("cas")
		deplNginx1 := createReadyDeployment("nginx-ingress")
		deplNginx2 := createReadyDeployment("nginx-static")
		deplPostfix := createReadyDeployment("postfix")

		clientMock := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(devDoguMap.ToConfigMap(), redmineCr, deplRedmine, deplPostgres, deplCas, deplNginx1, deplNginx2, deplPostfix).
			Build()
		preErr := clientMock.Get(testCtx, redmineCr.GetObjectKey(), devDoguMap.ToConfigMap())
		assert.False(t, errors.IsNotFound(preErr))

		sut := newTestDoguUpgradeManager(clientMock, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)
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

		recorderMock := mocks.NewEventRecorder(t)
		recorderMock.On("Event", redmineCr, corev1.EventTypeNormal, upgrade.UpgradeEventReason, "Checking premises...")
		recorderMock.On("Eventf", redmineCr, corev1.EventTypeNormal, upgrade.UpgradeEventReason, "Executing upgrade from %s to %s...", "4.2.3-10", "4.2.3-11")

		localFetcher := mocks.NewLocalDoguFetcher(t)
		localFetcher.On("FetchInstalled", "redmine").Return(redmineDoguInstalled, nil)

		resourceFetcher := mocks.NewResourceDoguFetcher(t)
		resourceFetcher.On("FetchWithResource", testCtx, redmineCr).Return(redmineDoguUpgrade, nil, nil)

		premChecker := mocks.NewPremisesChecker(t)
		premChecker.On("Check", testCtx, redmineCr, redmineDoguInstalled, redmineDoguUpgrade).Return(nil)

		upgradeExec := mocks.NewUpgradeExecutor(t)
		upgradeExec.On("Upgrade", testCtx, redmineCr, redmineDoguUpgrade).Return(assert.AnError)

		deplRedmine := createReadyDeployment("redmine")
		deplPostgres := createReadyDeployment("postgresql")
		deplCas := createReadyDeployment("cas")
		deplNginx1 := createReadyDeployment("nginx-ingress")
		deplNginx2 := createReadyDeployment("nginx-static")
		deplPostfix := createReadyDeployment("postfix")

		clientMock := fake.NewClientBuilder().
			WithScheme(getTestScheme()).
			WithObjects(redmineCr, deplRedmine, deplPostgres, deplCas, deplNginx1, deplNginx2, deplPostfix).
			Build()

		sut := newTestDoguUpgradeManager(clientMock, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)
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

		recorderMock := mocks.NewEventRecorder(t)
		recorderMock.On("Event", redmineCr, corev1.EventTypeNormal, upgrade.UpgradeEventReason, "Checking premises...")

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
			WithObjects(redmineCr, deplRedmine, deplPostgres, deplCas, deplNginx1, deplNginx2, deplPostfix).
			Build()

		sut := newTestDoguUpgradeManager(clientMock, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)
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

		recorderMock := mocks.NewEventRecorder(t)

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
			WithObjects(redmineCr, deplRedmine, deplPostgres, deplCas, deplNginx1, deplNginx2, deplPostfix).
			Build()

		sut := newTestDoguUpgradeManager(clientMock, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)
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

		recorderMock := mocks.NewEventRecorder(t)

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
			WithObjects(redmineCr, deplRedmine, deplPostgres, deplCas, deplNginx1, deplNginx2, deplPostfix).
			Build()

		sut := newTestDoguUpgradeManager(clientMock, recorderMock, localFetcher, resourceFetcher, premChecker, upgradeExec)
		sut.resourceDoguFetcher = resourceFetcher

		// when
		err := sut.Upgrade(testCtx, redmineCr)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		// any other mocks assert their expectations during t.CleanUp()
	})
}
