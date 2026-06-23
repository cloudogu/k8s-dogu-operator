package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/warpmenuentry"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type WarpMenuEntryStep struct {
	warpMenuEntryManager warpMenuEntryManager
	serviceInterface     serviceInterface
}

func NewWarpMenuEntryStep(warpMenuEntryManager warpmenuentry.Manager) *WarpMenuEntryStep {
	return &WarpMenuEntryStep{
		warpMenuEntryManager: warpMenuEntryManager,
	}
}

func (wmes *WarpMenuEntryStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {

	logrus.Infof("manoj , inside deployment method [%s]", doguResource.Name)
	logger := log.FromContext(ctx).WithName("WarpMenuEntryStep")
	logger.Info("Entered WarpMenuEntryStep")

	if err := wmes.warpMenuEntryManager.EnsureWarpMenuEntry(ctx, doguResource); err != nil {
		logger.Error(err, "An error occured in EnsureWarpMenuEntry")
		return steps.RequeueWithError(err)
	}
	logger.Info("WarpMenuEntry creation/updation succeeded")
	return steps.Continue()

}
