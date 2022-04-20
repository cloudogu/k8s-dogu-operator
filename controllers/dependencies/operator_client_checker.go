package dependencies

import (
	"github.com/cloudogu/cesapp/v4/core"
	"github.com/coreos/go-semver/semver"
)

const K8sDoguOperatorClientDependencyName = "k8s-dogu-operator"

type ClientDependencyChecker struct {
	Version *semver.Version
}

func NewClientDependencyChecker(version string) *ClientDependencyChecker {
	return &ClientDependencyChecker{
		Version: semver.New(version),
	}
}

// CheckAllDependencies looks into all client dependencies (mandatory- and optional ones) and checks weather they're
// all installed an that in the correct version
func (cc *ClientDependencyChecker) CheckAllDependencies(dogu core.Dogu) error {
	var allProblems error
	//
	//errMandatoryDependencies := cc.CheckMandatoryDependencies(dogu)
	//errOptionalDependencies := cc.CheckOptionalDependencies(dogu)
	//
	//if errMandatoryDependencies != nil || errOptionalDependencies != nil {
	//	allProblems = multierror.Append(errMandatoryDependencies, errOptionalDependencies)
	//}
	return allProblems
}

func (cc *ClientDependencyChecker) CheckMandatoryDependencies(dogu core.Dogu) error {
	//dependencies := dogu.GetDependenciesOfType(core.DependencyTypeClient)
	//
	//for _, dependency := range dependencies {
	//	if dependency.Name == K8sDoguOperatorClientDependencyName {
	//		return checkDependencies(core.DependencyTypeClient, false, cc.packageManager, []core.Dependency{dependency})
	//	}
	//}

	return nil
}

func (cc *ClientDependencyChecker) CheckOptionalDependencies(dogu core.Dogu) error {
	//dependencies := dogu.GetOptionalDependenciesOfType(core.DependencyTypeClient)
	//
	//for _, dependency := range dependencies {
	//	if dependency.Name == K8sDoguOperatorClientDependencyName {
	//		return checkDependencies(core.DependencyTypeClient, true, cc.packageManager, []core.Dependency{dependency})
	//	}
	//}

	return nil
}
