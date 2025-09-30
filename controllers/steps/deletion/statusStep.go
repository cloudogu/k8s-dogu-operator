package deletion

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ReasonDeleting = "Deleting"
)

type StatusStep struct {
	doguInterface doguInterface
}

func NewStatusStep(doguInterface doguClient.DoguInterface) *StatusStep {
	return &StatusStep{doguInterface: doguInterface}
}

func (s *StatusStep) Run(ctx context.Context, resource *v2.Dogu) steps.StepResult {
	resource.Status.Status = v2.DoguStatusDeleting
	resource.Status.Health = v2.UnavailableHealthStatus

	lastTransitionTime := steps.Now()
	const message = "The dogu is being deleted."
	meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
		Type:               v2.ConditionHealthy,
		Status:             metav1.ConditionFalse,
		Reason:             ReasonDeleting,
		Message:            message,
		LastTransitionTime: lastTransitionTime.Rfc3339Copy(),
	})
	meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
		Type:               v2.ConditionReady,
		Status:             metav1.ConditionFalse,
		Reason:             ReasonDeleting,
		Message:            message,
		LastTransitionTime: lastTransitionTime.Rfc3339Copy(),
	})

	var err error
	resource, err = s.doguInterface.UpdateStatus(ctx, resource, metav1.UpdateOptions{})
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to update status of dogu when deleting: %w", err))
	}

	return steps.Continue()
}
