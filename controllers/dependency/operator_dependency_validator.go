package dependency

import (
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const K8sDoguOperatorClientDependencyName = "k8s-dogu-operator"

// OperatorDependencyValidator is responsible to validate the `k8s-dogu-operator` client dependency for dogus
type OperatorDependencyValidator struct {
	Version *core.Version
}

// NewOperatorDependencyValidator creates a new operator dependency validator
func NewOperatorDependencyValidator(version *core.Version) *OperatorDependencyValidator {
	return &OperatorDependencyValidator{
		Version: version,
	}
}

// ValidateAllDependencies looks into all client dependencies (mandatory- and optional ones) and checks weather they're
// all installed an that in the correct version
func (odv *OperatorDependencyValidator) ValidateAllDependencies(dogu *core.Dogu) error {
	var allProblems error

	errMandatoryDependencies := odv.validateMandatoryDependencies(dogu)
	errOptionalDependencies := odv.validateOptionalDependencies(dogu)

	if errMandatoryDependencies != nil || errOptionalDependencies != nil {
		allProblems = multierror.Append(errMandatoryDependencies, errOptionalDependencies)
	}
	return allProblems
}

func (odv *OperatorDependencyValidator) checkVersion(dependency core.Dependency) (bool, error) {
	comparator, err := core.ParseVersionComparator(dependency.Version)
	if err != nil {
		return false, fmt.Errorf("failed to parse dependency version: %w", err)
	}

	allows, err := comparator.Allows(*odv.Version)
	if err != nil {
		return false, fmt.Errorf("failed to compare dependency version with operator version: %w", err)
	}
	return allows, nil
}

func (odv *OperatorDependencyValidator) validateMandatoryDependencies(dogu *core.Dogu) error {
	dependencies := dogu.GetDependenciesOfType(core.DependencyTypeClient)

	for _, dependency := range dependencies {
		if dependency.Name == K8sDoguOperatorClientDependencyName {
			allows, err := odv.checkVersion(dependency)
			if err != nil {
				return fmt.Errorf("failed to check version: %w", err)
			}

			if !allows {
				dependencyError := ErrorDependencyValidation{
					SourceError: errors.Errorf("%s parsed Version does not fulfill version requirement of %s dogu %s", dependency.Version, odv.Version.Raw, dependency.Name),
					Dependency:  dependency,
				}
				return &dependencyError
			}
		}
	}

	return nil
}

func (odv *OperatorDependencyValidator) validateOptionalDependencies(dogu *core.Dogu) error {
	dependencies := dogu.GetOptionalDependenciesOfType(core.DependencyTypeClient)

	for _, dependency := range dependencies {
		if dependency.Name == K8sDoguOperatorClientDependencyName {
			_, err := odv.checkVersion(dependency)
			if err != nil {
				return fmt.Errorf("failed to check version: %w", err)
			}
		}
	}

	return nil
}
