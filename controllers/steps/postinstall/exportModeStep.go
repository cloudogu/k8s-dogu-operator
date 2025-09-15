package postinstall

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/manager"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type ExportModeStep struct {
	exportManager exportManager
}

func (ems *ExportModeStep) Priority() int {
	return 3500
}

func NewExportModeStep(doguExportManager manager.DoguExportManager) *ExportModeStep {
	return &ExportModeStep{
		exportManager: doguExportManager,
	}
}

func (ems *ExportModeStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	err := ems.exportManager.UpdateExportMode(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}
	return steps.Continue()
}
