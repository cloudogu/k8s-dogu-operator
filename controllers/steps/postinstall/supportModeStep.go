package postinstall

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

type SupportModeStep struct {
	supportManager supportManager
}

func NewSupportModeStep(manager supportManager) *SupportModeStep {
	return &SupportModeStep{
		supportManager: manager,
	}
}

func (sms *SupportModeStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	_, err = sms.supportManager.HandleSupportMode(ctx, doguResource)
	return 0, err
}
