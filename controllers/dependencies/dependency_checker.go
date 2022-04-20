package dependencies

import (
	"github.com/cloudogu/cesapp/v4/core"
	"github.com/cloudogu/cesapp/v4/dependencies"
	"github.com/cloudogu/cesapp/v4/registry"
	"github.com/hashicorp/go-multierror"
)

const K8sDoguOperatorClientDependencyName = "k8s-dogu-operator"

type DependencyChecker struct {
	DoguDependencyValidator     *dependencies.DoguDependencyChecker `json:"dogu_dependency_validator"`
	OperatorDependencyValidator *operatorDependencyValidator        `json:"operator_dependency_validator"`
}

func NewDependencyChecker(version *core.Version, doguRegistry registry.DoguRegistry) *DependencyChecker {
	operatorDependencyValidator := newOperatorDependencyValidator(version)
	doguDependencyValidator := dependencies.NewDoguDependencyChecker(doguRegistry)

	return &DependencyChecker{
		DoguDependencyValidator:     doguDependencyValidator,
		OperatorDependencyValidator: operatorDependencyValidator,
	}
}

func (dc *DependencyChecker) ValidateDependencies(dogu *core.Dogu) error {
	var result error
	err := dc.DoguDependencyValidator.CheckAllDependencies(*dogu)
	if err != nil {
		result = multierror.Append(result, err)
	}

	err = dc.OperatorDependencyValidator.CheckAllDependencies(*dogu)
	if err != nil {
		result = multierror.Append(result, err)
	}

	return result // contains all errors that occurred when checking the dependencies.
}
