package v1_test

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/mock"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	eventV1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	extMocks "github.com/cloudogu/k8s-dogu-operator/internal/thirdParty/mocks"
)

var testDogu = &v1.Dogu{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "k8s.cloudogu.com/v1",
		Kind:       "Dogu",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "dogu",
		Namespace: "ecosystem",
	},
	Spec: v1.DoguSpec{
		Name:          "namespace/dogu",
		Version:       "1.2.3-4",
		UpgradeConfig: v1.UpgradeConfig{},
	},
	Status: v1.DoguStatus{Status: ""},
}
var testCtx = context.Background()

func TestDoguStatus_GetRequeueTime(t *testing.T) {
	tests := []struct {
		requeueCount        time.Duration
		expectedRequeueTime time.Duration
	}{
		{requeueCount: time.Second, expectedRequeueTime: time.Second * 2},
		{requeueCount: time.Second * 17, expectedRequeueTime: time.Second * 34},
		{requeueCount: time.Minute, expectedRequeueTime: time.Minute * 2},
		{requeueCount: time.Minute * 7, expectedRequeueTime: time.Minute * 14},
		{requeueCount: time.Minute * 45, expectedRequeueTime: time.Hour*1 + time.Minute*30},
		{requeueCount: time.Hour * 2, expectedRequeueTime: time.Hour * 4},
		{requeueCount: time.Hour * 3, expectedRequeueTime: time.Hour * 6},
		{requeueCount: time.Hour * 5, expectedRequeueTime: time.Hour * 6},
		{requeueCount: time.Hour * 100, expectedRequeueTime: time.Hour * 6},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("calculate next requeue time for current time %s", tt.requeueCount), func(t *testing.T) {
			// given
			ds := &v1.DoguStatus{
				RequeueTime: tt.requeueCount,
			}

			// when
			actualRequeueTime := ds.NextRequeue()

			// then
			assert.Equal(t, tt.expectedRequeueTime, actualRequeueTime)
		})
	}
}

func TestDoguStatus_ResetRequeueTime(t *testing.T) {
	t.Run("reset requeue time", func(t *testing.T) {
		// given
		ds := &v1.DoguStatus{
			RequeueTime: time.Hour * 3,
		}

		// when
		ds.ResetRequeueTime()

		// then
		assert.Equal(t, v1.RequeueTimeInitialRequeueTime, ds.RequeueTime)
	})
}

func TestDogu_GetSecretObjectKey(t *testing.T) {
	// given
	ds := &v1.Dogu{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myspecialdogu",
			Namespace: "testnamespace",
		},
	}

	// when
	key := ds.GetSecretObjectKey()

	// then
	assert.Equal(t, "myspecialdogu-secrets", key.Name)
	assert.Equal(t, "testnamespace", key.Namespace)
}

func Test_Dogu_ChangeState(t *testing.T) {
	ctx := context.TODO()

	t.Run("should set the dogu resource's status to upgrade", func(t *testing.T) {
		sut := &v1.Dogu{}
		mockClient := extMocks.NewK8sClient(t)
		statusMock := extMocks.NewK8sSubResourceWriter(t)
		mockClient.EXPECT().Get(ctx, sut.GetObjectKey(), sut).Return(nil)
		mockClient.EXPECT().Status().Return(statusMock)
		statusMock.On("Update", ctx, sut).Return(nil)

		// when
		err := sut.ChangeState(ctx, mockClient, v1.DoguStatusUpgrading)

		// then
		require.NoError(t, err)
		assert.Equal(t, v1.DoguStatusUpgrading, sut.Status.Status)
	})
	t.Run("should fail on get error", func(t *testing.T) {
		sut := &v1.Dogu{}
		mockClient := extMocks.NewK8sClient(t)
		mockClient.EXPECT().Get(ctx, sut.GetObjectKey(), sut).Return(assert.AnError)

		// when
		err := sut.ChangeState(ctx, mockClient, v1.DoguStatusUpgrading)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should fail on client error", func(t *testing.T) {
		sut := &v1.Dogu{}
		mockClient := extMocks.NewK8sClient(t)
		statusMock := extMocks.NewK8sSubResourceWriter(t)
		mockClient.EXPECT().Get(ctx, sut.GetObjectKey(), sut).Return(nil)
		mockClient.EXPECT().Status().Return(statusMock)
		statusMock.On("Update", ctx, sut).Return(assert.AnError)

		// when
		err := sut.ChangeState(ctx, mockClient, v1.DoguStatusUpgrading)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
}

func Test_Dogu_UpdateInstalledVersion(t *testing.T) {
	ctx := context.TODO()

	t.Run("should set the dogu resource's installed Version", func(t *testing.T) {
		sut := &v1.Dogu{Spec: v1.DoguSpec{Version: "0.2.1"}}
		mockClient := extMocks.NewK8sClient(t)
		statusMock := extMocks.NewK8sSubResourceWriter(t)
		emptyDogu := &v1.Dogu{}
		mockClient.EXPECT().Get(ctx, sut.GetObjectKey(), emptyDogu).RunAndReturn(func(ctx context.Context, name types.NamespacedName, object client.Object, option ...client.GetOption) error {
			doguPtr := object.(*v1.Dogu)
			*doguPtr = v1.Dogu{Spec: v1.DoguSpec{Version: "0.2.1"}}
			return nil
		})
		mockClient.EXPECT().Status().Return(statusMock)
		statusMock.On("Update", ctx, sut).Return(nil)

		// when
		err := sut.UpdateInstalledVersion(ctx, mockClient)

		// then
		require.NoError(t, err)
		assert.Equal(t, "0.2.1", sut.Status.InstalledVersion)
	})
	t.Run("should fail on get error", func(t *testing.T) {
		sut := &v1.Dogu{}
		mockClient := extMocks.NewK8sClient(t)
		mockClient.EXPECT().Get(ctx, sut.GetObjectKey(), sut).Return(assert.AnError)

		// when
		err := sut.UpdateInstalledVersion(ctx, mockClient)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should fail on client error", func(t *testing.T) {
		sut := &v1.Dogu{}
		mockClient := extMocks.NewK8sClient(t)
		statusMock := extMocks.NewK8sSubResourceWriter(t)
		mockClient.EXPECT().Get(ctx, sut.GetObjectKey(), sut).Return(nil)
		mockClient.EXPECT().Status().Return(statusMock)
		statusMock.On("Update", ctx, sut).Return(assert.AnError)

		// when
		err := sut.UpdateInstalledVersion(ctx, mockClient)

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
}

func Test_Dogu_UpdateStatusWithRetry(t *testing.T) {
	ctx := context.TODO()

	t.Run("should set the dogu resource's status to upgrade", func(t *testing.T) {
		sut := &v1.Dogu{}
		mockClient := extMocks.NewK8sClient(t)
		statusMock := extMocks.NewK8sSubResourceWriter(t)
		mockClient.EXPECT().Get(ctx, sut.GetObjectKey(), sut).Return(nil)
		mockClient.EXPECT().Status().Return(statusMock)
		statusMock.On("Update", ctx, sut).Return(nil)

		// when
		err := sut.UpdateStatusWithRetry(ctx, mockClient, func(d *v1.Dogu) { d.Status.Status = v1.DoguStatusInstalled })

		// then
		require.NoError(t, err)
		assert.Equal(t, v1.DoguStatusInstalled, sut.Status.Status)
	})
	t.Run("should fail on get error", func(t *testing.T) {
		sut := &v1.Dogu{}
		mockClient := extMocks.NewK8sClient(t)
		mockClient.EXPECT().Get(ctx, sut.GetObjectKey(), sut).Return(assert.AnError)

		// when
		err := sut.UpdateStatusWithRetry(ctx, mockClient, func(d *v1.Dogu) { d.Status.Status = v1.DoguStatusInstalled })

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should fail on client error", func(t *testing.T) {
		sut := &v1.Dogu{}
		mockClient := extMocks.NewK8sClient(t)
		statusMock := extMocks.NewK8sSubResourceWriter(t)
		mockClient.EXPECT().Get(ctx, sut.GetObjectKey(), sut).Return(nil)
		mockClient.EXPECT().Status().Return(statusMock)
		statusMock.On("Update", ctx, sut).Return(assert.AnError)

		// when
		err := sut.UpdateStatusWithRetry(ctx, mockClient, func(d *v1.Dogu) { d.Status.Status = v1.DoguStatusInstalled })

		// then
		require.ErrorIs(t, err, assert.AnError)
	})
	t.Run("should retry on client conflict", func(t *testing.T) {
		sut := &v1.Dogu{}
		mockClient := extMocks.NewK8sClient(t)
		statusMock := extMocks.NewK8sSubResourceWriter(t)
		mockClient.EXPECT().Get(ctx, sut.GetObjectKey(), mock.AnythingOfType("*v1.Dogu")).Return(nil).Twice()
		mockClient.EXPECT().Status().Return(statusMock).Twice()
		statusError := errors.NewConflict(schema.GroupResource{}, "test", fmt.Errorf("Test Error"))
		statusMock.On("Update", ctx, sut).Return(statusError).Once()
		statusMock.On("Update", ctx, sut).Return(nil).Once()

		// when
		err := sut.UpdateStatusWithRetry(ctx, mockClient, func(d *v1.Dogu) { d.Status.Status = v1.DoguStatusInstalled })

		// then
		require.NoError(t, err)
		assert.Equal(t, v1.DoguStatusInstalled, sut.Status.Status)
	})
}

func TestDogu_GetObjectKey(t *testing.T) {
	actual := testDogu.GetObjectKey()

	expectedObjKey := client.ObjectKey{
		Namespace: "ecosystem",
		Name:      "dogu",
	}
	assert.Equal(t, expectedObjKey, actual)
}

func TestDogu_GetObjectMeta(t *testing.T) {
	actual := testDogu.GetObjectMeta()

	expectedObjKey := &metav1.ObjectMeta{
		Namespace: "ecosystem",
		Name:      "dogu",
	}
	assert.Equal(t, expectedObjKey, actual)
}

func TestDogu_GetDataVolumeName(t *testing.T) {
	actual := testDogu.GetDataVolumeName()

	assert.Equal(t, "dogu-data", actual)
}

func TestDogu_GetPrivateVolumeName(t *testing.T) {
	actual := testDogu.GetPrivateKeySecretName()

	assert.Equal(t, "dogu-private", actual)
}

func TestDogu_GetDevelopmentDoguMapKey(t *testing.T) {
	actual := testDogu.GetDevelopmentDoguMapKey()

	expectedKey := client.ObjectKey{
		Namespace: "ecosystem",
		Name:      "dogu-descriptor",
	}
	assert.Equal(t, expectedKey, actual)
}

func TestDogu_GetPrivateKeyObjectKey(t *testing.T) {
	actual := testDogu.GetPrivateKeyObjectKey()

	expectedKey := client.ObjectKey{
		Namespace: "ecosystem",
		Name:      "dogu-private",
	}
	assert.Equal(t, expectedKey, actual)
}

func TestCesMatchingLabels_Add(t *testing.T) {
	t.Run("should add to empty object", func(t *testing.T) {
		input := v1.CesMatchingLabels{"key": "value"}
		// when
		actual := v1.CesMatchingLabels{}.Add(input)

		// then
		require.NotEmpty(t, actual)
		expected := v1.CesMatchingLabels{"key": "value"}
		assert.Equal(t, expected, actual)
	})
	t.Run("should add to filed object", func(t *testing.T) {
		input := v1.CesMatchingLabels{"key2": "value2"}
		// when
		actual := v1.CesMatchingLabels{"key": "value"}.Add(input)

		// then
		require.NotEmpty(t, actual)
		expected := v1.CesMatchingLabels{"key": "value", "key2": "value2"}
		assert.Equal(t, expected, actual)
	})
}

func TestDogu_Labels(t *testing.T) {
	sut := v1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "ldap"},
		Spec: v1.DoguSpec{
			Name:    "official/ldap",
			Version: "1.2.3-4",
		},
	}

	t.Run("should return pod labels", func(t *testing.T) {
		actual := sut.GetPodLabels()

		expected := v1.CesMatchingLabels{"dogu.name": "ldap", "dogu.version": "1.2.3-4"}
		assert.Equal(t, expected, actual)
	})

	t.Run("should return dogu name label", func(t *testing.T) {
		// when
		actual := sut.GetDoguNameLabel()

		// then
		expected := v1.CesMatchingLabels{"dogu.name": "ldap"}
		assert.Equal(t, expected, actual)
	})
}

func TestDogu_GetPod(t *testing.T) {
	readyPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap-x2y3z45", Labels: testDogu.GetPodLabels()},
		Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}}},
	}
	cli := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(readyPod).Build()

	// when
	actual, err := testDogu.GetPod(testCtx, cli)

	// then
	require.NoError(t, err)
	exptectedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "ldap-x2y3z45", Labels: testDogu.GetPodLabels()},
		Status:     corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}}},
	}
	// ignore ResourceVersion which is introduced during getting pods from the K8s API
	actual.ResourceVersion = ""
	assert.Equal(t, exptectedPod, actual)
}

func TestDevelopmentDoguMap_DeleteFromCluster(t *testing.T) {
	t.Run("should delete a DevelopmentDogu cm", func(t *testing.T) {
		inputCm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "ldap-dev-dev-map"},
			Data:       map[string]string{"key": "le data"},
		}
		mockClient := extMocks.NewK8sClient(t)
		mockClient.EXPECT().Delete(testCtx, inputCm).Return(nil)
		sut := &v1.DevelopmentDoguMap{
			ObjectMeta: metav1.ObjectMeta{Name: "ldap-dev-dev-map"},
			Data:       map[string]string{"key": "le data"},
		}

		// when
		err := sut.DeleteFromCluster(testCtx, mockClient)

		// then
		require.NoError(t, err)
	})
	t.Run("should return an error", func(t *testing.T) {
		inputCm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "ldap-dev-dev-map"},
			Data:       map[string]string{"key": "le data"},
		}
		mockClient := extMocks.NewK8sClient(t)
		mockClient.EXPECT().Delete(testCtx, inputCm).Return(assert.AnError)
		sut := &v1.DevelopmentDoguMap{
			ObjectMeta: metav1.ObjectMeta{Name: "ldap-dev-dev-map"},
			Data:       map[string]string{"key": "le data"},
		}

		// when
		err := sut.DeleteFromCluster(testCtx, mockClient)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func getTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()

	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "k8s.cloudogu.com",
		Version: "v1",
		Kind:    "dogu",
	}, &v1.Dogu{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}, &appsv1.Deployment{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Secret",
	}, &corev1.Secret{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Service",
	}, &corev1.Service{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "PersistentVolumeClaim",
	}, &corev1.PersistentVolumeClaim{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	}, &corev1.ConfigMap{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Event",
	}, &eventV1.Event{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Pod",
	}, &corev1.Pod{})
	scheme.AddKnownTypeWithName(schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "PodList",
	}, &corev1.PodList{})

	return scheme
}

func TestDogu_GetPrivateKeySecret(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		expected := &corev1.Secret{
			TypeMeta:   metav1.TypeMeta{Kind: "Secret", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{Name: "dogu-private", Namespace: "ecosystem"},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(expected).Build()

		// when
		secret, err := testDogu.GetPrivateKeySecret(context.TODO(), fakeClient)

		// then
		require.NoError(t, err)
		assert.Equal(t, expected, secret)
	})

	t.Run("fail to get private key secret", func(t *testing.T) {
		// given
		fakeClient := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()

		// when
		_, err := testDogu.GetPrivateKeySecret(context.TODO(), fakeClient)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get private key secret for dogu")
	})
}

func Test_Dogu_SelectHealthStatus(t *testing.T) {
	t.Run("should select available if isAvailable", func(t *testing.T) {
		// when
		healthStatus := v1.SelectHealthStatus(true)

		// then
		assert.Equal(t, v1.AvailableHealthStatus, healthStatus)
	})
	t.Run("should select unavailable if not isAvailable", func(t *testing.T) {
		// when
		healthStatus := v1.SelectHealthStatus(false)

		// then
		assert.Equal(t, v1.UnavailableHealthStatus, healthStatus)
	})
}
