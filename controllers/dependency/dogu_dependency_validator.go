package dependency

import (
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/cesapp/v5/dependencies"
	"github.com/hashicorp/go-multierror"
)

// dependencyValidationError is returned when a given dependency cloud not be validated.
type dependencyValidationError struct {
	sourceError error
	dependency  core.Dependency
}

// doguDependencyChecker is used to  check a single dependency of a dogu
type doguDependencyChecker interface {
	CheckDoguDependency(dependency core.Dependency, optional bool) error
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
	doguDependencyChecker doguDependencyChecker
}

// NewDoguDependencyValidator creates a new dogu dependencies checker
func NewDoguDependencyValidator(doguRegistry registry.DoguRegistry) *doguDependencyValidator {
	doguDependencyChecker := dependencies.NewDoguDependencyChecker(doguRegistry)

	return &doguDependencyValidator{
		doguDependencyChecker: doguDependencyChecker,
	}
}

// ValidateAllDependencies validates mandatory and optional dogu dependencies
func (dc *doguDependencyValidator) ValidateAllDependencies(dogu *core.Dogu) error {
	var allProblems error

	deps := dogu.GetDependenciesOfType(core.DependencyTypeDogu)
	err := dc.validateDoguDependencies(deps, false)
	if err != nil {
		allProblems = multierror.Append(err)
	}

	optionalDeps := dogu.GetOptionalDependenciesOfType(core.DependencyTypeDogu)
	err = dc.validateDoguDependencies(optionalDeps, true)
	if err != nil {
		allProblems = multierror.Append(err)
	}

	return allProblems
}

func (dc *doguDependencyValidator) validateDoguDependencies(dependencies []core.Dependency, optional bool) error {
	var problems error

	for _, doguDependency := range dependencies {
		name := doguDependency.Name
		if name == "nginx" || name == "registrator" {
			continue
		}
		err := dc.doguDependencyChecker.CheckDoguDependency(doguDependency, optional)
		if err != nil {
			dependencyError := dependencyValidationError{
				sourceError: err,
				dependency:  doguDependency,
			}
			problems = multierror.Append(problems, &dependencyError)
		}
	}
	return problems
}
