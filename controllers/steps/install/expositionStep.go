package install

import (
	"context"
	"fmt"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exposition"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ExpositionStep struct {
	expositionManager expositionManager
	serviceInterface  serviceInterface
	expositionEnabled bool
}

func NewExpositionStep(expositionManager exposition.Manager, serviceInterface v1.ServiceInterface, operatorConfig *config.OperatorConfig) *ExpositionStep {
	return &ExpositionStep{
		expositionManager: expositionManager,
		serviceInterface:  serviceInterface,
		expositionEnabled: operatorConfig.ExpositionEnabled,
	}
}

func (es *ExpositionStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	logger := log.FromContext(ctx).WithName("expositionStep")

	if !es.expositionEnabled {
		logger.Info("Exposition is disabled, skipping exposition")
		return steps.Continue()
	}

	doguService, err := es.serviceInterface.Get(ctx, doguResource.Name, metav1.GetOptions{})
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to get dogu service for %q: %w", doguResource.Name, err))
	}

	if err := es.expositionManager.EnsureExposition(ctx, doguResource, doguService); err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}
