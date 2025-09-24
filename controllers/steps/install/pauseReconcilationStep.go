package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const conditionReasonPaused = "ReconcilationIsPaused"
const conditionReasonNotPaused = "ReconcilationIsRunning"
const conditionMessagePaused = "Reconcilation is paused because of spec change"
const conditionMessageNotPaused = "Reconcilation is currently running"

type PauseReconciliationStep struct {
	doguInterface doguInterface
}

func NewPauseReconciliationStep(doguInterface doguClient.DoguInterface) *PauseReconciliationStep {
	return &PauseReconciliationStep{
		doguInterface: doguInterface,
	}
}

func (prs *PauseReconciliationStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	condition := v1.Condition{
		Type:               v2.ConditionPauseReconciliation,
		Status:             v1.ConditionTrue,
		Reason:             conditionReasonPaused,
		Message:            conditionMessagePaused,
		LastTransitionTime: v1.Now().Rfc3339Copy(),
	}

	if !doguResource.Spec.PauseReconciliation {
		condition.Status = v1.ConditionFalse
		condition.Reason = conditionReasonNotPaused
		condition.Message = conditionMessageNotPaused
	}

	meta.SetStatusCondition(&doguResource.Status.Conditions, condition)
	doguResource, err := prs.doguInterface.UpdateStatus(ctx, doguResource, v1.UpdateOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if doguResource.Spec.PauseReconciliation {
		return steps.Abort()
	}

	return steps.Continue()
}
