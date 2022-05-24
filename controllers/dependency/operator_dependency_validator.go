package dependency

import (
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const K8sDoguOperatorClientDependencyName = "k8s-dogu-operator"

// operatorDependencyValidator is responsible to validate the `k8s-dogu-operator` client dependency for dogus
type operatorDependencyValidator struct {
	version *core.Version
}

// NewOperatorDependencyValidator creates a new operator dependency validator
func NewOperatorDependencyValidator(version *core.Version) *operatorDependencyValidator {
	return &operatorDependencyValidator{
		version: version,
	}
}

// ValidateAllDependencies looks into all client dependencies (mandatory- and optional ones) and checks weather they're
// all installed an that in the correct version
func (odv *operatorDependencyValidator) ValidateAllDependencies(dogu *core.Dogu) error {
	var allProblems error

	errMandatoryDependencies := odv.validateMandatoryDependencies(dogu)
	errOptionalDependencies := odv.validateOptionalDependencies(dogu)

	if errMandatoryDependencies != nil || errOptionalDependencies != nil {
		allProblems = multierror.Append(errMandatoryDependencies, errOptionalDependencies)
	}
	return allProblems
}

func (odv *operatorDependencyValidator) checkVersion(dependency core.Dependency) (bool, error) {
	comparator, err := core.ParseVersionComparator(dependency.Version)
	if err != nil {
		return false, fmt.Errorf("failed to parse dependency version: %w", err)
	}

	allows, err := comparator.Allows(*odv.version)
	if err != nil {
		return false, fmt.Errorf("failed to compare dependency version with operator version: %w", err)
	}
	return allows, nil
}

func (odv *operatorDependencyValidator) validateMandatoryDependencies(dogu *core.Dogu) error {
	dependencies := dogu.GetDependenciesOfType(core.DependencyTypeClient)

	for _, dependency := range dependencies {
		if dependency.Name == K8sDoguOperatorClientDependencyName {
			allows, err := odv.checkVersion(dependency)
			if err != nil {
				return fmt.Errorf("failed to check version: %w", err)
			}

			if !allows {
				dependencyError := errorDependencyValidation{
					sourceError: errors.Errorf("%s parsed version does not fulfill version requirement of %s dogu %s", dependency.Version, odv.version.Raw, dependency.Name),
					dependency:  dependency,
				}
				return &dependencyError
			}
		}
	}

	return nil
}

func (odv *operatorDependencyValidator) validateOptionalDependencies(dogu *core.Dogu) error {
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
