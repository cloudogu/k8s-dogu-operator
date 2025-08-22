package postinstall

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type SupportModeStep struct {
	supportManager supportManager
}

func NewSupportModeStep(manager supportManager) *SupportModeStep {
	return &SupportModeStep{
		supportManager: manager,
	}
}

func (sms *SupportModeStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	_, err := sms.supportManager.HandleSupportMode(ctx, doguResource)
	return steps.NewStepResultContinueIsTrueAndRequeueIsZero(err)
}
