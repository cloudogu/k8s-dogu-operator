package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/warpmenuentry"
)

type WarpMenuEntryStep struct {
	warpmenuEntryManager warpMenuEntryManager
}

func NewWarpMenuEntryStep(warpmenuEntryManager warpmenuentry.Manager) *WarpMenuEntryStep {
	return &WarpMenuEntryStep{warpmenuEntryManager: warpmenuEntryManager}
}

func (ws *WarpMenuEntryStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	if err := ws.warpmenuEntryManager.EnsureWarpMenuEntry(ctx, doguResource); err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
