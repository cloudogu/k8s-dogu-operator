package install

import (
	"context"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv3"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdatePendingStep runs last, after the source and release checks have completed.
// Paused or stopped dogus retain differences reported for the current generation.
// An older condition can be cleared only because preceding checks found no current
// difference; an incomplete comparison must stop the chain before reaching here.
type UpdatePendingStep struct {
	client K8sClient
}

func NewUpdatePendingStep(client K8sClient) *UpdatePendingStep {
	return &UpdatePendingStep{
		client: client,
	}
}

func (ups *UpdatePendingStep) Run(ctx context.Context, doguResource *v3beta1.Dogu) doguv3.StepResult {
	if doguResource.Spec.PauseReconciliation || doguResource.Spec.Stopped {
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
	reason, message := v3beta1.ReasonStopped, "No changes are pending for the stopped dogu"
	if doguResource.Spec.PauseReconciliation {
		reason, message = v3beta1.ReasonReconciliationPaused, "Reconciliation is paused"
	}
	return ups.setUpdatePendingCondition(ctx, doguResource, v1.ConditionFalse, reason, message)
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
