package workloads

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Workloads contains the live workloads of a dogu.
type Workloads struct {
	Deployments  []appsv1.Deployment
	StatefulSets []appsv1.StatefulSet
	DaemonSets   []appsv1.DaemonSet
}

// IsEmpty returns true if the dogu has no workloads at all.
func (w Workloads) IsEmpty() bool {
	return len(w.Deployments) == 0 && len(w.StatefulSets) == 0 && len(w.DaemonSets) == 0
}

// Volume is a live PVC together with the workloads whose pod templates mount it.
type Volume struct {
	PVC       corev1.PersistentVolumeClaim
	MountedBy []corev1.TypedLocalObjectReference
}

// Discovery finds the live resources of a dogu.
type Discovery interface {
	// Workloads returns all Deployments, StatefulSets and DaemonSets of the dogu in the given namespace.
	Workloads(ctx context.Context, namespace, doguName string) (Workloads, error)
	// Volumes returns the live PVCs in the given namespace that the workloads mount.
	Volumes(ctx context.Context, namespace string, workloads Workloads) ([]Volume, error)
}

type discovery struct {
	k8s client.Reader
}

// NewDiscovery creates a discovery that reads live resources with the given Kubernetes client.
func NewDiscovery(k8s client.Reader) Discovery {
	return &discovery{k8s: k8s}
}

// Workloads finds the workloads by the label k8s.cloudogu.com/dogu.name.
func (d *discovery) Workloads(ctx context.Context, namespace, doguName string) (Workloads, error) {
	opts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels{v3beta1.DoguLabelName: doguName},
	}

	deployments := &appsv1.DeploymentList{}
	if err := d.k8s.List(ctx, deployments, opts...); err != nil {
		return Workloads{}, fmt.Errorf("failed to list deployments of dogu %q: %w", doguName, err)
	}

	statefulSets := &appsv1.StatefulSetList{}
	if err := d.k8s.List(ctx, statefulSets, opts...); err != nil {
		return Workloads{}, fmt.Errorf("failed to list statefulsets of dogu %q: %w", doguName, err)
	}

	daemonSets := &appsv1.DaemonSetList{}
	if err := d.k8s.List(ctx, daemonSets, opts...); err != nil {
		return Workloads{}, fmt.Errorf("failed to list daemonsets of dogu %q: %w", doguName, err)
	}

	return Workloads{
		Deployments:  deployments.Items,
		StatefulSets: statefulSets.Items,
		DaemonSets:   daemonSets.Items,
	}, nil
}

// Volumes reads mounts from pod and claim templates, not from pods, so a stopped dogu's volumes are found too.
func (d *discovery) Volumes(ctx context.Context, namespace string, workloads Workloads) ([]Volume, error) {
	pvcs := &corev1.PersistentVolumeClaimList{}
	if err := d.k8s.List(ctx, pvcs, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list PVCs in namespace %q: %w", namespace, err)
	}

	var volumes []Volume
	for _, livePVC := range pvcs.Items {
		var mountedBy []corev1.TypedLocalObjectReference
		for _, deployment := range workloads.Deployments {
			if mountsClaim(deployment.Spec.Template, livePVC.Name) {
				mountedBy = append(mountedBy, workloadReference("Deployment", deployment.Name))
			}
		}
		for _, statefulSet := range workloads.StatefulSets {
			if mountsClaim(statefulSet.Spec.Template, livePVC.Name) || hasClaimTemplateFor(statefulSet, livePVC.Name) {
				mountedBy = append(mountedBy, workloadReference("StatefulSet", statefulSet.Name))
			}
		}
		for _, daemonSet := range workloads.DaemonSets {
			if mountsClaim(daemonSet.Spec.Template, livePVC.Name) {
				mountedBy = append(mountedBy, workloadReference("DaemonSet", daemonSet.Name))
			}
		}

		if len(mountedBy) > 0 {
			volumes = append(volumes, Volume{PVC: livePVC, MountedBy: mountedBy})
		}
	}

	return volumes, nil
}

func mountsClaim(template corev1.PodTemplateSpec, claimName string) bool {
	for _, volume := range template.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == claimName {
			return true
		}
	}

	return false
}

func hasClaimTemplateFor(statefulSet appsv1.StatefulSet, pvcName string) bool {
	for _, template := range statefulSet.Spec.VolumeClaimTemplates {
		if pvc.MatchesStatefulSetClaim(pvcName, template.Name, statefulSet.Name) {
			return true
		}
	}

	return false
}

func workloadReference(kind, name string) corev1.TypedLocalObjectReference {
	return corev1.TypedLocalObjectReference{APIGroup: ptr.To(appsv1.GroupName), Kind: kind, Name: name}
}
