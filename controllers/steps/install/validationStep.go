package install

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/upgrade"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const requeueAfterValidation = 5 * time.Second

type ValidationStep struct {
	premisesChecker               premisesChecker
	localDoguFetcher              localDoguFetcher
	resourceDoguFetcher           resourceDoguFetcher
	securityValidator             securityValidator
	doguAdditionalMountsValidator doguAdditionalMountsValidator
	dependencyValidator           upgrade.DependencyValidator
}

func NewValidationStep(mgrSet *util.ManagerSet, checker premisesChecker) *ValidationStep {
	return &ValidationStep{
		premisesChecker:     checker,
		localDoguFetcher:    mgrSet.LocalDoguFetcher,
		resourceDoguFetcher: mgrSet.ResourceDoguFetcher,
	}
}

func (vs *ValidationStep) Run(ctx context.Context, doguResource *v2.Dogu) (requeueAfter time.Duration, err error) {
	fromDogu, toDogu, _, err := vs.getDogusForUpgrade(ctx, doguResource)
	if err != nil {
		return requeueAfterValidation, err
	}
	if fromDogu != nil && toDogu != nil && fromDogu.Version != toDogu.Version {
		err = vs.premisesChecker.Check(ctx, doguResource, toDogu, fromDogu)
		if err != nil {
			return requeueAfterValidation, fmt.Errorf("failed a premise check: %w", err)
		}
		return 0, nil
	}

	err = vs.dependencyValidator.ValidateDependencies(ctx, toDogu)
	if err != nil {
		return requeueAfterValidation, err
	}

	err = vs.securityValidator.ValidateSecurity(toDogu, doguResource)
	if err != nil {
		return requeueAfterValidation, err
	}

	err = vs.doguAdditionalMountsValidator.ValidateAdditionalMounts(ctx, toDogu, doguResource)
	if err != nil {
		return requeueAfterValidation, nil
	}
	return 0, nil
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
