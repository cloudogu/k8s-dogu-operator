package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/restart"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/logging"
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
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const doguRestartV3ControllerName = "dogu-v3-restart"

// rolloutRequeueInterval re-checks a rollout that produces no further workload events, e.g. when the new pods
// never become ready and the workload status stops changing. Without it such a restart would wait forever.
const rolloutRequeueInterval = 30 * time.Second

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

// restartDecision is what one reconciliation decided for a restart: the conditions to set, the events to emit and
// when to look again. precheck and restart decide, record applies.
type restartDecision struct {
	conditions   []metav1.Condition
	events       []restartEvent
	requeueAfter time.Duration
	err          error
}

// restartEvent is an event of the restart.
type restartEvent struct {
	eventType string
	reason    string
	message   string
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

	dogu, blocked := r.precheck(ctx, doguRestart)
	if blocked != nil {
		if err := r.record(ctx, doguRestart, *blocked); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to record restart decision: %w", err)
		}

		return reconcile.Result{RequeueAfter: blocked.requeueAfter}, nil
	}

	decision := r.restart(ctx, doguRestart, dogu)
	if err := r.record(ctx, doguRestart, decision); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to record restart decision: %w", err)
	}

	return reconcile.Result{RequeueAfter: decision.requeueAfter}, nil
}

// precheck checks whether a dogu needs to be restarted.
// It returns either the dogu to restart or the decision that ends the reconciliation without a restart.
// A restart that is finished or belongs to a v2 dogu gets an empty decision, never nil, because nil means go ahead.
func (r *DoguRestartV3Reconciler) precheck(ctx context.Context, doguRestart *doguv2.DoguRestart) (*v3beta1.Dogu, *restartDecision) {
	if restart.IsTerminal(doguRestart) {
		return nil, &restartDecision{}
	}

	dogu := &v3beta1.Dogu{}
	if err := r.client.Get(ctx, client.ObjectKey{Namespace: doguRestart.Namespace, Name: doguRestart.Spec.DoguName}, dogu); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, &restartDecision{err: fmt.Errorf("failed to get dogu %q of dogu restart %q: %w",
				doguRestart.Spec.DoguName, client.ObjectKeyFromObject(doguRestart), err)}
		}

		return nil, new(doguNotFound(ctx, doguRestart))
	}

	// restarts of v2 dogus are handled by the DoguRestartReconciler
	if dogu.Spec.DoguApiVersion != v3beta1.DoguApiVersionV3 {
		return nil, &restartDecision{}
	}

	if len(doguRestart.Status.Conditions) == 0 {
		logging.Info(ctx, "Dogu restart requested", "dogu", dogu.Name)
	}

	if !isDoguIdle(dogu) {
		const message = "Waiting for the current operation of the dogu to finish."

		return nil, &restartDecision{
			conditions: []metav1.Condition{
				restartedCondition(metav1.ConditionFalse, doguv2.DoguRestartReasonDoguNotIdle, message),
				readyCondition(metav1.ConditionFalse, doguv2.DoguRestartReasonDoguNotIdle, message),
			},
			requeueAfter: requeueWaitTimeout,
		}
	}

	return dogu, nil
}

// doguNotFound decides what happens to a restart whose dogu does not exist.
func doguNotFound(ctx context.Context, doguRestart *doguv2.DoguRestart) restartDecision {
	// Let missing dogus be handled by the DoguRestartReconciler (DoguV2) - it sets the conditions to DoguNotFound.
	if len(doguRestart.Status.Conditions) == 0 {
		return restartDecision{requeueAfter: requeueWaitTimeout}
	}

	logging.Info(ctx, "Dogu restart aborted: dogu not found", "dogu", doguRestart.Spec.DoguName)

	const message = "Dogu to restart was not found."
	ready := readyCondition(metav1.ConditionFalse, doguv2.DoguRestartReasonDoguNotFound, message)

	return restartDecision{
		conditions: []metav1.Condition{
			restartedCondition(metav1.ConditionFalse, doguv2.DoguRestartReasonDoguNotFound, message),
			ready,
		},
		events: []restartEvent{terminalEvent(ready)},
	}
}

// restart sets the restart annotation on the workloads of the dogu and reports how far the restart got.
func (r *DoguRestartV3Reconciler) restart(ctx context.Context, doguRestart *doguv2.DoguRestart, dogu *v3beta1.Dogu) restartDecision {
	// the timestamp is derived from the restart, so reconciling the same restart again never restarts the dogu again
	restartAt := doguRestart.CreationTimestamp.Time

	superseded, err := r.restarter.Apply(ctx, dogu.Namespace, dogu.Spec.Name, restartAt)
	if err != nil {
		return restartDecision{
			conditions: []metav1.Condition{restartedCondition(metav1.ConditionFalse, doguv2.DoguRestartReasonPatchFailed, err.Error())},
			events:     []restartEvent{{corev1.EventTypeWarning, doguv2.DoguRestartReasonPatchFailed, "Failed to restart the workloads of the dogu."}},
			err:        err,
		}
	}

	if superseded {
		logging.Info(ctx, "Dogu restart skipped: superseded by a newer restart", "dogu", dogu.Name)

		const message = "The dogu was already restarted by a newer restart."
		ready := readyCondition(metav1.ConditionTrue, doguv2.DoguRestartReasonSuperseded, message)

		return restartDecision{
			conditions: []metav1.Condition{
				restartedCondition(metav1.ConditionTrue, doguv2.DoguRestartReasonSuperseded, message),
				ready,
			},
			events: []restartEvent{terminalEvent(ready)},
		}
	}

	const restartedMessage = "The restart annotation is set on all workloads of the dogu."
	decision := restartDecision{
		conditions: []metav1.Condition{restartedCondition(metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded, restartedMessage)},
	}

	// emit the Restarted event only once, when the condition becomes true; later reconciles during the rollout pass
	// here again. It has to be read before record updates the conditions.
	if !meta.IsStatusConditionTrue(doguRestart.Status.Conditions, doguv2.DoguRestartConditionRestarted) {
		logging.Info(ctx, "Dogu restart triggered: restart annotation set on all workloads", "dogu", dogu.Name)
		decision.events = append(decision.events, restartEvent{corev1.EventTypeNormal, doguv2.DoguRestartConditionRestarted, restartedMessage})
	}

	if dogu.Spec.Stopped {
		return decision.readyWith(readyCondition(metav1.ConditionTrue, doguv2.DoguRestartReasonDoguStopped,
			"The dogu is stopped. Its workloads use the restart once the dogu is started."))
	}

	rolledOut, message, err := r.restarter.RolledOut(ctx, dogu.Namespace, dogu.Spec.Name, restartAt)
	if err != nil {
		// the Restarted condition is still recorded, so that it is not lost when the rollout cannot be evaluated
		decision.err = fmt.Errorf("failed to check if workloads are rolled out: %w", err)

		return decision
	}

	if !rolledOut {
		// changes of the workloads trigger the next reconciliation; the requeue is only a safety net for
		// a rollout that stops producing them
		decision.conditions = append(decision.conditions,
			readyCondition(metav1.ConditionFalse, doguv2.DoguRestartReasonRolloutInProgress, message))
		decision.requeueAfter = rolloutRequeueInterval

		return decision
	}

	logging.Info(ctx, "Dogu restarted: all workloads rolled out", "dogu", dogu.Name)

	return decision.readyWith(readyCondition(metav1.ConditionTrue, doguv2.DoguRestartReasonSucceeded,
		"All workloads of the dogu are rolled out."))
}

// readyWith finishes the decision with a Ready condition and the event that reports it.
func (d restartDecision) readyWith(ready metav1.Condition) restartDecision {
	d.conditions = append(d.conditions, ready)
	d.events = append(d.events, terminalEvent(ready))

	return d
}

// record applies an decision to the cluster: it persists the conditions, emits the events and collects the garbage of
// a restart that reached a terminal state.
func (r *DoguRestartV3Reconciler) record(ctx context.Context, doguRestart *doguv2.DoguRestart, decision restartDecision) error {
	if decision.err != nil {
		// if a failed reconciliation happened, it should be recorded in any case
		r.emit(doguRestart, decision.events)

		return errors.Join(decision.err, r.updateConditions(ctx, doguRestart, decision.conditions...))
	}

	if err := r.updateConditions(ctx, doguRestart, decision.conditions...); err != nil {
		return err
	}

	r.emit(doguRestart, decision.events)

	// a restart that reached a terminal state might exceed the history limit -> trigger GC
	if restart.IsTerminal(doguRestart) {
		if err := r.garbageCollector.DoGarbageCollection(ctx, doguRestart.Spec.DoguName); err != nil {
			return fmt.Errorf("failed to do garbage collection for dogu restart %q: %w", client.ObjectKeyFromObject(doguRestart), err)
		}
	}

	return nil
}

func (r *DoguRestartV3Reconciler) emit(doguRestart *doguv2.DoguRestart, events []restartEvent) {
	for _, e := range events {
		r.recorder.Event(doguRestart, e.eventType, e.reason, e.message)
	}
}

// terminalEvent reports a finished restart. Its type follows the Ready condition: Normal when the restart
// succeeded, Warning otherwise.
func terminalEvent(ready metav1.Condition) restartEvent {
	eType := corev1.EventTypeNormal
	if ready.Status != metav1.ConditionTrue {
		eType = corev1.EventTypeWarning
	}

	reason := ready.Reason
	if ready.Reason == doguv2.DoguRestartReasonSucceeded {
		reason = doguv2.DoguRestartConditionReady
	}

	return restartEvent{eventType: eType, reason: reason, message: ready.Message}
}

func (r *DoguRestartV3Reconciler) updateConditions(ctx context.Context, doguRestart *doguv2.DoguRestart, conditions ...metav1.Condition) error {
	changed := false
	for _, condition := range conditions {
		condition.ObservedGeneration = doguRestart.Generation
		changed = meta.SetStatusCondition(&doguRestart.Status.Conditions, condition) || changed
	}
	if !changed {
		return nil
	}

	if err := r.client.Status().Update(ctx, doguRestart); err != nil {
		return fmt.Errorf("failed to update conditions of dogu restart %q: %w", client.ObjectKeyFromObject(doguRestart), err)
	}

	logging.Debug(ctx, "updated conditions of dogu restart", "doguRestart", client.ObjectKeyFromObject(doguRestart))

	return nil
}

// restartedCondition builds the Restarted condition, which tells whether the workloads carry the restartAt annotation.
func restartedCondition(status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{Type: doguv2.DoguRestartConditionRestarted, Status: status, Reason: reason, Message: message}
}

// readyCondition builds the Ready condition, which tells whether the restart is finished.
func readyCondition(status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{Type: doguv2.DoguRestartConditionReady, Status: status, Reason: reason, Message: message}
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

	if v3beta1.IsOperationInProgress(ready.Reason) {
		return false
	}

	return true
}

// newDoguRestartRequestMapper creates a mapping from a workload to the restarts of its v3 dogu that wait for the rollout.
func newDoguRestartRequestMapper(k8s client.Reader) handler.MapFunc {
	return func(ctx context.Context, workload client.Object) []reconcile.Request {
		doguName, ok := workload.GetLabels()[v3beta1.DoguLabelName]
		if !ok {
			return nil
		}

		restarts := &doguv2.DoguRestartList{}
		if err := k8s.List(ctx, restarts, client.InNamespace(workload.GetNamespace())); err != nil {
			logging.Error(ctx, err, "failed to list dogu restarts to map workload", "workload", workload.GetName(), "dogu", doguName)

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
	if ready == nil {
		return false
	}

	if ready.Status == metav1.ConditionTrue {
		return false
	}

	return ready.Reason == doguv2.DoguRestartReasonRolloutInProgress
}
