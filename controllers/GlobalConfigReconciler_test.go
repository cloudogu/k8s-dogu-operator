package controllers

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/config"
)

func TestNewGlobalConfigReconciler(t *testing.T) {
	// given
	restartManagerMock := newMockDoguRestartManager(t)
	configMapMock := newMockConfigMapInterface(t)
	doguInterfaceMock := newMockDoguInterface(t)
	podInterfaceMock := newMockPodInterface(t)
	client := fake.NewClientBuilder().Build()
	managerMock := newMockCtrlManager(t)
	managerMock.EXPECT().GetControllerOptions().Return(config.Controller{})
	managerMock.EXPECT().GetScheme().Return(getTestScheme())
	managerMock.EXPECT().GetLogger().Return(logr.Logger{})
	managerMock.EXPECT().Add(mock.Anything).Return(nil)
	managerMock.EXPECT().GetCache().Return(nil)

	// when
	reconciler, err := NewGlobalConfigReconciler(restartManagerMock, configMapMock, doguInterfaceMock, podInterfaceMock, client, nil, managerMock)

	// then
	assert.NoError(t, err)
	assert.NotEmpty(t, reconciler)
}
