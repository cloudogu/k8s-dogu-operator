package controllers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v2/internal/cloudogu"
	extMocks "github.com/cloudogu/k8s-dogu-operator/v2/internal/thirdParty/mocks"
)

func TestNewDoguVolumeManager(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		recorder := extMocks.NewEventRecorder(t)
		cli := extMocks.NewK8sClient(t)

		// when
		result := NewDoguVolumeManager(cli, recorder)

		// then
		require.NotNil(t, result)
	})
}

type errAsyncExecutor struct{}

func (e *errAsyncExecutor) AddStep(cloudogu.AsyncStep) {}

func (e *errAsyncExecutor) Execute(context.Context, *k8sv2.Dogu, string) error {
	return assert.AnError
}

type asyncExecutor struct{}

func (e *asyncExecutor) AddStep(cloudogu.AsyncStep) {}

func (e *asyncExecutor) Execute(context.Context, *k8sv2.Dogu, string) error {
	return nil
}

func Test_doguVolumeManager_SetDoguDataVolumeSize(t *testing.T) {
	t.Run("error setting initial state", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		manager := &doguVolumeManager{client: fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()}

		// when
		err := manager.SetDoguDataVolumeSize(context.TODO(), dogu)

		// then
		require.Error(t, err)
	})

	t.Run("failed to execute steps", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		executor := &errAsyncExecutor{}
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithStatusSubresource(&k8sv2.Dogu{}).WithObjects(dogu).Build()
		manager := &doguVolumeManager{client: client, asyncExecutor: executor}

		// when
		err := manager.SetDoguDataVolumeSize(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, assert.AnError, err)

		errDogu := &k8sv2.Dogu{}
		err = client.Get(context.TODO(), dogu.GetObjectKey(), errDogu)
		require.NoError(t, err)
		assert.Equal(t, "resizing PVC", errDogu.Status.Status)
	})

	t.Run("success", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		executor := &asyncExecutor{}
		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithStatusSubresource(&k8sv2.Dogu{}).WithObjects(dogu).Build()
		manager := &doguVolumeManager{client: client, asyncExecutor: executor}

		// when
		err := manager.SetDoguDataVolumeSize(context.TODO(), dogu)

		// then
		require.NoError(t, err)
		errDogu := &k8sv2.Dogu{}
		err = client.Get(context.TODO(), dogu.GetObjectKey(), errDogu)
		require.NoError(t, err)
		assert.Equal(t, "installed", errDogu.Status.Status)
	})
}

func Test_scaleUpStep_Execute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)

		replicas := int32(0)
		deploy := &appsv1.Deployment{ObjectMeta: *dogu.GetObjectMeta(), Spec: appsv1.DeploymentSpec{Replicas: &replicas}}

		client := fake.NewClientBuilder().WithScheme(getTestScheme()).WithStatusSubresource(&k8sv2.Dogu{}).WithObjects(deploy, dogu).Build()
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Eventf", dogu, "Normal", "VolumeExpansion", "Scale deployment to %d replicas...", int32(1))
		sut := &scaleUpStep{client: client, eventRecorder: recorder, replicas: 1}

		// when
		state, err := sut.Execute(context.TODO(), dogu)

		// then
		require.NoError(t, err)
		deploy, err = dogu.GetDeployment(context.TODO(), client)
		require.NoError(t, err)
		assert.Equal(t, int32(1), *deploy.Spec.Replicas)
		resultDogu := &k8sv2.Dogu{}
		err = client.Get(context.TODO(), dogu.GetObjectKey(), resultDogu)
		require.NoError(t, err)
		assert.Equal(t, "", resultDogu.Status.RequeuePhase)
		assert.Equal(t, "finished", state)
	})

	t.Run("fail to get deployment", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		client := fake.NewClientBuilder().Build()
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Eventf", dogu, "Normal", "VolumeExpansion", "Scale deployment to %d replicas...", int32(0))
		sut := &scaleUpStep{client: client, eventRecorder: recorder}

		// when
		state, err := sut.Execute(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get deployment for dogu ldap")
		assert.Equal(t, "Scale up", state)
	})
}

func Test_scaleDownStep_Execute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)

		replicas := int32(2)
		deploy := &appsv1.Deployment{ObjectMeta: *dogu.GetObjectMeta(), Spec: appsv1.DeploymentSpec{Replicas: &replicas}}

		client := fake.NewClientBuilder().WithObjects(deploy).Build()
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Eventf", dogu, "Normal", "VolumeExpansion", "Scale deployment to %d replicas...", int32(0))
		sus := &scaleUpStep{}
		sut := &scaleDownStep{client: client, eventRecorder: recorder, scaleUpStep: sus}

		// when
		state, err := sut.Execute(context.TODO(), dogu)

		// then
		require.NoError(t, err)
		deploy, err = dogu.GetDeployment(context.TODO(), client)
		require.NoError(t, err)
		assert.Equal(t, int32(0), *deploy.Spec.Replicas)
		assert.Equal(t, "Edit PVC", state)
	})

	t.Run("fail to get deployment", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		client := fake.NewClientBuilder().Build()
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Eventf", dogu, "Normal", "VolumeExpansion", "Scale deployment to %d replicas...", int32(0))
		sus := &scaleUpStep{}
		sut := &scaleDownStep{client: client, eventRecorder: recorder, scaleUpStep: sus}

		// when
		state, err := sut.Execute(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get deployment for dogu ldap")
		assert.Equal(t, "", state)
	})
}

func Test_editPVCStep_Execute(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		oldSize := dogu.Spec.Resources.DataVolumeSize
		defer func() {
			dogu.Spec.Resources.DataVolumeSize = oldSize
		}()
		dogu.Spec.Resources.DataVolumeSize = "1Gi"
		requests := make(map[corev1.ResourceName]resource.Quantity)
		requests[corev1.ResourceStorage] = resource.MustParse("0.5Gi")
		doguPvc := &corev1.PersistentVolumeClaim{ObjectMeta: *dogu.GetObjectMeta(), Spec: corev1.PersistentVolumeClaimSpec{Resources: corev1.VolumeResourceRequirements{Requests: requests}}}
		client := fake.NewClientBuilder().WithObjects(doguPvc).Build()
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Update dogu data PVC request storage...")
		sut := &editPVCStep{client: client, eventRecorder: recorder}
		wantedCapacity := resource.MustParse("1Gi")

		// when
		state, err := sut.Execute(context.TODO(), dogu)

		// then
		require.NoError(t, err)
		pvc, err := dogu.GetDataPVC(context.TODO(), client)
		require.NoError(t, err)
		assert.True(t, pvc.Spec.Resources.Requests.Storage().Equal(wantedCapacity))
		assert.Equal(t, "Wait for resize", state)
	})

	t.Run("fail to get dogu pvc", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		oldSize := dogu.Spec.Resources.DataVolumeSize
		defer func() {
			dogu.Spec.Resources.DataVolumeSize = oldSize
		}()
		dogu.Spec.Resources.DataVolumeSize = "1Gi"
		client := fake.NewClientBuilder().Build()
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Update dogu data PVC request storage...")
		sut := &editPVCStep{client: client, eventRecorder: recorder}

		// when
		stage, err := sut.Execute(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get data pvc for dogu ldap")
		assert.Equal(t, "Edit PVC", stage)
	})

	t.Run("fail to parse quantity", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		client := fake.NewClientBuilder().Build()
		recorder := extMocks.NewEventRecorder(t)
		sut := &editPVCStep{client: client, eventRecorder: recorder}

		// when
		stage, err := sut.Execute(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse to quantity")
		assert.Equal(t, "Edit PVC", stage)
	})
}

func Test_checkIfPVCIsResizedStep_execute(t *testing.T) {
	t.Run("success for capacity available", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		dogu.Spec.Resources.DataVolumeSize = "1Gi"
		requests := map[corev1.ResourceName]resource.Quantity{}
		requests[corev1.ResourceStorage] = resource.MustParse("1Gi")
		doguPvc := &corev1.PersistentVolumeClaim{ObjectMeta: *dogu.GetObjectMeta(), Status: corev1.PersistentVolumeClaimStatus{Capacity: requests}, Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{Requests: requests}}}
		client := fake.NewClientBuilder().WithObjects(doguPvc).Build()
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Wait for pvc to be resized...")
		sut := &checkIfPVCIsResizedStep{client: client, eventRecorder: recorder}

		// when
		state, err := sut.Execute(context.TODO(), dogu)

		// then
		require.NoError(t, err)
		assert.Equal(t, "Scale up", state)
	})

	t.Run("success for condition FileSystemResizePending has status true", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		dogu.Spec.Resources.DataVolumeSize = "1Gi"
		requests := map[corev1.ResourceName]resource.Quantity{}
		requests[corev1.ResourceStorage] = resource.MustParse("1Gi")
		doguPvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: *dogu.GetObjectMeta(),
			Status: corev1.PersistentVolumeClaimStatus{
				Conditions: []corev1.PersistentVolumeClaimCondition{
					{Type: corev1.PersistentVolumeClaimFileSystemResizePending, Status: corev1.ConditionTrue},
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				Resources: corev1.VolumeResourceRequirements{Requests: requests},
			},
		}
		client := fake.NewClientBuilder().WithObjects(doguPvc).Build()
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Wait for pvc to be resized...")
		sut := &checkIfPVCIsResizedStep{client: client, eventRecorder: recorder}

		// when
		state, err := sut.Execute(context.TODO(), dogu)

		// then
		require.NoError(t, err)
		assert.Equal(t, "Scale up", state)
	})

	t.Run("fail to get pvc", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		client := fake.NewClientBuilder().Build()
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Wait for pvc to be resized...")
		sut := &checkIfPVCIsResizedStep{client: client, eventRecorder: recorder}
		wantedCapacity := resource.MustParse("2Gi")

		// when
		err := sut.waitForPVCResize(context.TODO(), dogu, wantedCapacity)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to get data pvc for dogu ldap")
	})

	t.Run("fail to parse quantity", func(t *testing.T) {
		dogu := readDoguCr(t, ldapCrBytes)
		dogu.Spec.Resources.DataVolumeSize = "1Gsdfsdfi"
		client := fake.NewClientBuilder().Build()
		recorder := extMocks.NewEventRecorder(t)
		sut := &checkIfPVCIsResizedStep{client: client, eventRecorder: recorder}

		// when
		stage, err := sut.Execute(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to parse to quantity")
		assert.Equal(t, "Wait for resize", stage)
	})

	t.Run("should return requeue error if status of condition FileSystemResizePending is false", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		dogu.Spec.Resources.DataVolumeSize = "1Gi"
		requests := map[corev1.ResourceName]resource.Quantity{}
		requests[corev1.ResourceStorage] = resource.MustParse("0.5Gi")
		doguPvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: *dogu.GetObjectMeta(),
			Status: corev1.PersistentVolumeClaimStatus{
				Conditions: []corev1.PersistentVolumeClaimCondition{
					{Type: corev1.PersistentVolumeClaimFileSystemResizePending, Status: corev1.ConditionFalse},
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				Resources: corev1.VolumeResourceRequirements{Requests: requests},
			},
		}
		client := fake.NewClientBuilder().WithObjects(doguPvc).Build()
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Wait for pvc to be resized...")
		sut := &checkIfPVCIsResizedStep{client: client, eventRecorder: recorder}

		// when
		_, err := sut.Execute(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "pvc resizing is in progress")
	})

	t.Run("should return requeue error if size is not changed", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		dogu.Spec.Resources.DataVolumeSize = "1Gi"
		requests := map[corev1.ResourceName]resource.Quantity{}
		requests[corev1.ResourceStorage] = resource.MustParse("0.5Gi")
		doguPvc := &corev1.PersistentVolumeClaim{ObjectMeta: *dogu.GetObjectMeta(), Status: corev1.PersistentVolumeClaimStatus{Capacity: requests}, Spec: corev1.PersistentVolumeClaimSpec{
			Resources: corev1.VolumeResourceRequirements{Requests: requests}}}
		client := fake.NewClientBuilder().WithObjects(doguPvc).Build()
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Wait for pvc to be resized...")
		sut := &checkIfPVCIsResizedStep{client: client, eventRecorder: recorder}

		// when
		_, err := sut.Execute(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "pvc resizing is in progress")
	})

	t.Run("should return requeue error if size is not changed and there is no condition FileSystemResizePending", func(t *testing.T) {
		// given
		dogu := readDoguCr(t, ldapCrBytes)
		dogu.Spec.Resources.DataVolumeSize = "1Gi"
		requests := map[corev1.ResourceName]resource.Quantity{}
		requests[corev1.ResourceStorage] = resource.MustParse("0.5Gi")
		doguPvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: *dogu.GetObjectMeta(),
			Status: corev1.PersistentVolumeClaimStatus{
				Conditions: []corev1.PersistentVolumeClaimCondition{
					{Type: corev1.PersistentVolumeClaimResizing, Status: corev1.ConditionTrue},
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				Resources: corev1.VolumeResourceRequirements{Requests: requests},
			},
		}
		client := fake.NewClientBuilder().WithObjects(doguPvc).Build()
		recorder := extMocks.NewEventRecorder(t)
		recorder.On("Event", dogu, "Normal", "VolumeExpansion", "Wait for pvc to be resized...")
		sut := &checkIfPVCIsResizedStep{client: client, eventRecorder: recorder}

		// when
		_, err := sut.Execute(context.TODO(), dogu)

		// then
		require.Error(t, err)
		assert.ErrorContains(t, err, "pvc resizing is in progress")
	})
}

func Test_notResizedError(t *testing.T) {
	err := &notResizedError{
		state:       "state",
		requeueTime: time.Second * 5,
	}

	require.True(t, err.Requeue())
	assert.Equal(t, "state", err.GetState())
	assert.Equal(t, time.Second*5, err.GetRequeueTime())
}
