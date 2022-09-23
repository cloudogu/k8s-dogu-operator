package controllers

import (
	"context"
	"testing"

	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDoguManager_Delete(t *testing.T) {
	// given
	inputDogu := &k8sv1.Dogu{}
	inputContext := context.Background()
	deleteManager := mocks.NewDeleteManager(t)
	deleteManager.On("Delete", inputContext, inputDogu).Return(nil)
	eventRecorder := &mocks.EventRecorder{}
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
	eventRecorder := &mocks.EventRecorder{}
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
	eventRecorder := &mocks.EventRecorder{}
	m := DoguManager{upgradeManager: upgradeManager, recorder: eventRecorder}

	eventRecorder.On("Event", inputDogu, corev1.EventTypeNormal, UpgradeEventReason, "Starting upgrade...")

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
		cesRegistry := &cesmocks.Registry{}
		globalConfig := &cesmocks.ConfigurationContext{}
		doguRegistry := &cesmocks.DoguRegistry{}
		eventRecorder := &mocks.EventRecorder{}
		globalConfig.On("Exists", "key_provider").Return(true, nil)
		cesRegistry.On("GlobalConfig").Return(globalConfig)
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		// when
		doguManager, err := NewDoguManager(client, operatorConfig, cesRegistry, eventRecorder)

		// then
		require.NoError(t, err)
		require.NotNil(t, doguManager)
		mock.AssertExpectationsForObjects(t, cesRegistry, globalConfig)
	})

	t.Run("successfully set default key provider", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = "test"
		cesRegistry := &cesmocks.Registry{}
		globalConfig := &cesmocks.ConfigurationContext{}
		doguRegistry := &cesmocks.DoguRegistry{}
		eventRecorder := &mocks.EventRecorder{}
		globalConfig.On("Exists", "key_provider").Return(false, nil)
		globalConfig.On("Set", "key_provider", "pkcs1v15").Return(nil)
		cesRegistry.On("GlobalConfig").Return(globalConfig)
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		// when
		doguManager, err := NewDoguManager(client, operatorConfig, cesRegistry, eventRecorder)

		// then
		require.NoError(t, err)
		require.NotNil(t, doguManager)
		mock.AssertExpectationsForObjects(t, cesRegistry, globalConfig)
	})

	t.Run("failed to query existing key provider", func(t *testing.T) {
		// given
		client := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		operatorConfig := &config.OperatorConfig{}
		eventRecorder := &mocks.EventRecorder{}
		operatorConfig.Namespace = "test"
		cesRegistry := &cesmocks.Registry{}
		globalConfig := &cesmocks.ConfigurationContext{}
		globalConfig.On("Exists", "key_provider").Return(true, assert.AnError)
		cesRegistry.On("GlobalConfig").Return(globalConfig)

		// when
		doguManager, err := NewDoguManager(client, operatorConfig, cesRegistry, eventRecorder)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
		assert.ErrorIs(t, err, assert.AnError)
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
		eventRecorder := &mocks.EventRecorder{}

		// when
		doguManager, err := NewDoguManager(client, operatorConfig, cesRegistry, eventRecorder)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to set default key provider")
		mock.AssertExpectationsForObjects(t, cesRegistry, globalConfig)
	})
}
