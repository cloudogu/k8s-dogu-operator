package initfx

import (
	"testing"

	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

const namespace = "namespace"

func TestNewConfigMapInterface(t *testing.T) {
	t.Run("should successfully create config map interface", func(t *testing.T) {
		// given
		coreV1Mock := newMockCoreV1Interface(t)
		configMapMock := newMockConfigMapInterface(t)
		coreV1Mock.EXPECT().ConfigMaps(namespace).Return(configMapMock)
		clientSetMock := newMockClientSet(t)
		clientSetMock.EXPECT().CoreV1().Return(coreV1Mock)
		operatorConfig := &config.OperatorConfig{Namespace: namespace}

		// when
		configMapInt := NewConfigMapInterface(clientSetMock, operatorConfig)

		// then
		assert.NotNil(t, configMapInt)
	})
}

func TestNewDeploymentInterface(t *testing.T) {
	t.Run("should successfully create deployment interface", func(t *testing.T) {
		// given
		appV1Mock := newMockAppsV1Interface(t)
		deploymentMock := newMockDeploymentInterface(t)
		appV1Mock.EXPECT().Deployments(namespace).Return(deploymentMock)
		clientSetMock := newMockClientSet(t)
		clientSetMock.EXPECT().AppsV1().Return(appV1Mock)
		operatorConfig := &config.OperatorConfig{Namespace: namespace}

		// when
		configMapInt := NewDeploymentInterface(clientSetMock, operatorConfig)

		// then
		assert.NotNil(t, configMapInt)
	})
}

func TestNewDoguInterface(t *testing.T) {
	t.Run("should successfully create dogu interface", func(t *testing.T) {
		// given
		doguClientMock := newMockDoguInterface(t)
		ecosystemClientMock := newMockEcosystemClient(t)
		ecosystemClientMock.EXPECT().Dogus(namespace).Return(doguClientMock)
		operatorConfig := &config.OperatorConfig{Namespace: namespace}

		// when
		interf := NewDoguInterface(ecosystemClientMock, operatorConfig)

		// then
		assert.NotNil(t, interf)
	})
}

func TestNewDoguRestartInterface(t *testing.T) {
	t.Run("should successfully create dogu restart interface", func(t *testing.T) {
		// given
		doguRestartClientMock := newMockDoguRestartInterface(t)
		ecosystemClientMock := newMockEcosystemClient(t)
		ecosystemClientMock.EXPECT().DoguRestarts(namespace).Return(doguRestartClientMock)
		operatorConfig := &config.OperatorConfig{Namespace: namespace}

		// when
		interf := NewDoguRestartInterface(ecosystemClientMock, operatorConfig)

		// then
		assert.NotNil(t, interf)
	})
}

func TestNewEventRecorder(t *testing.T) {
	t.Run("should successfully create dogu restart interface", func(t *testing.T) {
		// given
		mgrMock := newMockK8sManager(t)
		recorderMock := newMockEventRecorder(t)
		mgrMock.EXPECT().GetEventRecorderFor("k8s-dogu-operator").Return(recorderMock)

		// when
		eventRecorder := NewEventRecorder(mgrMock)

		// then
		assert.NotNil(t, eventRecorder)
	})
}

func TestNewK8sClient(t *testing.T) {
	t.Run("should successfully create dogu restart interface", func(t *testing.T) {
		// given
		mgrMock := newMockK8sManager(t)
		mgrMock.EXPECT().GetClient().Return(newMockK8sClient(t))

		// when
		k8sCl := NewK8sClient(mgrMock)

		// then
		assert.NotNil(t, k8sCl)
	})
}

func TestNewPersistentVolumeClaimInterface(t *testing.T) {
	t.Run("should successfully create pvc interface", func(t *testing.T) {
		// given
		coreV1Mock := newMockCoreV1Interface(t)
		pvcMock := newMockPvcInterface(t)
		coreV1Mock.EXPECT().PersistentVolumeClaims(namespace).Return(pvcMock)
		clientSetMock := newMockClientSet(t)
		clientSetMock.EXPECT().CoreV1().Return(coreV1Mock)
		operatorConfig := &config.OperatorConfig{Namespace: namespace}

		// when
		pvcInt := NewPersistentVolumeClaimInterface(clientSetMock, operatorConfig)

		// then
		assert.NotNil(t, pvcInt)
	})
}

func TestNewPodInterface(t *testing.T) {
	t.Run("should successfully create pod interface", func(t *testing.T) {
		// given
		coreV1Mock := newMockCoreV1Interface(t)
		podMock := newMockPodInterface(t)
		coreV1Mock.EXPECT().Pods(namespace).Return(podMock)
		clientSetMock := newMockClientSet(t)
		clientSetMock.EXPECT().CoreV1().Return(coreV1Mock)
		operatorConfig := &config.OperatorConfig{Namespace: namespace}

		// when
		podInt := NewPodInterface(clientSetMock, operatorConfig)

		// then
		assert.NotNil(t, podInt)
	})
}

func TestNewRestClient(t *testing.T) {
	t.Run("should successfully create rest client", func(t *testing.T) {
		// given
		coreV1Mock := newMockCoreV1Interface(t)
		coreV1Mock.EXPECT().RESTClient().Return(newMockRestInterface(t))
		clientSetMock := newMockClientSet(t)
		clientSetMock.EXPECT().CoreV1().Return(coreV1Mock)

		// when
		restClient := NewRestClient(clientSetMock)

		// then
		assert.NotNil(t, restClient)
	})
}

func TestNewScheme(t *testing.T) {
	t.Run("should successfully create scheme", func(t *testing.T) {
		// given
		mgrMock := newMockK8sManager(t)
		mgrMock.EXPECT().GetScheme().Return(&runtime.Scheme{})

		// when
		s := NewScheme(mgrMock)

		// then
		assert.NotNil(t, s)
	})
}

func TestNewSecretInterface(t *testing.T) {
	t.Run("should successfully create pod interface", func(t *testing.T) {
		// given
		coreV1Mock := newMockCoreV1Interface(t)
		secretMock := newMockSecretInterface(t)
		coreV1Mock.EXPECT().Secrets(namespace).Return(secretMock)
		clientSetMock := newMockClientSet(t)
		clientSetMock.EXPECT().CoreV1().Return(coreV1Mock)
		operatorConfig := &config.OperatorConfig{Namespace: namespace}

		// when
		secretInt := NewSecretInterface(clientSetMock, operatorConfig)

		// then
		assert.NotNil(t, secretInt)
	})
}

func TestNewServiceInterface(t *testing.T) {
	t.Run("should successfully create pod interface", func(t *testing.T) {
		// given
		coreV1Mock := newMockCoreV1Interface(t)
		serviceMock := newMockServiceInterface(t)
		coreV1Mock.EXPECT().Services(namespace).Return(serviceMock)
		clientSetMock := newMockClientSet(t)
		clientSetMock.EXPECT().CoreV1().Return(coreV1Mock)
		operatorConfig := &config.OperatorConfig{Namespace: namespace}

		// when
		podInt := NewServiceInterface(clientSetMock, operatorConfig)

		// then
		assert.NotNil(t, podInt)
	})
}

func Test_newEcoSystemClientSet(t *testing.T) {
	t.Run("should successfully create ecosystem client set", func(t *testing.T) {
		// given
		operatorConfig := &rest.Config{}

		// when
		cliSet, err := newEcoSystemClientSet(operatorConfig)

		// then
		assert.NotNil(t, cliSet)
		assert.NoError(t, err)
	})
}

func Test_newKubernetesClientSet(t *testing.T) {
	t.Run("should successfully create kubernetes client set", func(t *testing.T) {
		// given
		operatorConfig := &rest.Config{}

		// when
		cliSet, err := newKubernetesClientSet(operatorConfig)

		// then
		assert.NotNil(t, cliSet)
		assert.NoError(t, err)
	})
}
