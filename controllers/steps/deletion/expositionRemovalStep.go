package deletion

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exposition"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type ExpositionRemoverStep struct {
	expositionManager expositionManager
}

func NewExpositionRemoverStep(expositionManager exposition.Manager) *ExpositionRemoverStep {
	return &ExpositionRemoverStep{expositionManager: expositionManager}
}

func (ers *ExpositionRemoverStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	if err := ers.expositionManager.RemoveExposition(ctx, doguResource.GetSimpleDoguName()); err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
