package postinstall

import (
	"context"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

type ExportModeStep struct {
	exportManager exportManager
}

func NewExportModeStep(manager exportManager) *ExportModeStep {
	return &ExportModeStep{
		exportManager: manager,
	}
}

func (ems *ExportModeStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	return 0, ems.exportManager.UpdateExportMode(ctx, doguResource)
}
