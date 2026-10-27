package workloads

import (
	"context"
	"fmt"
	"slices"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Detacher removes the workloads and pods that mount volumes, without deleting a volume.
// It identifies workloads and owners by UID, as Helm may recreate a deleted workload under the same name.
type Detacher struct {
	// reader lists pods uncached, so a lagging cache cannot hide a mounting pod.
	reader client.Reader
	writer client.Writer
}

func NewDetacher(reader client.Reader, writer client.Writer) *Detacher {
	return &Detacher{reader: reader, writer: writer}
}

// ForeignOwnedClaims finds claims that could be deleted during the resize process. Only StatefulSets are deleted with
// OrphanPropagation, which keeps their owned claims safe. Other unexpectedly owned claims have to be protected and get
// reported by this function.
func ForeignOwnedClaims(workloads Workloads, volumes []Volume, claims []string) (failedClaims, foreignOwners []string) {
	for _, volume := range volumes {
		if !slices.Contains(claims, volume.PVC.Name) {
			continue
		}
		var allowed []types.UID
		for _, ref := range volume.MountedBy {
			if workload, _ := workloads.find(ref); ref.Kind == "StatefulSet" && workload != nil {
				allowed = append(allowed, workload.GetUID())
			}
		}
		for _, owner := range volume.PVC.OwnerReferences {
			if slices.Contains(allowed, owner.UID) {
				continue
			}
			if !slices.Contains(failedClaims, volume.PVC.Name) {
				failedClaims = append(failedClaims, volume.PVC.Name)
			}
			foreignOwners = append(foreignOwners, fmt.Sprintf("%s (%s/%s)", volume.PVC.Name, owner.Kind, owner.Name))
		}
	}

	return failedClaims, foreignOwners
}

// DeleteWorkloads deletes the workloads mounting the claims plus the outdated StatefulSets, as Helm cannot upgrade their immutable claim templates.
func (d *Detacher) DeleteWorkloads(ctx context.Context, workloads Workloads, volumes []Volume, claims []string, outdatedStatefulSets []types.UID) ([]string, error) {
	var refs []corev1.TypedLocalObjectReference
	for _, volume := range volumes {
		if slices.Contains(claims, volume.PVC.Name) {
			refs = append(refs, volume.MountedBy...)
		}
	}
	// matched by UID, as the checker reads cached and may still see a StatefulSet that Helm has recreated meanwhile
	for _, statefulSet := range workloads.StatefulSets {
		if slices.Contains(outdatedStatefulSets, statefulSet.UID) {
			refs = append(refs, corev1.TypedLocalObjectReference{Kind: "StatefulSet", Name: statefulSet.Name})
		}
	}

	var deleted []string
	for _, ref := range refs {
		name := ref.Kind + "/" + ref.Name
		if slices.Contains(deleted, name) {
			continue
		}
		workload, propagation := workloads.find(ref)
		if workload == nil {
			continue
		}
		uid := workload.GetUID()
		err := d.writer.Delete(ctx, workload, client.Preconditions{UID: &uid}, client.PropagationPolicy(propagation))
		if err != nil && !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to delete %s: %w", name, err)
		}
		deleted = append(deleted, name)
	}

	return deleted, nil
}

// DeleteMountingPods deletes the running pods mounting the claims, found by claim name as they carry no dogu label and may have outlived their workload.
func (d *Detacher) DeleteMountingPods(ctx context.Context, namespace string, claims []string) (pods, newlyDeleted []string, err error) {
	if len(claims) == 0 {
		return nil, nil, nil
	}

	podList := &corev1.PodList{}
	if err := d.reader.List(ctx, podList, client.InNamespace(namespace)); err != nil {
		return nil, nil, fmt.Errorf("failed to list pods in namespace %q: %w", namespace, err)
	}

	for i := range podList.Items {
		pod := &podList.Items[i]
		// skip finished pods to keep the logs
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed || !podMountsAny(pod, claims) {
			continue
		}
		if err := d.writer.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
			return nil, nil, fmt.Errorf("failed to delete pod %q: %w", pod.Name, err)
		}
		pods = append(pods, pod.Name)
		if pod.DeletionTimestamp == nil {
			newlyDeleted = append(newlyDeleted, pod.Name)
		}
	}

	return pods, newlyDeleted, nil
}

// find returns the workload a volume reference points to, along with the DeletionPropagation;
// StatefulSets are orphaned so their PVCs survive.
func (w Workloads) find(ref corev1.TypedLocalObjectReference) (client.Object, metav1.DeletionPropagation) {
	switch ref.Kind {
	case "Deployment":
		for i := range w.Deployments {
			if w.Deployments[i].Name == ref.Name {
				return &w.Deployments[i], metav1.DeletePropagationBackground
			}
		}
	case "DaemonSet":
		for i := range w.DaemonSets {
			if w.DaemonSets[i].Name == ref.Name {
				return &w.DaemonSets[i], metav1.DeletePropagationBackground
			}
		}
	case "StatefulSet":
		for i := range w.StatefulSets {
			if w.StatefulSets[i].Name == ref.Name {
				return &w.StatefulSets[i], metav1.DeletePropagationOrphan
			}
		}
	}

	return nil, ""
}

func podMountsAny(pod *corev1.Pod, claims []string) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil && slices.Contains(claims, volume.PersistentVolumeClaim.ClaimName) {
			return true
		}
	}

	return false
}
