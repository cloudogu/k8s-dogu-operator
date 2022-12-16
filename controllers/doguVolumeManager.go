package controllers

import (
	"context"
	"fmt"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	VolumeExpansionEventReason        = "VolumeExpansion"
	ErrorOnVolumeExpansionEventReason = "ErrVolumeExpansion"
)

const (
	startConditionPVC           = ""
	startConditionScaleDown     = "Scale down"
	startConditionWaitForResize = "Wait for resize"
	startConditionScaleUp       = "Scale up"
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

type doguVolumeManager struct {
	client        client.Client
	eventRecorder record.EventRecorder
}

// NewDoguVolumeManager creates a new instance of the doguVolumeManager.
func NewDoguVolumeManager(client client.Client, eventRecorder record.EventRecorder) *doguVolumeManager {
	return &doguVolumeManager{
		client:        client,
		eventRecorder: eventRecorder,
		asyncExecutor: asyncExecutor,
	}
}

func createAsyncSteps(executor internal.AsyncExecutor, client client.Client, recorder record.EventRecorder) {
	executor.AddStep(&editPVCStep{client: client, eventRecorder: recorder})
	scaleUp := &scaleUpStep{client: client, eventRecorder: recorder, replicas: 1}
	executor.AddStep(&scaleDownStep{client: client, eventRecorder: recorder, scaleUpStep: scaleUp})
	executor.AddStep(&checkIfPVCIsResizedStep{client: client, eventRecorder: recorder})
	executor.AddStep(scaleUp)
}

// SetDoguDataVolumeSize sets the quantity from the doguResource in the dogu data PVC.
func (d *doguVolumeManager) SetDoguDataVolumeSize(ctx context.Context, doguResource *k8sv1.Dogu) error {
	err := doguResource.ChangeState(ctx, d.client, k8sv1.DoguStatusPVCResizing)
	if err != nil {
		return err
	}

	err = d.asyncExecutor.Execute(ctx, doguResource, doguResource.Status.RequeuePhase)
	if err != nil {
		return err
	}

	return doguResource.ChangeState(ctx, d.client, k8sv1.DoguStatusInstalled)
}

type editPVCStep struct {
	client        client.Client
	eventRecorder record.EventRecorder
}

func (e *editPVCStep) GetStartCondition() string {
	return startConditionPVC
}

func (e *editPVCStep) Execute(ctx context.Context, dogu *k8sv1.Dogu) (string, error) {
	quantity, err := getQuantityForDogu(dogu)
	if err != nil {
		return e.GetStartCondition(), err
	}

	err = e.updatePVCQuantity(ctx, dogu, quantity)
	if err != nil {
		return e.GetStartCondition(), err
	}

	return startConditionScaleDown, nil
}

func (e *editPVCStep) updatePVCQuantity(ctx context.Context, doguResource *k8sv1.Dogu, quantity resource.Quantity) error {
	e.eventRecorder.Event(doguResource, corev1.EventTypeNormal, VolumeExpansionEventReason, "Update dogu data PVC request storage...")
	pvc, err := doguResource.GetDataPVC(ctx, e.client)
	if err != nil {
		return err
	}
	pvc.Spec.Resources.Requests[corev1.ResourceStorage] = quantity
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

func (s *scaleDownStep) GetStartCondition() string {
	return startConditionScaleDown
}

func (s *scaleDownStep) Execute(ctx context.Context, dogu *k8sv1.Dogu) (string, error) {
	oldReplicas, err := scaleDeployment(ctx, s.client, s.eventRecorder, dogu, 0)
	if err != nil {
		return s.GetStartCondition(), err
	}
	s.scaleUpStep.replicas = oldReplicas

	return startConditionWaitForResize, nil
}

type scaleUpStep struct {
	client        client.Client
	eventRecorder record.EventRecorder
	replicas      int32
}

func (s *scaleUpStep) GetStartCondition() string {
	return startConditionScaleUp
}

func (s *scaleUpStep) Execute(ctx context.Context, dogu *k8sv1.Dogu) (string, error) {
	_, err := scaleDeployment(ctx, s.client, s.eventRecorder, dogu, s.replicas)
	if err != nil {
		return s.GetStartCondition(), err
	}

	return async.FinishedState, nil
}

func scaleDeployment(ctx context.Context, client client.Client, recorder record.EventRecorder, doguResource *k8sv1.Dogu, newReplicas int32) (oldReplicas int32, err error) {
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

func getQuantityForDogu(dogu *k8sv1.Dogu) (resource.Quantity, error) {
	size := dogu.Spec.Resources.DataVolumeSize
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return resource.Quantity{}, fmt.Errorf("failed to parse to quantity: %w", err)
	}

	return quantity, nil
}

type checkIfPVCIsResizedStep struct {
	client        client.Client
	eventRecorder record.EventRecorder
}

func (w *checkIfPVCIsResizedStep) GetStartCondition() string {
	return startConditionWaitForResize
}

func (w *checkIfPVCIsResizedStep) Execute(ctx context.Context, dogu *k8sv1.Dogu) (string, error) {
	quantity, err := getQuantityForDogu(dogu)
	if err != nil {
		return w.GetStartCondition(), err
	}

	return startConditionScaleUp, w.waitForPVCResize(ctx, dogu, quantity)
}

func (w *checkIfPVCIsResizedStep) waitForPVCResize(ctx context.Context, doguResource *k8sv1.Dogu, quantity resource.Quantity) error {
	w.eventRecorder.Event(doguResource, corev1.EventTypeNormal, VolumeExpansionEventReason, "Wait for pvc to be resized...")
	pvc, err := doguResource.GetDataPVC(ctx, w.client)
	if err != nil {
		return err
	}

	resized := isPvcStorageResized(pvc, quantity)
	if !resized {
		return notResizedError{
			state:       w.GetStartCondition(),
			requeueTime: time.Minute * 1,
		}
	}

	return nil
}

func isPvcStorageResized(pvc *corev1.PersistentVolumeClaim, quantity resource.Quantity) bool {
	return pvc.Status.Capacity.Storage().Equal(quantity)
}
