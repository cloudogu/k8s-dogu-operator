package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/usecase"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	appsv1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ReasonReconcileSuccess = "ReconcileSuccess"
	ReasonReconcileFail    = "ReconcileFail"
	ReasonHasToReconcile   = "HasToReconcile"
)

type DoguReconciler struct {
	client            client.Client
	doguChangeHandler DoguUsecase
	doguDeleteHandler DoguUsecase
	doguInterface     doguInterface
	externalEvents    <-chan event.TypedGenericEvent[*doguv2.Dogu]
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
	doguChangeHandler *usecase.DoguInstallOrChangeUseCase,
	doguDeleteHandler *usecase.DoguDeleteUseCase,
	doguInterface doguClient.DoguInterface,
	externalEvents <-chan event.TypedGenericEvent[*doguv2.Dogu],
	manager manager.Manager,
) (*DoguReconciler, error) {
	r := &DoguReconciler{
		client:            k8sClient,
		doguChangeHandler: doguChangeHandler,
		doguDeleteHandler: doguDeleteHandler,
		doguInterface:     doguInterface,
		externalEvents:    externalEvents,
	}
	err := r.setupWithManager(manager)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *DoguReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	doguResource := &doguv2.Dogu{}
	err := r.client.Get(ctx, req.NamespacedName, doguResource)
	if err != nil {
		logger.Error(err, fmt.Sprintf("failed to get doguResource: %s", err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var requeueAfter time.Duration
	var cont bool
	if doguResource.GetDeletionTimestamp().IsZero() {
		requeueAfter, cont, err = r.doguChangeHandler.HandleUntilApplied(ctx, doguResource)
	} else {
		requeueAfter, cont, err = r.doguDeleteHandler.HandleUntilApplied(ctx, doguResource)
	}

	getDoguResourceErr := r.client.Get(ctx, req.NamespacedName, doguResource)
	if getDoguResourceErr != nil {
		return ctrl.Result{}, errors.Join(fmt.Errorf("failed to get doguResource %q: %w", req.NamespacedName, getDoguResourceErr), err)
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

	return ctrl.Result{RequeueAfter: requeueAfter}, errors.Join(getDoguResourceErr, err)
}

// SetupWithManager sets up the controller with the manager.
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
