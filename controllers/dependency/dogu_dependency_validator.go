package dependency

import (
	"context"
	"errors"
	"fmt"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-registry-lib/dogu"

	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
)

// dependencyValidationError is returned when a given dependency cloud not be validated.
type dependencyValidationError struct {
	sourceError error
	dependency  core.Dependency
}

// Report returns the error in string representation
func (e *dependencyValidationError) Error() string {
	return fmt.Sprintf("failed to resolve dependency: %v, source error: %s", e.dependency, e.sourceError.Error())
}

// Requeue determines if the current dogu operation should be requeue when this error was responsible for its failure
func (e *dependencyValidationError) Requeue() bool {
	return true
}

// doguDependencyValidator is responsible to check if all dogu dependencies are valid for a given dogu
type doguDependencyValidator struct {
	fetcher cloudogu.LocalDoguFetcher
}

// NewDoguDependencyValidator creates a new dogu dependencies checker
func NewDoguDependencyValidator(localDoguRegistry dogu.LocalRegistry) *doguDependencyValidator {
	doguDependencyChecker := cesregistry.NewLocalDoguFetcher(localDoguRegistry)

	return &doguDependencyValidator{
		fetcher: doguDependencyChecker,
	}
}

// ValidateAllDependencies validates mandatory and optional dogu dependencies
func (dc *doguDependencyValidator) ValidateAllDependencies(ctx context.Context, dogu *core.Dogu) error {
	var allProblems error

	deps := dogu.GetDependenciesOfType(core.DependencyTypeDogu)
	err := dc.validateDoguDependencies(ctx, deps, false)
	if err != nil {
		allProblems = errors.Join(allProblems, err)
	}

	optionalDeps := dogu.GetOptionalDependenciesOfType(core.DependencyTypeDogu)
	err = dc.validateDoguDependencies(ctx, optionalDeps, true)
	if err != nil {
		allProblems = errors.Join(allProblems, err)
	}

	return allProblems
}

func (dc *doguDependencyValidator) validateDoguDependencies(ctx context.Context, dependencies []core.Dependency, optional bool) error {
	var problems error

	for _, doguDependency := range dependencies {
		err := dc.checkDoguDependency(ctx, doguDependency, optional)
		if err != nil {
			dependencyError := dependencyValidationError{
				sourceError: err,
				dependency:  doguDependency,
			}
			problems = errors.Join(problems, &dependencyError)
		}
	}
	return problems
}

func (dc *doguDependencyValidator) checkDoguDependency(ctx context.Context, doguDependency core.Dependency, optional bool) error {
	log.FromContext(ctx).Info(fmt.Sprintf("checking dogu dependency %s:%s", doguDependency.Name, doguDependency.Version))

	localDependency, err := dc.fetcher.FetchInstalled(ctx, doguDependency.Name)
	if err != nil {
		if optional && apierrors.IsNotFound(err) {
			return nil // not installed => no error as this is ok for optional dependencies
		}
		return fmt.Errorf("failed to resolve dependencies %s: %w", doguDependency.Name, err)
	}

	if localDependency == nil {
		if optional {
			return nil // not installed => no error as this is ok for optional dependencies
		}
		return fmt.Errorf("dependency %s seems not to be installed", doguDependency.Name)
	}

	// it does not count as an error if no version is specified as the field is optional
	if doguDependency.Version == "" {
		return nil
	}

	localDependencyVersion, err := core.ParseVersion(localDependency.Version)
	if err != nil {
		return fmt.Errorf("failed to parse version of dependency %s: %w", localDependency.Name, err)
	}

	comparator, err := core.ParseVersionComparator(doguDependency.Version)
	if err != nil {
		return fmt.Errorf("failed to parse ParseVersionComparator of version %s for doguDependency %s: %w", doguDependency.Version, doguDependency.Name, err)
	}

	allows, err := comparator.Allows(localDependencyVersion)
	if err != nil {
		return fmt.Errorf("an error occurred when comparing the versions: %w", err)
	}
	if !allows {
		return fmt.Errorf("%s parsed Version does not fulfill version requirement of %s dogu %s", localDependency.Version, doguDependency.Version, doguDependency.Name)
	}

	return nil
}
