package deletion

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type DeleteOutOfHealthConfigMapStep struct {
	doguHealthStatusUpdater doguHealthStatusUpdater
}

func NewDeleteOutOfHealthConfigMapStep(doguHealthStatusUpdater health.DoguHealthStatusUpdater) *DeleteOutOfHealthConfigMapStep {
	return &DeleteOutOfHealthConfigMapStep{
		doguHealthStatusUpdater: doguHealthStatusUpdater,
	}
}

func (dhc *DeleteOutOfHealthConfigMapStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	err := dhc.doguHealthStatusUpdater.DeleteDoguOutOfHealthConfigMap(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}
	return steps.Continue()
}
