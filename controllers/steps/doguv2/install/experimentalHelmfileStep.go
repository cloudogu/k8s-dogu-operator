package install

import (
	"context"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ExperimentalHelmfileStep struct {
	operatorConfig *config.OperatorConfig
}

func NewExperimentalHelmfileStep(config *config.OperatorConfig) *ExperimentalHelmfileStep {
	return &ExperimentalHelmfileStep{
		operatorConfig: config,
	}
}

func (e *ExperimentalHelmfileStep) Run(ctx context.Context, resource *doguv2.Dogu) steps.StepResult {
	if !e.operatorConfig.ExperimentalHelmfileSupport {
		return steps.Continue()
	}

	logger := log.FromContext(ctx).
		WithName("experimentalHelmfileStep").
		WithValues("dogu", resource.Name)
	logger.Info("handling helmfile dogu")

	return steps.Abort()
}
