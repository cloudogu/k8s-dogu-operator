package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	authRegApiV1 "github.com/cloudogu/k8s-auth-registration-lib/api/v1"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	doguClientV2 "github.com/cloudogu/k8s-dogu-lib/v3/client/typed/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	expositionv1 "github.com/cloudogu/k8s-exposition-lib/api/v1"
	warpmenuentryv1 "github.com/cloudogu/k8s-warp-menu-entry-lib/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

// The DoguReconciler knows where the [*doguv2.Dogu] is at all times. It knows this because it knows where it isn't.
// By subtracting where it is from where it isn't, or where it isn't from where it is (whichever is greater), it obtains
// a difference, or deviation. The guidance subsystem uses deviations to generate corrective commands to drive the
// [*doguv2.Dogu] from a position where it is to a position where it isn't, and arriving at a position where it wasn't,
// it now is. Consequently, the position where it is, is now the position that it wasn't, and it follows that the
// position that it was, is now the position that it isn't. In the event that the position that it is in is not the
// position that it wasn't, the system has acquired a variation, the variation being the difference between where the
// [*doguv2.Dogu] is, and where it wasn't. If variation is considered to be a significant factor, it too may be
// corrected by the GEA. However, the DoguReconciler must also know where the [*doguv2.Dogu] was.
// The [*doguv2.Dogu] guidance computer scenario works as follows. Because a variation has modified some of the
// information the [*doguv2.Dogu] has obtained, it is not sure just where it is. However, it is sure where it isn't,
// within reason, and it knows where it was. It now subtracts where it should be from where it wasn't, or vice-versa,
// and by differentiating this from the algebraic sum of where it shouldn't be, and where it was, it is able to obtain
// the deviation and its variation, which is called error.
type DoguReconciler struct {
	client                  client.Client
	doguChangeHandler       DoguInstallOrChangeUseCase
	doguDeleteHandler       DoguDeleteUseCase
	doguInterface           doguInterface
	requeueHandlerV2        RequeueHandlerV2
	requeueHandlerV3        RequeueHandlerV3
	externalEvents          <-chan event.TypedGenericEvent[*doguv2.Dogu]
	eventRecorder           eventRecorder
	authRegistrationEnabled bool
	expositionEnabled       bool
	warpMenuEntryEnabled    bool
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

// NewDoguReconciler creates the component necessary for applying the desired state of the dogu.
func NewDoguReconciler(
	k8sClient client.Client,
	doguChangeHandler DoguInstallOrChangeUseCase,
	doguDeleteHandler DoguDeleteUseCase,
	doguInterface doguClientV2.DoguInterface,
	requeueHandlerV2 RequeueHandlerV2,
	requeueHandlerV3 RequeueHandlerV3,
	externalEvents <-chan event.TypedGenericEvent[*doguv2.Dogu],
	recorder record.EventRecorder,
	manager manager.Manager,
	config *config.OperatorConfig,
) (*DoguReconciler, error) {
	r := &DoguReconciler{
		client:                  k8sClient,
		doguChangeHandler:       doguChangeHandler,
		doguDeleteHandler:       doguDeleteHandler,
		doguInterface:           doguInterface,
		requeueHandlerV2:        requeueHandlerV2,
		requeueHandlerV3:        requeueHandlerV3,
		externalEvents:          externalEvents,
		eventRecorder:           recorder,
		authRegistrationEnabled: config.AuthRegistrationEnabled,
		expositionEnabled:       config.ExpositionEnabled,
		warpMenuEntryEnabled:    config.WarpMenuEntryEnabled,
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
		if apierrors.IsNotFound(err) {
			// the dogu resource is gone; there is nothing left to reconcile or to requeue.
			return ctrl.Result{}, nil
		}
		return r.requeueHandlerV2.Handle(ctx, doguResource, err, 0)
	}

	if !doguResource.IsV2() {
		log.FromContext(ctx).Error(fmt.Errorf("dogu api version %q is not v2", req.NamespacedName), "the operator currently only supports v2 dogus.")
		return ctrl.Result{}, nil
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
		return r.requeueHandlerV2.Handle(ctx, doguResource, errors.Join(fmt.Errorf("failed to get doguResource %q: %w", req.NamespacedName, getDoguResourceErr), err), 0)
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

	if !doguResource.IsV2() {
		v3Dogu := &v3beta1.Dogu{}
		err := doguResource.ConvertTo(v3Dogu)
		if err != nil {
			log.FromContext(ctx).Error(err, "failed to convert v2 dogu to v3 dogu", "dogu", doguResource.Spec.Name)
			// do not reconcile, end here and have your admin look into the problem
			return ctrl.Result{}, nil
		}
		return r.requeueHandlerV3.Handle(ctx, v3Dogu, errs, requeueAfter)
	}

	return r.requeueHandlerV2.Handle(ctx, doguResource, errs, requeueAfter)
}

// Helper function to simplify mocking for SetupWebhookWithManager
var webhookRegister = func(mgr ctrlManager) error {
	err := (&v3beta1.Dogu{}).SetupWebhookWithManager(mgr)
	if err != nil {
		return fmt.Errorf("failed to setup dogu webhook with manager: %w", err)
	}
	return nil
}

// setupWithManager sets up the controller with the manager.
// The dogu controller should be triggered when resources on which a dogu cr has an OwnerReference change.
// These resource types are listed here with owns.
// In addition, the dogu reconciler can be triggered via an events channel.
// This is intended, for example, for the GlobalConfigReconciler to reconcile the dogus again.
func (r *DoguReconciler) setupWithManager(mgr ctrlManager) error {
	// register webhook server for roundtrip conversion (v2, v3beta1)
	if err := webhookRegister(mgr); err != nil {
		return err
	}

	controllerBuilder := ctrl.NewControllerManagedBy(mgr).
		For(&doguv2.Dogu{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&coreV1.ConfigMap{}).
		Owns(&coreV1.Secret{}).
		Owns(&coreV1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&coreV1.PersistentVolumeClaim{}).
		Owns(&netv1.NetworkPolicy{}).
		Owns(&coreV1.Pod{}).
		WatchesRawSource(source.Channel(r.externalEvents, &handler.TypedEnqueueRequestForObject[*doguv2.Dogu]{}))
	if r.authRegistrationEnabled {
		controllerBuilder = controllerBuilder.Owns(&authRegApiV1.AuthRegistration{})
	}
	if r.expositionEnabled {
		controllerBuilder = controllerBuilder.Owns(&expositionv1.Exposition{})
	}
	if r.warpMenuEntryEnabled {
		controllerBuilder = controllerBuilder.Owns(&warpmenuentryv1.WarpMenuEntry{})
	}
	return controllerBuilder.Complete(r)
}

func (r *DoguReconciler) setReadyCondition(ctx context.Context, doguResource *doguv2.Dogu, status metav1.ConditionStatus, reason, message string) error {
	logger := log.FromContext(ctx)
	condition := metav1.Condition{
		Type:               doguv2.ConditionReady,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: doguResource.Generation,
	}

	var err error
	updatedDoguResource, err := r.doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status doguv2.DoguStatus) doguv2.DoguStatus {
		meta.SetStatusCondition(&status.Conditions, condition)
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		logger.Error(err, "Failed to update dogu resource")
		return err
	}
	*doguResource = *updatedDoguResource
	logger.Info("Updated dogu resource successfully!")
	return nil
}
