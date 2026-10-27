package workloads

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func mountingPod(name, claimName string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: objectMeta(name, testNamespace, ""), Spec: corev1.PodSpec{Volumes: []corev1.Volume{{
		Name:                  "data",
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: claimName},
	}}}}
}

func dataVolume(owners ...metav1.OwnerReference) Volume {
	pvc := corev1.PersistentVolumeClaim{ObjectMeta: objectMeta("data", testNamespace, "")}
	pvc.OwnerReferences = owners
	return Volume{PVC: pvc, MountedBy: []corev1.TypedLocalObjectReference{workloadReference("StatefulSet", "nexus")}}
}

func TestForeignOwnedClaims(t *testing.T) {
	statefulSet := appsv1.StatefulSet{ObjectMeta: objectMeta("nexus", testNamespace, testDoguName)}
	statefulSet.UID = "nexus-uid"
	workloads := Workloads{StatefulSets: []appsv1.StatefulSet{statefulSet}}

	t.Run("allows a claim owned by its mounting StatefulSet", func(t *testing.T) {
		failed, owners := ForeignOwnedClaims(
			workloads,
			[]Volume{dataVolume(metav1.OwnerReference{Kind: "StatefulSet", Name: "nexus", UID: "nexus-uid"})},
			[]string{"data"},
		)

		assert.Empty(t, failed)
		assert.Empty(t, owners)
	})

	t.Run("reports a claim owned by another object", func(t *testing.T) {
		failed, owners := ForeignOwnedClaims(
			workloads,
			[]Volume{dataVolume(metav1.OwnerReference{Kind: "Job", Name: "backup", UID: "job-uid"})},
			[]string{"data"},
		)

		assert.Equal(t, []string{"data"}, failed)
		assert.Equal(t, []string{"data (Job/backup)"}, owners)
	})
}

func TestDetacher_DeleteWorkloads(t *testing.T) {
	statefulSet := &appsv1.StatefulSet{ObjectMeta: objectMeta("nexus", testNamespace, testDoguName)}
	statefulSet.UID = "nexus-uid"
	volumes := []Volume{dataVolume()}

	t.Run("orphans the mounting StatefulSet and keeps its pods", func(t *testing.T) {
		var propagation metav1.DeletionPropagation
		c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(statefulSet.DeepCopy(), mountingPod("nexus-0", "data")).
			WithInterceptorFuncs(interceptor.Funcs{Delete: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
				options := &client.DeleteOptions{}
				options.ApplyOptions(opts)
				propagation = *options.PropagationPolicy
				return c.Delete(ctx, obj, opts...)
			}}).Build()

		deleted, err := NewDetacher(c, c).DeleteWorkloads(
			context.Background(),
			Workloads{StatefulSets: []appsv1.StatefulSet{*statefulSet}},
			volumes,
			[]string{"data"},
			nil,
		)

		require.NoError(t, err)
		assert.Equal(t, []string{"StatefulSet/nexus"}, deleted)
		assert.Equal(t, metav1.DeletePropagationOrphan, propagation)
		require.NoError(t, c.Get(context.Background(), client.ObjectKey{Namespace: testNamespace, Name: "nexus-0"}, &corev1.Pod{}))
	})

	t.Run("replaces an outdated StatefulSet without claims", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(statefulSet.DeepCopy()).Build()

		deleted, err := NewDetacher(c, c).DeleteWorkloads(
			context.Background(),
			Workloads{StatefulSets: []appsv1.StatefulSet{*statefulSet}},
			nil,
			nil,
			[]types.UID{"nexus-uid"},
		)

		require.NoError(t, err)
		assert.Equal(t, []string{"StatefulSet/nexus"}, deleted)
	})
}

func TestDetacher_DeleteMountingPods(t *testing.T) {
	t.Run("deletes only the pods mounting the claims", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(mountingPod("nexus-0", "data"), mountingPod("other-0", "other")).Build()

		pods, newlyDeleted, err := NewDetacher(c, c).DeleteMountingPods(context.Background(), testNamespace, []string{"data"})

		require.NoError(t, err)
		assert.Equal(t, []string{"nexus-0"}, pods)
		assert.Equal(t, []string{"nexus-0"}, newlyDeleted)
	})

	t.Run("lists no pods without claims", func(t *testing.T) {
		c := fake.NewClientBuilder().WithScheme(testScheme(t)).
			WithInterceptorFuncs(interceptor.Funcs{List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				t.Fatal("pods must not be listed without claims")
				return nil
			}}).Build()

		pods, newlyDeleted, err := NewDetacher(c, c).DeleteMountingPods(context.Background(), testNamespace, nil)

		require.NoError(t, err)
		assert.Empty(t, pods)
		assert.Empty(t, newlyDeleted)
	})
}
