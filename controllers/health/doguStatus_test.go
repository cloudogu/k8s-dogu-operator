package health

import (
	"fmt"
	"testing"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1api "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewDoguStatusUpdater(t *testing.T) {
	// given
	recorderMock := newMockEventRecorder(t)
	configMapInterfaceMock := newMockConfigMapInterface(t)
	podInterfaceMock := newMockPodInterface(t)

	// when
	actual := NewDoguStatusUpdater(recorderMock, configMapInterfaceMock, podInterfaceMock)

	// then
	assert.Same(t, podInterfaceMock, actual.podInterface)
	assert.Same(t, recorderMock, actual.recorder)
	assert.Same(t, configMapInterfaceMock, actual.configMapInterface)
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
		podClientMock := newMockPodInterface(t)
		cmClientMock := newMockConfigMapInterface(t)

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
		sut := &DoguStatusUpdater{podInterface: podClientMock, configMapInterface: cmClientMock}

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

		podClientMock := newMockPodInterface(t)
		cmClientMock := newMockConfigMapInterface(t)

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
		sut := &DoguStatusUpdater{podInterface: podClientMock, configMapInterface: cmClientMock}

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

		podClientMock := newMockPodInterface(t)
		cmClientMock := newMockConfigMapInterface(t)

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
		sut := &DoguStatusUpdater{podInterface: podClientMock, configMapInterface: cmClientMock}

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

		podClientMock := newMockPodInterface(t)
		cmClientMock := newMockConfigMapInterface(t)

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
		sut := &DoguStatusUpdater{podInterface: podClientMock, configMapInterface: cmClientMock}

		// when
		err := sut.UpdateHealthConfigMap(testCtx, deployment, doguJson)

		// then
		require.NoError(t, err)
		assert.Empty(t, testCM.Data["ldap"])
	})
	t.Run("should throw error if not able to get configmap", func(t *testing.T) {
		// given
		cmClientMock := newMockConfigMapInterface(t)

		cmClientMock.EXPECT().Get(testCtx, healthConfigMapName, metav1api.GetOptions{}).Return(nil, assert.AnError)

		sut := &DoguStatusUpdater{configMapInterface: cmClientMock}

		// when
		err := sut.UpdateHealthConfigMap(testCtx, deployment, &core.Dogu{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get health state configMap")
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should throw error if not able to get pod list of deployment", func(t *testing.T) {
		// given
		podClientMock := newMockPodInterface(t)
		cmClientMock := newMockConfigMapInterface(t)

		cmClientMock.EXPECT().Get(testCtx, healthConfigMapName, metav1api.GetOptions{}).Return(&corev1.ConfigMap{}, nil)

		podClientMock.EXPECT().List(testCtx, metav1api.ListOptions{
			LabelSelector: metav1api.FormatLabelSelector(deployment.Spec.Selector),
		}).Return(nil, assert.AnError)

		sut := &DoguStatusUpdater{podInterface: podClientMock, configMapInterface: cmClientMock}

		// when
		err := sut.UpdateHealthConfigMap(testCtx, deployment, &core.Dogu{})

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get all pods for the deployment")
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should throw error if not able to update configmap", func(t *testing.T) {
		// given
		podClientMock := newMockPodInterface(t)
		cmClientMock := newMockConfigMapInterface(t)

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
		sut := &DoguStatusUpdater{podInterface: podClientMock, configMapInterface: cmClientMock}

		// when
		err := sut.UpdateHealthConfigMap(testCtx, deployment, doguJson)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to update health state in health configMap")
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestDoguStatusUpdater_DeleteDoguOutOfHealthConfigMap(t *testing.T) {
	type fields struct {
		recorderFn           func(t *testing.T) record.EventRecorder
		configMapInterfaceFn func(t *testing.T) configMapInterface
		podInterfaceFn       func(t *testing.T) podInterface
	}
	tests := []struct {
		name    string
		fields  fields
		dogu    *v2.Dogu
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should try to delete dogu out of health configmap if it is already deleted",
			fields: fields{
				recorderFn: func(t *testing.T) record.EventRecorder {
					return newMockEventRecorder(t)
				},
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					cm := &corev1.ConfigMap{
						ObjectMeta: metav1api.ObjectMeta{
							Name: healthConfigMapName,
						},
						Data: map[string]string{
							"cas":  "",
							"ldap": "",
						},
					}
					mck.EXPECT().Get(testCtx, healthConfigMapName, metav1api.GetOptions{}).Return(cm, nil)
					mck.EXPECT().Update(testCtx, cm, metav1api.UpdateOptions{}).Return(cm, nil)
					return mck
				},
				podInterfaceFn: func(t *testing.T) podInterface {
					return newMockPodInterface(t)
				},
			},
			dogu:    &v2.Dogu{ObjectMeta: metav1api.ObjectMeta{Name: "test"}},
			wantErr: assert.NoError,
		},
		{
			name: "should delete dogu out of health config map",
			fields: fields{
				recorderFn: func(t *testing.T) record.EventRecorder {
					return newMockEventRecorder(t)
				},
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					getCm := &corev1.ConfigMap{
						ObjectMeta: metav1api.ObjectMeta{
							Name: healthConfigMapName,
						},
						Data: map[string]string{
							"cas":  "",
							"ldap": "",
							"test": "",
						},
					}
					updatedCm := &corev1.ConfigMap{
						ObjectMeta: metav1api.ObjectMeta{
							Name: healthConfigMapName,
						},
						Data: map[string]string{
							"cas":  "",
							"ldap": "",
						},
					}
					mck.EXPECT().Get(testCtx, healthConfigMapName, metav1api.GetOptions{}).Return(getCm, nil)
					mck.EXPECT().Update(testCtx, updatedCm, metav1api.UpdateOptions{}).Return(updatedCm, nil)
					return mck
				},
				podInterfaceFn: func(t *testing.T) podInterface {
					return newMockPodInterface(t)
				},
			},
			dogu:    &v2.Dogu{ObjectMeta: metav1api.ObjectMeta{Name: "test"}},
			wantErr: assert.NoError,
		},
		{
			name: "should fail to update health config map",
			fields: fields{
				recorderFn: func(t *testing.T) record.EventRecorder {
					return newMockEventRecorder(t)
				},
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					getCm := &corev1.ConfigMap{
						ObjectMeta: metav1api.ObjectMeta{
							Name: healthConfigMapName,
						},
						Data: map[string]string{
							"cas":  "",
							"ldap": "",
							"test": "",
						},
					}
					updatedCm := &corev1.ConfigMap{
						ObjectMeta: metav1api.ObjectMeta{
							Name: healthConfigMapName,
						},
						Data: map[string]string{
							"cas":  "",
							"ldap": "",
						},
					}
					mck.EXPECT().Get(testCtx, healthConfigMapName, metav1api.GetOptions{}).Return(getCm, nil)
					mck.EXPECT().Update(testCtx, updatedCm, metav1api.UpdateOptions{}).Return(nil, assert.AnError)
					return mck
				},
				podInterfaceFn: func(t *testing.T) podInterface {
					return newMockPodInterface(t)
				},
			},
			dogu:    &v2.Dogu{ObjectMeta: metav1api.ObjectMeta{Name: "test"}},
			wantErr: assert.Error,
		},
		{
			name: "should fail to create health config map if not exists",
			fields: fields{
				recorderFn: func(t *testing.T) record.EventRecorder {
					return newMockEventRecorder(t)
				},
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					createCm := &corev1.ConfigMap{
						ObjectMeta: metav1api.ObjectMeta{
							Name: healthConfigMapName,
						},
						Data: map[string]string{},
					}
					mck.EXPECT().Get(testCtx, healthConfigMapName, metav1api.GetOptions{}).Return(nil, errors.NewNotFound(schema.GroupResource{}, ""))
					mck.EXPECT().Create(testCtx, createCm, metav1api.CreateOptions{}).Return(createCm, nil)
					return mck
				},
				podInterfaceFn: func(t *testing.T) podInterface {
					return newMockPodInterface(t)
				},
			},
			dogu:    &v2.Dogu{ObjectMeta: metav1api.ObjectMeta{Name: "test"}},
			wantErr: assert.NoError,
		},
		{
			name: "should create health config map if not exists",
			fields: fields{
				recorderFn: func(t *testing.T) record.EventRecorder {
					return newMockEventRecorder(t)
				},
				configMapInterfaceFn: func(t *testing.T) configMapInterface {
					mck := newMockConfigMapInterface(t)
					createCm := &corev1.ConfigMap{
						ObjectMeta: metav1api.ObjectMeta{
							Name: healthConfigMapName,
						},
						Data: map[string]string{},
					}
					mck.EXPECT().Get(testCtx, healthConfigMapName, metav1api.GetOptions{}).Return(nil, errors.NewNotFound(schema.GroupResource{}, ""))
					mck.EXPECT().Create(testCtx, createCm, metav1api.CreateOptions{}).Return(createCm, nil)
					return mck
				},
				podInterfaceFn: func(t *testing.T) podInterface {
					return newMockPodInterface(t)
				},
			},
			dogu:    &v2.Dogu{ObjectMeta: metav1api.ObjectMeta{Name: "test"}},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsw := &DoguStatusUpdater{
				recorder:           tt.fields.recorderFn(t),
				configMapInterface: tt.fields.configMapInterfaceFn(t),
				podInterface:       tt.fields.podInterfaceFn(t),
			}
			tt.wantErr(t, dsw.DeleteDoguOutOfHealthConfigMap(testCtx, tt.dogu), fmt.Sprintf("DeleteDoguOutOfHealthConfigMap(%v, %v)", testCtx, tt.dogu))
		})
	}
}
