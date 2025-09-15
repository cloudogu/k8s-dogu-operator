package deletion

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type UnregisterDoguVersionStep struct {
	doguRegistrator doguRegistrator
}

func (udvs *UnregisterDoguVersionStep) Priority() int {
	return 5900
}

func NewUnregisterDoguVersionStep(registrator cesregistry.DoguRegistrator) *UnregisterDoguVersionStep {
	return &UnregisterDoguVersionStep{
		doguRegistrator: registrator,
	}
}

func (udvs *UnregisterDoguVersionStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	err := udvs.doguRegistrator.UnregisterDogu(ctx, doguResource.Name)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to register dogu: %w", err))
	}
	return steps.Continue()
}
