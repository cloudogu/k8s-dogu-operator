package v2_test

import (
	"context"
	"fmt"
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

	v2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
	extMocks "github.com/cloudogu/k8s-dogu-operator/v2/internal/thirdParty/mocks"
)

var testDogu = &v2.Dogu{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "k8s.cloudogu.com/v2",
		Kind:       "Dogu",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "dogu",
		Namespace: "ecosystem",
	},
	Spec: v2.DoguSpec{
		Name:          "namespace/dogu",
		Version:       "1.2.3-4",
		UpgradeConfig: v2.UpgradeConfig{},
	},
	Status: v2.DoguStatus{Status: ""},
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
			ds := &v2.DoguStatus{
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
		ds := &v2.DoguStatus{
			RequeueTime: time.Hour * 3,
		}

		// when
		ds.ResetRequeueTime()

		// then
		assert.Equal(t, v2.RequeueTimeInitialRequeueTime, ds.RequeueTime)
	})
}

func TestDogu_GetSecretObjectKey(t *testing.T) {
	// given
	ds := &v2.Dogu{
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
		input := v2.CesMatchingLabels{"key": "value"}
		// when
		actual := v2.CesMatchingLabels{}.Add(input)

		// then
		require.NotEmpty(t, actual)
		expected := v2.CesMatchingLabels{"key": "value"}
		assert.Equal(t, expected, actual)
	})
	t.Run("should add to filed object", func(t *testing.T) {
		input := v2.CesMatchingLabels{"key2": "value2"}
		// when
		actual := v2.CesMatchingLabels{"key": "value"}.Add(input)

		// then
		require.NotEmpty(t, actual)
		expected := v2.CesMatchingLabels{"key": "value", "key2": "value2"}
		assert.Equal(t, expected, actual)
	})
}

func TestDogu_Labels(t *testing.T) {
	sut := v2.Dogu{
		ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "ldap"},
		Spec: v2.DoguSpec{
			Name:    "official/ldap",
			Version: "1.2.3-4",
		},
	}

	t.Run("should return pod labels", func(t *testing.T) {
		actual := sut.GetPodLabels()

		expected := v2.CesMatchingLabels{"dogu.name": "ldap", "dogu.version": "1.2.3-4"}
		assert.Equal(t, expected, actual)
	})

	t.Run("should return dogu name label", func(t *testing.T) {
		// when
		actual := sut.GetDoguNameLabel()

		// then
		expected := v2.CesMatchingLabels{"dogu.name": "ldap"}
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
		sut := &v2.DevelopmentDoguMap{
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
		sut := &v2.DevelopmentDoguMap{
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
		Version: "v2",
		Kind:    "dogu",
	}, &v2.Dogu{})
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

func TestDogu_ChangeRequeuePhaseWithRetry(t *testing.T) {
	t.Run("success on conflict", func(t *testing.T) {
		// given
		resourceVersion := "1"
		sut := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "postgresql",
				Namespace:       "ecosystem",
				ResourceVersion: resourceVersion,
			},
		}

		newDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "postgresql",
				Namespace:       "ecosystem",
				ResourceVersion: "2",
			},
			Status: v2.DoguStatus{
				RequeuePhase: "old",
			},
		}

		requeuePhase := "phase"
		fakeClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithStatusSubresource(&v2.Dogu{}).WithObjects(newDogu).Build()

		// when
		err := sut.ChangeRequeuePhaseWithRetry(testCtx, fakeClient, requeuePhase)

		// then
		require.NoError(t, err)
		assert.Equal(t, requeuePhase, sut.Status.RequeuePhase)
		assert.NotEqual(t, resourceVersion, sut.ResourceVersion)
	})

	t.Run("should return error on get error", func(t *testing.T) {
		// given
		resourceVersion := "1"
		sut := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "postgresql",
				Namespace:       "ecosystem",
				ResourceVersion: resourceVersion,
			},
		}

		requeuePhase := "phase"
		fakeClient := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()

		// when
		err := sut.ChangeRequeuePhaseWithRetry(testCtx, fakeClient, requeuePhase)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "dogus.k8s.cloudogu.com \"postgresql\" not found")
	})
}

func TestDogu_ChangeStateWithRetry(t *testing.T) {
	t.Run("success on conflict", func(t *testing.T) {
		// given
		resourceVersion := "1"
		sut := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "postgresql",
				Namespace:       "ecosystem",
				ResourceVersion: resourceVersion,
			},
		}

		newDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "postgresql",
				Namespace:       "ecosystem",
				ResourceVersion: "2",
			},
			Status: v2.DoguStatus{
				Status: "old",
			},
		}

		status := "status"
		fakeClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithStatusSubresource(&v2.Dogu{}).WithObjects(newDogu).Build()

		// when
		err := sut.ChangeStateWithRetry(testCtx, fakeClient, status)

		// then
		require.NoError(t, err)
		assert.Equal(t, status, sut.Status.Status)
		assert.NotEqual(t, resourceVersion, sut.ResourceVersion)
	})

	t.Run("should return error on get error", func(t *testing.T) {
		// given
		resourceVersion := "1"
		sut := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "postgresql",
				Namespace:       "ecosystem",
				ResourceVersion: resourceVersion,
			},
		}

		status := "status"
		fakeClient := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()

		// when
		err := sut.ChangeStateWithRetry(testCtx, fakeClient, status)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "dogus.k8s.cloudogu.com \"postgresql\" not found")
	})
}

func TestDogu_NextRequeueWithRetry(t *testing.T) {
	t.Run("success on conflict; requeue time was reset", func(t *testing.T) {
		// given
		sut := v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "postgresql",
				Namespace:       "ecosystem",
				ResourceVersion: "1",
			},
			Status: v2.DoguStatus{
				RequeueTime: time.Second * 40,
			},
		}

		newDogu := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name:            "postgresql",
				Namespace:       "ecosystem",
				ResourceVersion: "2",
			},
			Status: v2.DoguStatus{
				RequeueTime: 0,
			},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(newDogu).WithStatusSubresource(&v2.Dogu{}).Build()

		// when
		retry, err := sut.NextRequeueWithRetry(testCtx, fakeClient)

		// then
		require.NoError(t, err)
		assert.Equal(t, time.Second*10, retry)
	})

	t.Run("should return error on get error", func(t *testing.T) {
		// given
		sut := &v2.Dogu{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "postgresql",
				Namespace: "ecosystem",
			},
		}

		fakeClient := fake.NewClientBuilder().WithScheme(getTestScheme()).Build()

		// when
		_, err := sut.NextRequeueWithRetry(testCtx, fakeClient)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "dogus.k8s.cloudogu.com \"postgresql\" not found")
	})
}
