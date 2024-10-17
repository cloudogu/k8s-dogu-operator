package health

import (
	"context"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestDoguStatusUpdater_UpdateStatus(t *testing.T) {
	t.Run("should fail to get dogu resource", func(t *testing.T) {
		// given
		doguClientMock := newMockDoguInterface(t)
		doguClientMock.EXPECT().Get(testCtx, "my-dogu", metav1api.GetOptions{}).Return(nil, assert.AnError)
		ecosystemClientMock := newMockEcosystemInterface(t)
		ecosystemClientMock.EXPECT().Dogus(testNamespace).Return(doguClientMock)

		sut := &DoguStatusUpdater{ecosystemClient: ecosystemClientMock}

		// when
		err := sut.UpdateStatus(testCtx, types.NamespacedName{Name: "my-dogu", Namespace: testNamespace}, true)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get dogu resource \"test-namespace/my-dogu\"")
	})
	t.Run("should fail to update health status of dogu", func(t *testing.T) {
		// given
		dogu := &v2.Dogu{ObjectMeta: metav1api.ObjectMeta{Name: "my-dogu", Namespace: testNamespace}}

		doguClientMock := newMockDoguInterface(t)
		doguClientMock.EXPECT().Get(testCtx, "my-dogu", metav1api.GetOptions{}).Return(dogu, nil)
		doguClientMock.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, metav1api.UpdateOptions{}).Return(nil, assert.AnError).
			Run(func(ctx context.Context, dogu *v2.Dogu, modifyStatusFn func(v2.DoguStatus) v2.DoguStatus, opts metav1api.UpdateOptions) {
				status := modifyStatusFn(dogu.Status)
				assert.Equal(t, v2.DoguStatus{Status: "", RequeueTime: 0, RequeuePhase: "", Health: "available", Stopped: false}, status)
			})
		ecosystemClientMock := newMockEcosystemInterface(t)
		ecosystemClientMock.EXPECT().Dogus(testNamespace).Return(doguClientMock)

		recorderMock := newMockEventRecorder(t)
		recorderMock.EXPECT().Event(dogu, "Warning", "HealthStatusUpdate", "failed to update dogu \"test-namespace/my-dogu\" with current health status [\"\"] to desired health status [\"available\"]")

		sut := &DoguStatusUpdater{ecosystemClient: ecosystemClientMock, recorder: recorderMock}

		// when
		err := sut.UpdateStatus(testCtx, dogu.GetObjectKey(), true)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update dogu \"test-namespace/my-dogu\" with current health status [\"\"] to desired health status [\"available\"]")
	})
	t.Run("should succeed to update health status of dogu", func(t *testing.T) {
		t.Run("available", func(t *testing.T) {
			// given
			dogu := &v2.Dogu{ObjectMeta: metav1api.ObjectMeta{Name: "my-dogu", Namespace: testNamespace}}

			doguClientMock := newMockDoguInterface(t)
			doguClientMock.EXPECT().Get(testCtx, "my-dogu", metav1api.GetOptions{}).Return(dogu, nil)
			doguClientMock.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, metav1api.UpdateOptions{}).Return(nil, nil).
				Run(func(ctx context.Context, dogu *v2.Dogu, modifyStatusFn func(v2.DoguStatus) v2.DoguStatus, opts metav1api.UpdateOptions) {
					status := modifyStatusFn(dogu.Status)
					assert.Equal(t, v2.DoguStatus{Status: "", RequeueTime: 0, RequeuePhase: "", Health: "available", Stopped: false}, status)
				})
			ecosystemClientMock := newMockEcosystemInterface(t)
			ecosystemClientMock.EXPECT().Dogus(testNamespace).Return(doguClientMock)

			recorderMock := newMockEventRecorder(t)
			recorderMock.EXPECT().Eventf(dogu, "Normal", "HealthStatusUpdate", "successfully updated health status to %q", v2.AvailableHealthStatus)

			sut := &DoguStatusUpdater{ecosystemClient: ecosystemClientMock, recorder: recorderMock}

			// when
			err := sut.UpdateStatus(testCtx, dogu.GetObjectKey(), true)

			// then
			require.NoError(t, err)
		})
		t.Run("unavailable", func(t *testing.T) {
			// given
			dogu := &v2.Dogu{ObjectMeta: metav1api.ObjectMeta{Name: "my-dogu", Namespace: testNamespace}}

			doguClientMock := newMockDoguInterface(t)
			doguClientMock.EXPECT().Get(testCtx, "my-dogu", metav1api.GetOptions{}).Return(dogu, nil)
			doguClientMock.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, metav1api.UpdateOptions{}).Return(nil, nil).
				Run(func(ctx context.Context, dogu *v2.Dogu, modifyStatusFn func(v2.DoguStatus) v2.DoguStatus, opts metav1api.UpdateOptions) {
					status := modifyStatusFn(dogu.Status)
					assert.Equal(t, v2.DoguStatus{Status: "", RequeueTime: 0, RequeuePhase: "", Health: "unavailable", Stopped: false}, status)
				})
			ecosystemClientMock := newMockEcosystemInterface(t)
			ecosystemClientMock.EXPECT().Dogus(testNamespace).Return(doguClientMock)

			recorderMock := newMockEventRecorder(t)
			recorderMock.EXPECT().Eventf(dogu, "Normal", "HealthStatusUpdate", "successfully updated health status to %q", v2.UnavailableHealthStatus)

			sut := &DoguStatusUpdater{ecosystemClient: ecosystemClientMock, recorder: recorderMock}

			// when
			err := sut.UpdateStatus(testCtx, dogu.GetObjectKey(), false)

			// then
			require.NoError(t, err)
		})
	})
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
