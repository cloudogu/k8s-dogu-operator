package controllers

import (
	"context"
	"testing"

	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu/mocks"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDoguManager_HandleVolumeExpansion(t *testing.T) {
	// given
	dogu := &k8sv1.Dogu{}
	volumeManagerMock := mocks.NewVolumeManager(t)
	eventRecorderMock := extMocks.NewEventRecorder(t)
	manager := DoguManager{volumeManager: volumeManagerMock, recorder: eventRecorderMock}

	eventRecorderMock.On("Event", dogu, "Normal", "VolumeExpansion", "Start volume expansion...")
	volumeManagerMock.On("SetDoguDataVolumeSize", mock.Anything, mock.Anything).Return(nil)

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

	supportManagerMock.On("HandleSupportMode", mock.Anything, mock.Anything).Return(true, nil)

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
	deleteManager.On("Delete", inputContext, inputDogu).Return(nil)
	eventRecorder := extMocks.NewEventRecorder(t)
	m := DoguManager{deleteManager: deleteManager, recorder: eventRecorder}

	eventRecorder.On("Event", inputDogu, corev1.EventTypeNormal, "Deinstallation", "Starting deinstallation...")

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
	installManager.On("Install", inputContext, inputDogu).Return(nil)
	eventRecorder := extMocks.NewEventRecorder(t)
	m := DoguManager{installManager: installManager, recorder: eventRecorder}

	eventRecorder.On("Event", inputDogu, corev1.EventTypeNormal, InstallEventReason, "Starting installation...")

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
	upgradeManager.On("Upgrade", inputContext, inputDogu).Return(nil)
	eventRecorder := extMocks.NewEventRecorder(t)
	m := DoguManager{upgradeManager: upgradeManager, recorder: eventRecorder}

	eventRecorder.On("Event", inputDogu, corev1.EventTypeNormal, upgrade.EventReason, "Starting upgrade...")

	// when
	err := m.Upgrade(inputContext, inputDogu)

	// then
	assert.NoError(t, err)
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
		cesRegistry := cesmocks.NewRegistry(t)
		globalConfig := cesmocks.NewConfigurationContext(t)
		doguRegistry := cesmocks.NewDoguRegistry(t)
		eventRecorder := extMocks.NewEventRecorder(t)
		globalConfig.On("Exists", "key_provider").Return(true, nil)
		cesRegistry.On("GlobalConfig").Return(globalConfig)
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		// when
		doguManager, err := NewDoguManager(client, operatorConfig, cesRegistry, eventRecorder)

		// then
		require.NoError(t, err)
		require.NotNil(t, doguManager)
	})

	t.Run("successfully set default key provider", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := cesmocks.NewRegistry(t)
		globalConfig := cesmocks.NewConfigurationContext(t)
		doguRegistry := cesmocks.NewDoguRegistry(t)
		eventRecorder := extMocks.NewEventRecorder(t)
		globalConfig.On("Exists", "key_provider").Return(false, nil)
		globalConfig.On("Set", "key_provider", "pkcs1v15").Return(nil)
		cesRegistry.On("GlobalConfig").Return(globalConfig)
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		// when
		doguManager, err := NewDoguManager(client, operatorConfig, cesRegistry, eventRecorder)

		// then
		require.NoError(t, err)
		require.NotNil(t, doguManager)
	})

	t.Run("failed to query existing key provider", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		eventRecorder := extMocks.NewEventRecorder(t)
		operatorConfig.Namespace = "test"
		cesRegistry := cesmocks.NewRegistry(t)
		globalConfig := cesmocks.NewConfigurationContext(t)
		globalConfig.On("Exists", "key_provider").Return(true, assert.AnError)
		cesRegistry.On("GlobalConfig").Return(globalConfig)

		// when
		doguManager, err := NewDoguManager(client, operatorConfig, cesRegistry, eventRecorder)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("failed to set default key provider", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := cesmocks.NewRegistry(t)
		globalConfig := cesmocks.NewConfigurationContext(t)
		globalConfig.On("Exists", "key_provider").Return(false, nil)
		globalConfig.On("Set", "key_provider", "pkcs1v15").Return(assert.AnError)
		cesRegistry.On("GlobalConfig").Return(globalConfig)
		eventRecorder := extMocks.NewEventRecorder(t)

		// when
		doguManager, err := NewDoguManager(client, operatorConfig, cesRegistry, eventRecorder)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to set default key provider")
	})
}
