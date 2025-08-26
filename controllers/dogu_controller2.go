package controllers

import (
	"context"
	"fmt"
	"time"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/logging"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/usecase"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type doguReconciler2 struct {
	client            client.Client
	doguChangeHandler DoguUsecase
	doguDeleteHandler DoguUsecase
}

func NewDoguReconciler2(client client.Client, ecosystemClient doguClient.EcoSystemV2Interface, operatorConfig *config.OperatorConfig, eventRecorder record.EventRecorder) (*doguReconciler2, error) {
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
	return &doguReconciler2{
		client:            client,
		doguChangeHandler: usecase.NewDoguInstallOrChangeUseCaseEmpty(client, mgrSet, configRepos, eventRecorder, operatorConfig.Namespace),
		doguDeleteHandler: nil,
	}, nil
}

func (r *doguReconciler2) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	doguResource := &doguv2.Dogu{}
	err := r.client.Get(ctx, req.NamespacedName, doguResource)
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get doguResource: %s", err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	var requeueAfter time.Duration
	if doguResource.GetDeletionTimestamp().IsZero() {
		requeueAfter, err = r.doguDeleteHandler.HandleUntilApplied(ctx, doguResource)
	} else {
		requeueAfter, err = r.doguChangeHandler.HandleUntilApplied(ctx, doguResource)
	}
	return ctrl.Result{RequeueAfter: requeueAfter}, err
}

// SetupWithManager sets up the controller with the manager.
func (r *doguReconciler2) SetupWithManager(mgr ctrl.Manager) error {
	var eventFilter predicate.Predicate
	eventFilter = predicate.GenerationChangedPredicate{}
	if logging.CurrentLogLevel == logrus.TraceLevel {
		recorder := mgr.GetEventRecorderFor(k8sDoguOperatorFieldManagerName)
		eventFilter = doguResourceChangeDebugPredicate{recorder: recorder}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&doguv2.Dogu{}).
		// Since we don't want to process dogus with same spec we use a generation change predicate
		// as a filter to reduce the reconcile calls.
		// The predicate implements a function that will be invoked of every update event that
		// the k8s api will fire. On writing the objects spec field the k8s api
		// increments the generation field. The function compares this field from the old
		// and new dogu resource. If they are equal the reconcile loop will not be called.
		WithEventFilter(eventFilter).
		Complete(r)
}
