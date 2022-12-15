package controllers

import (
	"context"
	"github.com/cloudogu/k8s-dogu-operator/internal/mocks/external"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestNewDoguVolumeManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		recorder := external.NewEventRecorder(t)
		client := external.NewClient(t)

		// when
		result := NewDoguVolumeManager(client, recorder)

		// then
		require.NotNil(t, result)
	})
}

func Test_doguVolumeManager_SetDoguDataVolumeSize(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		oldSize := dogu.Spec.Resources.DataVolumeSize
		dogu.Spec.Resources.DataVolumeSize = "2Gi"
		defer func() {
			dogu.Spec.Resources.DataVolumeSize = oldSize
		}()

		replicas := int32(1)
		deploy := &appsv1.Deployment{ObjectMeta: *dogu.GetObjectMeta(), Spec: appsv1.DeploymentSpec{Replicas: &replicas}}

		requests := map[corev1.ResourceName]resource.Quantity{}
		requests[corev1.ResourceStorage] = resource.MustParse("1Gi")
		pvc := &corev1.PersistentVolumeClaim{ObjectMeta: *dogu.GetObjectMeta(),
			Spec: corev1.PersistentVolumeClaimSpec{Resources: corev1.ResourceRequirements{Requests: requests}}}

		client := fake.NewClientBuilder().WithObjects(deploy, pvc).Build()
		recorder := external.NewEventRecorder(t)
		recorder.On("Eventf", dogu, "Normal", "VolumeExpansion", "Scale deployment to %d replicas...", int32(0))
		recorder.On("Eventf", dogu, "Normal", "VolumeExpansion", "Scale deployment to %d replicas...", int32(1))
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Update dogu data PVC request storage...")
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Wait for pvc to be resized...")

		pvc, err := dogu.GetDataPVC(context.TODO(), client)
		require.NoError(t, err)
		wantRequests := map[corev1.ResourceName]resource.Quantity{}
		wantRequests[corev1.ResourceStorage] = resource.MustParse("2Gi")
		pvc.Status.Capacity = wantRequests
		err = client.Status().Update(context.TODO(), pvc)
		require.NoError(t, err)
		sut := &doguVolumeManager{client: client, eventRecorder: recorder}

		// when
		err = sut.SetDoguDataVolumeSize(context.TODO(), dogu)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to parse quantity", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		oldSize := dogu.Spec.Resources.DataVolumeSize
		dogu.Spec.Resources.DataVolumeSize = "2wrong"
		defer func() {
			dogu.Spec.Resources.DataVolumeSize = oldSize
		}()
		sut := &doguVolumeManager{}

		// when
		err := sut.SetDoguDataVolumeSize(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse 2wrong to quantity")
	})

	t.Run("fail to update pvc", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		oldSize := dogu.Spec.Resources.DataVolumeSize
		dogu.Spec.Resources.DataVolumeSize = "2Gi"
		defer func() {
			dogu.Spec.Resources.DataVolumeSize = oldSize
		}()

		client := fake.NewClientBuilder().Build()
		recorder := external.NewEventRecorder(t)
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Update dogu data PVC request storage...")

		sut := &doguVolumeManager{client: client, eventRecorder: recorder}

		// when
		err := sut.SetDoguDataVolumeSize(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get data pvc for dogu ldap")
	})

	t.Run("fail to scale down deployment", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		oldSize := dogu.Spec.Resources.DataVolumeSize
		dogu.Spec.Resources.DataVolumeSize = "2Gi"
		defer func() {
			dogu.Spec.Resources.DataVolumeSize = oldSize
		}()

		requests := map[corev1.ResourceName]resource.Quantity{}
		requests[corev1.ResourceStorage] = resource.MustParse("1Gi")
		pvc := &corev1.PersistentVolumeClaim{ObjectMeta: *dogu.GetObjectMeta(),
			Spec: corev1.PersistentVolumeClaimSpec{Resources: corev1.ResourceRequirements{Requests: requests}}}

		client := fake.NewClientBuilder().WithObjects(pvc).Build()
		recorder := external.NewEventRecorder(t)
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Update dogu data PVC request storage...")
		recorder.On("Eventf", dogu, "Normal", "VolumeExpansion", "Scale deployment to %d replicas...", int32(0))

		sut := &doguVolumeManager{client: client, eventRecorder: recorder}

		// when
		err := sut.SetDoguDataVolumeSize(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get deployment for dogu ldap")
	})
}

func Test_doguVolumeManager_scaleDeployment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)

		replicas := int32(2)
		deploy := &appsv1.Deployment{ObjectMeta: *dogu.GetObjectMeta(), Spec: appsv1.DeploymentSpec{Replicas: &replicas}}

		requests := map[corev1.ResourceName]resource.Quantity{}
		requests[corev1.ResourceStorage] = resource.MustParse("1Gi")
		pvc := &corev1.PersistentVolumeClaim{ObjectMeta: *dogu.GetObjectMeta(), Spec: corev1.PersistentVolumeClaimSpec{Resources: corev1.ResourceRequirements{Requests: requests}}}

		client := fake.NewClientBuilder().WithObjects(deploy, pvc).Build()
		recorder := external.NewEventRecorder(t)
		recorder.On("Eventf", dogu, "Normal", "VolumeExpansion", "Scale deployment to %d replicas...", int32(1))
		sut := &doguVolumeManager{client: client, eventRecorder: recorder}

		// when
		oldReplicas, err := sut.scaleDeployment(context.TODO(), dogu, 1)

		// then
		require.NoError(t, err)
		assert.Equal(t, int32(2), oldReplicas)
		deploy, err = dogu.GetDeployment(context.TODO(), client)
		require.NoError(t, err)
		assert.Equal(t, int32(1), *deploy.Spec.Replicas)
	})

	t.Run("fail to get deployment", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		client := fake.NewClientBuilder().Build()
		recorder := external.NewEventRecorder(t)
		recorder.On("Eventf", dogu, "Normal", "VolumeExpansion", "Scale deployment to %d replicas...", int32(1))
		sut := &doguVolumeManager{client: client, eventRecorder: recorder}

		// when
		_, err := sut.scaleDeployment(context.TODO(), dogu, 1)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get deployment for dogu ldap")
	})
}

func Test_doguVolumeManager_updatePVCQuantity(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		requests := map[corev1.ResourceName]resource.Quantity{}
		requests[corev1.ResourceStorage] = resource.MustParse("1Gi")
		doguPvc := &corev1.PersistentVolumeClaim{ObjectMeta: *dogu.GetObjectMeta(), Spec: corev1.PersistentVolumeClaimSpec{Resources: corev1.ResourceRequirements{Requests: requests}}}
		client := fake.NewClientBuilder().WithObjects(doguPvc).Build()
		recorder := external.NewEventRecorder(t)
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Update dogu data PVC request storage...")
		sut := &doguVolumeManager{client: client, eventRecorder: recorder}
		wantedCapacity := resource.MustParse("2Gi")

		// when
		err := sut.updatePVCQuantity(context.TODO(), dogu, wantedCapacity)

		// then
		require.NoError(t, err)
		pvc, err := dogu.GetDataPVC(context.TODO(), client)
		require.NoError(t, err)
		assert.True(t, pvc.Spec.Resources.Requests.Storage().Equal(wantedCapacity))
	})

	t.Run("fail to get dogu pvc", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		client := fake.NewClientBuilder().Build()
		recorder := external.NewEventRecorder(t)
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Update dogu data PVC request storage...")
		sut := &doguVolumeManager{client: client, eventRecorder: recorder}
		wantedCapacity := resource.MustParse("2Gi")

		// when
		err := sut.updatePVCQuantity(context.TODO(), dogu, wantedCapacity)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get data pvc for dogu ldap")
	})
}

func Test_doguVolumeManager_waitForPVCResize(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		requests := map[corev1.ResourceName]resource.Quantity{}
		requests[corev1.ResourceStorage] = resource.MustParse("1Gi")
		doguPvc := &corev1.PersistentVolumeClaim{ObjectMeta: *dogu.GetObjectMeta(), Spec: corev1.PersistentVolumeClaimSpec{Resources: corev1.ResourceRequirements{Requests: requests}}}
		client := fake.NewClientBuilder().WithObjects(doguPvc).Build()
		recorder := external.NewEventRecorder(t)
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Wait for pvc to be resized...")
		sut := &doguVolumeManager{client: client, eventRecorder: recorder}
		wantedCapacity := resource.MustParse("2Gi")

		wantRequests := map[corev1.ResourceName]resource.Quantity{}
		wantRequests[corev1.ResourceStorage] = resource.MustParse("2Gi")
		doguPvc.Status.Capacity = wantRequests
		err := client.Status().Update(context.TODO(), doguPvc)
		require.NoError(t, err)

		// when
		err = sut.waitForPVCResize(context.TODO(), dogu, wantedCapacity)

		// then
		require.NoError(t, err)
	})

	t.Run("fail to get pvc ", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		client := fake.NewClientBuilder().Build()
		recorder := external.NewEventRecorder(t)
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Wait for pvc to be resized...")
		sut := &doguVolumeManager{client: client, eventRecorder: recorder}
		wantedCapacity := resource.MustParse("2Gi")

		// when
		err := sut.waitForPVCResize(context.TODO(), dogu, wantedCapacity)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to wait for resizing data PVC for dogu ldap")
	})
}

func Test_isPvcStorageResized(t *testing.T) {
	t.Run("success on equal request storage capacity", func(t *testing.T) {
		// given
		resources := make(map[corev1.ResourceName]resource.Quantity)
		quantity := resource.MustParse("2Gi")
		resources[corev1.ResourceStorage] = quantity
		pvc := &corev1.PersistentVolumeClaim{Status: corev1.PersistentVolumeClaimStatus{Capacity: resources}}

		// when
		result := isPvcStorageResized(pvc, quantity)

		// then
		require.True(t, result)
	})

	t.Run("fail on unequal request storage capacity", func(t *testing.T) {
		// given
		resources := make(map[corev1.ResourceName]resource.Quantity)
		quantity := resource.MustParse("2Gi")
		resources[corev1.ResourceStorage] = quantity
		pvc := &corev1.PersistentVolumeClaim{Status: corev1.PersistentVolumeClaimStatus{Capacity: resources}}

		// when
		result := isPvcStorageResized(pvc, resource.MustParse("3Gi"))

		// then
		require.False(t, result)
	})
}
