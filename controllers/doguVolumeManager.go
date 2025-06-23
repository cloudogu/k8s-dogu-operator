package controllers

import (
	"context"
	"fmt"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/async"

	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	opresource "github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
)

const (
	// VolumeExpansionEventReason is the reason string for firing volume expansion events.
	VolumeExpansionEventReason = "VolumeExpansion"
	// ErrorOnVolumeExpansionEventReason is the error string for firing volume expansion error events.
	ErrorOnVolumeExpansionEventReason = "ErrVolumeExpansion"
)

const (
	startConditionPVC           = ""
	startConditionEditPvc       = "Edit PVC"
	startConditionWaitForResize = "Wait for resize"
	startConditionScaleUp       = "Scale up"
	startConditionValidate      = "Validate Conditions"
)

type notResizedError struct {
	state       string
	requeueTime time.Duration
}

// GetState returns the target state for the dogu.
func (nre notResizedError) GetState() string {
	return nre.state
}

// GetRequeueTime returns the requeue time.
func (nre notResizedError) GetRequeueTime() time.Duration {
	return nre.requeueTime
}

// Requeue indicates if the error should reconcile the dogu again.
func (nre notResizedError) Requeue() bool {
	return true
}

// Error returns the error in readable format.
func (nre notResizedError) Error() string {
	return "pvc resizing is in progress"
}

// doguVolumeManager is currently used for resizing PVCs from dogus.
// To do this it uses an asyncExecutor with defined steps.
// The order of the steps is:
// 1. editPVCStep - Edits the size from the PVC
// 2. scaleDownStep - Kills all pods from the dogu.
// 3. checkIfPVCIsResizedStep - Waits until the storage controller resizes the volume.
// 4. scaleUpStep - Starts the terminated pods from the dogu.
type doguVolumeManager struct {
	client        client.Client
	doguInterface doguClient.DoguInterface
	eventRecorder record.EventRecorder
	asyncExecutor async.AsyncExecutor
}

// NewDoguVolumeManager creates a new instance of the doguVolumeManager.
func NewDoguVolumeManager(client client.Client, eventRecorder record.EventRecorder, doguInterface doguClient.DoguInterface) *doguVolumeManager {
	asyncExecutor := async.NewDoguExecutionController()
	createAsyncSteps(asyncExecutor, client, eventRecorder, doguInterface)

	return &doguVolumeManager{
		client:        client,
		doguInterface: doguInterface,
		eventRecorder: eventRecorder,
		asyncExecutor: asyncExecutor,
	}
}

func createAsyncSteps(executor async.AsyncExecutor, client client.Client, recorder record.EventRecorder, doguInterface doguClient.DoguInterface) {
	scaleUp := &scaleUpStep{client: client, eventRecorder: recorder, replicas: 1}
	executor.AddStep(&scaleDownStep{client: client, eventRecorder: recorder, scaleUpStep: scaleUp})
	executor.AddStep(&editPVCStep{client: client, eventRecorder: recorder, doguInterface: doguInterface})
	executor.AddStep(&checkIfPVCIsResizedStep{client: client, eventRecorder: recorder})
	executor.AddStep(scaleUp)
	//executor.AddStep(&dataVolumeSizeStep{client: client, eventRecorder: recorder, doguInterface: doguInterface})
}

// SetDoguDataVolumeSize sets the quantity from the doguResource in the dogu data PVC.
func (d *doguVolumeManager) SetDoguDataVolumeSize(ctx context.Context, doguResource *doguv2.Dogu) error {
	err := doguResource.ChangeStateWithRetry(ctx, d.client, doguv2.DoguStatusPVCResizing)
	if err != nil {
		return err
	}

	err = d.asyncExecutor.Execute(ctx, doguResource, doguResource.Status.RequeuePhase)
	if err != nil {
		return err
	}

	return doguResource.ChangeStateWithRetry(ctx, d.client, doguv2.DoguStatusInstalled)
}

type editPVCStep struct {
	client        client.Client
	doguInterface doguClient.DoguInterface
	eventRecorder record.EventRecorder
}

// GetStartCondition returns the condition required to start the step.
func (e *editPVCStep) GetStartCondition() string {
	return startConditionEditPvc
}

// Execute executes the step and returns the next state and if the step fails an error.
// The error can be a requeueable error so that the step will be executed again.
func (e *editPVCStep) Execute(ctx context.Context, dogu *doguv2.Dogu) (string, error) {
	quantity, err := dogu.GetMinDataVolumeSize()
	if err != nil {
		return e.GetStartCondition(), fmt.Errorf("failed to parse data volume size: %w", err)
	}

	err = e.updatePVCQuantity(ctx, dogu, quantity)
	if err != nil {
		return e.GetStartCondition(), err
	}

	return startConditionWaitForResize, nil
}

func (e *editPVCStep) updatePVCQuantity(ctx context.Context, doguResource *doguv2.Dogu, quantity resource.Quantity) error {
	e.eventRecorder.Event(doguResource, corev1.EventTypeNormal, VolumeExpansionEventReason, "Update dogu data PVC request storage...")
	pvc, err := doguResource.GetDataPVC(ctx, e.client)
	if err != nil {
		return err
	}

	// Update Status before Resizing - this should set the condition to false
	// because the new Minsize is larger than the actual current size before the resizing is finished
	_ = opresource.SetCurrentDataVolumeSize(ctx, e.doguInterface, e.client, doguResource, pvc)

	// It is necessary to create a new map because just setting a new quantity results in an exception.
	pvc.Spec.Resources.Requests = map[corev1.ResourceName]resource.Quantity{corev1.ResourceStorage: quantity}
	err = e.client.Update(ctx, pvc)
	if err != nil {
		return fmt.Errorf("failed to update PVC %s: %w", pvc.Name, err)
	}

	return err
}

type scaleDownStep struct {
	client        client.Client
	eventRecorder record.EventRecorder
	scaleUpStep   *scaleUpStep
}

// GetStartCondition returns the condition required to start the step.
func (s *scaleDownStep) GetStartCondition() string {
	return startConditionPVC
}

// Execute executes the step and returns the next state and if the step fails an error.
// The error can be a requeueable error so that the step will be executed again.
func (s *scaleDownStep) Execute(ctx context.Context, dogu *doguv2.Dogu) (string, error) {
	oldReplicas, err := scaleDeployment(ctx, s.client, s.eventRecorder, dogu, 0)
	if err != nil {
		return s.GetStartCondition(), err
	}
	s.scaleUpStep.replicas = oldReplicas

	return startConditionEditPvc, nil
}

type scaleUpStep struct {
	client        client.Client
	eventRecorder record.EventRecorder
	replicas      int32
}

// GetStartCondition returns the condition required to start the step.
func (s *scaleUpStep) GetStartCondition() string {
	return startConditionScaleUp
}

// Execute executes the step and returns the next state and if the step fails an error.
// The error can be a requeueable error so that the step will be executed again.
func (s *scaleUpStep) Execute(ctx context.Context, dogu *doguv2.Dogu) (string, error) {
	_, err := scaleDeployment(ctx, s.client, s.eventRecorder, dogu, s.replicas)
	if err != nil {
		return s.GetStartCondition(), err
	}

	return startConditionValidate, nil
}

func scaleDeployment(ctx context.Context, client client.Client, recorder record.EventRecorder, doguResource *doguv2.Dogu, newReplicas int32) (oldReplicas int32, err error) {
	recorder.Eventf(doguResource, corev1.EventTypeNormal, VolumeExpansionEventReason, "Scale deployment to %d replicas...", newReplicas)
	deployment, err := doguResource.GetDeployment(ctx, client)
	if err != nil {
		return 0, err
	}

	oldReplicas = *deployment.Spec.Replicas
	*deployment.Spec.Replicas = newReplicas
	err = client.Update(ctx, deployment)
	if err != nil {
		return 0, fmt.Errorf("failed to scale deployment for dogu %s: %w", doguResource.Name, err)
	}

	return oldReplicas, err
}

type checkIfPVCIsResizedStep struct {
	client        client.Client
	eventRecorder record.EventRecorder
}

// GetStartCondition returns the condition required to start the step.
func (c *checkIfPVCIsResizedStep) GetStartCondition() string {
	return startConditionWaitForResize
}

// Execute executes the step and returns the next state and if the step fails an error.
// The error can be a requeueable error so that the step will be executed again.
func (c *checkIfPVCIsResizedStep) Execute(ctx context.Context, dogu *doguv2.Dogu) (string, error) {
	quantity, err := dogu.GetMinDataVolumeSize()
	if err != nil {
		return c.GetStartCondition(), fmt.Errorf("failed to parse data volume size: %w", err)
	}

	return startConditionScaleUp, c.waitForPVCResize(ctx, dogu, quantity)
}

func (c *checkIfPVCIsResizedStep) waitForPVCResize(ctx context.Context, doguResource *doguv2.Dogu, quantity resource.Quantity) error {
	c.eventRecorder.Event(doguResource, corev1.EventTypeNormal, VolumeExpansionEventReason, "Wait for pvc to be resized...")
	pvc, err := doguResource.GetDataPVC(ctx, c.client)
	if err != nil {
		return err
	}
	resized := isPvcStorageResized(pvc, quantity)
	if !resized {
		return notResizedError{
			state:       c.GetStartCondition(),
			requeueTime: time.Minute * 1,
		}
	}

	return nil
}

func isPvcStorageResized(pvc *corev1.PersistentVolumeClaim, quantity resource.Quantity) bool {
	if isPvcResizeApplicable(pvc) {
		return true
	}

	// Longhorn works this way and does not add the Condition "FileSystemResizePending" to the PVC
	// see https://github.com/longhorn/longhorn/issues/2749
	isRequestedCapacityAvailable := pvc.Status.Capacity.Storage().Value() >= quantity.Value()
	return isRequestedCapacityAvailable
}

// isPvcResizeApplicable checks if the filesystem resize is "pending", which means that the pod has to be (re)started to actually resize the volume.
// see https://kubernetes.io/blog/2018/07/12/resizing-persistent-volumes-using-kubernetes/#file-system-expansion
func isPvcResizeApplicable(pvc *corev1.PersistentVolumeClaim) bool {
	for _, condition := range pvc.Status.Conditions {
		if condition.Type == corev1.PersistentVolumeClaimFileSystemResizePending && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

type dataVolumeSizeStep struct {
	client        client.Client
	doguInterface doguClient.DoguInterface
	eventRecorder record.EventRecorder
}

func (d *dataVolumeSizeStep) GetStartCondition() string {
	return startConditionValidate
}

// Execute executes the step and returns the next state and if the step fails an error.
// The error can be a requeueable error so that the step will be executed again.
func (d *dataVolumeSizeStep) Execute(ctx context.Context, dogu *doguv2.Dogu) (string, error) {
	logger := log.FromContext(ctx)
	logger.Info("Start Validate Volume Size..")
	pvc, err := dogu.GetDataPVC(ctx, d.client)
	if err != nil {
		return "", err
	}

	currentSize := pvc.Status.Capacity.Storage()

	minDataSize, err := dogu.GetMinDataVolumeSize()
	if err != nil {
		logger.Error(err, "failed to get min data volume size")
		return "", err
	}
	if minDataSize.Value() > currentSize.Value() {
		logger.Info("resize not finished yet... requeue")
		return "", notResizedError{
			state:       d.GetStartCondition(),
			requeueTime: time.Minute * 1,
		}
	}

	err = opresource.SetCurrentDataVolumeSize(ctx, d.doguInterface, d.client, dogu, pvc)

	if err != nil {
		return "", err
	}

	// Finish Resizing
	err = dogu.ChangeRequeuePhaseWithRetry(ctx, d.client, "")
	if err != nil {
		return "", err
	}

	return async.FinishedState, nil
}
