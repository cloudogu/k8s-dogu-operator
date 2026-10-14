package install

import (
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	flux "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func stoppedDoguClient(t *testing.T, objects ...client.Object) (*v3beta1.Dogu, client.Client) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, v3beta1.AddToScheme(scheme))
	require.NoError(t, flux.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	dogu := &v3beta1.Dogu{
		ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: "ecosystem", Generation: 7},
		Spec:       v3beta1.DoguSpec{Name: "scm", Version: "1.2.3", Stopped: true},
	}
	objects = append(objects, dogu)
	k8s := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(dogu).WithObjects(objects...).Build()
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(dogu), dogu))
	return dogu, k8s
}

func TestStopWorkloadsStep(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "ecosystem", UID: "web-uid", Labels: map[string]string{v3beta1.DoguLabelName: "scm"}},
		Spec:       appsv1.DeploymentSpec{Replicas: ptr.To[int32](3)},
	}
	release := &flux.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: "ecosystem", Generation: 3},
		Status:     flux.HelmReleaseStatus{ObservedGeneration: 3},
	}

	t.Run("waits for suspension without touching workloads", func(t *testing.T) {
		dogu, k8s := stoppedDoguClient(t, deployment.DeepCopy(), release.DeepCopy())
		result := NewStopWorkloadsStep(k8s).Run(t.Context(), dogu)
		require.NoError(t, result.Err)
		assert.Positive(t, result.RequeueAfter)
		actual := &appsv1.Deployment{}
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(deployment), actual))
		assert.Equal(t, int32(3), *actual.Spec.Replicas)
		assert.False(t, dogu.Status.Stopped)
	})

	t.Run("does not stop existing workloads without a HelmRelease", func(t *testing.T) {
		dogu, k8s := stoppedDoguClient(t, deployment.DeepCopy())
		result := NewStopWorkloadsStep(k8s).Run(t.Context(), dogu)
		require.NoError(t, result.Err)
		assert.Positive(t, result.RequeueAfter)
		actual := &appsv1.Deployment{}
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(deployment), actual))
		assert.Equal(t, int32(3), *actual.Spec.Replicas)
		assert.False(t, dogu.Status.Stopped)
	})

	t.Run("waits for Flux to acknowledge the suspended generation", func(t *testing.T) {
		suspended := release.DeepCopy()
		suspended.Spec.Suspend = true
		suspended.Generation++ // The earlier Helm action has only observed generation 3.
		statefulSet := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "ecosystem", UID: "data-uid", Labels: deployment.Labels},
			Spec: appsv1.StatefulSetSpec{
				Replicas: ptr.To[int32](2),
				PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
					WhenScaled: appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
				},
			},
		}
		dogu, k8s := stoppedDoguClient(t, deployment.DeepCopy(), suspended, statefulSet)
		step := NewStopWorkloadsStep(k8s)
		result := step.Run(t.Context(), dogu)
		require.NoError(t, result.Err)
		assert.Positive(t, result.RequeueAfter)
		actual := &appsv1.Deployment{}
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(deployment), actual))
		assert.Equal(t, int32(3), *actual.Spec.Replicas)
		actualSet := &appsv1.StatefulSet{}
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(statefulSet), actualSet))
		assert.Equal(t, statefulSet.Spec, actualSet.Spec)
		assert.False(t, dogu.Status.Stopped)

		// A stale Reconciling condition can remain after suspension; only the
		// top-level observed generation acknowledges the controller handoff.
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(suspended), suspended))
		suspended.Status.ObservedGeneration = suspended.Generation
		suspended.Status.Conditions = []metav1.Condition{{Type: "Reconciling", Status: metav1.ConditionTrue, Reason: "Progressing"}}
		require.NoError(t, k8s.Update(t.Context(), suspended))
		result = step.Run(t.Context(), dogu)
		require.NoError(t, result.Err)
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(deployment), actual))
		assert.Zero(t, *actual.Spec.Replicas)
	})

	t.Run("stops all owned workloads and waits for unlabeled Pods", func(t *testing.T) {
		suspended := release.DeepCopy()
		suspended.Spec.Suspend = true
		statefulSet := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "ecosystem", UID: "data-uid", Labels: deployment.Labels},
			Spec:       appsv1.StatefulSetSpec{Replicas: ptr.To[int32](2)},
		}
		foreign := deployment.DeepCopy()
		foreign.Name, foreign.UID = "foreign", "foreign-uid"
		foreign.Labels = map[string]string{v3beta1.DoguLabelName: "other"}
		replicaSet := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
			Name: "web-rs", Namespace: "ecosystem", UID: "rs-uid",
			OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "Deployment", Name: deployment.Name, UID: deployment.UID, Controller: ptr.To(true)}},
		}}
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
			Name: "web-pod", Namespace: "ecosystem", UID: "pod-uid",
			OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: replicaSet.Name, UID: replicaSet.UID, Controller: ptr.To(true)}},
		}}
		statefulPod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
			Name: "data-0", Namespace: "ecosystem", UID: "data-pod-uid",
			OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "StatefulSet", Name: statefulSet.Name, UID: statefulSet.UID, Controller: ptr.To(true)}},
		}}
		dogu, k8s := stoppedDoguClient(t, deployment.DeepCopy(), suspended, statefulSet, foreign, replicaSet, pod, statefulPod)
		step := NewStopWorkloadsStep(k8s)
		result := step.Run(t.Context(), dogu)
		require.NoError(t, result.Err)
		assert.Positive(t, result.RequeueAfter)
		actual := &appsv1.Deployment{}
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(deployment), actual))
		assert.Zero(t, *actual.Spec.Replicas)
		actualSet := &appsv1.StatefulSet{}
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(statefulSet), actualSet))
		assert.Zero(t, *actualSet.Spec.Replicas)
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(foreign), actual))
		assert.Equal(t, int32(3), *actual.Spec.Replicas)

		// A second pass must still wait for the Pod, not just zero replica specs.
		result = step.Run(t.Context(), dogu)
		require.NoError(t, result.Err)
		assert.Positive(t, result.RequeueAfter)
		assert.False(t, dogu.Status.Stopped)
		require.NoError(t, k8s.Delete(t.Context(), pod))
		result = step.Run(t.Context(), dogu)
		require.NoError(t, result.Err)
		assert.Positive(t, result.RequeueAfter, "the StatefulSet Pod must also be gone")
		assert.False(t, dogu.Status.Stopped)
		require.NoError(t, k8s.Delete(t.Context(), statefulPod))
		result = step.Run(t.Context(), dogu)
		require.NoError(t, result.Err)
		assert.True(t, result.Continue)
		assert.True(t, dogu.Status.Stopped)
		assert.True(t, meta.IsStatusConditionTrue(dogu.Status.Conditions, v3beta1.ConditionStopped))
	})

	t.Run("retains StatefulSet claims before scaling down", func(t *testing.T) {
		suspended := release.DeepCopy()
		suspended.Spec.Suspend = true
		statefulSet := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "ecosystem", UID: "data-uid", Labels: deployment.Labels},
			Spec: appsv1.StatefulSetSpec{
				Replicas: ptr.To[int32](2),
				PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
					WhenScaled: appsv1.DeletePersistentVolumeClaimRetentionPolicyType, WhenDeleted: appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "volume"}}},
			},
		}
		claim := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
			Name: "volume-data-1", Namespace: "ecosystem", UID: "claim-uid",
			OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "Pod", Name: "data-1", UID: "data-pod-uid"}},
		}}
		dogu, k8s := stoppedDoguClient(t, suspended, statefulSet, claim)
		step := NewStopWorkloadsStep(k8s)
		for range 2 {
			result := step.Run(t.Context(), dogu)
			require.NoError(t, result.Err)
			assert.Positive(t, result.RequeueAfter)
			actual := &appsv1.StatefulSet{}
			require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(statefulSet), actual))
			assert.Equal(t, int32(2), *actual.Spec.Replicas)
			assert.Equal(t, appsv1.RetainPersistentVolumeClaimRetentionPolicyType, actual.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled)
			assert.Equal(t, appsv1.DeletePersistentVolumeClaimRetentionPolicyType, actual.Spec.PersistentVolumeClaimRetentionPolicy.WhenDeleted)
		}
		// Simulate the StatefulSet controller removing the scale-down owner.
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(claim), claim))
		claim.OwnerReferences = nil
		require.NoError(t, k8s.Update(t.Context(), claim))
		result := step.Run(t.Context(), dogu)
		require.NoError(t, result.Err)
		actual := &appsv1.StatefulSet{}
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(statefulSet), actual))
		assert.Zero(t, *actual.Spec.Replicas)
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(claim), &corev1.PersistentVolumeClaim{}))
	})

	t.Run("does not scale a StatefulSet whose claim is already being deleted", func(t *testing.T) {
		suspended := release.DeepCopy()
		suspended.Spec.Suspend = true
		statefulSet := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "ecosystem", UID: "data-uid", Labels: deployment.Labels},
			Spec: appsv1.StatefulSetSpec{
				Replicas:             ptr.To[int32](2),
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "volume"}}},
			},
		}
		claim := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
			Name: "volume-data-1", Namespace: "ecosystem", Finalizers: []string{"test.cloudogu.com/hold"},
		}}
		dogu, k8s := stoppedDoguClient(t, suspended, statefulSet, claim)
		require.NoError(t, k8s.Delete(t.Context(), claim))

		result := NewStopWorkloadsStep(k8s).Run(t.Context(), dogu)
		require.ErrorContains(t, result.Err, "already being deleted")
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(statefulSet), statefulSet))
		assert.Equal(t, int32(2), *statefulSet.Spec.Replicas)
		assert.False(t, dogu.Status.Stopped)
	})

	t.Run("waits after changing retention without blocking other StatefulSets", func(t *testing.T) {
		suspended := release.DeepCopy()
		suspended.Spec.Suspend = true
		protecting := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "a-data", Namespace: "ecosystem", UID: "data-uid", Labels: deployment.Labels},
			Spec: appsv1.StatefulSetSpec{
				Replicas: ptr.To[int32](2),
				PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
					WhenScaled: appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "volume"}}},
			},
		}
		ready := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "z-ready", Namespace: "ecosystem", UID: "ready-uid", Labels: deployment.Labels},
			Spec:       appsv1.StatefulSetSpec{Replicas: ptr.To[int32](3)},
		}
		claim := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "volume-a-data-0", Namespace: "ecosystem"}}
		dogu, k8s := stoppedDoguClient(t, suspended, protecting, ready, claim)
		step := NewStopWorkloadsStep(k8s)
		result := step.Run(t.Context(), dogu)
		require.NoError(t, result.Err)
		assert.Positive(t, result.RequeueAfter)
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(protecting), protecting))
		assert.Equal(t, appsv1.RetainPersistentVolumeClaimRetentionPolicyType, protecting.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled)
		assert.Equal(t, int32(2), *protecting.Spec.Replicas, "retention changes must wait even without Pod-owned claims")
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(ready), ready))
		assert.Zero(t, *ready.Spec.Replicas, "one waiting StatefulSet must not block another")

		result = step.Run(t.Context(), dogu)
		require.NoError(t, result.Err)
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(protecting), protecting))
		assert.Zero(t, *protecting.Spec.Replicas)
	})

	t.Run("stopped first installation has no running workloads", func(t *testing.T) {
		dogu, k8s := stoppedDoguClient(t)
		result := NewStopWorkloadsStep(k8s).Run(t.Context(), dogu)
		require.NoError(t, result.Err)
		assert.True(t, result.Continue)
		assert.True(t, dogu.Status.Stopped)
	})
}

func TestStopWorkloadsStep_DisappearedReplicaSet(t *testing.T) {
	for _, tc := range []struct {
		name      string
		podLabels map[string]string
		foreignRS bool
		wantWait  bool
	}{
		{name: "terminating Pod outlives its ReplicaSet", podLabels: map[string]string{"app": "web", "component": "backend"}, wantWait: true},
		{name: "nonmatching orphan belongs elsewhere", podLabels: map[string]string{"app": "web", "component": "frontend"}},
		{name: "resolvable foreign owner takes precedence over selector", podLabels: map[string]string{"app": "web", "component": "backend"}, foreignRS: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "ecosystem", UID: "web-uid", Labels: map[string]string{v3beta1.DoguLabelName: "scm"}},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.To[int32](0),
					Selector: &metav1.LabelSelector{
						MatchLabels:      map[string]string{"app": "web"},
						MatchExpressions: []metav1.LabelSelectorRequirement{{Key: "component", Operator: metav1.LabelSelectorOpIn, Values: []string{"backend"}}},
					},
				},
			}
			replicaSet := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
				Name: "old-web", Namespace: "ecosystem", UID: "old-rs-uid",
				OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "Deployment", Name: deployment.Name, UID: deployment.UID, Controller: ptr.To(true)}},
			}}
			if tc.foreignRS {
				replicaSet.OwnerReferences[0].UID = "foreign-deployment-uid"
			}
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				Name: "old-web-pod", Namespace: "ecosystem", UID: "old-pod-uid", Labels: tc.podLabels, Finalizers: []string{"test.cloudogu.com/hold"},
				OwnerReferences: []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: "ReplicaSet", Name: replicaSet.Name, UID: replicaSet.UID, Controller: ptr.To(true)}},
			}}
			release := &flux.HelmRelease{
				ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: "ecosystem", Generation: 4},
				Spec:       flux.HelmReleaseSpec{Suspend: true},
				Status:     flux.HelmReleaseStatus{ObservedGeneration: 4},
			}
			dogu, k8s := stoppedDoguClient(t, deployment, release, replicaSet, pod)
			require.NoError(t, k8s.Delete(t.Context(), pod))
			require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(pod), pod))
			require.False(t, pod.DeletionTimestamp.IsZero())
			if !tc.foreignRS {
				require.NoError(t, k8s.Delete(t.Context(), replicaSet))
			}
			// A fresh step has no in-memory owner history to rely on.
			result := NewStopWorkloadsStep(k8s).Run(t.Context(), dogu)
			require.NoError(t, result.Err)
			assert.Equal(t, !tc.wantWait, result.Continue)
			assert.Equal(t, !tc.wantWait, dogu.Status.Stopped)
			if tc.wantWait {
				assert.Positive(t, result.RequeueAfter)
				pod.Finalizers = nil
				require.NoError(t, k8s.Update(t.Context(), pod))
				result = NewStopWorkloadsStep(k8s).Run(t.Context(), dogu)
				require.NoError(t, result.Err)
				assert.True(t, result.Continue)
				assert.True(t, meta.IsStatusConditionTrue(dogu.Status.Conditions, v3beta1.ConditionStopped))
			}
		})
	}
}
