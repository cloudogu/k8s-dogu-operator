package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/warpmenuentry"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type WarpMenuEntryStep struct {
	warpmenuEntryManager warpMenuEntryManager
	warpMenuEntryEnabled bool
}

func NewWarpMenuEntryStep(warpmenuEntryManager warpmenuentry.Manager, operatorConfig *config.OperatorConfig) *WarpMenuEntryStep {
	return &WarpMenuEntryStep{
		warpmenuEntryManager: warpmenuEntryManager,
		warpMenuEntryEnabled: operatorConfig.WarpMenuEntryEnabled,
	}
}

func (ws *WarpMenuEntryStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	logger := log.FromContext(ctx).WithName("warpMenuEntryStep")
	if !ws.warpMenuEntryEnabled {
		logger.Info("WarpMenuEntry is disabled, skipping warpMenuEntry")
		return steps.Continue()
	}

	if err := ws.warpmenuEntryManager.EnsureWarpMenuEntry(ctx, doguResource); err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
