package deletion

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/warpmenuentry"
)

type WarpMenuEntryRemoverStep struct {
	warpMenuEntryManager warpMenuEntryManager
}

func NewWarpMenuEntryRemoverStep(warpMenuEntryManager warpmenuentry.Manager) *WarpMenuEntryRemoverStep {
	return &WarpMenuEntryRemoverStep{warpMenuEntryManager: warpMenuEntryManager}
}

func (wrs *WarpMenuEntryRemoverStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	if err := wrs.warpMenuEntryManager.RemoveWarpMenuEntry(ctx, doguResource.GetSimpleDoguName()); err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
