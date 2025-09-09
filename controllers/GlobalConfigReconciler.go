package controllers

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	client2 "github.com/cloudogu/k8s-dogu-lib/v2/client"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const globalConfigMapName = "global-config"

var clientSetGetter = func(c *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(c)
}

type globalConfigReconciler struct {
	doguRestartManager *doguRestartManager
	configMapInterface configMapInterface
	doguInterface      doguInterface
	podInterface       podInterface
	client             client.Client
	doguEvents         chan<- event.TypedGenericEvent[*v2.Dogu]
}

func NewGlobalConfigReconciler(ecosystemClient client2.EcoSystemV2Interface, client client.Client, namespace string, doguEvents chan<- event.TypedGenericEvent[*v2.Dogu]) (GenericReconciler, error) {
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
		configMapInterface: clientSet.CoreV1().ConfigMaps(namespace),
		doguInterface:      ecosystemClient.Dogus(namespace),
		podInterface:       clientSet.CoreV1().Pods(namespace),
		client:             client,
		doguEvents:         doguEvents,
	}, nil
}

func (r *globalConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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
		deployment, err := dogu.GetDeployment(ctx, r.client)
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return ctrl.Result{}, err
		}
		doguLastStartingTime, err := r.getDeploymentLastStartingTime(ctx, deployment)
		if err != nil {
			return ctrl.Result{}, err
		}
		if doguLastStartingTime != nil && doguLastStartingTime.Before(cmLastUpdateTime.Time) {
			err := r.doguRestartManager.RestartDogu(ctx, &dogu)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		r.doguEvents <- event.TypedGenericEvent[*v2.Dogu]{Object: &dogu}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *globalConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
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

func (r *globalConfigReconciler) getDeploymentLastStartingTime(ctx context.Context, deployment *appsv1.Deployment) (*time.Time, error) {
	labelSelector := metav1.FormatLabelSelector(deployment.Spec.Selector)

	pods, err := r.podInterface.List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return nil, err
	}

	var lastTimeStarted *time.Time
	for _, pod := range pods.Items {
		if pod.Status.StartTime != nil {
			startTime := pod.Status.StartTime.Time
			if lastTimeStarted == nil || startTime.After(*lastTimeStarted) {
				lastTimeStarted = &startTime
			}
		}
	}
	return lastTimeStarted, nil
}

func (r *globalConfigReconciler) getConfigMapLastUpdatedTime(cm *corev1.ConfigMap) *metav1.Time {
	timestamp := cm.GetCreationTimestamp()
	latest := &timestamp

	for _, managedFields := range cm.GetManagedFields() {
		if managedFields.Time != nil && managedFields.Time.After(latest.Time) {
			latest = managedFields.Time
		}
	}
	return latest
}
