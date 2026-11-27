package workloads

import (
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	flux "github.com/fluxcd/helm-controller/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var scalerReleaseKey = client.ObjectKey{Namespace: "ecosystem", Name: "scm"}

func scalerClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, flux.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

// liveWorkloads reads the given workloads back, as the scaler expects current resource versions.
func liveWorkloads(t *testing.T, k8s client.Client, deployments []*appsv1.Deployment, statefulSets []*appsv1.StatefulSet) Workloads {
	t.Helper()
	var found Workloads
	for _, deployment := range deployments {
		actual := appsv1.Deployment{}
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(deployment), &actual))
		found.Deployments = append(found.Deployments, actual)
	}
	for _, statefulSet := range statefulSets {
		actual := appsv1.StatefulSet{}
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(statefulSet), &actual))
		found.StatefulSets = append(found.StatefulSets, actual)
	}
	return found
}

func replicasOf(t *testing.T, k8s client.Client, obj client.Object) int32 {
	t.Helper()
	require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(obj), obj))
	switch workload := obj.(type) {
	case *appsv1.Deployment:
		return *workload.Spec.Replicas
	case *appsv1.StatefulSet:
		return *workload.Spec.Replicas
	}
	t.Fatalf("unsupported workload %T", obj)
	return 0
}

func TestScaler_ScaleDown(t *testing.T) {
	doguLabels := map[string]string{v3beta1.DoguLabelName: "scm"}
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "ecosystem", UID: "web-uid", Labels: doguLabels},
		Spec:       appsv1.DeploymentSpec{Replicas: ptr.To[int32](3)},
	}
	release := &flux.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{Name: "scm", Namespace: "ecosystem", Generation: 3},
		Status:     flux.HelmReleaseStatus{ObservedGeneration: 3},
	}
	suspendedRelease := func() *flux.HelmRelease {
		suspended := release.DeepCopy()
		suspended.Spec.Suspend = true
		return suspended
	}

	t.Run("waits for suspension without touching workloads", func(t *testing.T) {
		k8s := scalerClient(t, deployment.DeepCopy(), release.DeepCopy())

		state, err := NewScaler(k8s).ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, []*appsv1.Deployment{deployment}, nil))

		require.NoError(t, err)
		assert.Equal(t, WaitingForSuspension, state)
		assert.Equal(t, int32(3), replicasOf(t, k8s, deployment.DeepCopy()))
	})

	t.Run("does not scale existing workloads without a HelmRelease", func(t *testing.T) {
		k8s := scalerClient(t, deployment.DeepCopy())

		state, err := NewScaler(k8s).ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, []*appsv1.Deployment{deployment}, nil))

		require.NoError(t, err)
		assert.Equal(t, WaitingForRelease, state)
		assert.Equal(t, int32(3), replicasOf(t, k8s, deployment.DeepCopy()))
	})

	t.Run("reports scaled down without HelmRelease and workloads", func(t *testing.T) {
		k8s := scalerClient(t)

		state, err := NewScaler(k8s).ScaleDown(t.Context(), scalerReleaseKey, Workloads{})

		require.NoError(t, err)
		assert.Equal(t, ScaledDown, state)
	})

	t.Run("waits for Flux to acknowledge the suspended generation", func(t *testing.T) {
		suspended := suspendedRelease()
		suspended.Generation++ // The earlier Helm action has only observed generation 3.
		statefulSet := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "ecosystem", UID: "data-uid", Labels: doguLabels},
			Spec: appsv1.StatefulSetSpec{
				Replicas: ptr.To[int32](2),
				PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
					WhenScaled: appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
				},
			},
		}
		k8s := scalerClient(t, deployment.DeepCopy(), suspended, statefulSet)
		scaler := NewScaler(k8s)
		deployments, statefulSets := []*appsv1.Deployment{deployment}, []*appsv1.StatefulSet{statefulSet}

		state, err := scaler.ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, deployments, statefulSets))

		require.NoError(t, err)
		assert.Equal(t, WaitingForSuspension, state)
		assert.Equal(t, int32(3), replicasOf(t, k8s, deployment.DeepCopy()))
		actualSet := &appsv1.StatefulSet{}
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(statefulSet), actualSet))
		assert.Equal(t, statefulSet.Spec, actualSet.Spec)

		// A stale Reconciling condition can remain after suspension; only the
		// top-level observed generation acknowledges the controller handoff.
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(suspended), suspended))
		suspended.Status.ObservedGeneration = suspended.Generation
		suspended.Status.Conditions = []metav1.Condition{{Type: "Reconciling", Status: metav1.ConditionTrue, Reason: "Progressing"}}
		require.NoError(t, k8s.Update(t.Context(), suspended))

		state, err = scaler.ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, deployments, statefulSets))

		require.NoError(t, err)
		assert.Equal(t, ScalingDown, state)
		assert.Zero(t, replicasOf(t, k8s, deployment.DeepCopy()))
	})

	t.Run("scales only the given workloads and waits for their Pods", func(t *testing.T) {
		statefulSet := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "ecosystem", UID: "data-uid", Labels: doguLabels},
			Spec:       appsv1.StatefulSetSpec{Replicas: ptr.To[int32](2)},
		}
		other := deployment.DeepCopy()
		other.Name, other.UID = "other", "other-uid"
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
		k8s := scalerClient(t, deployment.DeepCopy(), suspendedRelease(), statefulSet, other, replicaSet, pod, statefulPod)
		scaler := NewScaler(k8s)
		deployments, statefulSets := []*appsv1.Deployment{deployment}, []*appsv1.StatefulSet{statefulSet}

		state, err := scaler.ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, deployments, statefulSets))

		require.NoError(t, err)
		assert.Equal(t, ScalingDown, state)
		assert.Zero(t, replicasOf(t, k8s, deployment.DeepCopy()))
		assert.Zero(t, replicasOf(t, k8s, statefulSet.DeepCopy()))
		assert.Equal(t, int32(3), replicasOf(t, k8s, other), "workloads not passed in must not be scaled")

		// A second pass must still wait for the Pod, not just zero replica specs.
		state, err = scaler.ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, deployments, statefulSets))
		require.NoError(t, err)
		assert.Equal(t, ScalingDown, state)

		require.NoError(t, k8s.Delete(t.Context(), pod))
		state, err = scaler.ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, deployments, statefulSets))
		require.NoError(t, err)
		assert.Equal(t, ScalingDown, state, "the StatefulSet Pod must also be gone")

		require.NoError(t, k8s.Delete(t.Context(), statefulPod))
		state, err = scaler.ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, deployments, statefulSets))
		require.NoError(t, err)
		assert.Equal(t, ScaledDown, state)
	})

	t.Run("retains StatefulSet claims before scaling down", func(t *testing.T) {
		statefulSet := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "ecosystem", UID: "data-uid", Labels: doguLabels},
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
		k8s := scalerClient(t, suspendedRelease(), statefulSet, claim)
		scaler := NewScaler(k8s)
		statefulSets := []*appsv1.StatefulSet{statefulSet}
		for range 2 {
			state, err := scaler.ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, nil, statefulSets))
			require.NoError(t, err)
			assert.Equal(t, ScalingDown, state)
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

		_, err := scaler.ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, nil, statefulSets))

		require.NoError(t, err)
		assert.Zero(t, replicasOf(t, k8s, statefulSet.DeepCopy()))
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(claim), &corev1.PersistentVolumeClaim{}))
	})

	t.Run("does not scale a StatefulSet whose claim is already being deleted", func(t *testing.T) {
		statefulSet := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "ecosystem", UID: "data-uid", Labels: doguLabels},
			Spec: appsv1.StatefulSetSpec{
				Replicas:             ptr.To[int32](2),
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "volume"}}},
			},
		}
		claim := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{
			Name: "volume-data-1", Namespace: "ecosystem", Finalizers: []string{"test.cloudogu.com/hold"},
		}}
		k8s := scalerClient(t, suspendedRelease(), statefulSet, claim)
		require.NoError(t, k8s.Delete(t.Context(), claim))

		_, err := NewScaler(k8s).ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, nil, []*appsv1.StatefulSet{statefulSet}))

		require.ErrorContains(t, err, "already being deleted")
		assert.Equal(t, int32(2), replicasOf(t, k8s, statefulSet.DeepCopy()))
	})

	t.Run("waits after changing retention without blocking other StatefulSets", func(t *testing.T) {
		protecting := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "a-data", Namespace: "ecosystem", UID: "data-uid", Labels: doguLabels},
			Spec: appsv1.StatefulSetSpec{
				Replicas: ptr.To[int32](2),
				PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
					WhenScaled: appsv1.DeletePersistentVolumeClaimRetentionPolicyType,
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "volume"}}},
			},
		}
		ready := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Name: "z-ready", Namespace: "ecosystem", UID: "ready-uid", Labels: doguLabels},
			Spec:       appsv1.StatefulSetSpec{Replicas: ptr.To[int32](3)},
		}
		claim := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "volume-a-data-0", Namespace: "ecosystem"}}
		k8s := scalerClient(t, suspendedRelease(), protecting, ready, claim)
		scaler := NewScaler(k8s)
		statefulSets := []*appsv1.StatefulSet{protecting, ready}

		state, err := scaler.ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, nil, statefulSets))

		require.NoError(t, err)
		assert.Equal(t, ScalingDown, state)
		actual := &appsv1.StatefulSet{}
		require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(protecting), actual))
		assert.Equal(t, appsv1.RetainPersistentVolumeClaimRetentionPolicyType, actual.Spec.PersistentVolumeClaimRetentionPolicy.WhenScaled)
		assert.Equal(t, int32(2), *actual.Spec.Replicas, "retention changes must wait even without Pod-owned claims")
		assert.Zero(t, replicasOf(t, k8s, ready.DeepCopy()), "one waiting StatefulSet must not block another")

		_, err = scaler.ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, nil, statefulSets))

		require.NoError(t, err)
		assert.Zero(t, replicasOf(t, k8s, protecting.DeepCopy()))
	})
}

func TestScaler_ScaleDown_DisappearedReplicaSet(t *testing.T) {
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
			k8s := scalerClient(t, deployment, release, replicaSet, pod)
			require.NoError(t, k8s.Delete(t.Context(), pod))
			require.NoError(t, k8s.Get(t.Context(), client.ObjectKeyFromObject(pod), pod))
			require.False(t, pod.DeletionTimestamp.IsZero())
			if !tc.foreignRS {
				require.NoError(t, k8s.Delete(t.Context(), replicaSet))
			}
			deployments := []*appsv1.Deployment{deployment}
			wantState := ScaledDown
			if tc.wantWait {
				wantState = ScalingDown
			}

			// A fresh scaler has no in-memory owner history to rely on.
			state, err := NewScaler(k8s).ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, deployments, nil))

			require.NoError(t, err)
			assert.Equal(t, wantState, state)
			if tc.wantWait {
				pod.Finalizers = nil
				require.NoError(t, k8s.Update(t.Context(), pod))
				state, err = NewScaler(k8s).ScaleDown(t.Context(), scalerReleaseKey, liveWorkloads(t, k8s, deployments, nil))
				require.NoError(t, err)
				assert.Equal(t, ScaledDown, state)
			}
		})
	}
}
