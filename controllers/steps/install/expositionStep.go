package install

import (
	"context"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exposition"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ExpositionStep struct {
	expositionManager expositionManager
	expositionEnabled bool
}

func NewExpositionStep(expositionManager exposition.Manager, operatorConfig *config.OperatorConfig) *ExpositionStep {
	return &ExpositionStep{
		expositionManager: expositionManager,
		expositionEnabled: operatorConfig.ExpositionEnabled,
	}
}

func (es *ExpositionStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	logger := log.FromContext(ctx).WithName("expositionStep")

	if !es.expositionEnabled {
		logger.Info("Exposition is disabled, skipping exposition")
		return steps.Continue()
	}

	if err := es.expositionManager.EnsureExposition(ctx, doguResource); err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
