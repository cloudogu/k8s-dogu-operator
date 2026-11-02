package pvc

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Volume is a live PVC together with the workloads whose pod templates mount it.
type Volume struct {
	PVC       corev1.PersistentVolumeClaim
	MountedBy []corev1.TypedLocalObjectReference
}

// VolumeStatuses reports the volumes sorted by claim name, with the rendered size as desired size and failedClaims as ResizeFailed.
func VolumeStatuses(renderedChart []*unstructured.Unstructured, namespace string, volumes []Volume, failedClaims []string) ([]v3beta1.VolumeStatus, error) {
	livePVCs := make([]corev1.PersistentVolumeClaim, 0, len(volumes))
	for _, volume := range volumes {
		livePVCs = append(livePVCs, volume.PVC)
	}
	desiredSizes, err := DesiredSizes(renderedChart, namespace, livePVCs)
	if err != nil {
		return nil, fmt.Errorf("failed to read desired volume sizes: %w", err)
	}

	statuses := make([]v3beta1.VolumeStatus, 0, len(volumes))
	for _, volume := range volumes {
		state := volumeState(&volume.PVC)
		if slices.Contains(failedClaims, volume.PVC.Name) {
			state = v3beta1.VolumeStatusStateResizeFailed
		}
		statuses = append(statuses, v3beta1.VolumeStatus{
			ClaimName:   volume.PVC.Name,
			Size:        volume.PVC.Status.Capacity.Storage().DeepCopy(),
			DesiredSize: desiredSizes[client.ObjectKeyFromObject(&volume.PVC)],
			State:       state,
			MountedBy:   volume.MountedBy,
		})
	}
	slices.SortFunc(statuses, func(a, b v3beta1.VolumeStatus) int {
		return strings.Compare(a.ClaimName, b.ClaimName)
	})

	return statuses, nil
}

func volumeState(claim *corev1.PersistentVolumeClaim) v3beta1.VolumeStatusState {
	if HasTrueCondition(claim, corev1.PersistentVolumeClaimControllerResizeError, corev1.PersistentVolumeClaimNodeResizeError) {
		return v3beta1.VolumeStatusStateResizeFailed
	}

	if claim.Spec.Resources.Requests.Storage().Cmp(*claim.Status.Capacity.Storage()) > 0 ||
		HasTrueCondition(claim, corev1.PersistentVolumeClaimResizing, corev1.PersistentVolumeClaimFileSystemResizePending) {
		return v3beta1.VolumeStatusStateResizing
	}

	return v3beta1.VolumeStatusStateReady
}
