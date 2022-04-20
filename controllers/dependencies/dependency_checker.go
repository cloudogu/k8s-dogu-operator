package dependencies

import (
	"github.com/cloudogu/cesapp/v4/core"
	"github.com/cloudogu/cesapp/v4/dependencies"
	"github.com/cloudogu/cesapp/v4/registry"
	"github.com/hashicorp/go-multierror"
)

const K8sDoguOperatorClientDependencyName = "k8s-dogu-operator"

// DependencyValidator is a composite validator responsible to validate the dogu and client dependencies of dogus.
type DependencyValidator struct {
	DoguRegistry                registry.DoguRegistry        `json:"dogu_registry"`
	OperatorDependencyValidator *operatorDependencyValidator `json:"operator_dependency_validator"`
}

// NewDependencyValidator create a new composite validator checking the dogu and client dependencies
func NewDependencyValidator(version *core.Version, doguRegistry registry.DoguRegistry) *DependencyValidator {
	operatorDependencyValidator := newOperatorDependencyValidator(version)

	return &DependencyValidator{
		DoguRegistry:                doguRegistry,
		OperatorDependencyValidator: operatorDependencyValidator,
	}
}

// ValidateDependencies validates all kinds of dependencies for dogus. An error is returned when any invalid
// dependencies were detected.
func (dc *DependencyValidator) ValidateDependencies(dogu *core.Dogu) error {
	var result error

	doguChecker := dependencies.NewDoguDependencyChecker(dc.DoguRegistry)
	err := doguChecker.CheckAllDependencies(*dogu)
	if err != nil {
		result = multierror.Append(result, err)
	}

	err = dc.OperatorDependencyValidator.ValidateAllDependencies(*dogu)
	if err != nil {
		result = multierror.Append(result, err)
	}

	return result // contains all errors that occurred when checking the dependencies.
}
