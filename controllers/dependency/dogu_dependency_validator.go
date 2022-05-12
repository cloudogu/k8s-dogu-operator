package dependency

import (
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	"github.com/cloudogu/cesapp/v4/dependencies"
	"github.com/cloudogu/cesapp/v4/registry"
	"github.com/hashicorp/go-multierror"
)

// ErrorDependencyValidation is returned when a given dependency cloud not be validated.
type ErrorDependencyValidation struct {
	SourceError error
	Dependency  core.Dependency
}

// doguDependencyChecker is used to  check a single dependency of a dogu
type doguDependencyChecker interface {
	CheckDoguDependency(dependency core.Dependency, optional bool) error
}

// Report returns the error in string representation
func (e *ErrorDependencyValidation) Error() string {
	return fmt.Sprintf("failed to resolve dependency: %v, source error: %s", e.Dependency, e.SourceError.Error())
}

// Report constructs a simple human readable message
func (e *ErrorDependencyValidation) Report() string {
	return fmt.Sprintf("failed to resolve dependency: %v", e.Dependency)
}

// Requeue determines if the current dogu operation should be requeue when this error was responsible for its failure
func (e *ErrorDependencyValidation) Requeue() bool {
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
			dependencyError := ErrorDependencyValidation{
				SourceError: err,
				Dependency:  doguDependency,
			}
			problems = multierror.Append(problems, &dependencyError)
		}
	}
	return problems
}
