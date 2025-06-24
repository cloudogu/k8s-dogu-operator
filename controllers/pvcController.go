package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	metav1api "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PvcReconciler watches every pvc object with a dogu.name label in the cluster and sets the Min-Data-size condition for the corresponding dogu
type PvcReconciler struct {
	client             K8sClient
	k8sClientSet       ClientSet
	ecoSystemClientSet ecosystemInterface
}

func NewPvcReconciler(client K8sClient, k8sClientSet ClientSet, ecoSystemClientSet ecosystemInterface) *PvcReconciler {
	return &PvcReconciler{
		client:             client,
		k8sClientSet:       k8sClientSet,
		ecoSystemClientSet: ecoSystemClientSet,
	}
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (pr *PvcReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("start reconciling pvc")

	pvc, err := pr.k8sClientSet.CoreV1().PersistentVolumeClaims(request.Namespace).Get(ctx, request.Name, metav1api.GetOptions{})
	if err != nil {
		return finishOrRequeue(logger,
			client.IgnoreNotFound(
				fmt.Errorf("failed to get pvc %q: %w", request.NamespacedName, err),
			),
		)
	}

	doguName := getDoguLabel(pvc)
	logger.Info(fmt.Sprintf("reconciling pvc for %s", doguName))

	dogu, err := pr.ecoSystemClientSet.Dogus(request.Namespace).Get(ctx, doguName, metav1api.GetOptions{})
	if err != nil {
		return finishOrRequeue(logger, fmt.Errorf("failed to get dogu %q: %w", doguName, err))
	}

	err = resource.SetCurrentDataVolumeSize(ctx, pr.ecoSystemClientSet.Dogus(request.Namespace), pr.client, dogu, pvc)
	if err != nil {
		return finishOrRequeue(logger, fmt.Errorf("failed to update data size for pvc %q: %w", request.NamespacedName, err))
	}

	return finishOperation()
}

func (r *PvcReconciler) getEventFilter() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.TypedCreateEvent[client.Object]) bool {
			return false
		},
		DeleteFunc: func(e event.TypedDeleteEvent[client.Object]) bool {
			return false
		},
		GenericFunc: func(e event.TypedGenericEvent[client.Object]) bool {
			return false
		},
		UpdateFunc: func(e event.TypedUpdateEvent[client.Object]) bool {

			if !hasDoguLabel(e.ObjectNew) {
				return false
			}

			if e.ObjectNew.(*v1.PersistentVolumeClaim).Status.Capacity.Storage().Value() != e.ObjectOld.(*v1.PersistentVolumeClaim).Status.Capacity.Storage().Value() {
				return true
			}

			return false
		},
	}
}

// SetupWithManager sets up the controller with the manager.
func (r *PvcReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var eventFilter predicate.Predicate

	eventFilter = r.getEventFilter()

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.PersistentVolumeClaim{}).
		// Only reconcile dogu PVCs whose storage capacity changed.
		WithEventFilter(eventFilter).
		Complete(r)
}
