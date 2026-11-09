package controllers

import (
	"context"
	"fmt"
	"maps"
	"slices"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/pvc"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const doguVolumeLabelControllerName = "dogu-v3-volume-labels"

// DoguVolumeLabelReconciler ensures that the volumes of v3 dogus carry the dogu labels.
type DoguVolumeLabelReconciler struct {
	client client.Client
}

// NewDoguVolumeLabelReconciler creates the volume label reconciler and registers it with the manager if dogu v3 is enabled.
func NewDoguVolumeLabelReconciler(
	k8sClient client.Client,
	manager manager.Manager,
	config *config.OperatorConfig,
) (*DoguVolumeLabelReconciler, error) {
	r := newDoguVolumeLabelReconciler(k8sClient)
	if !config.DoguV3Enabled {
		return r, nil
	}

	if err := r.setupWithManager(manager); err != nil {
		return nil, fmt.Errorf("failed to setup %s controller: %w", doguVolumeLabelControllerName, err)
	}
	return r, nil
}

func newDoguVolumeLabelReconciler(k8sClient client.Client) *DoguVolumeLabelReconciler {
	return &DoguVolumeLabelReconciler{client: k8sClient}
}

// setupWithManager registers the controller with the manager.
// It reconciles a volume when the volume changes and when a dogu workload changes that mounts or generates it.
func (r *DoguVolumeLabelReconciler) setupWithManager(mgr ctrlManager) error {
	doguWorkloadChanges := builder.WithPredicates(doguWorkloadChangedPredicate())
	toVolumes := handler.EnqueueRequestsFromMapFunc(newDoguVolumeRequestMapper(mgr.GetClient()))

	return ctrl.NewControllerManagedBy(mgr).
		Named(doguVolumeLabelControllerName).
		For(&corev1.PersistentVolumeClaim{}).
		Watches(&appsv1.Deployment{}, toVolumes, doguWorkloadChanges).
		Watches(&appsv1.StatefulSet{}, toVolumes, doguWorkloadChanges).
		Watches(&appsv1.DaemonSet{}, toVolumes, doguWorkloadChanges).
		Complete(r)
}

// doguWorkloadChangedPredicate passes dogu workload events that can change volume ownership, dropping status-only updates.
func doguWorkloadChangedPredicate() predicate.Predicate {
	hasDoguLabel := predicate.NewPredicateFuncs(func(object client.Object) bool {
		_, ok := object.GetLabels()[v3beta1.DoguLabelName]
		return ok
	})
	// include LabelChangedPredicate as a safety-net if dogu-label is changed on a workload
	return predicate.And(hasDoguLabel, predicate.Or[client.Object](predicate.GenerationChangedPredicate{}, predicate.LabelChangedPredicate{}))
}

// Reconcile sets the dogu labels on a volume that a v3 dogu workload mounts or generates from a StatefulSet claim template.
func (r *DoguVolumeLabelReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	volume := &corev1.PersistentVolumeClaim{}
	if err := r.client.Get(ctx, req.NamespacedName, volume); err != nil {
		if apierrors.IsNotFound(err) {
			logging.Debug(ctx, "Volume not found; skipping dogu labels", "volume", req.NamespacedName)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get volume %q: %w", req.NamespacedName, err)
	}
	if !volume.DeletionTimestamp.IsZero() {
		logging.Debug(ctx, "Volume is being deleted; skipping dogu labels", "volume", req.NamespacedName)
		return reconcile.Result{}, nil
	}

	doguNames, err := r.matchDogus(ctx, volume)
	if err != nil {
		return reconcile.Result{}, err
	}

	switch len(doguNames) {
	case 0:
		logging.Debug(ctx, "Volume belongs to no dogu; skipping dogu labels", "volume", req.NamespacedName)
		return reconcile.Result{}, nil
	case 1:
		return reconcile.Result{}, r.ensureDoguLabels(ctx, volume, doguNames[0])
	default:
		logging.Info(ctx, "WARNING: volume belongs to more than one dogu; leaving its dogu labels unchanged", "volume", req.NamespacedName, "dogus", doguNames)
		return reconcile.Result{}, nil
	}
}

// matchDogus returns the names of the dogus whose workloads mount the volume or generate it from a StatefulSet claim template.
func (r *DoguVolumeLabelReconciler) matchDogus(ctx context.Context, volume *corev1.PersistentVolumeClaim) ([]string, error) {
	doguWorkloadOpts := []client.ListOption{client.InNamespace(volume.Namespace), client.HasLabels{v3beta1.DoguLabelName}}
	doguNames := map[string]struct{}{}

	statefulSets := &appsv1.StatefulSetList{}
	if err := r.client.List(ctx, statefulSets, doguWorkloadOpts...); err != nil {
		return nil, fmt.Errorf("failed to list dogu statefulsets in namespace %q: %w", volume.Namespace, err)
	}
	for _, statefulSet := range statefulSets.Items {
		if slices.Contains(claimNames(statefulSet.Spec.Template.Spec), volume.Name) {
			doguNames[statefulSet.Labels[v3beta1.DoguLabelName]] = struct{}{}
		}
		for _, template := range statefulSet.Spec.VolumeClaimTemplates {
			if pvc.MatchesStatefulSetClaim(volume.Name, template.Name, statefulSet.Name) {
				doguNames[statefulSet.Labels[v3beta1.DoguLabelName]] = struct{}{}
			}
		}
	}

	deployments := &appsv1.DeploymentList{}
	if err := r.client.List(ctx, deployments, doguWorkloadOpts...); err != nil {
		return nil, fmt.Errorf("failed to list dogu deployments in namespace %q: %w", volume.Namespace, err)
	}
	for _, deployment := range deployments.Items {
		if slices.Contains(claimNames(deployment.Spec.Template.Spec), volume.Name) {
			doguNames[deployment.Labels[v3beta1.DoguLabelName]] = struct{}{}
		}
	}

	daemonSets := &appsv1.DaemonSetList{}
	if err := r.client.List(ctx, daemonSets, doguWorkloadOpts...); err != nil {
		return nil, fmt.Errorf("failed to list dogu daemonsets in namespace %q: %w", volume.Namespace, err)
	}
	for _, daemonSet := range daemonSets.Items {
		if slices.Contains(claimNames(daemonSet.Spec.Template.Spec), volume.Name) {
			doguNames[daemonSet.Labels[v3beta1.DoguLabelName]] = struct{}{}
		}
	}

	return slices.Sorted(maps.Keys(doguNames)), nil
}

// claimNames returns the names of the volumes the pod spec mounts by claimName.
func claimNames(podSpec corev1.PodSpec) []string {
	var names []string
	for _, volume := range podSpec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			names = append(names, volume.PersistentVolumeClaim.ClaimName)
		}
	}
	return names
}

// ensureDoguLabels patches only the dogu labels of the volume, and only if one of them is missing or wrong.
func (r *DoguVolumeLabelReconciler) ensureDoguLabels(ctx context.Context, volume *corev1.PersistentVolumeClaim, doguName string) error {
	if volume.Labels[v3beta1.DoguLabelName] == doguName && volume.Labels[doguv2.DoguLabelName] == doguName {
		logging.Debug(ctx, "Volume dogu labels already correct", "volume", client.ObjectKeyFromObject(volume), "dogu", doguName)
		return nil
	}

	original := volume.DeepCopy()
	if volume.Labels == nil {
		volume.Labels = map[string]string{}
	}
	volume.Labels[v3beta1.DoguLabelName] = doguName
	volume.Labels[doguv2.DoguLabelName] = doguName
	if err := r.client.Patch(ctx, volume, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to patch dogu labels of volume %q: %w", client.ObjectKeyFromObject(volume), err)
	}
	previous := map[string]string{
		v3beta1.DoguLabelName: original.Labels[v3beta1.DoguLabelName],
		doguv2.DoguLabelName:  original.Labels[doguv2.DoguLabelName],
	}
	logging.Info(ctx, "Volume dogu labels set", "volume", client.ObjectKeyFromObject(volume), "dogu", doguName, "previousLabels", previous)
	return nil
}

// newDoguVolumeRequestMapper creates a mapping from a workload to the volumes it mounts and, for a StatefulSet, the existing volumes generated from its claim templates.
func newDoguVolumeRequestMapper(k8s client.Reader) handler.MapFunc {
	return func(ctx context.Context, object client.Object) []reconcile.Request {
		var podSpec corev1.PodSpec
		var requests []reconcile.Request
		switch workload := object.(type) {
		case *appsv1.Deployment:
			podSpec = workload.Spec.Template.Spec
		case *appsv1.StatefulSet:
			podSpec = workload.Spec.Template.Spec
			requests = claimTemplateVolumeRequests(ctx, k8s, workload)
		case *appsv1.DaemonSet:
			podSpec = workload.Spec.Template.Spec
		default:
			return nil
		}

		for _, claimName := range claimNames(podSpec) {
			requests = append(requests, reconcile.Request{Namespace: object.GetNamespace(), Name: claimName})
		}
		return requests
	}
}

// claimTemplateVolumeRequests returns requests for the existing volumes generated from the claim templates of the StatefulSet.
func claimTemplateVolumeRequests(ctx context.Context, k8s client.Reader, statefulSet *appsv1.StatefulSet) []reconcile.Request {
	if len(statefulSet.Spec.VolumeClaimTemplates) == 0 {
		return nil
	}

	volumes := &corev1.PersistentVolumeClaimList{}
	if err := k8s.List(ctx, volumes, client.InNamespace(statefulSet.Namespace)); err != nil {
		// a map function cannot return an error; the event for this statefulset is lost
		logging.Error(ctx, err, "failed to list volumes to map statefulset", "statefulset", statefulSet.Name)
		return nil
	}

	var requests []reconcile.Request
	for _, volume := range volumes.Items {
		for _, template := range statefulSet.Spec.VolumeClaimTemplates {
			if pvc.MatchesStatefulSetClaim(volume.Name, template.Name, statefulSet.Name) {
				requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&volume)})
				break
			}
		}
	}
	return requests
}
