package postinstall

import (
	"context"
	"fmt"
	"time"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	opresource "github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const requeueAfterVolume = 10 * time.Second
const ActualVolumeSizeMeetsMinDataSize = "ActualVolumeSizeMeetsMinDataSize"

type VolumeExpanderStep struct {
	client           k8sClient
	doguInterface    doguInterface
	localDoguFetcher localDoguFetcher
}

func NewVolumeExpanderStep(client client.Client, doguInterface doguClient.DoguInterface, fetcher cesregistry.LocalDoguFetcher) *VolumeExpanderStep {
	return &VolumeExpanderStep{
		client:           client,
		doguInterface:    doguInterface,
		localDoguFetcher: fetcher,
	}
}

func (vs *VolumeExpanderStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	dogu, err := vs.localDoguFetcher.FetchInstalled(ctx, cescommons.SimpleName(doguResource.Name))
	if err != nil {
		return steps.RequeueWithError(err)
	}
	if !hasPvc(dogu) {
		err = vs.setSuccessCondition(ctx, doguResource)
		if err != nil {
			return steps.RequeueWithError(err)
		}
		return steps.Continue()
	}
	pvc, err := doguResource.GetDataPVC(ctx, vs.client)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	quantity, err := doguResource.GetMinDataVolumeSize()
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if vs.isPvcStorageResized(pvc, quantity) {
		err = vs.setSuccessCondition(ctx, doguResource)
		if err != nil {
			return steps.RequeueWithError(err)
		}
		return steps.Continue()
	}

	if pvc.Spec.Size() == quantity.Size() {
		return steps.RequeueAfter(requeueAfterVolume)
	}

	err = opresource.SetCurrentDataVolumeSize(ctx, vs.doguInterface, vs.client, doguResource, pvc)
	if err != nil {
		steps.RequeueWithError(err)
	}

	// It is necessary to create a new map because just setting a new quantity results in an exception.
	pvc.Spec.Resources.Requests = map[corev1.ResourceName]resource.Quantity{corev1.ResourceStorage: quantity}
	err = vs.client.Update(ctx, pvc)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to update PVC %s: %w", pvc.Name, err))
	}

	return steps.RequeueAfter(requeueAfterVolume)
}

func (vs *VolumeExpanderStep) isPvcStorageResized(pvc *corev1.PersistentVolumeClaim, quantity resource.Quantity) bool {
	// Longhorn works this way and does not add the Condition "FileSystemResizePending" to the PVC
	// see https://github.com/longhorn/longhorn/issues/2749
	isRequestedCapacityAvailable := pvc.Status.Capacity.Storage().Value() >= quantity.Value()
	return isRequestedCapacityAvailable
}

func hasPvc(dogu *core.Dogu) bool {
	for _, volume := range dogu.Volumes {
		if volume.NeedsBackup {
			return true
		}
	}
	return false
}

func (vs *VolumeExpanderStep) setSuccessCondition(ctx context.Context, doguResource *v2.Dogu) error {
	condition := v1.Condition{
		Type:               v2.ConditionMeetsMinVolumeSize,
		Status:             v1.ConditionTrue,
		Reason:             ActualVolumeSizeMeetsMinDataSize,
		Message:            "Current VolumeSize meets the configured minimum VolumeSize",
		LastTransitionTime: v1.Now().Rfc3339Copy(),
	}

	doguResource, err := vs.doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status v2.DoguStatus) v2.DoguStatus {
		meta.SetStatusCondition(&status.Conditions, condition)
		return status
	}, v1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
