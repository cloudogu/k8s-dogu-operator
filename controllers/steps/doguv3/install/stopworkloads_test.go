package install

import (
	"errors"
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func stoppedDoguClient(t *testing.T) (*v3beta1.Dogu, client.Client) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, v3beta1.AddToScheme(scheme))
	dogu := &v3beta1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: "ecosystem", Generation: 7},
		Spec:       v3beta1.DoguSpec{Name: "scm", Version: "1.2.3", Stopped: true},
	}
	k8s := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(dogu).WithObjects(dogu).Build()
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(dogu), dogu))
	return dogu, k8s
}

func TestStopWorkloadsStep(t *testing.T) {
	releaseKey := client.ObjectKey{Namespace: "ecosystem", Name: "scm"}
	found := workloads.Workloads{Deployments: []appsv1.Deployment{{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "ecosystem"}}}}

	newStep := func(t *testing.T, k8s client.Client) (*StopWorkloadsStep, *MockVolumeDiscovery, *MockWorkloadScaler) {
		discovery, scaler := NewMockVolumeDiscovery(t), NewMockWorkloadScaler(t)
		return &StopWorkloadsStep{k8sClient: k8s, discovery: discovery, scaler: scaler}, discovery, scaler
	}

	t.Run("pause skips stopping even when stopped is requested", func(t *testing.T) {
		dogu, k8s := stoppedDoguClient(t)
		dogu.Spec.PauseReconciliation = true
		before := dogu.DeepCopy()
		step, _, _ := newStep(t, k8s)

		result := step.Run(t.Context(), dogu)

		require.NoError(t, result.Err)
		assert.True(t, result.Continue)
		assert.Equal(t, before.Status, dogu.Status)
	})

	t.Run("does nothing for a running dogu", func(t *testing.T) {
		dogu, k8s := stoppedDoguClient(t)
		dogu.Spec.Stopped = false
		step, _, _ := newStep(t, k8s)

		result := step.Run(t.Context(), dogu)

		require.NoError(t, result.Err)
		assert.True(t, result.Continue)
	})

	t.Run("requeues with error when workloads cannot be discovered", func(t *testing.T) {
		dogu, k8s := stoppedDoguClient(t)
		step, discovery, _ := newStep(t, k8s)
		discovery.EXPECT().Workloads(mock.Anything, "ecosystem", "scm").Return(workloads.Workloads{}, errors.New("list failed"))

		result := step.Run(t.Context(), dogu)

		require.ErrorContains(t, result.Err, "list failed")
		assert.Equal(t, v3beta1.ReasonStopping, result.ReadyReason)
		assert.False(t, dogu.Status.Stopped)
	})

	t.Run("requeues with error when scaling fails", func(t *testing.T) {
		dogu, k8s := stoppedDoguClient(t)
		step, discovery, scaler := newStep(t, k8s)
		discovery.EXPECT().Workloads(mock.Anything, "ecosystem", "scm").Return(found, nil)
		scaler.EXPECT().ScaleDown(mock.Anything, releaseKey, found).Return(workloads.ScalingDown, errors.New("patch failed"))

		result := step.Run(t.Context(), dogu)

		require.ErrorContains(t, result.Err, "patch failed")
		assert.Equal(t, v3beta1.ReasonStopping, result.ReadyReason)
		assert.False(t, dogu.Status.Stopped)
	})

	t.Run("requeues with error for an unknown scale state without touching the status", func(t *testing.T) {
		dogu, k8s := stoppedDoguClient(t)
		before := dogu.DeepCopy()
		step, discovery, scaler := newStep(t, k8s)
		discovery.EXPECT().Workloads(mock.Anything, "ecosystem", "scm").Return(found, nil)
		scaler.EXPECT().ScaleDown(mock.Anything, releaseKey, found).Return(workloads.ScaleState(99), nil)

		result := step.Run(t.Context(), dogu)

		require.ErrorContains(t, result.Err, "unknown scale state 99")
		assert.Equal(t, v3beta1.ReasonStopping, result.ReadyReason)
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(dogu), dogu))
		assert.Equal(t, before.Status, dogu.Status)
		assert.Equal(t, before.ResourceVersion, dogu.ResourceVersion)
	})

	for _, tc := range []struct {
		state   workloads.ScaleState
		message string
	}{
		{state: workloads.WaitingForSuspension, message: "Waiting for HelmRelease suspension before stopping workloads"},
		{state: workloads.WaitingForRelease, message: "Waiting for a suspended HelmRelease for existing workloads"},
	} {
		t.Run("preserves the status while "+tc.message, func(t *testing.T) {
			dogu, k8s := stoppedDoguClient(t)
			meta.SetStatusCondition(&dogu.Status.Conditions, metav1.Condition{
				Type: v3beta1.ConditionUpdatePending, Status: metav1.ConditionTrue, Reason: v3beta1.ReasonStopped,
				Message: "Chart or values changes are pending until the dogu starts", ObservedGeneration: dogu.Generation,
			})
			require.NoError(t, k8s.Status().Update(t.Context(), dogu))
			before := dogu.DeepCopy()
			step, discovery, scaler := newStep(t, k8s)
			discovery.EXPECT().Workloads(mock.Anything, "ecosystem", "scm").Return(found, nil)
			scaler.EXPECT().ScaleDown(mock.Anything, releaseKey, found).Return(tc.state, nil)

			result := step.Run(t.Context(), dogu)

			require.NoError(t, result.Err)
			assert.Positive(t, result.RequeueAfter)
			assert.Equal(t, v3beta1.ReasonStopping, result.ReadyReason)
			assert.Equal(t, tc.message, result.ReadyMessage)
			require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(dogu), dogu))
			assert.Equal(t, before.Status, dogu.Status)
			assert.Equal(t, before.ResourceVersion, dogu.ResourceVersion)
		})
	}

	t.Run("reports partially stopped while scaling down", func(t *testing.T) {
		dogu, k8s := stoppedDoguClient(t)
		step, discovery, scaler := newStep(t, k8s)
		discovery.EXPECT().Workloads(mock.Anything, "ecosystem", "scm").Return(found, nil)
		scaler.EXPECT().ScaleDown(mock.Anything, releaseKey, found).Return(workloads.ScalingDown, nil)

		result := step.Run(t.Context(), dogu)

		require.NoError(t, result.Err)
		assert.Positive(t, result.RequeueAfter)
		assert.Equal(t, "Waiting for all dogu workloads and Pods to stop", result.ReadyMessage)
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(dogu), dogu))
		assert.False(t, dogu.Status.Stopped)
		condition := meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionStopped)
		require.NotNil(t, condition)
		assert.Equal(t, metav1.ConditionFalse, condition.Status)
		assert.Equal(t, v3beta1.ReasonPartiallyStopped, condition.Reason)
	})

	t.Run("reports stopped once scaled down", func(t *testing.T) {
		dogu, k8s := stoppedDoguClient(t)
		step, discovery, scaler := newStep(t, k8s)
		discovery.EXPECT().Workloads(mock.Anything, "ecosystem", "scm").Return(found, nil)
		scaler.EXPECT().ScaleDown(mock.Anything, releaseKey, found).Return(workloads.ScaledDown, nil)

		result := step.Run(t.Context(), dogu)

		require.NoError(t, result.Err)
		assert.True(t, result.Continue)
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(dogu), dogu))
		assert.True(t, dogu.Status.Stopped)
		assert.True(t, meta.IsStatusConditionTrue(dogu.Status.Conditions, v3beta1.ConditionStopped))
	})

	t.Run("does not update an already stopped status", func(t *testing.T) {
		dogu, k8s := stoppedDoguClient(t)
		step, discovery, scaler := newStep(t, k8s)
		discovery.EXPECT().Workloads(mock.Anything, "ecosystem", "scm").Return(found, nil)
		scaler.EXPECT().ScaleDown(mock.Anything, releaseKey, found).Return(workloads.ScaledDown, nil)
		require.True(t, step.Run(t.Context(), dogu).Continue)
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(dogu), dogu))
		before := dogu.DeepCopy()

		result := step.Run(t.Context(), dogu)

		assert.True(t, result.Continue)
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(dogu), dogu))
		assert.Equal(t, before.ResourceVersion, dogu.ResourceVersion)
	})
}

func TestNewStopWorkloadsStep(t *testing.T) {
	_, k8s := stoppedDoguClient(t)

	step := NewStopWorkloadsStep(k8s)

	assert.NotNil(t, step.discovery)
	assert.NotNil(t, step.scaler)
}
