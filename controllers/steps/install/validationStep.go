package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/additionalMount"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/security"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/upgrade"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ValidationStep struct {
	premisesChecker               premisesChecker
	localDoguFetcher              localDoguFetcher
	resourceDoguFetcher           resourceDoguFetcher
	securityValidator             securityValidator
	doguAdditionalMountsValidator doguAdditionalMountsValidator
	dependencyValidator           dependencyValidator
}

func (vs *ValidationStep) Priority() int {
	return 5200
}

func NewValidationStep(
	checker upgrade.PremisesChecker,
	localDoguFetcher cesregistry.LocalDoguFetcher,
	resourceDoguFetcher cesregistry.ResourceDoguFetcher,
	dependencyValidator dependency.Validator,
	securityValidator security.Validator,
	doguAdditionalMountsValidator additionalMount.Validator,
) *ValidationStep {
	return &ValidationStep{
		premisesChecker:               checker,
		localDoguFetcher:              localDoguFetcher,
		resourceDoguFetcher:           resourceDoguFetcher,
		dependencyValidator:           dependencyValidator,
		securityValidator:             securityValidator,
		doguAdditionalMountsValidator: doguAdditionalMountsValidator,
	}
}

func (vs *ValidationStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	fromDogu, toDogu, _, err := vs.getDogusForUpgrade(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	if fromDogu != nil && toDogu != nil && fromDogu.Version != toDogu.Version {
		err = vs.premisesChecker.Check(ctx, doguResource, toDogu, fromDogu)
		if err != nil {
			return steps.RequeueWithError(fmt.Errorf("failed a premise check: %w", err))
		}

		return steps.Continue()
	}

	err = vs.dependencyValidator.ValidateDependencies(ctx, toDogu)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	err = vs.securityValidator.ValidateSecurity(toDogu, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	err = vs.doguAdditionalMountsValidator.ValidateAdditionalMounts(ctx, toDogu, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	return steps.Continue()
}

func (vs *ValidationStep) getDogusForUpgrade(ctx context.Context, doguResource *v2.Dogu) (*core.Dogu, *core.Dogu, *v2.DevelopmentDoguMap, error) {
	logger := log.FromContext(ctx).
		WithName("ValidationStep.getDogusForUpgrade").
		WithValues("doguName", doguResource.Name)
	fromDogu, err := vs.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		logger.Info("dogu ist not installed. Installation will be started.")
	}

	toDogu, developmentDoguMap, err := vs.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to fetch dogu descriptor: %w", err)
	}

	return fromDogu, toDogu, developmentDoguMap, nil
}
