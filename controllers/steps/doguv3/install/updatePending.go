package install

import (
	"context"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdatePendingStep resets any UpdatePending condition to false, if the PauseReconciliation flag is set to false.
// It also resets the condition if the PauseReconciliation flag is set to true, but the UpdatePending condition is
// only set for an older resource generation - because that indicates, that no changes would be made to OCIRepository
// or HelmRelease based on the _current_ dogu resource generation.
type UpdatePendingStep struct {
	client K8sClient
}

func NewUpdatePendingStep(client K8sClient) *UpdatePendingStep {
	return &UpdatePendingStep{
		client: client,
	}
}

func (ups *UpdatePendingStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) doguv3.StepResult {
	if doguResource.Spec.PauseReconciliation {
		return ups.checkUpdatePendingCondition(ctx, doguResource)
	}
	return ups.setUpdatePendingCondition(ctx, doguResource, v1.ConditionFalse, v3beta1.ReasonNotSuspended, "Reconciliation is enabled")
}

func (ups *UpdatePendingStep) checkUpdatePendingCondition(ctx context.Context, doguResource *v3beta1.Dogu) doguv3.StepResult {
	condition := meta.FindStatusCondition(doguResource.Status.Conditions, v3beta1.ConditionUpdatePending)
	if condition == nil {
		return doguv3.Continue()
	}
	if condition.ObservedGeneration == doguResource.Generation {
		return doguv3.Continue()
	}
	return ups.setUpdatePendingCondition(ctx, doguResource, v1.ConditionFalse, v3beta1.ReasonReconciliationPaused, "Reconciliation is paused")
}

func (ups *UpdatePendingStep) setUpdatePendingCondition(ctx context.Context, doguResource *v3beta1.Dogu, status v1.ConditionStatus, reason string, message string) doguv3.StepResult {
	if meta.SetStatusCondition(&doguResource.Status.Conditions, v1.Condition{
		Type:               v3beta1.ConditionUpdatePending,
		Status:             status,
		ObservedGeneration: doguResource.Generation,
		Reason:             reason,
		Message:            message,
	}) {
		err := ups.client.Status().Update(ctx, doguResource)
		if err != nil {
			return doguv3.RequeueWithError(err, reason)
		}
	}
	return doguv3.Continue()
}
