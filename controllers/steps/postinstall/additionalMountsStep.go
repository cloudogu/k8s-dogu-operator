package postinstall

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type AdditionalMountsStep struct {
	additionalMountManager
}

func NewAdditionalMountsStep(manager additionalMountManager) *AdditionalMountsStep {
	return &AdditionalMountsStep{
		additionalMountManager: manager,
	}
}

func (ams *AdditionalMountsStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	changed, err := ams.AdditionalMountsChanged(ctx, doguResource)
	if err != nil {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
	}
	if changed {
		return steps.NewStepResultContinueIsTrueAndRequeueIsZero(ams.UpdateAdditionalMounts(ctx, doguResource))
	}
	return steps.StepResult{}
}
