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
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

type ValidationStep struct {
	doguHealthChecker             doguHealthChecker
	localDoguFetcher              localDoguFetcher
	securityValidator             securityValidator
	doguAdditionalMountsValidator doguAdditionalMountsValidator
	dependencyValidator           dependencyValidator
	recorder                      eventRecorder
}

func NewValidationStep(
	healthChecker health.DoguHealthChecker,
	fetcher cesregistry.LocalDoguFetcher,
	dependencyValidator dependency.Validator,
	securityValidator security.Validator,
	doguAdditionalMountsValidator additionalMount.Validator,
	recorder record.EventRecorder,
) *ValidationStep {
	return &ValidationStep{
		doguHealthChecker:             healthChecker,
		localDoguFetcher:              fetcher,
		dependencyValidator:           dependencyValidator,
		securityValidator:             securityValidator,
		doguAdditionalMountsValidator: doguAdditionalMountsValidator,
		recorder:                      recorder,
	}
}

func (vs *ValidationStep) Run(ctx context.Context, doguResource *v2.Dogu) steps.StepResult {
	fromDogu, err := vs.localDoguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil && !errors.IsNotFoundError(err) {
		return steps.RequeueWithError(err)
	}
	unallowedDowngrade, err := vs.shouldAbortBecauseOfUnallowedDowngrade(fromDogu, doguResource)
	if err != nil {
		return steps.RequeueWithError(err)
	}
	if unallowedDowngrade {
		return steps.Abort()
	}

	toDogu, err := vs.localDoguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return steps.RequeueWithError(fmt.Errorf("failed to fetch dogu descriptor for %q: %w", doguResource.Name, err))
	}

	if fromDogu != nil {
		changeNamespace := doguResource.Spec.UpgradeConfig.AllowNamespaceSwitch
		err = vs.checkDoguIdentity(fromDogu, toDogu, changeNamespace)
		if err != nil {
			return steps.RequeueWithError(err)
		}
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

func (vs *ValidationStep) checkDoguIdentity(localDogu *core.Dogu, remoteDogu *core.Dogu, namespaceChange bool) error {
	if localDogu.GetSimpleName() != remoteDogu.GetSimpleName() {
		return fmt.Errorf("dogus must have the same name (%s=%s)", localDogu.GetSimpleName(), remoteDogu.GetSimpleName())
	}

	if !namespaceChange && localDogu.GetNamespace() != remoteDogu.GetNamespace() {
		return fmt.Errorf("dogus must have the same namespace (%s=%s)", localDogu.GetNamespace(), remoteDogu.GetNamespace())
	}

	return nil
}

func (vs *ValidationStep) shouldAbortBecauseOfUnallowedDowngrade(fromDogu *core.Dogu, doguResource *v2.Dogu) (bool, error) {
	if fromDogu != nil {
		older, err := isOlder(doguResource.Spec.Version, fromDogu.Version)
		if err != nil {
			return false, err
		}
		if older && !doguResource.Spec.UpgradeConfig.ForceUpgrade {
			vs.recorder.Eventf(doguResource, v1.EventTypeWarning, InstallEventReason, "Downgrade is not allowed. Please install another version of dogu %s", doguResource.Name)
			return true, nil
		}
	}
	return false, nil
}
