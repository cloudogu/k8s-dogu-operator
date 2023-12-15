package health

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/api/ecoSystem"
	metav1api "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"
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
func NewDoguChecker(doguClient ecoSystem.DoguInterface, localFetcher cloudogu.LocalDoguFetcher) *doguChecker {
	return &doguChecker{
		doguClient:        doguClient,
		doguLocalRegistry: localFetcher,
	}
}

type doguChecker struct {
	doguClient        ecoSystem.DoguInterface
	doguLocalRegistry cloudogu.LocalDoguFetcher
}

// CheckByName returns nil if the dogu resource's health status says it's available.
// If the dogu is unhealthy, an error of type *health.DoguHealthError is returned:
//
//	var doguHealthError *health.DoguHealthError
//	if errors.As(err, doguHealthError) { ... }
func (dc *doguChecker) CheckByName(ctx context.Context, doguName string) error {
	doguResource, err := dc.doguClient.Get(ctx, doguName, metav1api.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get dogu resource %q: %w", doguName, err)
	}

	if doguResource.Status.Health != k8sv1.AvailableHealthStatus {
		return NewDoguHealthError(fmt.Errorf("dogu %q appears unhealthy",
			doguResource.Name))
	}

	return nil
}

// CheckDependenciesRecursive checks mandatory and optional dogu dependencies for health and returns an error if at
// least one dogu is not healthy.
func (dc *doguChecker) CheckDependenciesRecursive(ctx context.Context, localDoguRoot *core.Dogu) error {
	var result error

	err := dc.checkMandatoryRecursive(ctx, localDoguRoot)
	if err != nil {
		result = errors.Join(result, err)
	}

	err = dc.checkOptionalRecursive(ctx, localDoguRoot)
	if err != nil {
		result = errors.Join(result, err)
	}

	return result
}

func (dc *doguChecker) checkMandatoryRecursive(ctx context.Context, localDogu *core.Dogu) error {
	var result error

	for _, dependency := range localDogu.GetDependenciesOfType(core.DependencyTypeDogu) {
		localDependencyDoguName := dependency.Name

		dependencyDogu, err := dc.doguLocalRegistry.FetchInstalled(localDependencyDoguName)
		if err != nil {
			err2 := fmt.Errorf("error getting registry key for %s: %w", localDependencyDoguName, err)
			result = errors.Join(result, err2)
			// with no dogu information at hand we have no data on dependencies and must continue with the next dogu
			continue
		}

		err = dc.CheckByName(ctx, localDependencyDoguName)
		if err != nil {
			result = errors.Join(result, err)
		}

		err = dc.CheckDependenciesRecursive(ctx, dependencyDogu)
		if err != nil {
			result = errors.Join(result, err)
		}
	}

	return result
}

func (dc *doguChecker) checkOptionalRecursive(ctx context.Context, localDogu *core.Dogu) error {
	const optional = true
	var result error

	for _, dependency := range localDogu.GetOptionalDependenciesOfType(core.DependencyTypeDogu) {
		localDependencyDoguName := dependency.Name

		dependencyDogu, err := dc.doguLocalRegistry.FetchInstalled(localDependencyDoguName)
		if err != nil {
			if optional && registry.IsKeyNotFoundError(err) {
				// optional dogus may not be installed, so continue and do nothing
			} else {
				// with no dogu information at hand we have no data on dependencies and must continue with the next dogu
				result = errors.Join(result, err)
			}
			continue
		}

		err = dc.CheckByName(ctx, localDependencyDoguName)
		if err != nil {
			result = errors.Join(result, err)
		}

		err = dc.CheckDependenciesRecursive(ctx, dependencyDogu)
		if err != nil {
			result = errors.Join(result, err)
		}
	}

	return result
}
