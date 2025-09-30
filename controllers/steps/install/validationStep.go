package install

import (
	"context"
	"fmt"

	"github.com/cloudogu/ces-commons-lib/errors"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/additionalMount"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/health"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/security"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ValidationStep struct {
	doguHealthChecker             doguHealthChecker
	resourceDoguFetcher           resourceDoguFetcher
	localDoguFetcher              localDoguFetcher
	securityValidator             securityValidator
	doguAdditionalMountsValidator doguAdditionalMountsValidator
	dependencyValidator           dependencyValidator
}

func NewValidationStep(
	healthChecker health.DoguHealthChecker,
	resourceDoguFetcher cesregistry.ResourceDoguFetcher,
	localDoguFetcher cesregistry.LocalDoguFetcher,
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
		localDoguFetcher:              localDoguFetcher,
	}
}

func (vs *ValidationStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	logger := log.FromContext(ctx)
	fromDogu, err := vs.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil && !errors.IsNotFoundError(err) {
		return steps.RequeueWithError(err)
	}
	if fromDogu != nil {
		older, err := isOlder(doguResource.Spec.Version, fromDogu.Version)
		if err != nil {
			return steps.RequeueWithError(err)
		}
		if older && !doguResource.Spec.UpgradeConfig.ForceUpgrade {
			logger.Info(fmt.Sprintf("Downgrade is not allowed. Please install another version of dogu %s", doguResource.Name))
			return steps.Abort()
		}
	}

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

func isOlder(version1Raw, version2Raw string) (bool, error) {
	version1, err := core.ParseVersion(version1Raw)
	if err != nil {
		return false, err
	}

	version2, err := core.ParseVersion(version2Raw)
	if err != nil {
		return false, err
	}

	return version1.IsOlderThan(version2), nil
}
