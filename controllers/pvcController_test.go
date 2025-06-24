package controllers

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func TestNewPvcReconciler(t *testing.T) {
	t.Run("should not be empty", func(t *testing.T) {
		// given
		clientMock := NewMockK8sClient(t)
		clientSetMock := NewMockClientSet(t)
		ecoSystemClientMock := newMockEcosystemInterface(t)

		// when
		actual := NewPvcReconciler(clientMock, clientSetMock, ecoSystemClientMock)

		// then
		assert.NotEmpty(t, actual)
	})
}

func Test_pvcReconciler_SetupWithManager(t *testing.T) {
	t.Run("should fail", func(t *testing.T) {
		// given
		sut := &PvcReconciler{}

		// when
		err := sut.SetupWithManager(nil)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "must provide a non-nil Manager")
	})
	t.Run("should succeed", func(t *testing.T) {
		// given
		ctrlManMock := newMockControllerManager(t)
		ctrlManMock.EXPECT().GetControllerOptions().Return(config.Controller{})
		ctrlManMock.EXPECT().GetScheme().Return(createPvcScheme(t))
		logger := log.FromContext(testCtx)
		ctrlManMock.EXPECT().GetLogger().Return(logger)
		ctrlManMock.EXPECT().Add(mock.Anything).Return(nil)
		ctrlManMock.EXPECT().GetCache().Return(nil)

		sut := &PvcReconciler{}

		// when
		err := sut.SetupWithManager(ctrlManMock)

		// then
		require.NoError(t, err)
	})
	t.Run("events", func(t *testing.T) {
		// given

		sut := &PvcReconciler{}

		funcs := sut.getEventFilter()

		createEvent := event.TypedCreateEvent[client.Object]{}
		assert.False(t, funcs.Create(createEvent))

		e1 := event.TypedCreateEvent[client.Object]{}
		assert.False(t, funcs.Create(e1))

		e2 := event.TypedDeleteEvent[client.Object]{}
		assert.False(t, funcs.Delete(e2))

		e3 := event.TypedGenericEvent[client.Object]{}
		assert.False(t, funcs.Generic(e3))

		labels := make(map[string]string)
		newDoguPvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Labels: labels},
		}

		e4 := event.TypedUpdateEvent[client.Object]{
			ObjectNew: newDoguPvc,
		}

		assert.False(t, funcs.Update(e4))

		labels["dogu.name"] = "my-dogu"

		requests := make(map[corev1.ResourceName]resource.Quantity)
		requests[corev1.ResourceStorage] = resource.MustParse("1Gi")
		newDoguPvc = &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Labels: labels},
			Spec:       corev1.PersistentVolumeClaimSpec{Resources: corev1.VolumeResourceRequirements{Requests: requests}},
			Status:     corev1.PersistentVolumeClaimStatus{Capacity: requests},
		}

		oldDoguPvc := &corev1.PersistentVolumeClaim{
			Spec:   corev1.PersistentVolumeClaimSpec{Resources: corev1.VolumeResourceRequirements{Requests: requests}},
			Status: corev1.PersistentVolumeClaimStatus{Capacity: requests},
		}

		e4 = event.TypedUpdateEvent[client.Object]{
			ObjectNew: newDoguPvc,
			ObjectOld: oldDoguPvc,
		}
		assert.False(t, funcs.Update(e4))

		oldrequests := make(map[corev1.ResourceName]resource.Quantity)
		oldrequests[corev1.ResourceStorage] = resource.MustParse("2Gi")
		oldDoguPvc = &corev1.PersistentVolumeClaim{
			Spec:   corev1.PersistentVolumeClaimSpec{Resources: corev1.VolumeResourceRequirements{Requests: oldrequests}},
			Status: corev1.PersistentVolumeClaimStatus{Capacity: oldrequests},
		}

		e4 = event.TypedUpdateEvent[client.Object]{
			ObjectNew: newDoguPvc,
			ObjectOld: oldDoguPvc,
		}
		assert.True(t, funcs.Update(e4))

	})

}

func createPvcScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	gv, err := schema.ParseGroupVersion("apps/v2")
	assert.NoError(t, err)

	scheme.AddKnownTypes(gv, &corev1.PersistentVolumeClaim{})
	return scheme
}

func TestPvcReconciler_Reconcile(t *testing.T) {
	t.Run("should fail to get pvc", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-dogu", Namespace: testNamespace}}

		clientMock := NewMockK8sClient(t)
		pvcClientMock := newMockPvcInterface(t)
		pvcClientMock.EXPECT().Get(testCtx, "my-dogu", metav1.GetOptions{}).Return(nil, assert.AnError)
		coreV1Client := newMockCoreV1Interface(t)
		coreV1Client.EXPECT().PersistentVolumeClaims(testNamespace).Return(pvcClientMock)
		clientSetMock := NewMockClientSet(t)
		clientSetMock.EXPECT().CoreV1().Return(coreV1Client)

		sut := &PvcReconciler{
			client:       clientMock,
			k8sClientSet: clientSetMock,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get pvc \"test-namespace/my-dogu\"")
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should fail to get doguname", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-dogu", Namespace: testNamespace}}

		clientMock := NewMockK8sClient(t)
		pvcClientMock := newMockPvcInterface(t)
		dogu := &doguv2.Dogu{}
		doguPvc := &corev1.PersistentVolumeClaim{ObjectMeta: *dogu.GetObjectMeta()}
		pvcClientMock.EXPECT().Get(testCtx, "my-dogu", metav1.GetOptions{}).Return(doguPvc, nil)
		coreV1Client := newMockCoreV1Interface(t)
		coreV1Client.EXPECT().PersistentVolumeClaims(testNamespace).Return(pvcClientMock)
		clientSetMock := NewMockClientSet(t)
		clientSetMock.EXPECT().CoreV1().Return(coreV1Client)

		ecosystemClient := newMockEcosystemInterface(t)
		doguClient := newMockDoguInterface(t)
		ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClient)

		doguClient.EXPECT().Get(testCtx, "", mock.Anything).Return(nil, assert.AnError)

		sut := &PvcReconciler{
			client:             clientMock,
			k8sClientSet:       clientSetMock,
			ecoSystemClientSet: ecosystemClient,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to get dogu \"\"")
		assert.Equal(t, ctrl.Result{}, actual)
	})
	t.Run("should fail to set pvc size", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-dogu", Namespace: testNamespace}}

		clientMock := NewMockK8sClient(t)
		pvcClientMock := newMockPvcInterface(t)
		dogu := &doguv2.Dogu{}
		doguPvc := &corev1.PersistentVolumeClaim{ObjectMeta: *dogu.GetObjectMeta()}
		pvcClientMock.EXPECT().Get(testCtx, "my-dogu", metav1.GetOptions{}).Return(doguPvc, nil)
		coreV1Client := newMockCoreV1Interface(t)
		coreV1Client.EXPECT().PersistentVolumeClaims(testNamespace).Return(pvcClientMock)
		clientSetMock := NewMockClientSet(t)
		clientSetMock.EXPECT().CoreV1().Return(coreV1Client)

		ecosystemClient := newMockEcosystemInterface(t)
		doguClient := newMockDoguInterface(t)
		ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClient)

		doguClient.EXPECT().Get(testCtx, "", mock.Anything).Return(dogu, nil)

		doguClient.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, mock.Anything).Return(nil, assert.AnError)

		sut := &PvcReconciler{
			client:             clientMock,
			k8sClientSet:       clientSetMock,
			ecoSystemClientSet: ecosystemClient,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
		assert.ErrorContains(t, err, "failed to update data size for pvc \"test-namespace/my-dogu\"")
		assert.Equal(t, ctrl.Result{}, actual)
	})

	t.Run("success", func(t *testing.T) {
		// given
		request := ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-dogu", Namespace: testNamespace}}

		clientMock := NewMockK8sClient(t)
		pvcClientMock := newMockPvcInterface(t)
		dogu := &doguv2.Dogu{}
		doguPvc := &corev1.PersistentVolumeClaim{ObjectMeta: *dogu.GetObjectMeta()}
		pvcClientMock.EXPECT().Get(testCtx, "my-dogu", metav1.GetOptions{}).Return(doguPvc, nil)
		coreV1Client := newMockCoreV1Interface(t)
		coreV1Client.EXPECT().PersistentVolumeClaims(testNamespace).Return(pvcClientMock)
		clientSetMock := NewMockClientSet(t)
		clientSetMock.EXPECT().CoreV1().Return(coreV1Client)

		ecosystemClient := newMockEcosystemInterface(t)
		doguClient := newMockDoguInterface(t)
		ecosystemClient.EXPECT().Dogus(testNamespace).Return(doguClient)

		doguClient.EXPECT().Get(testCtx, "", mock.Anything).Return(dogu, nil)

		doguClient.EXPECT().UpdateStatusWithRetry(testCtx, dogu, mock.Anything, mock.Anything).Return(dogu, nil)

		sut := &PvcReconciler{
			client:             clientMock,
			k8sClientSet:       clientSetMock,
			ecoSystemClientSet: ecosystemClient,
		}

		// when
		actual, err := sut.Reconcile(testCtx, request)

		// then
		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, actual)
	})

}
