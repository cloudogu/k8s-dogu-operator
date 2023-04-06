package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/annotation"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/logging"
	"github.com/cloudogu/k8s-dogu-operator/controllers/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"

	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const operatorEventReason = "OperationThresholding"

const (
	InstallEventReason        = "Installation"
	ErrorOnInstallEventReason = "ErrInstallation"
)
const (
	DeinstallEventReason      = "Deinstallation"
	ErrorDeinstallEventReason = "ErrDeinstallation"
)

const (
	SupportEventReason        = "Support"
	ErrorOnSupportEventReason = "ErrSupport"
)

const (
	RequeueEventReason        = "Requeue"
	ErrorOnRequeueEventReason = "ErrRequeue"
)

const handleRequeueErrMsg = "failed to handle requeue: %w"

type operation string

const (
	Install                            = operation("Install")
	Upgrade                            = operation("Upgrade")
	Delete                             = operation("Delete")
	Ignore                             = operation("Ignore")
	ExpandVolume                       = operation("ExpandVolume")
	ChangeAdditionalIngressAnnotations = operation("ChangeAdditionalIngressAnnotations")
)

// doguReconciler reconciles a Dogu object
type doguReconciler struct {
	client             client.Client
	doguManager        cloudogu.DoguManager
	doguRequeueHandler requeueHandler
	recorder           record.EventRecorder
	fetcher            cloudogu.LocalDoguFetcher
}

// requeueHandler abstracts the process to decide whether a requeue process should be done based on received errors.
type requeueHandler interface {
	// Handle takes an error and handles the requeue process for the current dogu operation.
	Handle(ctx context.Context, contextMessage string, doguResource *k8sv1.Dogu, err error, onRequeue func(dogu *k8sv1.Dogu)) (result ctrl.Result, requeueErr error)
}

// NewDoguReconciler creates a new reconciler instance for the dogu resource
func NewDoguReconciler(client client.Client, doguManager cloudogu.DoguManager, eventRecorder record.EventRecorder, namespace string, localRegistry registry.DoguRegistry) (*doguReconciler, error) {
	doguRequeueHandler, err := NewDoguRequeueHandler(client, eventRecorder, namespace)
	if err != nil {
		return nil, err
	}

	localDoguFetcher := cesregistry.NewLocalDoguFetcher(localRegistry)
	return &doguReconciler{
		client:             client,
		doguManager:        doguManager,
		doguRequeueHandler: doguRequeueHandler,
		recorder:           eventRecorder,
		fetcher:            localDoguFetcher,
	}, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *doguReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	doguResource := &k8sv1.Dogu{}
	err := r.client.Get(ctx, req.NamespacedName, doguResource)
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get doguResource: %s", err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info(fmt.Sprintf("Dogu %s/%s has been found", doguResource.Namespace, doguResource.Name))

	if doguResource.Status.Status != k8sv1.DoguStatusNotInstalled {
		supportResult, err := r.handleSupportMode(ctx, doguResource)
		if supportResult != nil {
			return *supportResult, err
		}
	}

	requiredOperation, err := r.evaluateRequiredOperation(ctx, doguResource)
	if err != nil {
		return requeueWithError(fmt.Errorf("failed to evaluate required operation: %w", err))
	}
	logger.Info(fmt.Sprintf("Required operation for Dogu %s/%s is: %s; Status %+v", doguResource.Namespace, doguResource.Name, requiredOperation, doguResource.Status))

	switch requiredOperation {
	case Install:
		return r.performInstallOperation(ctx, doguResource)
	case Upgrade:
		return r.performUpgradeOperation(ctx, doguResource)
	case Delete:
		return r.performDeleteOperation(ctx, doguResource)
	case Ignore:
		return finishOperation()
	case ExpandVolume:
		return r.performVolumeOperation(ctx, doguResource)
	case ChangeAdditionalIngressAnnotations:
		return r.performAddititionalIngressAnnotationsOperation(ctx, doguResource)
	default:
		return finishOperation()
	}
}

func (r *doguReconciler) handleSupportMode(ctx context.Context, doguResource *k8sv1.Dogu) (*ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("Handling support flag for dogu %s", doguResource.Name))
	supportModeChanged, err := r.doguManager.HandleSupportMode(ctx, doguResource)
	if err != nil {
		printError := strings.ReplaceAll(err.Error(), "\n", "")
		r.recorder.Eventf(doguResource, v1.EventTypeWarning, ErrorOnSupportEventReason, "Handling of support mode failed.", printError)
		return &ctrl.Result{}, fmt.Errorf("failed to handle support mode: %w", err)
	}

	// Do not care about other operations if the mode has changed. Data changes with activated support mode won't and shouldn't be processed.
	logger.Info(fmt.Sprintf("Check if support mode changed for dogu %s", doguResource.Name))
	if supportModeChanged {
		r.recorder.Event(doguResource, v1.EventTypeNormal, SupportEventReason, "Support mode has changed. Ignoring other events.")
		return &ctrl.Result{}, nil
	}

	// Do not care about other operations if the support mode is currently active.
	if doguResource.Spec.SupportMode {
		logger.Info(fmt.Sprintf("Support mode is currently active for dogu %s", doguResource.Name))
		r.recorder.Event(doguResource, v1.EventTypeNormal, SupportEventReason, "Support mode is active. Ignoring other events.")
		return &ctrl.Result{}, nil
	}

	return nil, nil
}

func (r *doguReconciler) evaluateRequiredOperation(ctx context.Context, doguResource *k8sv1.Dogu) (operation, error) {
	logger := log.FromContext(ctx)
	if doguResource.DeletionTimestamp != nil && !doguResource.DeletionTimestamp.IsZero() {
		return Delete, nil
	}

	switch doguResource.Status.Status {
	case k8sv1.DoguStatusNotInstalled:
		return Install, nil
	case k8sv1.DoguStatusPVCResizing:
		return ExpandVolume, nil
	case k8sv1.DoguStatusInstalled:
		isVolumeExpansion, err := r.checkForVolumeExpansion(ctx, doguResource)
		if err != nil {
			return Ignore, err
		}

		if isVolumeExpansion {
			return ExpandVolume, nil
		}

		ingressAnnotationsChanged, err := r.checkForAdditionalIngressAnnotations(ctx, doguResource)
		if err != nil {
			return Ignore, err
		}

		if ingressAnnotationsChanged {
			return ChangeAdditionalIngressAnnotations, nil
		}

		// Checking if the resource spec field has changed is unnecessary because we
		// use a predicate to filter update events where specs don't change
		upgradeable, err := checkUpgradeability(doguResource, r.fetcher)
		if err != nil {
			printError := strings.ReplaceAll(err.Error(), "\n", "")
			r.recorder.Eventf(doguResource, v1.EventTypeWarning, operatorEventReason, "Could not check if dogu needs to be upgraded: %s", printError)

			return Ignore, err
		}

		if upgradeable {
			return Upgrade, nil
		}

		return Ignore, nil
	case k8sv1.DoguStatusInstalling:
		return Ignore, nil
	case k8sv1.DoguStatusDeleting:
		return Ignore, nil
	default:
		logger.Info(fmt.Sprintf("Found unknown operation for dogu status: %s", doguResource.Status.Status))
		return Ignore, nil
	}
}

func (r *doguReconciler) checkForVolumeExpansion(ctx context.Context, doguResource *k8sv1.Dogu) (bool, error) {
	doguPvc := &v1.PersistentVolumeClaim{}
	err := r.client.Get(ctx, doguResource.GetObjectKey(), doguPvc)
	if apierrors.IsNotFound(err) {
		// no persistent volume claim -> no volume for the dogu -> no expansion possible
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to get persistent volume claim of dogu [%s]: %w", doguResource.Name, err)
	}

	dataVolumeSize := doguResource.Spec.Resources.DataVolumeSize
	if dataVolumeSize == "" {
		return false, nil
	}

	doguTargetDataVolumeSize := resource.MustParse(k8sv1.DefaultVolumeSize)
	size, err := resource.ParseQuantity(dataVolumeSize)
	if err != nil {
		return false, fmt.Errorf("failed to parse resource volume size: %w", err)
	}

	if !size.IsZero() {
		doguTargetDataVolumeSize = size
	}

	if doguTargetDataVolumeSize.Value() > doguPvc.Spec.Resources.Requests.Storage().Value() {
		return true, nil
	} else if doguTargetDataVolumeSize.Value() == doguPvc.Spec.Resources.Requests.Storage().Value() {
		return false, nil
	} else {
		return false, fmt.Errorf("invalid dogu state for dogu [%s] as requested volume size is [%s] while "+
			"existing volume is [%s], shrinking of volumes is not allowed", doguResource.Name,
			doguTargetDataVolumeSize.String(), doguPvc.Spec.Resources.Requests.Storage().String())
	}
}

func (r *doguReconciler) checkForAdditionalIngressAnnotations(ctx context.Context, doguResource *k8sv1.Dogu) (bool, error) {
	doguService := &v1.Service{}
	err := r.client.Get(ctx, doguResource.GetObjectKey(), doguService)
	if err != nil {
		return false, fmt.Errorf("failed to get service of dogu [%s]: %w", doguResource.Name, err)
	}

	annotationsJson, exists := doguService.Annotations[annotation.AdditionalIngressAnnotationsAnnotation]
	annotations := k8sv1.IngressAnnotations(nil)
	if exists {
		err = json.Unmarshal([]byte(annotationsJson), &annotations)
		if err != nil {
			return false, fmt.Errorf("failed to get additional ingress annotations from service of dogu [%s]: %w", doguResource.Name, err)
		}
	}

	if reflect.DeepEqual(annotations, doguResource.Spec.AdditionalIngressAnnotations) {
		return false, nil
	} else {
		return true, nil
	}
}

// SetupWithManager sets up the controller with the manager.
func (r *doguReconciler) SetupWithManager(mgr ctrl.Manager) error {
	var eventFilter predicate.Predicate
	eventFilter = predicate.GenerationChangedPredicate{}
	if logging.CurrentLogLevel == logrus.TraceLevel {
		recorder := mgr.GetEventRecorderFor(k8sDoguOperatorFieldManagerName)
		eventFilter = doguResourceChangeDebugPredicate{recorder: recorder}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sv1.Dogu{}).
		// Since we don't want to process dogus with same spec we use a generation change predicate
		// as a filter to reduce the reconcile calls.
		// The predicate implements a function that will be invoked of every update event that
		// the k8s api will fire. On writing the objects spec field the k8s api
		// increments the generation field. The function compares this field from the old
		// and new dogu resource. If they are equal the reconcile loop will not be called.
		WithEventFilter(eventFilter).
		Complete(r)
}

func (r *doguReconciler) performOperation(ctx context.Context, doguResource *k8sv1.Dogu,
	eventProperties operationEventProperties, requeueDoguStatus string,
	operation func(context.Context, *k8sv1.Dogu) error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	operationError := operation(ctx, doguResource)
	contextMessageOnError := fmt.Sprintf("failed to %s dogu %s", eventProperties.operationVerb, doguResource.Name)

	if operationError != nil {
		printError := strings.ReplaceAll(operationError.Error(), "\n", "")
		logger.Error(operationError, fmt.Sprintf("%s failed", eventProperties.operationName))
		r.recorder.Eventf(doguResource, v1.EventTypeWarning, eventProperties.errorReason,
			"%s failed. Reason: %s.", eventProperties.operationName, printError)
	} else {
		r.recorder.Eventf(doguResource, v1.EventTypeNormal, eventProperties.successReason, "%s successful.",
			eventProperties.operationName)
	}

	result, handleErr := r.doguRequeueHandler.Handle(ctx, contextMessageOnError, doguResource, operationError,
		func(dogu *k8sv1.Dogu) {
			doguResource.Status.Status = requeueDoguStatus
		})
	if handleErr != nil {
		r.recorder.Eventf(doguResource, v1.EventTypeWarning, ErrorOnRequeueEventReason,
			"Failed to requeue the %s.", strings.ToLower(eventProperties.operationName))
		return requeueWithError(fmt.Errorf(handleRequeueErrMsg, handleErr))
	}

	return requeueOrFinishOperation(result)
}

type operationEventProperties struct {
	successReason string
	errorReason   string
	operationName string
	operationVerb string
}

func (r *doguReconciler) performInstallOperation(ctx context.Context, doguResource *k8sv1.Dogu) (ctrl.Result, error) {
	installOperationEventProps := operationEventProperties{
		successReason: InstallEventReason,
		errorReason:   ErrorOnInstallEventReason,
		operationName: "Installation",
		operationVerb: "install",
	}
	return r.performOperation(ctx, doguResource, installOperationEventProps, k8sv1.DoguStatusNotInstalled,
		r.doguManager.Install)
}

func (r *doguReconciler) performDeleteOperation(ctx context.Context, doguResource *k8sv1.Dogu) (ctrl.Result, error) {
	deleteOperationEventProps := operationEventProperties{
		successReason: DeinstallEventReason,
		errorReason:   ErrorDeinstallEventReason,
		operationName: "Deinstallation",
		operationVerb: "delete",
	}
	return r.performOperation(ctx, doguResource, deleteOperationEventProps, k8sv1.DoguStatusInstalled,
		r.doguManager.Delete)
}

func (r *doguReconciler) performUpgradeOperation(ctx context.Context, doguResource *k8sv1.Dogu) (ctrl.Result, error) {
	upgradeOperationEventProps := operationEventProperties{
		successReason: upgrade.EventReason,
		errorReason:   upgrade.ErrorOnFailedUpgradeEventReason,
		operationName: "Upgrade",
		operationVerb: "upgrade",
	}
	// revert to Installed in case of requeueing after an error so that a upgrade
	// can be tried again.
	return r.performOperation(ctx, doguResource, upgradeOperationEventProps, k8sv1.DoguStatusInstalled,
		r.doguManager.Upgrade)
}

func (r *doguReconciler) performVolumeOperation(ctx context.Context, doguResource *k8sv1.Dogu) (ctrl.Result, error) {
	volumeExpansionOperationEventProps := operationEventProperties{
		successReason: VolumeExpansionEventReason,
		errorReason:   ErrorOnVolumeExpansionEventReason,
		operationName: "VolumeExpansion",
		operationVerb: "expand volume",
	}

	// revert to resizing in case of requeueing after an error so that the size check can be done again.
	return r.performOperation(ctx, doguResource, volumeExpansionOperationEventProps, k8sv1.DoguStatusPVCResizing, r.doguManager.SetDoguDataVolumeSize)
}

func (r *doguReconciler) performAddititionalIngressAnnotationsOperation(ctx context.Context, doguResource *k8sv1.Dogu) (ctrl.Result, error) {
	additionalIngressAnnotationsOperationEventProps := operationEventProperties{
		successReason: AdditionalIngressAnnotationsChangeEventReason,
		errorReason:   ErrorOnAdditionalIngressAnnotationsChangeEventReason,
		operationName: "AdditionalIngressAnnotationsChange",
		operationVerb: "change additional ingress annotations",
	}

	// revert to Installed in case of requeueing after an error so that the change check can be done again.
	return r.performOperation(ctx, doguResource, additionalIngressAnnotationsOperationEventProps, k8sv1.DoguStatusInstalled, r.doguManager.SetDoguAdditionalIngressAnnotations)
}

func checkUpgradeability(doguResource *k8sv1.Dogu, fetcher cloudogu.LocalDoguFetcher) (bool, error) {
	fromDogu, err := fetcher.FetchInstalled(doguResource.Name)
	if err != nil {
		return false, err
	}

	checker := &upgradeChecker{}
	toDogu := &core.Dogu{Name: doguResource.Spec.Name, Version: doguResource.Spec.Version}

	return checker.IsUpgradeable(fromDogu, toDogu, doguResource.Spec.UpgradeConfig.ForceUpgrade)
}

// requeueWithError is a syntax sugar function to express that every non-nil error will result in a requeue
// operation.
//
// Use requeueOrFinishOperation() if the reconciler should requeue the operation because of the result instead of an
// error.
// Use finishOperation() if the reconciler should not requeue the operation.
func requeueWithError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}

// requeueOrFinishOperation is a syntax sugar function to express that the there is no error to handle but the result
// controls whether the current operation should be finished or requeued.
//
// Use requeueWithError() if the reconciler should requeue the operation because of a non-nil error.
// Use finishOperation() if the reconciler should not requeue the operation.
func requeueOrFinishOperation(result ctrl.Result) (ctrl.Result, error) {
	return result, nil
}

// finishOperation is a syntax sugar function to express that the current operation should be finished and not be
// requeued. This can happen if the operation was successful or even if an unhandleable error occurred which prevents
// requeueing.
//
// Use requeueOrFinishOperation() or requeueWithError() if the reconciler should requeue the operation.
func finishOperation() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

type doguResourceChangeDebugPredicate struct {
	predicate.Funcs
	recorder record.EventRecorder
}

// Update implements default UpdateEvent filter for validating generation change.
func (cp doguResourceChangeDebugPredicate) Update(e event.UpdateEvent) bool {

	objectDiff, objectInQuestion := buildResourceDiff(e.ObjectOld, e.ObjectNew)
	cp.recorder.Event(objectInQuestion, v1.EventTypeNormal, "Debug", objectDiff)

	if e.ObjectOld == nil {
		ctrl.Log.Error(nil, "Update event has no old object to update", "event", e)
		return false
	}

	if e.ObjectNew == nil {
		ctrl.Log.Error(nil, "Update event has no new object for update", "event", e)
		return false
	}

	return e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration()
}

func buildResourceDiff(objOld client.Object, objNew client.Object) (string, client.Object) {
	var aOld client.Object
	var aNew client.Object

	// both values can be nil during creation or deletion (though not at the same time)
	// take care to provide a proper diff in any of these cases
	var objectInQuestion client.Object
	if objOld != nil {
		aOld = objOld
		objectInQuestion = aOld
	}
	if objNew != nil {
		aNew = objNew
		objectInQuestion = aNew
	}

	diff := cmp.Diff(aOld, aNew)

	return strings.ReplaceAll(diff, "\u00a0", " "), objectInQuestion
}
