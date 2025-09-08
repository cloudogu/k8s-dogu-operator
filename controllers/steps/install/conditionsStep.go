package install

import (
	"context"
	"fmt"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
}

type ConditionsStep struct {
	doguInterface doguInterface
}

func NewConditionsStep(mgrSet *util.ManagerSet, namespace string) *ConditionsStep {
	return &ConditionsStep{
		doguInterface: mgrSet.EcosystemClient.Dogus(namespace),
	}
}

func (cs *ConditionsStep) Run(ctx context.Context, doguResource *doguv2.Dogu) steps.StepResult {
	logger := log.FromContext(ctx)
	if doguResource.Status.Conditions != nil {
		doguResource.Status.Conditions = make([]v1.Condition, 0)
	}
	if len(doguResource.Status.Conditions) == len(expectedConditions) {
		logger.Info(fmt.Sprintf("All Conditions for dogu %s are set", doguResource.GetSimpleDoguName()))
		return steps.Continue()
	}

	existingConditions := sets.NewString()
	conditions := doguResource.Status.Conditions
	for _, condition := range doguResource.Status.Conditions {
		existingConditions.Insert(condition.Type)
	}

	for _, condition := range expectedConditions {
		if !existingConditions.Has(condition) {
			conditions = append(conditions, v1.Condition{
				Type:               condition,
				Status:             v1.ConditionUnknown,
				Reason:             ConditionReason,
				Message:            ConditionMessage,
				LastTransitionTime: v1.Now(),
			})
		}
	}
	doguResource.Status.Conditions = conditions

	_, err := cs.doguInterface.UpdateStatus(ctx, doguResource, v1.UpdateOptions{})
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
