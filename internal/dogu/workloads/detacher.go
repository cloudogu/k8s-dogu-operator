package workloads

import (
	"context"
	"fmt"
	"slices"

	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
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

// ForeignOwnedClaims reports the claims that unmounting would endanger, as "claim (Kind/name)" per foreign owner.
// Only StatefulSets are deleted with OrphanPropagation, which keeps the claims they own safe; any other owner may delete its claim.
func ForeignOwnedClaims(workloads Workloads, volumes []pvc.Volume, claims []string) []string {
	var foreignOwners []string
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
			if !slices.Contains(allowed, owner.UID) {
				foreignOwners = append(foreignOwners, fmt.Sprintf("%s (%s/%s)", volume.PVC.Name, owner.Kind, owner.Name))
			}
		}
	}

	return foreignOwners
}

// ReplaceStatefulSets deletes the outdated StatefulSets, as Helm cannot upgrade their immutable claim templates; the release recreates them.
func (d *Detacher) ReplaceStatefulSets(ctx context.Context, workloads Workloads, outdatedStatefulSets []types.UID) ([]string, error) {
	var refs []corev1.TypedLocalObjectReference
	// matched by UID, as the checker reads cached and may still see a StatefulSet that Helm has recreated meanwhile
	for _, statefulSet := range workloads.StatefulSets {
		if slices.Contains(outdatedStatefulSets, statefulSet.UID) {
			refs = append(refs, corev1.TypedLocalObjectReference{Kind: "StatefulSet", Name: statefulSet.Name})
		}
	}

	return d.deleteWorkloads(ctx, workloads, refs)
}

// Unmounted is what Unmount deleted in one pass; it is done once all three are empty.
type Unmounted struct {
	Workloads []string
	// Pods are the pods still mounting the claims, NewlyDeletedPods those of them deleted in this pass.
	Pods             []string
	NewlyDeletedPods []string
}

// Unmount deletes the workloads mounting the claims and, once these are gone, the pods still mounting the claims, as orphaned StatefulSets leave their pods running.
func (d *Detacher) Unmount(ctx context.Context, namespace string, workloads Workloads, volumes []pvc.Volume, claims []string) (Unmounted, error) {
	if len(claims) == 0 {
		return Unmounted{}, nil
	}

	var refs []corev1.TypedLocalObjectReference
	for _, volume := range volumes {
		if slices.Contains(claims, volume.PVC.Name) {
			refs = append(refs, volume.MountedBy...)
		}
	}
	deleted, err := d.deleteWorkloads(ctx, workloads, refs)
	if err != nil || len(deleted) > 0 {
		return Unmounted{Workloads: deleted}, err
	}

	pods, newlyDeleted, err := d.deleteMountingPods(ctx, namespace, claims)
	return Unmounted{Pods: pods, NewlyDeletedPods: newlyDeleted}, err
}

func (d *Detacher) deleteWorkloads(ctx context.Context, workloads Workloads, refs []corev1.TypedLocalObjectReference) ([]string, error) {
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

// deleteMountingPods deletes the running pods mounting the claims, found by claim name as they carry no dogu label and may have outlived their workload.
func (d *Detacher) deleteMountingPods(ctx context.Context, namespace string, claims []string) (pods, newlyDeleted []string, err error) {
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
