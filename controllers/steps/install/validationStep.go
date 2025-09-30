package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/additionalMount"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/security"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
)

type ValidationStep struct {
	doguHealthChecker             doguHealthChecker
	resourceDoguFetcher           resourceDoguFetcher
	securityValidator             securityValidator
	doguAdditionalMountsValidator doguAdditionalMountsValidator
	dependencyValidator           dependencyValidator
}

func NewValidationStep(
	healthChecker health.DoguHealthChecker,
	resourceDoguFetcher cesregistry.ResourceDoguFetcher,
	dependencyValidator dependency.Validator,
	securityValidator security.Validator,
	doguAdditionalMountsValidator additionalMount.Validator,
) *ValidationStep {
	return &ValidationStep{
		doguHealthChecker:             healthChecker,
		resourceDoguFetcher:           resourceDoguFetcher,
		dependencyValidator:           dependencyValidator,
		securityValidator:             securityValidator,
		doguAdditionalMountsValidator: doguAdditionalMountsValidator,
	}
}

func (vs *ValidationStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	toDogu, _, err := vs.getDogusForUpgrade(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	err = vs.dependencyValidator.ValidateDependencies(ctx, toDogu)
	if err != nil {
		return steps.RequeueWithError(err)
	}

	err = vs.doguHealthChecker.CheckDependenciesRecursive(ctx, toDogu, doguResource.Namespace)
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

func (vs *ValidationStep) getDogusForUpgrade(ctx context.Context, doguResource *v2.Dogu) (*core.Dogu, *v2.DevelopmentDoguMap, error) {
	toDogu, developmentDoguMap, err := vs.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch dogu descriptor: %w", err)
	}

	return toDogu, developmentDoguMap, nil
}
