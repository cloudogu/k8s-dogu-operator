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
// It reconciles a volume when the volume changes and when a dogu StatefulSet changes whose claim templates generate it.
func (r *DoguVolumeLabelReconciler) setupWithManager(mgr ctrlManager) error {
	hasDoguLabel := builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
		_, ok := object.GetLabels()[v3beta1.DoguLabelName]
		return ok
	}))

	return ctrl.NewControllerManagedBy(mgr).
		Named(doguVolumeLabelControllerName).
		For(&corev1.PersistentVolumeClaim{}).
		Watches(&appsv1.StatefulSet{}, handler.EnqueueRequestsFromMapFunc(newDoguVolumeRequestMapper(mgr.GetClient())), hasDoguLabel).
		Complete(r)
}

// Reconcile sets the dogu labels on a volume generated from the claim template of a v3 dogu StatefulSet.
func (r *DoguVolumeLabelReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	volume := &corev1.PersistentVolumeClaim{}
	if err := r.client.Get(ctx, req.NamespacedName, volume); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get volume %q: %w", req.NamespacedName, err)
	}
	if !volume.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}

	doguNames, err := r.matchDogus(ctx, volume)
	if err != nil {
		return reconcile.Result{}, err
	}

	switch len(doguNames) {
	case 0:
		return reconcile.Result{}, nil
	case 1:
		return reconcile.Result{}, r.ensureDoguLabels(ctx, volume, doguNames[0])
	default:
		logging.Info(ctx, "WARNING: volume belongs to more than one dogu; leaving its dogu labels unchanged", "volume", req.NamespacedName, "dogus", doguNames)
		return reconcile.Result{}, nil
	}
}

// matchDogus returns the sorted, distinct names of the dogus whose StatefulSet claim templates generate the volume.
func (r *DoguVolumeLabelReconciler) matchDogus(ctx context.Context, volume *corev1.PersistentVolumeClaim) ([]string, error) {
	statefulSets := &appsv1.StatefulSetList{}
	if err := r.client.List(ctx, statefulSets, client.InNamespace(volume.Namespace), client.HasLabels{v3beta1.DoguLabelName}); err != nil {
		return nil, fmt.Errorf("failed to list dogu statefulsets in namespace %q: %w", volume.Namespace, err)
	}

	doguNames := map[string]struct{}{}
	for _, statefulSet := range statefulSets.Items {
		for _, template := range statefulSet.Spec.VolumeClaimTemplates {
			if pvc.MatchesStatefulSetClaim(volume.Name, template.Name, statefulSet.Name) {
				doguNames[statefulSet.Labels[v3beta1.DoguLabelName]] = struct{}{}
			}
		}
	}
	// Sorted is helpful to not introduce log/status/event flicker
	return slices.Sorted(maps.Keys(doguNames)), nil
}

// ensureDoguLabels patches only the dogu labels of the volume, and only if one of them is missing or wrong.
func (r *DoguVolumeLabelReconciler) ensureDoguLabels(ctx context.Context, volume *corev1.PersistentVolumeClaim, doguName string) error {
	if volume.Labels[v3beta1.DoguLabelName] == doguName && volume.Labels[doguv2.DoguLabelName] == doguName {
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
	return nil
}

// newDoguVolumeRequestMapper creates a mapping from a StatefulSet to the existing volumes generated from its claim templates.
func newDoguVolumeRequestMapper(k8s client.Reader) handler.MapFunc {
	return func(ctx context.Context, object client.Object) []reconcile.Request {
		statefulSet, ok := object.(*appsv1.StatefulSet)
		if !ok || len(statefulSet.Spec.VolumeClaimTemplates) == 0 {
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
}
