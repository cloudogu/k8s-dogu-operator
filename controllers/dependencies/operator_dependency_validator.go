package dependencies

import (
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	"github.com/hashicorp/go-multierror"
)

type operatorDependencyValidator struct {
	Version *core.Version
}

func newOperatorDependencyValidator(version *core.Version) *operatorDependencyValidator {
	return &operatorDependencyValidator{
		Version: version,
	}
}

// CheckAllDependencies looks into all client dependencies (mandatory- and optional ones) and checks weather they're
// all installed an that in the correct version
func (cc *operatorDependencyValidator) CheckAllDependencies(dogu core.Dogu) error {
	var allProblems error

	errMandatoryDependencies := cc.CheckMandatoryDependencies(dogu)
	errOptionalDependencies := cc.CheckOptionalDependencies(dogu)

	if errMandatoryDependencies != nil || errOptionalDependencies != nil {
		allProblems = multierror.Append(errMandatoryDependencies, errOptionalDependencies)
	}
	return allProblems
}

func (cc *operatorDependencyValidator) CheckMandatoryDependencies(dogu core.Dogu) error {
	dependencies := dogu.GetDependenciesOfType(core.DependencyTypeClient)

	for _, dependency := range dependencies {
		if dependency.Name == K8sDoguOperatorClientDependencyName {
			allows, err := cc.checkVersion(dependency)
			if err != nil {
				return fmt.Errorf("failed to check version: %w", err)
			}

			if !allows {
				return fmt.Errorf("%s parsed Version does not fulfill version requirement of %s for %s %s",
					cc.Version.Raw, dependency.Version, dependency.Type, dependency.Name)
			}
		}
	}

	return nil
}

func (cc *operatorDependencyValidator) checkVersion(dependency core.Dependency) (bool, error) {
	comparator, err := core.ParseVersionComparator(dependency.Version)
	if err != nil {
		return false, fmt.Errorf("failed to parse dependency version: %w", err)
	}

	allows, err := comparator.Allows(*cc.Version)
	if err != nil {
		return false, fmt.Errorf("failed to compare dependency version with operator version: %w", err)
	}
	return allows, nil
}

func (cc *operatorDependencyValidator) CheckOptionalDependencies(dogu core.Dogu) error {
	dependencies := dogu.GetOptionalDependenciesOfType(core.DependencyTypeClient)

	for _, dependency := range dependencies {
		if dependency.Name == K8sDoguOperatorClientDependencyName {
			_, err := cc.checkVersion(dependency)
			if err != nil {
				return fmt.Errorf("failed to check version: %w", err)
			}
		}
	}

	return nil
}
