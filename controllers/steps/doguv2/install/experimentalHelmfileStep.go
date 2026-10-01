package install

import (
	"context"

	doguv2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	steps "github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps/doguv2"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ExperimentalHelmfileStep struct {
	operatorConfig *config.OperatorConfig
	doguFetcher    resourceDoguFetcher
}

func NewExperimentalHelmfileStep(
	config *config.OperatorConfig,
	fetcher cesregistry.ResourceDoguFetcher,
) *ExperimentalHelmfileStep {
	return &ExperimentalHelmfileStep{
		operatorConfig: config,
		doguFetcher:    fetcher,
	}
}

func (e *ExperimentalHelmfileStep) Run(ctx context.Context, resource *doguv2.Dogu) steps.StepResult {
	logger := log.FromContext(ctx).
		WithName("experimentalHelmfileStep").
		WithValues("dogu", resource.Name)

	if !e.operatorConfig.ExperimentalHelmfileSupport {
		return steps.Continue()
	}

	doguDescriptor, _, err := e.doguFetcher.FetchWithResource(ctx, resource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if doguDescriptor.Properties["kind"] != "Helmfile" {
		logger.Info("dogu is not a helmfile dogu, continuing")
		return steps.Continue()
	}

	logger.Info("handling helmfile dogu")

	return steps.Abort()
}
