package dependency

import (
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	"github.com/hashicorp/go-multierror"
)

// DependencyValidator is responsible to validate the dependencies of a dogu
type DependencyValidator interface {
	ValidateAllDependencies(dogu *core.Dogu) error
}

// CompositeDependencyValidator is a composite validator responsible to validate the dogu and client dependencies of dogus.
type CompositeDependencyValidator struct {
	Validators []DependencyValidator `json:"validators"`
}

// NewCompositeDependencyValidator create a new composite validator checking the dogu and client dependencies
func NewCompositeDependencyValidator(version *core.Version, doguRegistry registry.DoguRegistry) *CompositeDependencyValidator {
	validators := []DependencyValidator{}

	operatorDependencyValidator := NewOperatorDependencyValidator(version)
	validators = append(validators, operatorDependencyValidator)

	doguDependencyValidator := NewDoguDependencyValidator(doguRegistry)
	validators = append(validators, doguDependencyValidator)

	return &CompositeDependencyValidator{
		Validators: validators,
	}
}

// ValidateDependencies validates all kinds of dependencies for dogus. An error is returned when any invalid
// dependencies were detected.
func (dv *CompositeDependencyValidator) ValidateDependencies(dogu *core.Dogu) error {
	var result error

	for _, validator := range dv.Validators {
		err := validator.ValidateAllDependencies(dogu)
		if err != nil {
			result = multierror.Append(result, err)
		}
	}

	return result // contains all errors that occurred when checking the dependencies.
}
