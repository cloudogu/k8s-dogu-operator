package dependency

import (
	"context"
	"errors"
	"fmt"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	regLibErr "github.com/cloudogu/ces-commons-lib/errors"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
)

// dogus with special validation treatment
const (
	// dogus that are no longer supported in k8s CES LOP
	LegacyDoguNginx       = "nginx"
	LegacyDoguRegistrator = "registrator"
	// dogus that migrated from dogu to components
	ComponentDoguCas     = "cas"
	ComponentDoguPostfix = "postfix"
)

// dependencyValidationError is returned when a given dependency cloud not be validated.
type dependencyValidationError struct {
	sourceError error
	dependency  core.Dependency
}

// Report returns the error in string representation
func (e *dependencyValidationError) Error() string {
	return fmt.Sprintf("failed to resolve dependency: %v, source error: %s", e.dependency, e.sourceError.Error())
}

// Requeue determines if the current dogu operation should be requeue when this error was responsible for its failure
func (e *dependencyValidationError) Requeue() bool {
	return true
}

// doguDependencyValidator is responsible to check if all dogu dependencies are valid for a given dogu
type doguDependencyValidator struct {
	fetcher                       localDoguFetcher
	authRegistrationEnabled       bool
	disablePostfixDependencyCheck bool
}

// newDoguDependencyValidator creates a new dogu dependencies checker
func newDoguDependencyValidator(doguFetcher localDoguFetcher, config *config.OperatorConfig) *doguDependencyValidator {
	return &doguDependencyValidator{
		fetcher:                       doguFetcher,
		authRegistrationEnabled:       config.AuthRegistrationEnabled,
		disablePostfixDependencyCheck: config.DisablePostfixDependencyCheck,
	}
}

// ValidateAllDependencies validates mandatory and optional dogu dependencies
func (dc *doguDependencyValidator) ValidateAllDependencies(ctx context.Context, dogu *core.Dogu) error {
	var allProblems error

	deps := dogu.GetDependenciesOfType(core.DependencyTypeDogu)
	err := dc.validateDoguDependencies(ctx, deps, false)
	if err != nil {
		allProblems = errors.Join(allProblems, err)
	}

	optionalDeps := dogu.GetOptionalDependenciesOfType(core.DependencyTypeDogu)
	err = dc.validateDoguDependencies(ctx, optionalDeps, true)
	if err != nil {
		allProblems = errors.Join(allProblems, err)
	}

	return allProblems
}

func (dc *doguDependencyValidator) validateDoguDependencies(ctx context.Context, dependencies []core.Dependency, optional bool) error {
	var problems error

	for _, doguDependency := range dependencies {
		err := dc.checkDoguDependency(ctx, doguDependency, optional)
		if err != nil {
			dependencyError := dependencyValidationError{
				sourceError: err,
				dependency:  doguDependency,
			}
			problems = errors.Join(problems, &dependencyError)
		}
	}
	return problems
}

func (dc *doguDependencyValidator) checkDoguDependency(ctx context.Context, doguDependency core.Dependency, optional bool) error {
	logger := log.FromContext(ctx)
	if doguDependency.Name == LegacyDoguNginx || doguDependency.Name == LegacyDoguRegistrator {
		logger.Info(fmt.Sprintf("skipping legacy dogu dependency: %s", doguDependency.Name))
		return nil
	}

	if dc.authRegistrationEnabled && doguDependency.Name == ComponentDoguCas {
		logger.Info("skipping legacy dogu dependency for 'cas' because auth registration is enabled")
		return nil
	}

	if dc.disablePostfixDependencyCheck && doguDependency.Name == ComponentDoguPostfix {
		logger.Info("skipping legacy dogu dependency for 'postfix' because postfix is assumed to be installed as a component")
		return nil
	}

	logger.Info(fmt.Sprintf("checking dogu dependency %s:%s", doguDependency.Name, doguDependency.Version))

	localDependency, err := dc.fetcher.FetchInstalled(ctx, cescommons.SimpleName(doguDependency.Name))
	if err != nil {
		if optional && regLibErr.IsNotFoundError(err) {
			return nil // not installed => no error as this is ok for optional dependencies
		}
		return fmt.Errorf("failed to resolve dependency %q: %w", doguDependency.Name, err)
	}

	if localDependency == nil {
		if optional {
			return nil // not installed => no error as this is ok for optional dependencies
		}
		return fmt.Errorf("dependency %q seems not to be installed", doguDependency.Name)
	}

	// it does not count as an error if no version is specified as the field is optional
	if doguDependency.Version == "" {
		return nil
	}

	localDependencyVersion, err := core.ParseVersion(localDependency.Version)
	if err != nil {
		return fmt.Errorf("failed to parse version of dependency %q: %w", localDependency.Name, err)
	}

	comparator, err := core.ParseVersionComparator(doguDependency.Version)
	if err != nil {
		return fmt.Errorf("failed to parse ParseVersionComparator of version %q for doguDependency %q: %w", doguDependency.Version, doguDependency.Name, err)
	}

	allows, err := comparator.Allows(localDependencyVersion)
	if err != nil {
		return fmt.Errorf("an error occurred when comparing the versions: %w", err)
	}
	if !allows {
		return fmt.Errorf("%q parsed Version does not fulfill version requirement of %q dogu %q", localDependency.Version, doguDependency.Version, doguDependency.Name)
	}

	return nil
}
