package install

import (
	"context"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdatePendingStep resets any UpdatePending condition to false, if the PauseReconciliation flag is set to false.
type UpdatePendingStep struct {
	client K8sClient
}

func NewUpdatePendingStep(client K8sClient) *UpdatePendingStep {
	return &UpdatePendingStep{
		client: client,
	}
}

func (u *UpdatePendingStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) doguv3.StepResult {
	if doguResource.Spec.PauseReconciliation {
		return doguv3.Continue()
	}
	if meta.SetStatusCondition(&doguResource.Status.Conditions, v1.Condition{
		Type:               v3beta1.ConditionUpdatePending,
		Status:             v1.ConditionFalse,
		ObservedGeneration: doguResource.Generation,
		Reason:             v3beta1.ReasonReconciliationEnabled,
		Message:            "Reconciliation is enabled",
	}) {
		err := u.client.Status().Update(ctx, doguResource)
		if err != nil {
			return doguv3.RequeueWithError(err, v3beta1.ReasonReconciliationEnabled)
		}
	}
	return doguv3.Continue()
}
