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

// Report returns the error in string representation
func (e *ErrorDependencyValidation) Error() string {
	return fmt.Sprintf("failed to resolve to depdencies: %v, source error: %s", e.Dependency, e.SourceError.Error())
}

// Report constructs a simple human readable message
func (e *ErrorDependencyValidation) Report() string {
	return fmt.Sprintf("failed to resolve to depdencies: %v", e.Dependency)
}

// DoguDependencyValidator is responsible to check if all dogu dependencies are valid for a given dogu
type DoguDependencyValidator struct {
	DoguDependencyChecker *dependencies.DoguDependencyChecker
}

// NewDoguDependencyValidator creates a new dogu dependencies checker
func NewDoguDependencyValidator(doguRegistry registry.DoguRegistry) *DoguDependencyValidator {
	doguDependencyChecker := dependencies.NewDoguDependencyChecker(doguRegistry)

	return &DoguDependencyValidator{
		DoguDependencyChecker: doguDependencyChecker,
	}
}

func (dc *DoguDependencyValidator) ValidateAllDependencies(dogu *core.Dogu) error {
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

func (dc *DoguDependencyValidator) validateDoguDependencies(dependencies []core.Dependency, optional bool) error {
	var problems error

	for _, doguDependency := range dependencies {
		err := dc.DoguDependencyChecker.CheckDoguDependency(doguDependency, optional)
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
