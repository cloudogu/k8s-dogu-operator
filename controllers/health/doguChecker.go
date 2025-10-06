package health

import (
	"context"
	"errors"
	"fmt"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	regLibErr "github.com/cloudogu/ces-commons-lib/errors"
	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"

	metav1api "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// NewDoguHealthError creates a new dogu health error.
func NewDoguHealthError(err error) *DoguHealthError {
	return &DoguHealthError{err: err}
}

// DoguHealthError is a dogu validation error. Instances can be unwrapped. Instances can be type asserted.
type DoguHealthError struct {
	err error
}

// Unwrap returns the original error.
func (dhe *DoguHealthError) Unwrap() error {
	return dhe.err
}

// Error returns the full error message as string.
func (dhe *DoguHealthError) Error() string {
	return fmt.Errorf("dogu failed a health check: %w", dhe.err).Error()
}

// NewDoguChecker creates a checker for dogu health.
func NewDoguChecker(ecosystemClient doguClient.EcoSystemV2Interface, localFetcher cesregistry.LocalDoguFetcher) DoguHealthChecker {
	return &doguChecker{
		ecosystemClient:   ecosystemClient,
		doguLocalRegistry: localFetcher,
	}
}

type doguChecker struct {
	ecosystemClient   doguClient.EcoSystemV2Interface
	doguLocalRegistry localDoguFetcher
}

// CheckByName returns nil if the dogu resource's health status says it's available.
// If the dogu is unhealthy, an error of type *health.DoguHealthError is returned:
//
//	var doguHealthError *health.DoguHealthError
//	if errors.As(err, &doguHealthError) { ... }
func (dc *doguChecker) CheckByName(ctx context.Context, doguName types.NamespacedName) error {
	doguResource, err := dc.ecosystemClient.
		Dogus(doguName.Namespace).
		Get(ctx, doguName.Name, metav1api.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get dogu resource %q: %w", doguName, err)
	}

	if doguResource.Status.Health != k8sv2.AvailableHealthStatus {
		return NewDoguHealthError(fmt.Errorf("dogu %q appears unhealthy",
			doguResource.Name))
	}

	return nil
}

// CheckDependenciesRecursive checks mandatory and optional dogu dependencies for health and returns an error if at
// least one dogu is not healthy.
func (dc *doguChecker) CheckDependenciesRecursive(ctx context.Context, localDoguRoot *core.Dogu, namespace string) error {
	var errs []error

	err := dc.checkMandatoryRecursive(ctx, localDoguRoot, namespace)
	if err != nil {
		errs = append(errs, err)
	}

	err = dc.checkOptionalRecursive(ctx, localDoguRoot, namespace)
	if err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (dc *doguChecker) checkMandatoryRecursive(ctx context.Context, localDogu *core.Dogu, namespace string) error {
	var errs []error

	for _, dependency := range localDogu.GetDependenciesOfType(core.DependencyTypeDogu) {
		if dependency.Name == "nginx" || dependency.Name == "registrator" {
			continue
		}
		localDependencyDoguName := types.NamespacedName{Name: dependency.Name, Namespace: namespace}

		dependencyDogu, err := dc.doguLocalRegistry.FetchInstalled(ctx, cescommons.SimpleName(dependency.Name))
		if err != nil {
			errs = append(errs, fmt.Errorf("error fetching local dogu descriptor for dependency %q: %w", dependency.Name, err))
			// with no dogu information at hand we have no data on dependencies and must continue with the next dogu
			continue
		}

		err = dc.CheckByName(ctx, localDependencyDoguName)
		if err != nil {
			errs = append(errs, err)
		}

		err = dc.CheckDependenciesRecursive(ctx, dependencyDogu, namespace)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (dc *doguChecker) checkOptionalRecursive(ctx context.Context, localDogu *core.Dogu, namespace string) error {
	const optional = true
	var errs []error

	for _, dependency := range localDogu.GetOptionalDependenciesOfType(core.DependencyTypeDogu) {
		if dependency.Name == "nginx" || dependency.Name == "registrator" {
			continue
		}

		localDependencyDoguName := types.NamespacedName{Name: dependency.Name, Namespace: namespace}

		dependencyDogu, err := dc.doguLocalRegistry.FetchInstalled(ctx, cescommons.SimpleName(dependency.Name))
		if err != nil {
			if optional && regLibErr.IsNotFoundError(err) {
				// optional dogus may not be installed, so continue and do nothing
			} else {
				// with no dogu information at hand we have no data on dependencies and must continue with the next dogu
				errs = append(errs, fmt.Errorf("error fetching local dogu descriptor for dependency %q: %w", dependency.Name, err))
			}
			continue
		}

		err = dc.CheckByName(ctx, localDependencyDoguName)
		if err != nil {
			errs = append(errs, err)
		}

		err = dc.CheckDependenciesRecursive(ctx, dependencyDogu, namespace)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
