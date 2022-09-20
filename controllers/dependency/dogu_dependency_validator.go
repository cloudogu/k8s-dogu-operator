package dependency

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// errorDependencyValidation is returned when a given dependency cloud not be validated.
type errorDependencyValidation struct {
	sourceError error
	dependency  core.Dependency
}

// Report returns the error in string representation
func (e *errorDependencyValidation) Error() string {
	return fmt.Sprintf("failed to resolve dependency: %v, source error: %s", e.dependency, e.sourceError.Error())
}

// Requeue determines if the current dogu operation should be requeue when this error was responsible for its failure
func (e *errorDependencyValidation) Requeue() bool {
	return true
}

type localDoguFetcher interface {
	FetchInstalled(doguName string) (installedDogu *core.Dogu, err error)
}

// doguDependencyValidator is responsible to check if all dogu dependencies are valid for a given dogu
type doguDependencyValidator struct {
	fetcher localDoguFetcher
}

// NewDoguDependencyValidator creates a new dogu dependencies checker
func NewDoguDependencyValidator(localDoguRegistry registry.DoguRegistry) *doguDependencyValidator {
	doguDependencyChecker := cesregistry.NewDoguFetcher(nil, localDoguRegistry, nil)

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
		allProblems = multierror.Append(err)
	}

	optionalDeps := dogu.GetOptionalDependenciesOfType(core.DependencyTypeDogu)
	err = dc.validateDoguDependencies(ctx, optionalDeps, true)
	if err != nil {
		allProblems = multierror.Append(err)
	}

	return allProblems
}

func (dc *doguDependencyValidator) validateDoguDependencies(ctx context.Context, dependencies []core.Dependency, optional bool) error {
	var problems error

	for _, doguDependency := range dependencies {
		err := dc.checkDoguDependency(ctx, doguDependency, optional)
		if err != nil {
			dependencyError := errorDependencyValidation{
				sourceError: err,
				dependency:  doguDependency,
			}
			problems = multierror.Append(problems, &dependencyError)
		}
	}
	return problems
}

func (dc *doguDependencyValidator) checkDoguDependency(ctx context.Context, doguDependency core.Dependency, optional bool) error {
	log.FromContext(ctx).Info("checking dogu dependency %s:%s", doguDependency.Name, doguDependency.Version)

	localDependency, err := dc.fetcher.FetchInstalled(doguDependency.Name)
	if err != nil {
		if optional && registry.IsKeyNotFoundError(err) {
			return nil // not installed => no error as this is ok for optional dependencies
		}
		return errors.Wrapf(err, "failed to resolve dependencies %s", doguDependency.Name)
	}

	if localDependency == nil {
		if optional {
			return nil // not installed => no error as this is ok for optional dependencies
		}
		return errors.Errorf("dependency %s seems not to be installed", doguDependency.Name)
	}

	// it does not count as an error if no version is specified as the field is optional
	if doguDependency.Version != "" {
		localDependencyVersion, err := core.ParseVersion(localDependency.Version)
		if err != nil {
			return errors.Wrapf(err, "failed to parse version of dependency %s", localDependency.Name)
		}

		comparator, err := core.ParseVersionComparator(doguDependency.Version)
		if err != nil {
			return errors.Wrapf(err, "failed to parse ParseVersionComparator of version %s for doguDependency %s", doguDependency.Version, doguDependency.Name)
		}

		allows, err := comparator.Allows(localDependencyVersion)
		if err != nil {
			return errors.Wrapf(err, "An error occurred when comparing the versions")
		}
		if !allows {
			return errors.Errorf("%s parsed Version does not fulfill version requirement of %s dogu %s", localDependency.Version, doguDependency.Version, doguDependency.Name)
		}
	}
	return nil
}
