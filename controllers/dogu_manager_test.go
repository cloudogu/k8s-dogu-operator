package controllers

import (
	"context"
	"k8s.io/client-go/kubernetes"
	fake2 "k8s.io/client-go/kubernetes/fake"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
)

func TestDoguManager_SetDoguAdditionalIngressAnnotations(t *testing.T) {
	// given
	dogu := &k8sv1.Dogu{}
	recorder := extMocks.NewEventRecorder(t)
	recorder.EXPECT().Event(dogu, "Normal", "AdditionalIngressAnnotationsChange", "Start additional ingress annotations change...")
	annotationsManager := mocks.NewAdditionalIngressAnnotationsManager(t)
	annotationsManager.EXPECT().SetDoguAdditionalIngressAnnotations(mock.Anything, dogu).Return(nil)
	manager := DoguManager{ingressAnnotationsManager: annotationsManager, recorder: recorder}

	// when
	err := manager.SetDoguAdditionalIngressAnnotations(context.TODO(), dogu)

	// then
	require.Nil(t, err)
}

func TestDoguManager_HandleVolumeExpansion(t *testing.T) {
	// given
	dogu := &k8sv1.Dogu{}
	volumeManagerMock := mocks.NewVolumeManager(t)
	eventRecorderMock := extMocks.NewEventRecorder(t)
	manager := DoguManager{volumeManager: volumeManagerMock, recorder: eventRecorderMock}

	eventRecorderMock.EXPECT().Event(dogu, "Normal", "VolumeExpansion", "Start volume expansion...")
	volumeManagerMock.EXPECT().SetDoguDataVolumeSize(mock.Anything, mock.Anything).Return(nil)

	// when
	err := manager.SetDoguDataVolumeSize(context.TODO(), dogu)

	// then
	require.NoError(t, err)
}

func TestDoguManager_HandleSupportMode(t *testing.T) {
	// given
	dogu := &k8sv1.Dogu{}
	supportManagerMock := mocks.NewSupportManager(t)
	eventRecorderMock := extMocks.NewEventRecorder(t)
	manager := DoguManager{supportManager: supportManagerMock, recorder: eventRecorderMock}

	supportManagerMock.EXPECT().HandleSupportMode(mock.Anything, mock.Anything).Return(true, nil)

	// when
	result, err := manager.HandleSupportMode(context.TODO(), dogu)

	// then
	require.NoError(t, err)
	require.True(t, result)
}

func TestDoguManager_Delete(t *testing.T) {
	// given
	inputDogu := &k8sv1.Dogu{}
	inputContext := context.Background()
	deleteManager := mocks.NewDeleteManager(t)
	deleteManager.EXPECT().Delete(inputContext, inputDogu).Return(nil)
	eventRecorder := extMocks.NewEventRecorder(t)
	m := DoguManager{deleteManager: deleteManager, recorder: eventRecorder}

	eventRecorder.EXPECT().Event(inputDogu, corev1.EventTypeNormal, "Deinstallation", "Starting deinstallation...")

	// when
	err := m.Delete(inputContext, inputDogu)

	// then
	assert.NoError(t, err)
}

func TestDoguManager_Install(t *testing.T) {
	// given
	inputDogu := &k8sv1.Dogu{}
	inputContext := context.Background()
	installManager := mocks.NewInstallManager(t)
	installManager.EXPECT().Install(inputContext, inputDogu).Return(nil)
	eventRecorder := extMocks.NewEventRecorder(t)
	m := DoguManager{installManager: installManager, recorder: eventRecorder}

	eventRecorder.EXPECT().Event(inputDogu, corev1.EventTypeNormal, InstallEventReason, "Starting installation...")

	// when
	err := m.Install(inputContext, inputDogu)

	// then
	assert.NoError(t, err)
}

func TestDoguManager_Upgrade(t *testing.T) {
	// given
	inputDogu := &k8sv1.Dogu{}
	inputContext := context.Background()
	upgradeManager := mocks.NewUpgradeManager(t)
	upgradeManager.EXPECT().Upgrade(inputContext, inputDogu).Return(nil)
	eventRecorder := extMocks.NewEventRecorder(t)
	m := DoguManager{upgradeManager: upgradeManager, recorder: eventRecorder}

	eventRecorder.EXPECT().Event(inputDogu, corev1.EventTypeNormal, upgrade.EventReason, "Starting upgrade...")

	// when
	err := m.Upgrade(inputContext, inputDogu)

	// then
	assert.NoError(t, err)
}

func TestNewDoguManager(t *testing.T) {
	// override default controller method to retrieve a kube config
	oldGetConfigDelegate := ctrl.GetConfig
	oldClientSetGetter := clientSetGetter
	defer func() {
		ctrl.GetConfig = oldGetConfigDelegate
		clientSetGetter = oldClientSetGetter
	}()
	ctrl.GetConfig = func() (*rest.Config, error) {
		return &rest.Config{}, nil
	}
	ctrl.GetConfigOrDie = func() *rest.Config {
		getConfig, err := ctrl.GetConfig()
		if err != nil {
			panic(err)
		}
		return getConfig
	}

	t.Run("success", func(t *testing.T) {
		// given
		additionalImages := createConfigMap(
			config.OperatorAdditionalImagesConfigmapName,
			map[string]string{config.ChownInitImageConfigmapNameKey: "image:tag"})
		clientSetGetter = func(c *rest.Config) (kubernetes.Interface, error) {
			return fake2.NewSimpleClientset(additionalImages), nil
		}
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects().Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = testNamespace
		cesRegistry := cesmocks.NewRegistry(t)
		globalConfig := cesmocks.NewConfigurationContext(t)
		doguRegistry := cesmocks.NewDoguRegistry(t)
		eventRecorder := extMocks.NewEventRecorder(t)
		k8sClientSet := extMocks.ClientSet{}
		globalConfig.On("Exists", "key_provider").Return(true, nil)
		cesRegistry.On("GlobalConfig").Return(globalConfig)
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		ecosystemClientSetMock := mocks.NewEcosystemInterface(t)
		ecosystemClientSetMock.EXPECT().Dogus(testNamespace).Return(nil)

		// when
		doguManager, err := NewDoguManager(client, ecosystemClientSetMock, operatorConfig, cesRegistry, eventRecorder, &k8sClientSet)

		// then
		require.NoError(t, err)
		require.NotNil(t, doguManager)
	})

	t.Run("successfully set default key provider", func(t *testing.T) {
		// given
		additionalImages := createConfigMap(
			config.OperatorAdditionalImagesConfigmapName,
			map[string]string{config.ChownInitImageConfigmapNameKey: "image:tag"})
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(additionalImages).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = testNamespace
		cesRegistry := cesmocks.NewRegistry(t)
		globalConfig := cesmocks.NewConfigurationContext(t)
		doguRegistry := cesmocks.NewDoguRegistry(t)
		eventRecorder := extMocks.NewEventRecorder(t)
		k8sClientSet := extMocks.ClientSet{}
		globalConfig.On("Exists", "key_provider").Return(false, nil)
		globalConfig.On("Set", "key_provider", "pkcs1v15").Return(nil)
		cesRegistry.On("GlobalConfig").Return(globalConfig)
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		ecosystemClientSetMock := mocks.NewEcosystemInterface(t)
		ecosystemClientSetMock.EXPECT().Dogus(testNamespace).Return(nil)

		// when
		doguManager, err := NewDoguManager(client, ecosystemClientSetMock, operatorConfig, cesRegistry, eventRecorder, &k8sClientSet)

		// then
		require.NoError(t, err)
		require.NotNil(t, doguManager)
	})

	t.Run("failed to query existing key provider", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		eventRecorder := extMocks.NewEventRecorder(t)
		operatorConfig.Namespace = testNamespace
		cesRegistry := cesmocks.NewRegistry(t)
		globalConfig := cesmocks.NewConfigurationContext(t)
		k8sClientSet := extMocks.ClientSet{}
		globalConfig.On("Exists", "key_provider").Return(true, assert.AnError)
		cesRegistry.On("GlobalConfig").Return(globalConfig)

		// when
		doguManager, err := NewDoguManager(client, nil, operatorConfig, cesRegistry, eventRecorder, &k8sClientSet)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("failed to set default key provider", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = testNamespace
		cesRegistry := cesmocks.NewRegistry(t)
		globalConfig := cesmocks.NewConfigurationContext(t)
		k8sClientSet := extMocks.ClientSet{}
		globalConfig.On("Exists", "key_provider").Return(false, nil)
		globalConfig.On("Set", "key_provider", "pkcs1v15").Return(assert.AnError)
		cesRegistry.On("GlobalConfig").Return(globalConfig)
		eventRecorder := extMocks.NewEventRecorder(t)

		// when
		doguManager, err := NewDoguManager(client, nil, operatorConfig, cesRegistry, eventRecorder, &k8sClientSet)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set default key provider")
	})
}

func createConfigMap(name string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
		},
		Data: data,
	}
}
