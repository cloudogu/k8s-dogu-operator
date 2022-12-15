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

type doguVolumeManager struct {
	client        client.Client
	eventRecorder record.EventRecorder
}

// NewDoguVolumeManager creates a new instance of the doguVolumeManager.
func NewDoguVolumeManager(client client.Client, eventRecorder record.EventRecorder) *doguVolumeManager {
	return &doguVolumeManager{
		client:        client,
		eventRecorder: eventRecorder,
	}
}

// SetDoguDataVolumeSize sets the quantity from the doguResource in the dogu data PVC.
func (d *doguVolumeManager) SetDoguDataVolumeSize(ctx context.Context, doguResource *k8sv1.Dogu) error {
	size := doguResource.Spec.Resources.DataVolumeSize
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return fmt.Errorf("failed to parse %s to quantity: %w", size, err)
	}

	err = d.updatePVCQuantity(ctx, doguResource, quantity)
	if err != nil {
		return err
	}

	oldReplicas, err := d.scaleDeployment(ctx, doguResource, 0)
	if err != nil {
		return err
	}

	err = d.waitForPVCResize(ctx, doguResource, quantity)
	if err != nil {
		return err
	}

	_, err = d.scaleDeployment(ctx, doguResource, oldReplicas)
	if err != nil {
		return err
	}

	return nil
}

func (d *doguVolumeManager) waitForPVCResize(ctx context.Context, doguResource *k8sv1.Dogu, quantity resource.Quantity) error {
	d.eventRecorder.Event(doguResource, corev1.EventTypeNormal, VolumeExpansionEventReason, "Wait for pvc to be resized...")
	backoffMax24Hour := wait.Backoff{
		Duration: 30 * time.Second,
		Factor:   1.5,
		Jitter:   0,
		Steps:    99999,
		Cap:      24 * time.Hour,
	}
	err := wait.ExponentialBackoffWithContext(ctx, backoffMax24Hour, func() (done bool, err error) {
		// Event
		pvc, err := doguResource.GetDataPVC(ctx, d.client)
		if err != nil {
			return false, err
		}

		return isPvcStorageResized(pvc, quantity), nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for resizing data PVC for dogu %s: %w", doguResource.Name, err)
	}

	return nil
}

func (d *doguVolumeManager) scaleDeployment(ctx context.Context, doguResource *k8sv1.Dogu, newReplicas int32) (oldReplicas int32, err error) {
	d.eventRecorder.Eventf(doguResource, corev1.EventTypeNormal, VolumeExpansionEventReason, "Scale deployment to %d replicas...", newReplicas)
	deployment, err := doguResource.GetDeployment(ctx, d.client)
	if err != nil {
		return 0, err
	}

	oldReplicas = *deployment.Spec.Replicas
	*deployment.Spec.Replicas = newReplicas
	err = d.client.Update(ctx, deployment)
	if err != nil {
		return 0, fmt.Errorf("failed to scale deployment for dogu %s: %w", doguResource.Name, err)
	}
	return oldReplicas, err
}

func (d *doguVolumeManager) updatePVCQuantity(ctx context.Context, doguResource *k8sv1.Dogu, quantity resource.Quantity) error {
	d.eventRecorder.Event(doguResource, corev1.EventTypeNormal, VolumeExpansionEventReason, "Update dogu data PVC request storage...")
	pvc, err := doguResource.GetDataPVC(ctx, d.client)
	if err != nil {
		return err
	}
	pvc.Spec.Resources.Requests[corev1.ResourceStorage] = quantity
	err = d.client.Update(ctx, pvc)
	if err != nil {
		return fmt.Errorf("failed to update PVC %s: %w", pvc.Name, err)
	}
	return err
}

func isPvcStorageResized(pvc *corev1.PersistentVolumeClaim, quantity resource.Quantity) bool {
	return pvc.Status.Capacity.Storage().Equal(quantity)
}
