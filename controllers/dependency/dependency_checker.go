package dependency

import (
	"context"
	"errors"
	"github.com/cloudogu/k8s-dogu-operator/controllers/localregistry"

	"github.com/cloudogu/cesapp-lib/core"
)

// DependencyValidator is responsible to validate the dependencies of a dogu
type DependencyValidator interface {
	ValidateAllDependencies(ctx context.Context, dogu *core.Dogu) error
}

// CompositeDependencyValidator is a composite validator responsible to validate the dogu and client dependencies of dogus.
type CompositeDependencyValidator struct {
	Validators []DependencyValidator `json:"validators"`
}

// NewCompositeDependencyValidator create a new composite validator checking the dogu and client dependencies
func NewCompositeDependencyValidator(version *core.Version, doguRegistry localregistry.LocalDoguRegistry) *CompositeDependencyValidator {
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
func (dv *CompositeDependencyValidator) ValidateDependencies(ctx context.Context, dogu *core.Dogu) error {
	var result error

	for _, validator := range dv.Validators {
		err := validator.ValidateAllDependencies(ctx, dogu)
		if err != nil {
			result = errors.Join(result, err)
		}
	}

	return result // contains all errors that occurred when checking the dependencies.
}
