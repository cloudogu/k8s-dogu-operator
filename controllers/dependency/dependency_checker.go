package dependency

import (
	"context"
	"errors"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
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
func NewCompositeDependencyValidator(operatorConfig *config.OperatorConfig, doguFetcher cesregistry.LocalDoguFetcher) Validator {
	var validators []DependencyValidator

	operatorDependencyValidator := newOperatorDependencyValidator(operatorConfig.Version)
	validators = append(validators, operatorDependencyValidator)

	doguDependencyValidator := newDoguDependencyValidator(doguFetcher)
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
