package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/k8s-apply-lib/apply"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/usecase"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/k8s-registry-lib/repository"
	appsv1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

const k8sDoguOperatorFieldManagerName = "k8s-dogu-operator"

type doguReconciler struct {
	client            client.Client
	doguChangeHandler DoguUsecase
	doguDeleteHandler DoguUsecase
	doguInterface     doguInterface
}

func NewDoguReconciler(
	client client.Client,
	ecosystemClient doguClient.EcoSystemV2Interface,
	operatorConfig *config.OperatorConfig,
	eventRecorder record.EventRecorder,
	doguHealthStatusUpdater health.DoguHealthStatusUpdater,
	availabilityChecker *health.AvailabilityChecker,
) (DoguReconciler, error) {
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

	return &doguReconciler{
		client:            client,
		doguChangeHandler: usecase.NewDoguInstallOrChangeUseCase(client, mgrSet, configRepos, eventRecorder, operatorConfig.Namespace, doguHealthStatusUpdater, doguRestartMgr, availabilityChecker),
		doguDeleteHandler: usecase.NewDoguDeleteUsecase(client, mgrSet, configRepos, operatorConfig),
		doguInterface:     ecosystemClient.Dogus(operatorConfig.Namespace),
	}, nil
}

func (r *doguReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
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
		logger.Error(err, fmt.Sprintf("failed to get doguResource: %s", getDoguResourceErr))
		return ctrl.Result{}, err
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

	if getDoguResourceErr != nil {
		return ctrl.Result{RequeueAfter: requeueAfter}, getDoguResourceErr
	}

	return ctrl.Result{RequeueAfter: requeueAfter}, err
}

// SetupWithManager sets up the controller with the manager.
func (r *doguReconciler) SetupWithManager(mgr ctrl.Manager, externalEvents <-chan event.TypedGenericEvent[*doguv2.Dogu]) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&doguv2.Dogu{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&coreV1.ConfigMap{}).
		Owns(&coreV1.Secret{}).
		Owns(&coreV1.Service{}).
		Owns(&appsv1.Deployment{}).
		Owns(&coreV1.PersistentVolumeClaim{}).
		Owns(&netv1.NetworkPolicy{}).
		Owns(&coreV1.Pod{}).
		WatchesRawSource(source.Channel(externalEvents, &handler.TypedEnqueueRequestForObject[*doguv2.Dogu]{})).
		Complete(r)
}

func (r *doguReconciler) setReadyCondition(ctx context.Context, doguResource *doguv2.Dogu, status metav1.ConditionStatus, reason, message string) error {
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

func createMgrSet(ctx context.Context, restConfig *rest.Config, client client.Client, clientSet kubernetes.Interface, ecosystemClient doguClient.EcoSystemV2Interface, operatorConfig *config.OperatorConfig, configRepos util.ConfigRepositories) (*util.ManagerSet, error) {
	imageGetter := newAdditionalImageGetter(clientSet, operatorConfig.Namespace)
	additionalImageChownInitContainer, err := imageGetter.imageForKey(ctx, config.ChownInitImageConfigmapNameKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get additional images: %w", err)
	}

	additionalExportModeContainer, err := imageGetter.imageForKey(ctx, config.ExporterImageConfigmapNameKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get additional images: %w", err)
	}

	additionalMountsContainer, err := imageGetter.imageForKey(ctx, config.AdditionalMountsInitContainerImageConfigmapNameKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get additional images: %w", err)
	}

	additionalImages := map[string]string{config.ChownInitImageConfigmapNameKey: additionalImageChownInitContainer,
		config.ExporterImageConfigmapNameKey:                      additionalExportModeContainer,
		config.AdditionalMountsInitContainerImageConfigmapNameKey: additionalMountsContainer}

	applier, scheme, err := apply.New(restConfig, k8sDoguOperatorFieldManagerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create K8s applier: %w", err)
	}
	// we need this as we add dogu resource owner-references to every custom object.
	err = doguv2.AddToScheme(scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to add apply scheme: %w", err)
	}

	mgrSet, err := util.NewManagerSet(restConfig, client, clientSet, ecosystemClient, operatorConfig, configRepos, applier, additionalImages)
	if err != nil {
		return nil, fmt.Errorf("could not create manager set: %w", err)
	}
	return mgrSet, err
}

// createConfigRepositories creates the repositories for global, dogu and sensitive dogu configs that are based on
// k8s resources (configmaps / secrets)
func createConfigRepositories(clientSet kubernetes.Interface, namespace string) util.ConfigRepositories {
	configMapClient := clientSet.CoreV1().ConfigMaps(namespace)
	secretsClient := clientSet.CoreV1().Secrets(namespace)

	return util.ConfigRepositories{
		GlobalConfigRepository:  repository.NewGlobalConfigRepository(configMapClient),
		DoguConfigRepository:    repository.NewDoguConfigRepository(configMapClient),
		SensitiveDoguRepository: repository.NewSensitiveDoguConfigRepository(secretsClient),
	}
}
