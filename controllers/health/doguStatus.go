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
	"github.com/cloudogu/k8s-dogu-operator/retry"
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

	return retry.OnConflict(func() error {
		dogu, err := doguClient.Get(ctx, doguName.Name, metav1api.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get dogu resource %q: %w", doguName, err)
		}

		dogu.Status.Health = doguv1.SelectHealthStatus(isAvailable)

		_, err = doguClient.UpdateStatus(ctx, dogu, metav1api.UpdateOptions{})
		if err != nil {
			message := fmt.Sprintf("failed to update dogu %q with health status %q", doguName, dogu.Status.Health)
			dsw.recorder.Event(dogu, v1.EventTypeWarning, statusUpdateEventReason, message)
			return fmt.Errorf("%s: %w", message, err)
		}

		dsw.recorder.Eventf(dogu, v1.EventTypeNormal, statusUpdateEventReason, "successfully updated health status to %q", dogu.Status.Health)
		return nil
	})
}
