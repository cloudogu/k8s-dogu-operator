package install

import (
	"context"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ConditionReason  = "Initializing"
	ConditionMessage = "Controller is initializing, status not yet determined"
)

var expectedConditions = []string{
	doguv2.ConditionHealthy,
	doguv2.ConditionMeetsMinVolumeSize,
	doguv2.ConditionReady,
	doguv2.ConditionSupportMode,
	doguv2.ConditionPauseReconciliation,
}

type ConditionsStep struct {
	conditionUpdater ConditionUpdater
}

func NewConditionsStep(updater ConditionUpdater) *ConditionsStep {
	return &ConditionsStep{
		conditionUpdater: updater,
	}
}

func (cs *ConditionsStep) Run(ctx context.Context, doguResource *doguv2.Dogu) steps.StepResult {
	if doguResource.Status.Conditions == nil {
		doguResource.Status.Conditions = make([]v1.Condition, 0)
	}

	existingConditions := sets.NewString()
	for _, condition := range doguResource.Status.Conditions {
		existingConditions.Insert(condition.Type)
	}

	var conditions []v1.Condition
	for _, condition := range expectedConditions {
		if !existingConditions.Has(condition) {
			conditions = append(conditions, v1.Condition{
				Type:               condition,
				Status:             v1.ConditionUnknown,
				Reason:             ConditionReason,
				Message:            ConditionMessage,
				ObservedGeneration: doguResource.Generation,
			})
		}
	}

	if len(conditions) > 0 {
		err := cs.conditionUpdater.UpdateConditions(ctx, doguResource, conditions)
		if err != nil {
			return steps.RequeueWithError(err)
		}
	}

	return steps.Continue()
}
