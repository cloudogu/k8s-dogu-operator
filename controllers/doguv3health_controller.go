package controllers

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/health"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// DoguHealthReconciler keeps the Healthy condition of v3 dogus in sync with the state of their workloads.
// It is the only writer of the Healthy condition of v3 dogus.
type DoguHealthReconciler struct {
	client   client.Client
	checker  healthChecker
	recorder eventRecorder
}

func NewDoguHealthReconciler(k8sClient client.Client, checker healthChecker, recorder eventRecorder) *DoguHealthReconciler {
	return &DoguHealthReconciler{client: k8sClient, checker: checker, recorder: recorder}
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
		// TODO: Are we fine with a warning here in favor of more readable code?
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
