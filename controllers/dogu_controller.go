package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	appsv1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	RequeueEventReason          = "Requeue"
	ReconcileStartedEventReason = "ReconcileStarted"
)

const (
	ReasonReconcileSuccess = "ReconcileSuccess"
	ReasonReconcileFail    = "ReconcileFail"
	ReasonHasToReconcile   = "HasToReconcile"
)

type DoguReconciler struct {
	client            client.Client
	doguChangeHandler DoguInstallOrChangeUseCase
	doguDeleteHandler DoguDeleteUseCase
	doguInterface     doguInterface
	requeueHandler    RequeueHandler
	externalEvents    <-chan event.TypedGenericEvent[*doguv2.Dogu]
	eventRecorder     eventRecorder
}

func NewDoguEvents() chan event.TypedGenericEvent[*doguv2.Dogu] {
	return make(chan event.TypedGenericEvent[*doguv2.Dogu])
}

func NewDoguEventsIn(channel chan event.TypedGenericEvent[*doguv2.Dogu]) chan<- event.TypedGenericEvent[*doguv2.Dogu] {
	return channel
}

func NewDoguEventsOut(channel chan event.TypedGenericEvent[*doguv2.Dogu]) <-chan event.TypedGenericEvent[*doguv2.Dogu] {
	return channel
}

func NewDoguReconciler(
	k8sClient client.Client,
	doguChangeHandler DoguInstallOrChangeUseCase,
	doguDeleteHandler DoguDeleteUseCase,
	doguInterface doguClient.DoguInterface,
	requeueHandler RequeueHandler,
	externalEvents <-chan event.TypedGenericEvent[*doguv2.Dogu],
	recorder record.EventRecorder,
	manager manager.Manager,
) (*DoguReconciler, error) {
	r := &DoguReconciler{
		client:            k8sClient,
		doguChangeHandler: doguChangeHandler,
		doguDeleteHandler: doguDeleteHandler,
		doguInterface:     doguInterface,
		requeueHandler:    requeueHandler,
		externalEvents:    externalEvents,
		eventRecorder:     recorder,
	}
	err := r.setupWithManager(manager)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *DoguReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	doguResource := &doguv2.Dogu{}
	err := r.client.Get(ctx, req.NamespacedName, doguResource)
	if err != nil {
		return r.requeueHandler.Handle(ctx, doguResource, client.IgnoreNotFound(err), 0)
	}
	r.eventRecorder.Event(doguResource, coreV1.EventTypeNormal, ReconcileStartedEventReason, "reconciliation started")

	var requeueAfter time.Duration
	var cont bool
	if doguResource.GetDeletionTimestamp().IsZero() {
		requeueAfter, cont, err = r.doguChangeHandler.HandleUntilApplied(ctx, doguResource)
	} else {
		requeueAfter, cont, err = r.doguDeleteHandler.HandleUntilApplied(ctx, doguResource)
		err = client.IgnoreNotFound(err)
		if cont {
			return ctrl.Result{}, nil
		}
	}

	getDoguResourceErr := r.client.Get(ctx, req.NamespacedName, doguResource)
	if getDoguResourceErr != nil {
		return r.requeueHandler.Handle(ctx, doguResource, errors.Join(fmt.Errorf("failed to get doguResource %q: %w", req.NamespacedName, getDoguResourceErr), err), 0)
	}

	if requeueAfter != 0 {
		getDoguResourceErr = r.setReadyCondition(ctx, doguResource, metav1.ConditionFalse, ReasonHasToReconcile, fmt.Sprintf("The dogu resource has to be requeued after %d seconds.", requeueAfter))
	} else if err != nil {
		getDoguResourceErr = r.setReadyCondition(ctx, doguResource, metav1.ConditionFalse, ReasonReconcileFail, fmt.Sprintf("The dogu resource has to be requeued because of an error: %q.", err))
	} else if !cont {
		getDoguResourceErr = r.setReadyCondition(ctx, doguResource, metav1.ConditionFalse, ReasonReconcileFail, "The reconcile has been aborted")
	} else {
		getDoguResourceErr = r.setReadyCondition(ctx, doguResource, metav1.ConditionTrue, ReasonReconcileSuccess, "The dogu resource has been reconciled successfully and is ready.")
	}

	errs := errors.Join(getDoguResourceErr, err)

	return r.requeueHandler.Handle(ctx, doguResource, errs, requeueAfter)
}

// setupWithManager sets up the controller with the manager.
// The dogu controller should be triggered when resources on which a dogu cr has an OwnerReference change.
// These resource types are listed here with owns.
// In addition, the dogu reconciler can be triggered via an events channel.
// This is intended, for example, for the GlobalConfigReconciler to reconcile the dogus again.
func (r *DoguReconciler) setupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&doguv2.Dogu{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&coreV1.ConfigMap{}).
		Owns(&coreV1.Secret{}).
		Owns(&coreV1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&coreV1.PersistentVolumeClaim{}).
		Owns(&netv1.NetworkPolicy{}).
		Owns(&coreV1.Pod{}).
		WatchesRawSource(source.Channel(r.externalEvents, &handler.TypedEnqueueRequestForObject[*doguv2.Dogu]{})).
		Complete(r)
}

func (r *DoguReconciler) setReadyCondition(ctx context.Context, doguResource *doguv2.Dogu, status metav1.ConditionStatus, reason, message string) error {
	logger := log.FromContext(ctx)
	condition := metav1.Condition{
		Type:               doguv2.ConditionReady,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}
	meta.SetStatusCondition(&doguResource.Status.Conditions, condition)
	doguResource, err := r.doguInterface.UpdateStatus(ctx, doguResource, metav1.UpdateOptions{})
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to update dogu resource"))
		return err
	}
	logger.Info(fmt.Sprintf("Updated dogu resource successfully!"))
	return nil
}
