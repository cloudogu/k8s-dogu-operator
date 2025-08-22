package postinstall

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type ExportModeStep struct {
	exportManager exportManager
}

func NewExportModeStep(manager exportManager) *ExportModeStep {
	return &ExportModeStep{
		exportManager: manager,
	}
}

func (ems *ExportModeStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	return steps.NewStepResultContinueIsTrueAndRequeueIsZero(ems.exportManager.UpdateExportMode(ctx, doguResource))
}
