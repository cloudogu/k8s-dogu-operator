package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	resource2 "github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	appsv1 "k8s.io/api/apps/v1"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/annotation"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/logging"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/upgrade"
	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

const (
	FailedNameValidationEventReason              = "FailedNameValidation"
	FailedVolumeSizeParsingValidationEventReason = "FailedVolumeSizeParsingValidation"
	FailedVolumeSizeSIValidationEventReason      = "FailedVolumeSizeSIValidation"
)

const handleRequeueErrMsg = "failed to handle requeue: %w"

type operation string

func operationsContain(operations []operation, contain operation) bool {
	for _, op := range operations {
		if op == contain {
			return true
		}
	}
	return false
}

const (
	Install                            = operation("Install")
	Upgrade                            = operation("Upgrade")
	Delete                             = operation("Delete")
	Wait                               = operation("Wait")
	ExpandVolume                       = operation("ExpandVolume")
	ChangeAdditionalIngressAnnotations = operation("ChangeAdditionalIngressAnnotations")
	StartStopDogu                      = operation("StartStopDogu")
	ChangeSecurityContext              = operation("ChangeSecurityContext")
	ChangeExportMode                   = operation("ChangeExportMode")
	ChangeAdditionalMounts             = operation("ChangeAdditionalMounts")
)

const requeueWaitTimeout = 5 * time.Second

// doguReconciler reconciles a Dogu object
type doguReconciler struct {
	client             client.Client
	doguManager        CombinedDoguManager
	doguRequeueHandler requeueHandler
	recorder           record.EventRecorder
	fetcher            localDoguFetcher
	doguInterface      doguClient.DoguInterface
}

// NewDoguReconciler creates a new reconciler instance for the dogu resource
func NewDoguReconciler(client client.Client, doguInterface doguClient.DoguInterface, doguManager CombinedDoguManager, eventRecorder record.EventRecorder, namespace string, doguFetcher localDoguFetcher) (*doguReconciler, error) {
	doguRequeueHandler, err := NewDoguRequeueHandler(doguInterface, eventRecorder, namespace)
	if err != nil {
		return nil, err
	}

	return &doguReconciler{
		client:             client,
		doguManager:        doguManager,
		doguRequeueHandler: doguRequeueHandler,
		recorder:           eventRecorder,
		fetcher:            doguFetcher,
		doguInterface:      doguInterface,
	}, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *doguReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	doguResource := &doguv2.Dogu{}
	err := r.client.Get(ctx, req.NamespacedName, doguResource)
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get doguResource: %s", err))
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logger.Info(fmt.Sprintf("Dogu %s/%s has been found", doguResource.Namespace, doguResource.Name))

	valid := r.validateDogu(doguResource)
	if !valid {
		return finishOperation()
	}

	if doguResource.Status.Status != doguv2.DoguStatusNotInstalled {
		supportResult, err := r.handleSupportMode(ctx, doguResource)
		if supportResult != nil {
			return *supportResult, err
		}
	}

	requiredOperations, err := r.evaluateRequiredOperations(ctx, doguResource)
	if err != nil {
		return requeueWithError(fmt.Errorf("failed to evaluate required operation: %w", err))
	}

	return r.executeRequiredOperation(ctx, requiredOperations, doguResource)
}

func (r *doguReconciler) validateDogu(doguResource *doguv2.Dogu) bool {
	hasValidName := r.validateName(doguResource)
	hasValidVolumeSize := r.validateVolumeSize(doguResource)

	return hasValidName && hasValidVolumeSize
}

func (r *doguReconciler) executeRequiredOperation(ctx context.Context, requiredOperations []operation, doguResource *doguv2.Dogu) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	if len(requiredOperations) == 0 {
		logger.Info(fmt.Sprintf("Nothing to be done for Dogu %s/%s with Status %s", doguResource.Namespace, doguResource.Name, doguResource.Status.Status))
		return finishOperation()
	}

	logger.Info(fmt.Sprintf("Required operation for Dogu %s/%s is: %s; Status %+v", doguResource.Namespace, doguResource.Name, requiredOperations[0], doguResource.Status))

	requeueForMultipleOperations := len(requiredOperations) > 1
	switch requiredOperations[0] {
	case Wait:
		if requeueForMultipleOperations {
			return requeueOrFinishOperation(ctrl.Result{Requeue: true, RequeueAfter: requeueWaitTimeout})
		}

		return finishOperation()
	case Install:
		return r.performInstallOperation(ctx, doguResource)
	case Upgrade:
		return r.performUpgradeOperation(ctx, doguResource, requeueForMultipleOperations)
	case Delete:
		return r.performDeleteOperation(ctx, doguResource)
	case ExpandVolume:
		return r.performVolumeOperation(ctx, doguResource, requeueForMultipleOperations)
	case ChangeAdditionalIngressAnnotations:
		return r.performAdditionalIngressAnnotationsOperation(ctx, doguResource, requeueForMultipleOperations)
	case StartStopDogu:
		return r.performStartStopDoguOperation(ctx, doguResource, requeueForMultipleOperations)
	case ChangeSecurityContext:
		return r.performSecurityContextOperation(ctx, doguResource, requeueForMultipleOperations)
	case ChangeExportMode:
		return r.performExportModeOperation(ctx, doguResource, requeueForMultipleOperations)
	case ChangeAdditionalMounts:
		return r.performAdditionalMountsOperation(ctx, doguResource, requeueForMultipleOperations)
	default:
		return finishOperation()
	}
}

func (r *doguReconciler) handleSupportMode(ctx context.Context, doguResource *doguv2.Dogu) (*ctrl.Result, error) {
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
		if doguResource.Spec.SupportMode {
			r.recorder.Event(doguResource, v1.EventTypeNormal, SupportEventReason, "Support mode switched on. Ignoring other events.")
			return &ctrl.Result{}, nil
		}

		r.recorder.Event(doguResource, v1.EventTypeNormal, SupportEventReason, "Support mode switched off. Resuming processing of other events.")
		return &ctrl.Result{Requeue: true}, nil
	}

	// Do not care about other operations if the support mode is currently active.
	if doguResource.Spec.SupportMode {
		logger.Info(fmt.Sprintf("Support mode is currently active for dogu %s", doguResource.Name))
		r.recorder.Event(doguResource, v1.EventTypeNormal, SupportEventReason, "Support mode is active. Ignoring other events.")
		return &ctrl.Result{}, nil
	}

	return nil, nil
}

func (r *doguReconciler) evaluateRequiredOperations(ctx context.Context, doguResource *doguv2.Dogu) ([]operation, error) {
	logger := log.FromContext(ctx)
	if doguResource.DeletionTimestamp != nil && !doguResource.DeletionTimestamp.IsZero() {
		return []operation{Delete}, nil
	}

	var err error
	var operations []operation
	switch doguResource.Status.Status {
	case doguv2.DoguStatusNotInstalled:
		return []operation{Install}, nil
	case doguv2.DoguStatusStarting:
		operations = append(operations, StartStopDogu)
		operations, err = r.appendRequiredPostInstallOperations(ctx, doguResource, operations)
		if err != nil {
			return nil, err
		}
	case doguv2.DoguStatusStopping:
		operations = append(operations, StartStopDogu)
		operations, err = r.appendRequiredPostInstallOperations(ctx, doguResource, operations)
		if err != nil {
			return nil, err
		}
	case doguv2.DoguStatusPVCResizing:
		operations = append(operations, ExpandVolume)
		operations, err = r.appendRequiredPostInstallOperations(ctx, doguResource, operations)
		if err != nil {
			return nil, err
		}
	case doguv2.DoguStatusInstalling:
		fallthrough
	case doguv2.DoguStatusUpgrading:
		operations = append(operations, Wait)
		operations, err = r.appendRequiredPostInstallOperations(ctx, doguResource, operations)
		if err != nil {
			return nil, err
		}
	case doguv2.DoguStatusInstalled:
		operations, err = r.appendRequiredPostInstallOperations(ctx, doguResource, operations)
		if err != nil {
			return nil, err
		}
	case doguv2.DoguStatusChangingExportMode:
		operations = append(operations, ChangeExportMode)
		operations, err = r.appendRequiredPostInstallOperations(ctx, doguResource, operations)
		if err != nil {
			return nil, err
		}
	case doguv2.DoguStatusDeleting:
		return []operation{}, nil
	case doguv2.DoguStatusChangingDataMounts:
		operations = append(operations, ChangeAdditionalMounts)
		operations, err = r.appendRequiredPostInstallOperations(ctx, doguResource, operations)
		if err != nil {
			return nil, err
		}
	default:
		logger.Info(fmt.Sprintf("Cannot evaluate required operation for unknown dogu status: %s", doguResource.Status.Status))
		return []operation{}, nil
	}

	return operations, nil
}

func (r *doguReconciler) appendRequiredPostInstallOperations(ctx context.Context, doguResource *doguv2.Dogu, operations []operation) ([]operation, error) {
	if checkShouldStartStopDogu(doguResource) {
		operations = append(operations, StartStopDogu)
	}

	isVolumeExpansion, err := r.checkForVolumeExpansion(ctx, doguResource)
	if err != nil {
		return nil, err
	}

	if isVolumeExpansion && !operationsContain(operations, ExpandVolume) {
		operations = append(operations, ExpandVolume)
	}

	ingressAnnotationsChanged, err := r.checkForAdditionalIngressAnnotations(ctx, doguResource)
	if err != nil {
		return nil, err
	}

	if ingressAnnotationsChanged {
		operations = append(operations, ChangeAdditionalIngressAnnotations)
	}

	securityContextChanged, err := r.checkSecurityContextChanged(ctx, doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to check if security context is changed: %w", err)
	}

	if securityContextChanged {
		operations = append(operations, ChangeSecurityContext)
	}

	if checkShouldChangeExportMode(doguResource) && !operationsContain(operations, ChangeExportMode) {
		operations = append(operations, ChangeExportMode)
	}

	// Checking if the resource spec field has changed is unnecessary because we
	// use a predicate to filter update events where specs don't change
	upgradeable, err := checkUpgradeability(ctx, doguResource, r.fetcher)
	if err != nil {
		printError := strings.ReplaceAll(err.Error(), "\n", "")
		r.recorder.Eventf(doguResource, v1.EventTypeWarning, operatorEventReason, "Could not check if dogu needs to be upgraded: %s", printError)

		return nil, err
	}

	if upgradeable {
		operations = append(operations, Upgrade)
	} else {
		// ChangeAdditionalMounts operation should only be triggered if the dogu does not upgrade because the upgrade itself does this anyway.
		changed, err := r.doguManager.AdditionalMountsChanged(ctx, doguResource)
		if err != nil {
			return nil, err
		}
		if changed && !operationsContain(operations, ChangeAdditionalMounts) {
			operations = append(operations, ChangeAdditionalMounts)
		}
	}

	return operations, nil
}

func checkShouldStartStopDogu(doguResource *doguv2.Dogu) bool {
	return doguResource.Spec.Stopped != doguResource.Status.Stopped
}

func checkShouldChangeExportMode(doguResource *doguv2.Dogu) bool {
	return doguResource.Spec.ExportMode != doguResource.Status.ExportMode
}

func (r *doguReconciler) checkForVolumeExpansion(ctx context.Context, doguResource *doguv2.Dogu) (bool, error) {
	doguPvc := &v1.PersistentVolumeClaim{}
	err := r.client.Get(ctx, doguResource.GetObjectKey(), doguPvc)
	if apierrors.IsNotFound(err) {
		// no persistent volume claim -> no volume for the dogu -> no expansion possible
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to get persistent volume claim of dogu [%s]: %w", doguResource.Name, err)
	}

	doguTargetDataVolumeSize, err := doguResource.GetMinDataVolumeSize()
	if err != nil {
		return false, fmt.Errorf("failed to parse data volume size: %w", err)
	}

	if doguTargetDataVolumeSize.Value() > doguPvc.Status.Capacity.Storage().Value() {
		return true, nil
	} else {
		return false, nil
	}
}

func (r *doguReconciler) checkForAdditionalIngressAnnotations(ctx context.Context, doguResource *doguv2.Dogu) (bool, error) {
	doguService := &v1.Service{}
	err := r.client.Get(ctx, doguResource.GetObjectKey(), doguService)
	if err != nil {
		return false, fmt.Errorf("failed to get service of dogu [%s]: %w", doguResource.Name, err)
	}

	annotationsJson, exists := doguService.Annotations[annotation.AdditionalIngressAnnotationsAnnotation]
	annotations := doguv2.IngressAnnotations(nil)
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

func (r *doguReconciler) checkSecurityContextChanged(ctx context.Context, doguResource *doguv2.Dogu) (bool, error) {
	doguDescriptor, err := r.fetcher.FetchInstalled(ctx, cescommons.SimpleName(doguResource.Name))
	if err != nil {
		return false, fmt.Errorf("failed to get dogu descriptor %q: %w", doguResource.Name, err)
	}

	generator := &resource2.SecurityContextGenerator{}
	podSecurityContext, containerSecurityContext := generator.Generate(ctx, doguDescriptor, doguResource)
	slices.Sort(containerSecurityContext.Capabilities.Add)

	doguDeployment := &appsv1.Deployment{}
	err = r.client.Get(ctx, doguResource.GetObjectKey(), doguDeployment)
	if err != nil {
		return false, fmt.Errorf("failed to get deployment of dogu %q: %w", doguResource.Name, err)
	}

	if !reflect.DeepEqual(podSecurityContext, doguDeployment.Spec.Template.Spec.SecurityContext) {
		return true, nil
	}

	for _, container := range doguDeployment.Spec.Template.Spec.Containers {
		skipNonDoguContainersInPod := container.Name != doguResource.Name
		if skipNonDoguContainersInPod {
			continue
		}
		slices.Sort(container.SecurityContext.Capabilities.Add)
		if !reflect.DeepEqual(containerSecurityContext, container.SecurityContext) {
			return true, nil
		}
	}

	return false, nil
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

func (r *doguReconciler) performOperation(ctx context.Context, doguResource *doguv2.Dogu,
	eventProperties operationEventProperties, requeueDoguStatus string,
	operation func(context.Context, *doguv2.Dogu) error, shouldRequeue bool) (ctrl.Result, error) {
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
		func(dogu *doguv2.Dogu) error {
			_, err := r.doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status doguv2.DoguStatus) doguv2.DoguStatus {
				status.Status = requeueDoguStatus
				return status
			}, metav1.UpdateOptions{})
			return err
		})
	if handleErr != nil {
		r.recorder.Eventf(doguResource, v1.EventTypeWarning, ErrorOnRequeueEventReason,
			"Failed to requeue the %s.", strings.ToLower(eventProperties.operationName))
		return requeueWithError(fmt.Errorf(handleRequeueErrMsg, handleErr))
	}

	if shouldRequeue {
		result.Requeue = true
	}

	return requeueOrFinishOperation(result)
}

type operationEventProperties struct {
	successReason string
	errorReason   string
	operationName string
	operationVerb string
}

func (r *doguReconciler) performInstallOperation(ctx context.Context, doguResource *doguv2.Dogu) (ctrl.Result, error) {
	installOperationEventProps := operationEventProperties{
		successReason: InstallEventReason,
		errorReason:   ErrorOnInstallEventReason,
		operationName: "Installation",
		operationVerb: "install",
	}
	return r.performOperation(ctx, doguResource, installOperationEventProps, doguv2.DoguStatusNotInstalled,
		r.doguManager.Install, false)
}

func (r *doguReconciler) performDeleteOperation(ctx context.Context, doguResource *doguv2.Dogu) (ctrl.Result, error) {
	deleteOperationEventProps := operationEventProperties{
		successReason: DeinstallEventReason,
		errorReason:   ErrorDeinstallEventReason,
		operationName: "Deinstallation",
		operationVerb: "delete",
	}
	return r.performOperation(ctx, doguResource, deleteOperationEventProps, doguv2.DoguStatusInstalled,
		r.doguManager.Delete, false)
}

func (r *doguReconciler) performUpgradeOperation(ctx context.Context, doguResource *doguv2.Dogu, shouldRequeue bool) (ctrl.Result, error) {
	upgradeOperationEventProps := operationEventProperties{
		successReason: upgrade.EventReason,
		errorReason:   upgrade.ErrorOnFailedUpgradeEventReason,
		operationName: "Upgrade",
		operationVerb: "upgrade",
	}
	// revert to Installed in case of requeueing after an error so that a upgrade
	// can be tried again.
	return r.performOperation(ctx, doguResource, upgradeOperationEventProps, doguv2.DoguStatusInstalled,
		r.doguManager.Upgrade, shouldRequeue)
}

func (r *doguReconciler) performVolumeOperation(ctx context.Context, doguResource *doguv2.Dogu, shouldRequeue bool) (ctrl.Result, error) {
	volumeExpansionOperationEventProps := operationEventProperties{
		successReason: VolumeExpansionEventReason,
		errorReason:   ErrorOnVolumeExpansionEventReason,
		operationName: "VolumeExpansion",
		operationVerb: "expand volume",
	}

	// revert to resizing in case of requeueing after an error so that the size check can be done again.
	return r.performOperation(ctx, doguResource, volumeExpansionOperationEventProps, doguv2.DoguStatusPVCResizing, r.doguManager.SetDoguDataVolumeSize, shouldRequeue)
}

func (r *doguReconciler) performAdditionalIngressAnnotationsOperation(ctx context.Context, doguResource *doguv2.Dogu, shouldRequeue bool) (ctrl.Result, error) {
	additionalIngressAnnotationsOperationEventProps := operationEventProperties{
		successReason: AdditionalIngressAnnotationsChangeEventReason,
		errorReason:   ErrorOnAdditionalIngressAnnotationsChangeEventReason,
		operationName: "AdditionalIngressAnnotationsChange",
		operationVerb: "change additional ingress annotations",
	}

	// revert to Installed in case of requeueing after an error so that the change check can be done again.
	return r.performOperation(ctx, doguResource, additionalIngressAnnotationsOperationEventProps, doguv2.DoguStatusInstalled, r.doguManager.SetDoguAdditionalIngressAnnotations, shouldRequeue)
}

func (r *doguReconciler) performSecurityContextOperation(ctx context.Context, doguResource *doguv2.Dogu, shouldRequeue bool) (ctrl.Result, error) {
	deploymentWithSecurityContextOperationEventProps := operationEventProperties{
		successReason: SecurityContextChangeEventReason,
		errorReason:   ErrorOnSecurityContextChangeEventReason,
		operationName: "SecurityContextChange",
		operationVerb: "change security context",
	}

	// revert to Installed in case of requeueing after an error so that the change check can be done again.
	return r.performOperation(ctx, doguResource, deploymentWithSecurityContextOperationEventProps, doguv2.DoguStatusInstalled, r.doguManager.UpdateDeploymentWithSecurityContext, shouldRequeue)
}

func (r *doguReconciler) performStartStopDoguOperation(ctx context.Context, doguResource *doguv2.Dogu, shouldRequeue bool) (ctrl.Result, error) {
	status := doguv2.DoguStatusStarting
	if doguResource.Spec.Stopped {
		status = doguv2.DoguStatusStopping
	}

	return r.performOperation(ctx, doguResource, operationEventProperties{
		successReason: StartStopDoguEventReason,
		errorReason:   ErrorOnStartStopDoguEventReason,
		operationName: "StartStopDogu",
		operationVerb: "start/stop dogu",
	}, status, r.doguManager.StartStopDogu, shouldRequeue)
}

func (r *doguReconciler) performExportModeOperation(ctx context.Context, doguResource *doguv2.Dogu, shouldRequeue bool) (ctrl.Result, error) {
	return r.performOperation(ctx, doguResource, operationEventProperties{
		successReason: ChangeExportModeEventReason,
		errorReason:   ErrorOnChangeExportModeEventReason,
		operationName: "ChangeExportMode",
		operationVerb: "activate export-mode",
	}, doguv2.DoguStatusChangingExportMode, r.doguManager.UpdateExportMode, shouldRequeue)
}

func (r *doguReconciler) performAdditionalMountsOperation(ctx context.Context, doguResource *doguv2.Dogu, shouldRequeue bool) (ctrl.Result, error) {
	return r.performOperation(ctx, doguResource, operationEventProperties{
		successReason: ChangeDoguAdditionalMountsEventReason,
		errorReason:   ErrorOnChangeDoguAdditionalMountsEventReason,
		operationName: "ChangeAdditionalMounts",
		operationVerb: "change additional mounts",
	}, doguv2.DoguStatusChangingDataMounts, r.doguManager.UpdateAdditionalMounts, shouldRequeue)
}

func (r *doguReconciler) validateName(doguResource *doguv2.Dogu) (success bool) {
	simpleName := core.GetSimpleDoguName(doguResource.Spec.Name)

	if doguResource.Name != simpleName {
		r.recorder.Eventf(doguResource, v1.EventTypeWarning, FailedNameValidationEventReason, "Dogu resource does not follow naming rules: The dogu's simple name '%s' must be the same as the resource name '%s'.", simpleName, doguResource.Name)
		return false
	}

	return true
}

func (r *doguReconciler) validateVolumeSize(doguResource *doguv2.Dogu) (success bool) {
	size := doguResource.Spec.Resources.DataVolumeSize
	if len(size) == 0 {
		return true
	}

	_, err := resource.ParseQuantity(size)
	if err != nil {
		r.recorder.Eventf(doguResource, v1.EventTypeWarning, FailedVolumeSizeParsingValidationEventReason, "Dogu resource volume size parsing error: %s", size)
		return false
	}

	return true
}

func checkUpgradeability(ctx context.Context, doguResource *doguv2.Dogu, fetcher localDoguFetcher) (bool, error) {
	// only upgrade if the dogu is running
	if doguResource.Status.Stopped {
		return false, nil
	}

	fromDogu, err := fetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
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
