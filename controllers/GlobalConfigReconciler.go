package controllers

import (
	"context"

	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const globalConfigMapName = "global-config"

type globalConfigReconciler struct {
	doguRestartManager *doguRestartManager
}

func NewGlobalConfigReconciler(ecosystemClient ecosystemInterface, client client.Client, namespace string) (*globalConfigReconciler, error) {
	restConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := clientSetGetter(restConfig)
	if err != nil {
		return nil, err
	}
	return &globalConfigReconciler{
		doguRestartManager: NewDoguRestartManager(ecosystemClient, clientSet, client, namespace),
	}, nil
}

func (r *globalConfigReconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	err := r.doguRestartManager.RestartAllDogus(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *globalConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.ConfigMap{}).
		WithEventFilter(globalConfigPredicate()).
		Complete(r)
}

func globalConfigPredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.TypedCreateEvent[client.Object]) bool {
			return e.Object.GetName() == globalConfigMapName
		},
		DeleteFunc: func(e event.TypedDeleteEvent[client.Object]) bool {
			return e.Object.GetName() == globalConfigMapName
		},
		UpdateFunc: func(e event.TypedUpdateEvent[client.Object]) bool {
			return e.ObjectOld.GetName() == globalConfigMapName
		},
		GenericFunc: func(e event.TypedGenericEvent[client.Object]) bool {
			return e.Object.GetName() == globalConfigMapName
		},
	}
}
