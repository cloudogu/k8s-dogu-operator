package controllers

import (
	"context"
	"fmt"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	opresource "github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const requeueAfterVolume = 10 * time.Second
const scaleDownReplicas = 0
const scaleUpReplicas = 1

type VolumeStep struct {
	doguVolumeManager *doguVolumeManager
	client            client.Client
	doguInterface     doguInterface
}

func NewVolumeStep(client client.Client, eventRecorder record.EventRecorder, doguInterface doguClient.DoguInterface) *VolumeStep {
	return &VolumeStep{
		client:            client,
		doguVolumeManager: NewDoguVolumeManager(client, eventRecorder, doguInterface),
	}
}

func (vs *VolumeStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	// TODO Non blocking
	pvc, err := doguResource.GetDataPVC(ctx, vs.client)
	if err != nil {
		return 0, err
	}
	quantity, err := doguResource.GetMinDataVolumeSize()
	if err != nil {
		return 0, err
	}
	if vs.isPvcStorageResized(pvc, quantity) {
		return 0, nil
	}
	if !vs.isScaledDown(ctx, vs.client, doguResource) && !vs.isPvcResizeApplicable(pvc) {
		_, err := vs.scaleDeployment(ctx, vs.client, doguResource, scaleDownReplicas)
		if err != nil {
			return 0, fmt.Errorf("failed to scale down replicas: %w", err)
		}
	}

	if vs.isScaledDown(ctx, vs.client, doguResource) && !vs.isPvcResizeApplicable(pvc) {
		_ = opresource.SetCurrentDataVolumeSize(ctx, vs.doguInterface, vs.client, doguResource, pvc)

		// It is necessary to create a new map because just setting a new quantity results in an exception.
		pvc.Spec.Resources.Requests = map[corev1.ResourceName]resource.Quantity{corev1.ResourceStorage: quantity}
		err = vs.client.Update(ctx, pvc)
		if err != nil {
			return 0, fmt.Errorf("failed to update PVC %s: %w", pvc.Name, err)
		}
	}

	if vs.isScaledDown(ctx, vs.client, doguResource) && vs.isPvcResizeApplicable(pvc) {
		_, err := vs.scaleDeployment(ctx, vs.client, doguResource, scaleUpReplicas)
		if err != nil {
			return 0, fmt.Errorf("failed to scale down replicas: %w", err)
		}
	}

	return 0, nil
}

func (vs *VolumeStep) isPvcStorageResized(pvc *corev1.PersistentVolumeClaim, quantity resource.Quantity) bool {
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
func (vs *VolumeStep) isPvcResizeApplicable(pvc *corev1.PersistentVolumeClaim) bool {
	for _, condition := range pvc.Status.Conditions {
		if condition.Type == corev1.PersistentVolumeClaimFileSystemResizePending && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (vs *VolumeStep) scaleDeployment(ctx context.Context, client client.Client, doguResource *v2.Dogu, newReplicas int32) (oldReplicas int32, err error) {
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

func (vs *VolumeStep) isScaledDown(ctx context.Context, client client.Client, doguResource *v2.Dogu) bool {
	deployment, err := doguResource.GetDeployment(ctx, client)
	if err != nil {
		return false
	}
	return *deployment.Spec.Replicas == 0
}
