package health

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/internal/cloudogu"

	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
func NewDoguChecker(client client.Client, localFetcher cloudogu.LocalDoguFetcher) *doguChecker {
	return &doguChecker{
		client:            client,
		doguLocalRegistry: localFetcher,
	}
}

type doguChecker struct {
	client            client.Client
	doguLocalRegistry cloudogu.LocalDoguFetcher
}

// CheckWithResource returns nil if the dogu's replica exist and are ready. If the dogu is unhealthy an error of type
// *health.DoguHealthError is returned:
//
//	if e, ok := err.(*health.DoguHealthError); ok { ... }
func (dc *doguChecker) CheckWithResource(ctx context.Context, doguResource *k8sv1.Dogu) error {
	return dc.checkByNameAndK8sObjectKey(ctx, doguResource.Name, doguResource.GetObjectKey())
}

// CheckDependenciesRecursive checks mandatory and optional dogu dependencies for health and returns an error if at
// least one dogu is not healthy.
func (dc *doguChecker) CheckDependenciesRecursive(ctx context.Context, localDoguRoot *core.Dogu, currentK8sNamespace string) error {
	var result error

	err := dc.checkMandatoryRecursive(ctx, localDoguRoot, currentK8sNamespace)
	if err != nil {
		result = errors.Join(result, err)
	}

	err = dc.checkOptionalRecursive(ctx, localDoguRoot, currentK8sNamespace)
	if err != nil {
		result = errors.Join(result, err)
	}

	return result
}

func (dc *doguChecker) checkMandatoryRecursive(ctx context.Context, localDogu *core.Dogu, currentK8sNamespace string) error {
	var result error

	for _, dependency := range localDogu.GetDependenciesOfType(core.DependencyTypeDogu) {
		localDependencyDoguName := dependency.Name
		objectKey := getObjectKeyForDoguAndNamespace(localDependencyDoguName, currentK8sNamespace)

		dependencyDogu, err := dc.doguLocalRegistry.FetchInstalled(localDependencyDoguName)
		if err != nil {
			err2 := fmt.Errorf("error getting registry key for %s: %w", localDependencyDoguName, err)
			result = errors.Join(result, err2)
			// with no dogu information at hand we have no data on dependencies and must continue with the next dogu
			continue
		}

		err = dc.checkByNameAndK8sObjectKey(ctx, localDependencyDoguName, objectKey)
		if err != nil {
			result = errors.Join(result, err)
		}

		err = dc.CheckDependenciesRecursive(ctx, dependencyDogu, currentK8sNamespace)
		if err != nil {
			result = errors.Join(result, err)
		}
	}

	return result
}

func (dc *doguChecker) checkOptionalRecursive(ctx context.Context, localDogu *core.Dogu, currentK8sNamespace string) error {
	const optional = true
	var result error

	for _, dependency := range localDogu.GetOptionalDependenciesOfType(core.DependencyTypeDogu) {
		localDependencyDoguName := dependency.Name
		objectKey := getObjectKeyForDoguAndNamespace(localDependencyDoguName, currentK8sNamespace)

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

		err = dc.checkByNameAndK8sObjectKey(ctx, localDependencyDoguName, objectKey)
		if err != nil {
			result = errors.Join(result, err)
		}

		err = dc.CheckDependenciesRecursive(ctx, dependencyDogu, currentK8sNamespace)
		if err != nil {
			result = errors.Join(result, err)
		}
	}

	return result
}

func (dc *doguChecker) checkByNameAndK8sObjectKey(ctx context.Context, doguName string, objectKey client.ObjectKey) error {
	deployment := &v1.Deployment{}
	err := dc.client.Get(ctx, objectKey, deployment)
	if err != nil {
		return fmt.Errorf("dogu %s health check failed: %w", doguName, err)
	}

	deploymentStatus := deployment.Status
	if deploymentStatus.ReadyReplicas == 0 {
		return NewDoguHealthError(fmt.Errorf("dogu %s appears unhealthy (desired replicas: %d, ready: %d)",
			doguName, deploymentStatus.Replicas, deploymentStatus.ReadyReplicas))
	}

	return nil
}

func getObjectKeyForDoguAndNamespace(localDogu, currentK8sNamespace string) client.ObjectKey {
	return client.ObjectKey{
		Namespace: currentK8sNamespace,
		Name:      localDogu,
	}
}
