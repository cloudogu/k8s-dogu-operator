package controllers

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const globalConfigMapName = "global-config"

type GlobalConfigReconciler struct {
	doguInterface doguInterface
	doguEvents    chan<- event.TypedGenericEvent[*v2.Dogu]
}

func NewGlobalConfigReconciler(
	doguInterface doguClient.DoguInterface,
	doguEvents chan<- event.TypedGenericEvent[*v2.Dogu],
	manager manager.Manager,
) (*GlobalConfigReconciler, error) {
	r := &GlobalConfigReconciler{
		doguInterface: doguInterface,
		doguEvents:    doguEvents,
	}
	err := r.setupWithManager(manager)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *GlobalConfigReconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	doguList, err := r.doguInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, dogu := range doguList.Items {
		r.doguEvents <- event.TypedGenericEvent[*v2.Dogu]{Object: &dogu}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GlobalConfigReconciler) setupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
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
