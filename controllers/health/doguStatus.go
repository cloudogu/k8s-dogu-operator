package health

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1api "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	doguv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

const statusUpdateEventReason = "HealthStatusUpdate"

type DoguStatusUpdater struct {
	ecosystemClient ecoSystem.EcoSystemV1Alpha1Interface
	recorder        record.EventRecorder
}

func NewDoguStatusUpdater(ecosystemClient ecoSystem.EcoSystemV1Alpha1Interface, recorder record.EventRecorder) *DoguStatusUpdater {
	return &DoguStatusUpdater{ecosystemClient: ecosystemClient, recorder: recorder}
}

// UpdateStatus sets the health status of the dogu according to whether if it's available or not.
func (dsw *DoguStatusUpdater) UpdateStatus(ctx context.Context, doguName types.NamespacedName, isAvailable bool) error {
	doguClient := dsw.ecosystemClient.Dogus(doguName.Namespace)

	dogu, err := doguClient.Get(ctx, doguName.Name, metav1api.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get dogu resource %q: %w", doguName, err)
	}

	desiredHealthStatus := doguv1.UnavailableHealthStatus
	if isAvailable {
		desiredHealthStatus = doguv1.AvailableHealthStatus
	}

	_, err = doguClient.UpdateStatusWithRetry(ctx, dogu, func(status doguv1.DoguStatus) doguv1.DoguStatus {
		status.Health = desiredHealthStatus
		return status
	}, metav1api.UpdateOptions{})

	if err != nil {
		message := fmt.Sprintf("failed to update dogu %q with current health status [%q] to desired health status [%q]", doguName, dogu.Status.Health, desiredHealthStatus)
		dsw.recorder.Event(dogu, v1.EventTypeWarning, statusUpdateEventReason, message)
		return fmt.Errorf("%s: %w", message, err)
	}

	dsw.recorder.Eventf(dogu, v1.EventTypeNormal, statusUpdateEventReason, "successfully updated health status to %q", desiredHealthStatus)
	return nil
}
