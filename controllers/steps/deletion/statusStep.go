package deletion

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ReasonDeleting = "Deleting"
)

// The StatusStep sets the status of the dogu to deleting and the healthy and ready conditions to false.
type StatusStep struct {
	doguInterface doguInterface
}

func NewStatusStep(doguInterface doguClient.DoguInterface) *StatusStep {
	return &StatusStep{doguInterface: doguInterface}
}

func (s *StatusStep) Run(ctx context.Context, resource *v2.Dogu) steps.StepResult {
	resource, err := s.doguInterface.UpdateStatusWithRetry(ctx, resource, func(status v2.DoguStatus) v2.DoguStatus {
		status.Status = v2.DoguStatusDeleting
		status.Health = v2.UnavailableHealthStatus

		const message = "The dogu is being deleted."
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:               v2.ConditionHealthy,
			Status:             metav1.ConditionFalse,
			Reason:             ReasonDeleting,
			Message:            message,
			ObservedGeneration: resource.Generation,
		})
		meta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:               v2.ConditionReady,
			Status:             metav1.ConditionFalse,
			Reason:             ReasonDeleting,
			Message:            message,
			ObservedGeneration: resource.Generation,
		})

		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return steps.Continue()
		}
		return steps.RequeueWithError(fmt.Errorf("failed to update status of dogu when deleting: %w", err))
	}

	return steps.Continue()
}
