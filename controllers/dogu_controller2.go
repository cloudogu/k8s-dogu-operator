package controllers

import (
	"context"
	"fmt"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/usecase"
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
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ReasonReconcileSuccess = "ReconcileSuccess"
	ReasonReconcileFail    = "ReconcileFail"
	ReasonHasToReconcile   = "HasToReconcile"
)

type doguReconciler2 struct {
	client            client.Client
	doguChangeHandler DoguUsecase
	doguDeleteHandler DoguUsecase
	doguInterface     doguInterface
}

func NewDoguReconciler2(client client.Client, ecosystemClient doguClient.EcoSystemV2Interface, operatorConfig *config.OperatorConfig, eventRecorder record.EventRecorder, doguHealthStatusUpdater health.DoguHealthStatusUpdater, availabilityChecker *health.AvailabilityChecker) (*doguReconciler2, error) {
	ctx := context.Background()
	restConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}

	clientSet, err := clientSetGetter(restConfig)
	if err != nil {
		return nil, err
	}

	configRepos := createConfigRepositories(clientSet, operatorConfig.Namespace)
	// At this point, the operator's client is only ready AFTER the operator's Start(...) was called.
	// Instead we must use our own client to avoid an immediate cache error: "the cache is not started, can not read objects"
	mgrSet, err := createMgrSet(ctx, restConfig, client, clientSet, ecosystemClient, operatorConfig, configRepos)
	if err != nil {
		return nil, err
	}

	doguRestartMgr := NewDoguRestartManager(mgrSet.EcosystemClient, clientSet, client, operatorConfig.Namespace)

	return &doguReconciler2{
		client:            client,
		doguChangeHandler: usecase.NewDoguInstallOrChangeUseCase(client, mgrSet, configRepos, eventRecorder, operatorConfig.Namespace, doguHealthStatusUpdater, doguRestartMgr, availabilityChecker),
		doguDeleteHandler: usecase.NewDoguDeleteUsecase(client, mgrSet, configRepos, operatorConfig),
		doguInterface:     ecosystemClient.Dogus(operatorConfig.Namespace),
	}, nil
}

func (r *doguReconciler2) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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

	if err2 := r.client.Get(ctx, req.NamespacedName, doguResource); err2 != nil {
		logger.Error(err, fmt.Sprintf("failed to get doguResource: %s", err2))
		return ctrl.Result{}, err
	}

	if requeueAfter != 0 {
		err = r.setReadyCondition(ctx, doguResource, metav1.ConditionFalse, ReasonHasToReconcile, fmt.Sprintf("The dogu resource has to be requeued after %d seconds.", requeueAfter))
	} else if err != nil {
		err = r.setReadyCondition(ctx, doguResource, metav1.ConditionFalse, ReasonReconcileFail, fmt.Sprintf("The dogu resource has to be requeued because of an error: %q.", err))
	} else if !cont {
		err = r.setReadyCondition(ctx, doguResource, metav1.ConditionFalse, ReasonReconcileFail, "The reconcile has been aborted")
	} else {
		err = r.setReadyCondition(ctx, doguResource, metav1.ConditionTrue, ReasonReconcileSuccess, "The dogu resource has been reconciled successfully and is ready.")
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, err
}

// SetupWithManager sets up the controller with the manager.
func (r *doguReconciler2) SetupWithManager(mgr ctrl.Manager, externalEvents <-chan event.TypedGenericEvent[*doguv2.Dogu]) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&doguv2.Dogu{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&coreV1.ConfigMap{}).
		Owns(&coreV1.Secret{}).
		Owns(&coreV1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&coreV1.PersistentVolumeClaim{}).
		Owns(&netv1.NetworkPolicy{}).
		WatchesRawSource(source.Channel(externalEvents, &handler.TypedEnqueueRequestForObject[*doguv2.Dogu]{})).
		Complete(r)
}

func getReadyCondition(doguResource *doguv2.Dogu) *metav1.Condition {
	return meta.FindStatusCondition(doguResource.Status.Conditions, doguv2.ConditionReady)
}

func (r *doguReconciler2) setReadyCondition(ctx context.Context, doguResource *doguv2.Dogu, status metav1.ConditionStatus, reason, message string) error {
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
