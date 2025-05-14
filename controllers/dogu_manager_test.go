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

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/upgrade"
)

func TestDoguManager_SetDoguAdditionalIngressAnnotations(t *testing.T) {
	// given
	dogu := &doguv2.Dogu{}
	recorder := newMockEventRecorder(t)
	recorder.EXPECT().Event(dogu, "Normal", "AdditionalIngressAnnotationsChange", "Start additional ingress annotations change...")
	annotationsManager := newMockAdditionalIngressAnnotationsManager(t)
	annotationsManager.EXPECT().SetDoguAdditionalIngressAnnotations(mock.Anything, dogu).Return(nil)
	manager := DoguManager{ingressAnnotationsManager: annotationsManager, recorder: recorder}

	// when
	err := manager.SetDoguAdditionalIngressAnnotations(context.TODO(), dogu)

	// then
	require.Nil(t, err)
}

func TestDoguManager_HandleVolumeExpansion(t *testing.T) {
	// given
	dogu := &doguv2.Dogu{}
	volumeManagerMock := newMockVolumeManager(t)
	eventRecorderMock := newMockEventRecorder(t)
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
	dogu := &doguv2.Dogu{}
	supportManagerMock := newMockSupportManager(t)
	eventRecorderMock := newMockEventRecorder(t)
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
	inputDogu := &doguv2.Dogu{}
	inputContext := context.Background()
	deleteManager := newMockDeleteManager(t)
	deleteManager.EXPECT().Delete(inputContext, inputDogu).Return(nil)
	eventRecorder := newMockEventRecorder(t)
	m := DoguManager{deleteManager: deleteManager, recorder: eventRecorder}

	eventRecorder.EXPECT().Event(inputDogu, corev1.EventTypeNormal, "Deinstallation", "Starting deinstallation...")

	// when
	err := m.Delete(inputContext, inputDogu)

	// then
	assert.NoError(t, err)
}

func TestDoguManager_Install(t *testing.T) {
	// given
	inputDogu := &doguv2.Dogu{}
	inputContext := context.Background()
	installManager := newMockInstallManager(t)
	installManager.EXPECT().Install(inputContext, inputDogu).Return(nil)
	eventRecorder := newMockEventRecorder(t)
	m := DoguManager{installManager: installManager, recorder: eventRecorder}

	eventRecorder.EXPECT().Event(inputDogu, corev1.EventTypeNormal, InstallEventReason, "Starting installation...")

	// when
	err := m.Install(inputContext, inputDogu)

	// then
	assert.NoError(t, err)
}

func TestDoguManager_Upgrade(t *testing.T) {
	// given
	inputDogu := &doguv2.Dogu{}
	inputContext := context.Background()
	upgradeManager := newMockUpgradeManager(t)
	upgradeManager.EXPECT().Upgrade(inputContext, inputDogu).Return(nil)
	eventRecorder := newMockEventRecorder(t)
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
			map[string]string{config.ChownInitImageConfigmapNameKey: "image:tag", config.ExporterImageConfigmapNameKey: "exporter:0.0.1", config.DataSeederImageConfigmapNameKey: "data-seed:57.0.01"})
		clientSetGetter = func(c *rest.Config) (kubernetes.Interface, error) {
			return fake2.NewSimpleClientset(additionalImages), nil
		}
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects().Build()
		operatorConfig := &config.OperatorConfig{}
		operatorConfig.Namespace = testNamespace

		eventRecorder := newMockEventRecorder(t)

		ecosystemClientSetMock := newMockEcosystemInterface(t)
		ecosystemClientSetMock.EXPECT().Dogus(testNamespace).Return(nil)

		// when
		doguManager, err := NewDoguManager(client, ecosystemClientSetMock, operatorConfig, eventRecorder)

		// then
		require.NoError(t, err)
		require.NotNil(t, doguManager)
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

func TestDoguManager_UpdateExportMode(t *testing.T) {
	t.Run("should call UpdateExportMode on exportManager", func(t *testing.T) {
		doguResource := &doguv2.Dogu{}

		mockEM := newMockExportManager(t)
		mockEM.EXPECT().UpdateExportMode(testCtx, doguResource).Return(nil)

		mockRecorder := newMockEventRecorder(t)
		mockRecorder.EXPECT().Event(doguResource, corev1.EventTypeNormal, ChangeExportModeEventReason, "export-mode changing...")

		sut := &DoguManager{
			exportManager: mockEM,
			recorder:      mockRecorder,
		}

		err := sut.UpdateExportMode(testCtx, doguResource)

		require.NoError(t, err)
	})

	t.Run("should fail calling UpdateExportMode on exportManager", func(t *testing.T) {
		doguResource := &doguv2.Dogu{}

		mockEM := newMockExportManager(t)
		mockEM.EXPECT().UpdateExportMode(testCtx, doguResource).Return(assert.AnError)

		mockRecorder := newMockEventRecorder(t)

		sut := &DoguManager{
			exportManager: mockEM,
			recorder:      mockRecorder,
		}

		err := sut.UpdateExportMode(testCtx, doguResource)

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}
