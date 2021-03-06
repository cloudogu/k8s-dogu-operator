package controllers

import (
	"context"
	cesmocks "github.com/cloudogu/cesapp-lib/registry/mocks"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestDoguManager_Delete(t *testing.T) {
	// given
	inputDogu := &k8sv1.Dogu{}
	inputContext := context.Background()
	deleteManager := &mocks.DeleteManager{}
	deleteManager.On("Delete", inputContext, inputDogu).Return(nil)
	m := DoguManager{deleteManager: deleteManager}

	// when
	err := m.Delete(inputContext, inputDogu)

	// then
	assert.NoError(t, err)
	mock.AssertExpectationsForObjects(t, deleteManager)
}

func TestDoguManager_Install(t *testing.T) {
	// given
	inputDogu := &k8sv1.Dogu{}
	inputContext := context.Background()
	installManager := &mocks.InstallManager{}
	installManager.On("Install", inputContext, inputDogu).Return(nil)
	m := DoguManager{installManager: installManager}

	// when
	err := m.Install(inputContext, inputDogu)

	// then
	assert.NoError(t, err)
	mock.AssertExpectationsForObjects(t, installManager)
}

func TestDoguManager_Upgrade(t *testing.T) {
	// todo change to real test when upgrade is implemented.
	// given
	inputDogu := &k8sv1.Dogu{}
	inputContext := context.Background()
	m := DoguManager{}

	// when
	err := m.Upgrade(inputContext, inputDogu)

	// then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "currently not implemented")
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
		doguManager, err := NewDoguManager(client, operatorConfig, cesRegistry)

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
		globalConfig.On("Exists", "key_provider").Return(false, nil)
		globalConfig.On("Set", "key_provider", "pkcs1v15").Return(nil)
		cesRegistry.On("GlobalConfig").Return(globalConfig)
		cesRegistry.On("DoguRegistry").Return(doguRegistry)

		// when
		doguManager, err := NewDoguManager(client, operatorConfig, cesRegistry)

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
		doguManager, err := NewDoguManager(client, operatorConfig, cesRegistry)

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

		// when
		doguManager, err := NewDoguManager(client, operatorConfig, cesRegistry)

		// then
		require.Error(t, err)
		require.Nil(t, doguManager)
		assert.ErrorIs(t, err, assert.AnError)
		assert.Contains(t, err.Error(), "failed to set default key provider")
		mock.AssertExpectationsForObjects(t, cesRegistry, globalConfig)
	})
}
