package controllers

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	managers "github.com/cloudogu/k8s-dogu-operator/v3/controllers/manager"

	mgr "github.com/cloudogu/k8s-dogu-operator/v3/controllers/manager"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const globalConfigMapName = "global-config"

type GlobalConfigReconciler struct {
	doguRestartManager doguRestartManager
	configMapInterface configMapInterface
	doguInterface      doguInterface
	doguEvents         chan<- event.TypedGenericEvent[*v2.Dogu]
	deploymentManager  deploymentManager
}

func NewGlobalConfigReconciler(
	doguRestartManager managers.DoguRestartManager,
	configMapInterface v1.ConfigMapInterface,
	doguInterface doguClient.DoguInterface,
	doguEvents chan<- event.TypedGenericEvent[*v2.Dogu],
	manager manager.Manager,
	deploymentManager mgr.DeploymentManager,
) (*GlobalConfigReconciler, error) {
	r := &GlobalConfigReconciler{
		doguRestartManager: doguRestartManager,
		configMapInterface: configMapInterface,
		doguInterface:      doguInterface,
		doguEvents:         doguEvents,
		deploymentManager:  deploymentManager,
	}
	err := r.setupWithManager(manager)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *GlobalConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	cm, err := r.configMapInterface.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		logger.Error(err, fmt.Sprintf("failed to get doguResource: %s", err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	cmLastUpdateTime := r.getConfigMapLastUpdatedTime(cm)

	doguList, err := r.doguInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, dogu := range doguList.Items {
		var doguLastStartingTime *time.Time
		doguLastStartingTime, err = r.deploymentManager.GetLastStartingTime(ctx, dogu.Name)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return ctrl.Result{}, err
		}
		if doguLastStartingTime != nil && doguLastStartingTime.Before(cmLastUpdateTime.Time) {
			err = r.doguRestartManager.RestartDogu(ctx, &dogu)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

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

func (r *GlobalConfigReconciler) getConfigMapLastUpdatedTime(cm *corev1.ConfigMap) *metav1.Time {
	timestamp := cm.GetCreationTimestamp()
	latest := &timestamp

	for _, managedFields := range cm.GetManagedFields() {
		if managedFields.Time != nil && managedFields.Time.After(latest.Time) {
			latest = managedFields.Time
		}
	}
	return latest
}
