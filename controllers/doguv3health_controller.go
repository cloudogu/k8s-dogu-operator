package controllers

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/health"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const doguHealthControllerName = "dogu-v3-health"

// DoguHealthReconciler keeps the Healthy condition of v3 dogus in sync with the state of their workloads.
// It is the only writer of the Healthy condition of v3 dogus.
type DoguHealthReconciler struct {
	client   client.Client
	checker  healthChecker
	recorder eventRecorder
}

// NewDoguHealthReconciler creates the health reconciler and registers it with the manager if dogu v3 is enabled.
func NewDoguHealthReconciler(
	k8sClient client.Client,
	recorder record.EventRecorder,
	manager manager.Manager,
	config *config.OperatorConfig,
) (*DoguHealthReconciler, error) {
	r := newDoguHealthReconciler(k8sClient, health.NewChecker(workloads.NewDiscovery(k8sClient)), recorder)
	if !config.DoguV3Enabled {
		return r, nil
	}

	if err := r.setupWithManager(manager); err != nil {
		return nil, fmt.Errorf("failed to setup %s controller: %w", doguHealthControllerName, err)
	}
	return r, nil
}

func newDoguHealthReconciler(k8sClient client.Client, checker healthChecker, recorder eventRecorder) *DoguHealthReconciler {
	return &DoguHealthReconciler{client: k8sClient, checker: checker, recorder: recorder}
}

// setupWithManager registers the controller with the manager.
// It reconciles a v3 dogu when the dogu itself is created or its spec changes (e.g. stopped is toggled), and when one
// of its workloads changes. Workloads are mapped to their dogu by the dogu name label.
func (r *DoguHealthReconciler) setupWithManager(mgr ctrlManager) error {
	hasDoguLabel := builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
		_, ok := object.GetLabels()[v3beta1.DoguLabelName]
		return ok
	}))
	toDogu := handler.EnqueueRequestsFromMapFunc(newDoguRequestMapper(mgr.GetClient()))

	return ctrl.NewControllerManagedBy(mgr).
		Named(doguHealthControllerName).
		For(&v3beta1.Dogu{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&appsv1.Deployment{}, toDogu, hasDoguLabel).
		Watches(&appsv1.StatefulSet{}, toDogu, hasDoguLabel).
		Watches(&appsv1.DaemonSet{}, toDogu, hasDoguLabel).
		Complete(r)
}

// Reconcile derives the health of a v3 dogu from its workloads and persists it as Healthy condition.
func (r *DoguHealthReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	// the check runs inside the retry so that it always sees the latest spec
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dogu := &v3beta1.Dogu{}
		if err := r.client.Get(ctx, req.NamespacedName, dogu); err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed to get dogu %q: %w", req.NamespacedName, err)
		}

		if dogu.Spec.DoguApiVersion != v3beta1.DoguApiVersionV3 {
			return nil
		}

		state, err := r.checker.Check(ctx, dogu)
		if err != nil {
			return fmt.Errorf("failed to check health of dogu %q: %w", req.NamespacedName, err)
		}

		condition := toCondition(state, dogu.Generation)
		// send event only if status (not message or reason) changed;
		// a missing condition counts as not healthy, so a dogu that is still starting up emits no warning
		previousStatus := metav1.ConditionFalse
		if previous := meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionHealthy); previous != nil {
			previousStatus = previous.Status
		}
		statusChanged := previousStatus != condition.Status

		if !meta.SetStatusCondition(&dogu.Status.Conditions, condition) {
			return nil
		}

		if err := r.client.Status().Update(ctx, dogu); err != nil {
			return fmt.Errorf("failed to update Healthy condition of dogu %q: %w", req.NamespacedName, err)
		}

		if statusChanged {
			r.recorder.Event(dogu, eventType(condition.Status), condition.Reason, condition.Message)
		}
		return nil
	})

	return reconcile.Result{}, err
}

func toCondition(state health.State, generation int64) metav1.Condition {
	status := metav1.ConditionTrue
	if !state.Healthy {
		status = metav1.ConditionFalse
	}

	return metav1.Condition{
		Type:               v3beta1.ConditionHealthy,
		Status:             status,
		Reason:             state.Reason,
		Message:            state.Message,
		ObservedGeneration: generation,
	}
}

func eventType(status metav1.ConditionStatus) string {
	if status == metav1.ConditionTrue {
		return corev1.EventTypeNormal
	}
	return corev1.EventTypeWarning
}

// newDoguRequestMapper creates a mapping from a workload to the v3 dogus it belongs to.
// v3 workloads carry no owner reference to their dogu. Instead, we use the label k8s.cloudogu.com/dogu.name
// for identification.
func newDoguRequestMapper(k8s client.Reader) handler.MapFunc {
	return func(ctx context.Context, workload client.Object) []reconcile.Request {
		doguName, ok := workload.GetLabels()[v3beta1.DoguLabelName]
		if !ok {
			return nil
		}

		dogus := &v3beta1.DoguList{}
		if err := k8s.List(ctx, dogus, client.InNamespace(workload.GetNamespace())); err != nil {
			// a map function cannot return an error; the event for this workload is lost
			log.FromContext(ctx).Error(err, "failed to list dogus to map workload", "workload", workload.GetName(), "dogu", doguName)
			return nil
		}

		var requests []reconcile.Request
		for _, dogu := range dogus.Items {
			if dogu.Spec.DoguApiVersion == v3beta1.DoguApiVersionV3 && dogu.Spec.Name == doguName {
				requests = append(requests, reconcile.Request{Namespace: dogu.Namespace, Name: dogu.Name})
			}
		}
		return requests
	}
}
