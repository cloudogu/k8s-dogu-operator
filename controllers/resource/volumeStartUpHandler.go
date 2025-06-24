package resource

import (
	"context"
	"fmt"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/retry-lib/retry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ActualVolumeSizeMeetsMinDataSize = "ActualVolumeSizeMeetsMinDataSize"
	VolumeSizeNotMeetsMinDataSize    = "VolumeSizeNotMeetsMinDataSize"
)

type VolumeStartUpHandlerClient interface {
	client.Client
}

type VolumeStartUpHandlerDoguInterface interface {
	doguClient.DoguInterface
}

type VolumneStartupHandler struct {
	client        VolumeStartUpHandlerClient
	doguInterface VolumeStartUpHandlerDoguInterface
}

func NewVolumeStartupHandler(client client.Client, doguInterface doguClient.DoguInterface) *VolumneStartupHandler {
	return &VolumneStartupHandler{
		client:        client,
		doguInterface: doguInterface,
	}
}

func (v *VolumneStartupHandler) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.WithName("volume startup handler").Info("updating data volume size of all dogus on startup")

	list, err := v.doguInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, dogu := range list.Items {
		pvc, e := dogu.GetDataPVC(ctx, v.client)
		if e != nil {
			logger.Info(fmt.Sprintf("no pvc for dogu %s: %v", dogu.Name, e))
			continue
		}
		_ = SetCurrentDataVolumeSize(ctx, v.doguInterface, v.client, &dogu, pvc)
	}
	return nil
}

// SetCurrentDataVolumeSize set the current DataVolumeSize within the status of the dogu
func SetCurrentDataVolumeSize(ctx context.Context, doguInterface doguClient.DoguInterface, client client.Client, doguResource *doguv2.Dogu, pvc *corev1.PersistentVolumeClaim) error {
	logger := log.FromContext(ctx)

	// Check min size condition
	condition := metav1.Condition{
		Type:               doguv2.ConditionMeetsMinVolumeSize,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
	}
	minDataSize, err := doguResource.GetMinDataVolumeSize()
	if err != nil {
		logger.Error(err, "failed to get min data volume size")
		return err
	}
	var currentSize *resource.Quantity
	condition.Reason = ActualVolumeSizeMeetsMinDataSize
	currentSize = &minDataSize
	if pvc != nil {
		currentSize = pvc.Status.Capacity.Storage()
		// is minDataSize larger than currentsize
		if minDataSize.Cmp(*currentSize) > 0 {
			logger.Info(fmt.Sprintf("set condition for resizing %d - %d -> %v", currentSize.Value(), minDataSize.Value(), condition.Status))
			condition.Status = metav1.ConditionFalse
			condition.Message = fmt.Sprintf("Current VolumeSize '%d' is less then the configured minimum VolumeSize '%d'", currentSize.Value(), minDataSize.Value())
			condition.Reason = VolumeSizeNotMeetsMinDataSize
		}
		// Resize PVC is current dogu size is larger than current pvc-capacity
		// this might happen during backup and restore
		specsize := pvc.Spec.Resources.Requests.Storage()
		if specsize.Cmp(*currentSize) < 0 {
			logger.Info(fmt.Sprintf("set spec request size for pvc %d - %d", specsize.Value(), currentSize.Value()))
			specrequests := make(map[corev1.ResourceName]resource.Quantity)
			specrequests[corev1.ResourceStorage] = *currentSize
			pvc.Spec.Resources.Requests = specrequests
			err = retry.OnConflict(func() error {
				return client.Update(ctx, pvc)
			})
			if err != nil {
				logger.Error(err, "failed to update pvc size")
				return err
			}
		}
	}

	_, err = doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status doguv2.DoguStatus) doguv2.DoguStatus {
		meta.SetStatusCondition(&status.Conditions, condition)
		logger.Info(fmt.Sprintf("set data volume size %v for dogu %s", currentSize, doguResource.Name))
		status.DataVolumeSize = currentSize
		return status
	}, metav1.UpdateOptions{})

	if err != nil {
		logger.Error(err, "failed to update data volume size")
		return err
	}

	return nil
}
