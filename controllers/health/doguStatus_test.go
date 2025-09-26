package health

import (
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1api "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDoguStatusUpdater(t *testing.T) {
	// given
	ecosystemClientMock := newMockEcosystemInterface(t)
	recorderMock := newMockEventRecorder(t)

	// when
	actual := NewDoguStatusUpdater(ecosystemClientMock, recorderMock, nil)

	// then
	assert.NotEmpty(t, actual)
}

func TestDoguStatusUpdater_UpdateHealthConfigMap(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1api.ObjectMeta{
			Name:      "ldap",
			Namespace: testNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1api.LabelSelector{
				MatchLabels: map[string]string{"test": "halloWelt"},
			},
		},
	}
	testCM := &corev1.ConfigMap{}
	started := true
	podList := &corev1.PodList{
		Items: []corev1.Pod{
			{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{{
						Started: &started,
					}},
				},
			},
		},
	}

	t.Run("should succeed to update health config map", func(t *testing.T) {
		// given
		clientSetMock := newMockClientSet(t)
		coreV1Client := newMockCoreV1Interface(t)
		podClientMock := newMockPodInterface(t)
		cmClientMock := newMockConfigMapInterface(t)

		clientSetMock.EXPECT().CoreV1().Return(coreV1Client)

		coreV1Client.EXPECT().ConfigMaps(testNamespace).Return(cmClientMock)
		coreV1Client.EXPECT().Pods(testNamespace).Return(podClientMock)

		cmClientMock.EXPECT().Get(testCtx, healthConfigMapName, metav1api.GetOptions{}).Return(testCM, nil)
		cmClientMock.EXPECT().Update(testCtx, testCM, metav1api.UpdateOptions{}).Return(testCM, nil)

		podClientMock.EXPECT().List(testCtx, metav1api.ListOptions{
			LabelSelector: metav1api.FormatLabelSelector(deployment.Spec.Selector),
		}).Return(podList, nil)

		doguJson := &core.Dogu{
			HealthChecks: []core.HealthCheck{{
				Type: "state",
			}},
		}
		sut := &DoguStatusUpdater{k8sClientSet: clientSetMock}

		// when
		err := sut.UpdateHealthConfigMap(testCtx, deployment, doguJson)

		// then
		require.NoError(t, err)
		assert.Equal(t, "ready", testCM.Data["ldap"])
	})
	t.Run("should succeed to update health config map with custom state", func(t *testing.T) {
		// given
		testCM.Data = make(map[string]string)
		testCM.Data["ldap"] = "ready"

		clientSetMock := newMockClientSet(t)
		coreV1Client := newMockCoreV1Interface(t)
		podClientMock := newMockPodInterface(t)
		cmClientMock := newMockConfigMapInterface(t)

		clientSetMock.EXPECT().CoreV1().Return(coreV1Client)

		coreV1Client.EXPECT().ConfigMaps(testNamespace).Return(cmClientMock)
		coreV1Client.EXPECT().Pods(testNamespace).Return(podClientMock)

		cmClientMock.EXPECT().Get(testCtx, healthConfigMapName, metav1api.GetOptions{}).Return(testCM, nil)
		cmClientMock.EXPECT().Update(testCtx, testCM, metav1api.UpdateOptions{}).Return(testCM, nil)

		podClientMock.EXPECT().List(testCtx, metav1api.ListOptions{
			LabelSelector: metav1api.FormatLabelSelector(deployment.Spec.Selector),
		}).Return(podList, nil)

		doguJson := &core.Dogu{
			HealthChecks: []core.HealthCheck{{
				Type:  "state",
				State: "customReady123",
			}},
		}
		sut := &DoguStatusUpdater{k8sClientSet: clientSetMock}

		// when
		err := sut.UpdateHealthConfigMap(testCtx, deployment, doguJson)

		// then
		require.NoError(t, err)
		assert.Equal(t, "customReady123", testCM.Data["ldap"])
	})
	t.Run("should remove health state from config map if not started", func(t *testing.T) {
		// given
		testCM.Data = make(map[string]string)
		testCM.Data["ldap"] = "ready"
		started = false
		podList.Items[0].Status.ContainerStatuses[0].Started = &started

		clientSetMock := newMockClientSet(t)
		coreV1Client := newMockCoreV1Interface(t)
		podClientMock := newMockPodInterface(t)
		cmClientMock := newMockConfigMapInterface(t)

		clientSetMock.EXPECT().CoreV1().Return(coreV1Client)

		coreV1Client.EXPECT().ConfigMaps(testNamespace).Return(cmClientMock)
		coreV1Client.EXPECT().Pods(testNamespace).Return(podClientMock)

		cmClientMock.EXPECT().Get(testCtx, healthConfigMapName, metav1api.GetOptions{}).Return(testCM, nil)
		cmClientMock.EXPECT().Update(testCtx, testCM, metav1api.UpdateOptions{}).Return(testCM, nil)

		podClientMock.EXPECT().List(testCtx, metav1api.ListOptions{
			LabelSelector: metav1api.FormatLabelSelector(deployment.Spec.Selector),
		}).Return(podList, nil)

		doguJson := &core.Dogu{
			HealthChecks: []core.HealthCheck{{
				Type: "state",
			}},
		}
		sut := &DoguStatusUpdater{k8sClientSet: clientSetMock}

		// when
		err := sut.UpdateHealthConfigMap(testCtx, deployment, doguJson)

		// then
		require.NoError(t, err)
		assert.Empty(t, testCM.Data["ldap"])
	})
	t.Run("should do remove existing state if no healthcheck of type state", func(t *testing.T) {
		// given
		testCM.Data = make(map[string]string)
		testCM.Data["ldap"] = "ready"

		clientSetMock := newMockClientSet(t)
		coreV1Client := newMockCoreV1Interface(t)
		podClientMock := newMockPodInterface(t)
		cmClientMock := newMockConfigMapInterface(t)

		clientSetMock.EXPECT().CoreV1().Return(coreV1Client)

		coreV1Client.EXPECT().ConfigMaps(testNamespace).Return(cmClientMock)
		coreV1Client.EXPECT().Pods(testNamespace).Return(podClientMock)

		cmClientMock.EXPECT().Get(testCtx, healthConfigMapName, metav1api.GetOptions{}).Return(testCM, nil)
		cmClientMock.EXPECT().Update(testCtx, testCM, metav1api.UpdateOptions{}).Return(testCM, nil)

		podClientMock.EXPECT().List(testCtx, metav1api.ListOptions{
			LabelSelector: metav1api.FormatLabelSelector(deployment.Spec.Selector),
		}).Return(podList, nil)

		doguJson := &core.Dogu{
			HealthChecks: []core.HealthCheck{{
				Type: "tcp",
			}},
		}
		sut := &DoguStatusUpdater{k8sClientSet: clientSetMock}

		// when
		err := sut.UpdateHealthConfigMap(testCtx, deployment, doguJson)

		// then
		require.NoError(t, err)
		assert.Empty(t, testCM.Data["ldap"])
	})
	t.Run("should throw error if not able to get configmap", func(t *testing.T) {
		// given
		clientSetMock := newMockClientSet(t)
		coreV1Client := newMockCoreV1Interface(t)
		cmClientMock := newMockConfigMapInterface(t)

		clientSetMock.EXPECT().CoreV1().Return(coreV1Client)
		coreV1Client.EXPECT().ConfigMaps(testNamespace).Return(cmClientMock)
		cmClientMock.EXPECT().Get(testCtx, healthConfigMapName, metav1api.GetOptions{}).Return(nil, assert.AnError)

		sut := &DoguStatusUpdater{k8sClientSet: clientSetMock}

		// when
		err := sut.UpdateHealthConfigMap(testCtx, deployment, &core.Dogu{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get health state configMap")
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should throw error if not able to get pod list of deployment", func(t *testing.T) {
		// given
		clientSetMock := newMockClientSet(t)
		coreV1Client := newMockCoreV1Interface(t)
		podClientMock := newMockPodInterface(t)
		cmClientMock := newMockConfigMapInterface(t)

		clientSetMock.EXPECT().CoreV1().Return(coreV1Client)

		coreV1Client.EXPECT().ConfigMaps(testNamespace).Return(cmClientMock)
		coreV1Client.EXPECT().Pods(testNamespace).Return(podClientMock)

		cmClientMock.EXPECT().Get(testCtx, healthConfigMapName, metav1api.GetOptions{}).Return(&corev1.ConfigMap{}, nil)

		podClientMock.EXPECT().List(testCtx, metav1api.ListOptions{
			LabelSelector: metav1api.FormatLabelSelector(deployment.Spec.Selector),
		}).Return(nil, assert.AnError)

		sut := &DoguStatusUpdater{k8sClientSet: clientSetMock}

		// when
		err := sut.UpdateHealthConfigMap(testCtx, deployment, &core.Dogu{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get all pods for the deployment")
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should throw error if not able to update configmap", func(t *testing.T) {
		// given
		clientSetMock := newMockClientSet(t)
		coreV1Client := newMockCoreV1Interface(t)
		podClientMock := newMockPodInterface(t)
		cmClientMock := newMockConfigMapInterface(t)

		clientSetMock.EXPECT().CoreV1().Return(coreV1Client)

		coreV1Client.EXPECT().ConfigMaps(testNamespace).Return(cmClientMock)
		coreV1Client.EXPECT().Pods(testNamespace).Return(podClientMock)

		cmClientMock.EXPECT().Get(testCtx, healthConfigMapName, metav1api.GetOptions{}).Return(testCM, nil)
		cmClientMock.EXPECT().Update(testCtx, testCM, metav1api.UpdateOptions{}).Return(nil, assert.AnError)

		podClientMock.EXPECT().List(testCtx, metav1api.ListOptions{
			LabelSelector: metav1api.FormatLabelSelector(deployment.Spec.Selector),
		}).Return(podList, nil)

		doguJson := &core.Dogu{
			HealthChecks: []core.HealthCheck{{
				Type: "state",
			}},
		}
		sut := &DoguStatusUpdater{k8sClientSet: clientSetMock}

		// when
		err := sut.UpdateHealthConfigMap(testCtx, deployment, doguJson)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update health state in health configMap")
		assert.ErrorIs(t, err, assert.AnError)
	})
}
