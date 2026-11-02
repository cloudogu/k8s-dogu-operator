package pvc

import (
	"testing"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

// volumePVC requests "request" but has only "capacity", with the given conditions set to true.
func volumePVC(name, request, capacity string, conditions ...corev1.PersistentVolumeClaimConditionType) corev1.PersistentVolumeClaim {
	claim := *newPVC(name, request, nil)
	claim.Status.Capacity = corev1.ResourceList{corev1.ResourceStorage: resource.MustParse(capacity)}
	for _, condition := range conditions {
		claim.Status.Conditions = append(claim.Status.Conditions, corev1.PersistentVolumeClaimCondition{Type: condition, Status: corev1.ConditionTrue})
	}
	return claim
}

func TestVolumeStatuses(t *testing.T) {
	statefulSet := []corev1.TypedLocalObjectReference{{APIGroup: ptr.To(appsv1.GroupName), Kind: "StatefulSet", Name: "nexus"}}
	proxy := []corev1.TypedLocalObjectReference{{APIGroup: ptr.To(appsv1.GroupName), Kind: "Deployment", Name: "nexus-proxy"}}
	// the standalone claim "data" with 2Gi and the claim template "db" of StatefulSet "nexus" with 5Gi
	rendered := renderedObjects(t, newPVC("data", "2Gi", nil), newStatefulSet("nexus", nil, 0, newPVC("db", "5Gi", nil)))

	t.Run("reports size, desired size, state and mounting workloads of each volume sorted by claim name", func(t *testing.T) {
		volumes := []Volume{
			{PVC: volumePVC("existing", "1Gi", "1Gi"), MountedBy: proxy},
			{PVC: volumePVC("db-nexus-0", "5Gi", "3Gi"), MountedBy: statefulSet},
			{PVC: volumePVC("data", "2Gi", "2Gi"), MountedBy: statefulSet},
			{PVC: volumePVC("db-nexus-1", "5Gi", "5Gi", corev1.PersistentVolumeClaimFileSystemResizePending), MountedBy: statefulSet},
			{PVC: volumePVC("db-nexus-2", "5Gi", "3Gi", corev1.PersistentVolumeClaimResizing, corev1.PersistentVolumeClaimControllerResizeError), MountedBy: statefulSet},
			{PVC: volumePVC("db-nexus-3", "5Gi", "3Gi", corev1.PersistentVolumeClaimNodeResizeError), MountedBy: statefulSet},
		}

		statuses, err := VolumeStatuses(rendered, testNamespace, volumes, nil)

		require.NoError(t, err)
		assert.Equal(t, []v3beta1.VolumeStatus{
			{ClaimName: "data", Size: resource.MustParse("2Gi"), DesiredSize: resource.MustParse("2Gi"), State: v3beta1.VolumeStatusStateReady, MountedBy: statefulSet},
			{ClaimName: "db-nexus-0", Size: resource.MustParse("3Gi"), DesiredSize: resource.MustParse("5Gi"), State: v3beta1.VolumeStatusStateResizing, MountedBy: statefulSet},
			{ClaimName: "db-nexus-1", Size: resource.MustParse("5Gi"), DesiredSize: resource.MustParse("5Gi"), State: v3beta1.VolumeStatusStateResizing, MountedBy: statefulSet},
			{ClaimName: "db-nexus-2", Size: resource.MustParse("3Gi"), DesiredSize: resource.MustParse("5Gi"), State: v3beta1.VolumeStatusStateResizeFailed, MountedBy: statefulSet},
			{ClaimName: "db-nexus-3", Size: resource.MustParse("3Gi"), DesiredSize: resource.MustParse("5Gi"), State: v3beta1.VolumeStatusStateResizeFailed, MountedBy: statefulSet},
			{ClaimName: "existing", Size: resource.MustParse("1Gi"), State: v3beta1.VolumeStatusStateReady, MountedBy: proxy},
		}, statuses)
	})

	t.Run("reports the failed claims as ResizeFailed", func(t *testing.T) {
		volumes := []Volume{{PVC: volumePVC("data", "1Gi", "1Gi")}, {PVC: volumePVC("db-nexus-0", "5Gi", "5Gi")}}

		statuses, err := VolumeStatuses(rendered, testNamespace, volumes, []string{"data"})

		require.NoError(t, err)
		require.Len(t, statuses, 2)
		assert.Equal(t, v3beta1.VolumeStatusStateResizeFailed, statuses[0].State)
		assert.Equal(t, v3beta1.VolumeStatusStateReady, statuses[1].State)
	})

	t.Run("fails when the desired sizes cannot be read from the rendered chart", func(t *testing.T) {
		withoutRequest := newPVC("data", "2Gi", nil)
		withoutRequest.Spec.Resources.Requests = nil

		_, err := VolumeStatuses(renderedObjects(t, withoutRequest), testNamespace, []Volume{{PVC: volumePVC("data", "2Gi", "2Gi")}}, nil)

		assert.ErrorContains(t, err, "failed to read desired volume sizes")
	})
}
