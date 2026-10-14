package controllers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/restart"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const doguRestartV3ControllerName = "dogu-v3-restart"

// busyDoguReasons are the reasons of the Ready condition of a v3 dogu while an operation is in progress.
// A restart waits for these operations to finish, so that it does not roll out the workloads in the middle of them.
var busyDoguReasons = []string{
	v3beta1.ReasonInstalling,
	v3beta1.ReasonUpgrading,
	v3beta1.ReasonResizingPVC,
	v3beta1.ReasonChangingExportMode,
	v3beta1.ReasonStarting,
	v3beta1.ReasonStopping,
	v3beta1.ReasonDeleting,
}

// doguRestarter restarts the workloads of a dogu.
type doguRestarter interface {
	// Apply sets the restart annotation on all workloads of the dogu and returns true if the restart is superseded by a newer one.
	Apply(ctx context.Context, namespace, doguName string, restartAt time.Time) (superseded bool, err error)
	// RolledOut returns true if all workloads of the dogu are rolled out after the restart.
	RolledOut(ctx context.Context, namespace, doguName string, restartAt time.Time) (bool, string, error)
}

// DoguRestartV3Reconciler restarts v3 dogus by setting the restart annotation in the pod templates of their workloads.
// It does not change the HelmRelease of the dogu, so a restart creates no new Helm revision.
// Restarts of v3 dogus are tracked by the conditions Restarted and Ready, restarts of v2 dogus by their phase in the
// DoguRestartReconciler.
type DoguRestartV3Reconciler struct {
	client           client.Client
	restarter        doguRestarter
	garbageCollector DoguRestartGarbageCollector
	recorder         eventRecorder
}

// NewDoguRestartV3Reconciler creates the restart reconciler for v3 dogus and registers it with the manager if dogu v3 is enabled.
func NewDoguRestartV3Reconciler(
	k8sClient client.Client,
	recorder record.EventRecorder,
	gc DoguRestartGarbageCollector,
	manager manager.Manager,
	config *config.OperatorConfig,
) (*DoguRestartV3Reconciler, error) {
	r := newDoguRestartV3Reconciler(k8sClient, restart.NewRestarter(k8sClient), gc, recorder)
	if !config.DoguV3Enabled {
		return r, nil
	}

	if err := r.setupWithManager(manager); err != nil {
		return nil, fmt.Errorf("failed to setup %s controller: %w", doguRestartV3ControllerName, err)
	}
	return r, nil
}

func newDoguRestartV3Reconciler(k8sClient client.Client, restarter doguRestarter, gc DoguRestartGarbageCollector, recorder eventRecorder) *DoguRestartV3Reconciler {
	return &DoguRestartV3Reconciler{
		client:           k8sClient,
		restarter:        restarter,
		garbageCollector: gc,
		recorder:         recorder,
	}
}

// setupWithManager registers the controller with the manager.
// It reconciles a restart when it is created and when one of the workloads of its dogu changes, so that it notices
// when the rollout is finished. Workloads are mapped to their restarts by the dogu name label.
func (r *DoguRestartV3Reconciler) setupWithManager(mgr ctrlManager) error {
	hasDoguLabel := builder.WithPredicates(predicate.NewPredicateFuncs(func(object client.Object) bool {
		_, ok := object.GetLabels()[v3beta1.DoguLabelName]
		return ok
	}))

	toRestarts := handler.EnqueueRequestsFromMapFunc(newDoguRestartRequestMapper(mgr.GetClient()))

	return ctrl.NewControllerManagedBy(mgr).
		Named(doguRestartV3ControllerName).
		For(&doguv2.DoguRestart{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&appsv1.Deployment{}, toRestarts, hasDoguLabel).
		Watches(&appsv1.StatefulSet{}, toRestarts, hasDoguLabel).
		Watches(&appsv1.DaemonSet{}, toRestarts, hasDoguLabel).
		Complete(r)
}

// Reconcile restarts the workloads of a v3 dogu and tracks the progress in the conditions of the restart.
func (r *DoguRestartV3Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	doguRestart := &doguv2.DoguRestart{}
	if err := r.client.Get(ctx, req.NamespacedName, doguRestart); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// restarts with a phase belong to v2 dogus
	if doguRestart.Status.Phase != doguv2.RestartStatusPhaseNew {
		return reconcile.Result{}, nil
	}

	// a terminal restart never changes the workloads again, e.g. if a workload was recreated without the annotation
	if restart.IsTerminal(doguRestart) {
		return reconcile.Result{}, r.collectGarbage(ctx, doguRestart)
	}

	dogu := &v3beta1.Dogu{}
	if err := r.client.Get(ctx, client.ObjectKey{Namespace: doguRestart.Namespace, Name: doguRestart.Spec.DoguName}, dogu); err != nil {
		if !apierrors.IsNotFound(err) {
			return reconcile.Result{}, fmt.Errorf("failed to get dogu %q of dogu restart %q: %w", doguRestart.Spec.DoguName, req.NamespacedName, err)
		}

		// without a dogu the api version is unknown, so new restarts are left to the DoguRestartReconciler.
		// The dogu may only be missing in the cache, while the DoguRestartReconciler reads it from the API and ignores
		// the restart of the v3 dogu. Check again until the cache contains the dogu or the restart has a phase.
		// TODO when v2 is removed: This relies on the DoguRestartReconciler to set the phase "dogu not found". Without it,
		//  restarts of missing dogus are requeued forever. Handle this case here instead: confirm the missing dogu with
		//  an uncached read and set the conditions to DoguNotFound.
		if len(doguRestart.Status.Conditions) == 0 {
			return reconcile.Result{RequeueAfter: requeueWaitTimeout}, nil
		}

		return r.finish(ctx, doguRestart, metav1.ConditionFalse, doguv2.DoguRestartReasonDoguNotFound, "Dogu to restart was not found.")
	}

	if dogu.Spec.DoguApiVersion != v3beta1.DoguApiVersionV3 {
		return reconcile.Result{}, nil
	}

	if !isDoguIdle(dogu) {
		message := "Waiting for the current operation of the dogu to finish."
		err := r.updateConditions(ctx, doguRestart,
			newRestartCondition(doguRestart, doguv2.DoguRestartConditionRestarted, metav1.ConditionFalse, doguv2.DoguRestartReasonDoguNotIdle, message),
			newRestartCondition(doguRestart, doguv2.DoguRestartConditionReady, metav1.ConditionFalse, doguv2.DoguRestartReasonDoguNotIdle, message))

		return reconcile.Result{RequeueAfter: requeueWaitTimeout}, err
	}

	return r.restart(ctx, doguRestart, dogu)
}

func (r *DoguRestartV3Reconciler) restart(ctx context.Context, doguRestart *doguv2.DoguRestart, dogu *v3beta1.Dogu) (reconcile.Result, error) {
	// the timestamp is derived from the restart, so reconciling the same restart again never restarts the dogu again
	restartAt := doguRestart.CreationTimestamp.Time

	superseded, err := r.restarter.Apply(ctx, dogu.Namespace, dogu.Spec.Name, restartAt)
	if err != nil {
		r.recorder.Event(doguRestart, corev1.EventTypeWarning, doguv2.DoguRestartReasonPatchFailed, "Failed to restart the workloads of the dogu.")
		statusErr := r.updateConditions(ctx, doguRestart,
			newRestartCondition(doguRestart, doguv2.DoguRestartConditionRestarted, metav1.ConditionFalse, doguv2.DoguRestartReasonPatchFailed, err.Error()))

		return reconcile.Result{}, errors.Join(err, statusErr)
	}

	if superseded {
		return r.finish(ctx, doguRestart, metav1.ConditionTrue, doguv2.DoguRestartReasonSuperseded, "The dogu was already restarted by a newer restart.")
	}

	// emit the Restarting event only once, when the condition becomes true; later reconciles during the rollout pass here again.
	// It has to be checked before updateConditions, which changes the conditions in place.
	firstRestart := !meta.IsStatusConditionTrue(doguRestart.Status.Conditions, doguv2.DoguRestartConditionRestarted)

	// persist Restarted before evaluating the rollout, so it is not lost if the evaluation fails
	restartedMessage := "The restart annotation is set on all workloads of the dogu."
	restarted := newRestartCondition(doguRestart, doguv2.DoguRestartConditionRestarted, metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded, restartedMessage)
	if uErr := r.updateConditions(ctx, doguRestart, restarted); uErr != nil {
		return reconcile.Result{}, uErr
	}
	if firstRestart {
		r.recorder.Event(doguRestart, corev1.EventTypeNormal, "Restarting", restartedMessage)
	}

	if dogu.Spec.Stopped {
		return r.finishReady(ctx, doguRestart, restarted, doguv2.DoguRestartReasonDoguStopped, "The dogu is stopped. Its workloads use the restart once the dogu is started.")
	}

	rolledOut, message, err := r.restarter.RolledOut(ctx, dogu.Namespace, dogu.Spec.Name, restartAt)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to check if workloads are rolled out: %w", err)
	}

	if !rolledOut {
		// changes of the workloads trigger the next reconciliation, so no requeue is necessary
		return reconcile.Result{}, r.updateConditions(ctx, doguRestart, restarted,
			newRestartCondition(doguRestart, doguv2.DoguRestartConditionReady, metav1.ConditionFalse, doguv2.DoguRestartReasonRolloutInProgress, message))
	}

	return r.finishReady(ctx, doguRestart, restarted, doguv2.DoguRestartReasonSucceeded, "All workloads of the dogu are rolled out.")
}

// finish sets both conditions to the same terminal state.
func (r *DoguRestartV3Reconciler) finish(ctx context.Context, doguRestart *doguv2.DoguRestart, status metav1.ConditionStatus, reason, message string) (reconcile.Result, error) {
	restarted := newRestartCondition(doguRestart, doguv2.DoguRestartConditionRestarted, status, reason, message)
	ready := newRestartCondition(doguRestart, doguv2.DoguRestartConditionReady, status, reason, message)

	return r.finishWith(ctx, doguRestart, restarted, ready, reason, message)
}

// finishReady sets the Ready condition to true.
func (r *DoguRestartV3Reconciler) finishReady(ctx context.Context, doguRestart *doguv2.DoguRestart, restarted metav1.Condition, reason, message string) (reconcile.Result, error) {
	ready := newRestartCondition(doguRestart, doguv2.DoguRestartConditionReady, metav1.ConditionTrue, reason, message)
	eventReason := reason
	if reason == doguv2.DoguRestartReasonSucceeded {
		eventReason = "Restarted"
	}

	return r.finishWith(ctx, doguRestart, restarted, ready, eventReason, message)
}

func (r *DoguRestartV3Reconciler) finishWith(ctx context.Context, doguRestart *doguv2.DoguRestart, restarted, ready metav1.Condition, eventReason, message string) (reconcile.Result, error) {
	if err := r.updateConditions(ctx, doguRestart, restarted, ready); err != nil {
		return reconcile.Result{}, err
	}

	eType := corev1.EventTypeNormal
	if ready.Status != metav1.ConditionTrue {
		eType = corev1.EventTypeWarning
	}

	r.recorder.Event(doguRestart, eType, eventReason, message)

	return reconcile.Result{}, r.collectGarbage(ctx, doguRestart)
}

func (r *DoguRestartV3Reconciler) collectGarbage(ctx context.Context, doguRestart *doguv2.DoguRestart) error {
	if err := r.garbageCollector.DoGarbageCollection(ctx, doguRestart.Spec.DoguName); err != nil {
		return fmt.Errorf("failed to do garbage collection for dogu restart %q: %w", client.ObjectKeyFromObject(doguRestart), err)
	}

	return nil
}

func (r *DoguRestartV3Reconciler) updateConditions(ctx context.Context, doguRestart *doguv2.DoguRestart, conditions ...metav1.Condition) error {
	changed := false
	for _, condition := range conditions {
		changed = meta.SetStatusCondition(&doguRestart.Status.Conditions, condition) || changed
	}
	if !changed {
		return nil
	}

	if err := r.client.Status().Update(ctx, doguRestart); err != nil {
		return fmt.Errorf("failed to update conditions of dogu restart %q: %w", client.ObjectKeyFromObject(doguRestart), err)
	}

	log.FromContext(ctx).Info("updated conditions of dogu restart", "doguRestart", client.ObjectKeyFromObject(doguRestart))

	return nil
}

func newRestartCondition(doguRestart *doguv2.DoguRestart, conditionType string, status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: doguRestart.Generation,
	}
}

// isDoguIdle returns false while a change of the dogu is not reconciled yet or an operation of the dogu is in progress.
// A failed operation counts as idle, so a dogu can be restarted to recover from it.
func isDoguIdle(dogu *v3beta1.Dogu) bool {
	if dogu.DeletionTimestamp != nil {
		return false
	}

	// a Ready condition from before the latest spec change does not tell whether an operation is pending
	ready := meta.FindStatusCondition(dogu.Status.Conditions, v3beta1.ConditionReady)
	if ready == nil || ready.ObservedGeneration < dogu.Generation {
		return false
	}

	if ready.Status == metav1.ConditionTrue {
		return true
	}

	if slices.Contains(busyDoguReasons, ready.Reason) {
		return false
	}

	return true
}

// newDoguRestartRequestMapper creates a mapping from a workload to the restarts of its v3 dogu that wait for the rollout.
// Other restarts are reconciled on creation or by a requeue, so workload changes are irrelevant for them.
func newDoguRestartRequestMapper(k8s client.Reader) handler.MapFunc {
	return func(ctx context.Context, workload client.Object) []reconcile.Request {
		doguName, ok := workload.GetLabels()[v3beta1.DoguLabelName]
		if !ok {
			return nil
		}

		restarts := &doguv2.DoguRestartList{}
		if err := k8s.List(ctx, restarts, client.InNamespace(workload.GetNamespace())); err != nil {
			log.FromContext(ctx).Error(err, "failed to list dogu restarts to map workload", "workload", workload.GetName(), "dogu", doguName)

			return nil
		}

		var requests []reconcile.Request
		for i := range restarts.Items {
			item := &restarts.Items[i]
			if item.Spec.DoguName == doguName && isWaitingForRollout(item) {
				requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(item)})
			}
		}
		return requests
	}
}

func isWaitingForRollout(doguRestart *doguv2.DoguRestart) bool {
	ready := meta.FindStatusCondition(doguRestart.Status.Conditions, doguv2.DoguRestartConditionReady)
	return ready != nil && ready.Status == metav1.ConditionFalse && ready.Reason == doguv2.DoguRestartReasonRolloutInProgress
}
