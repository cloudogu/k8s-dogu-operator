package health

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/hashicorp/go-multierror"
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
func NewDoguChecker(client client.Client, doguLocalRegistry registry.DoguRegistry) *doguChecker {
	return &doguChecker{
		client:            client,
		doguLocalRegistry: doguLocalRegistry,
	}
}

type doguChecker struct {
	client            client.Client
	doguLocalRegistry registry.DoguRegistry
}

// CheckWithResource returns nil if the dogu's replica exist and are ready. If the dogu is unhealthy an error of type
// *health.DoguHealthError is returned:
//
//  if e, ok := err.(*health.DoguHealthError); ok { ... }
func (dc *doguChecker) CheckWithResource(ctx context.Context, doguResource *k8sv1.Dogu) error {
	return dc.checkByNameAndK8sObjectKey(ctx, doguResource.Name, doguResource.GetObjectKey())
}

// CheckDependenciesRecursive
func (dc *doguChecker) CheckDependenciesRecursive(ctx context.Context, localDoguRoot *core.Dogu, currentK8sNamespace string) error {
	var problems error

	err := dc.checkMandatoryRecursive(ctx, localDoguRoot, currentK8sNamespace)
	problems = multierror.Append(problems, err)

	err = dc.checkOptionalRecursive(ctx, localDoguRoot, currentK8sNamespace)
	problems = multierror.Append(problems, err)

	return problems
}

func (dc *doguChecker) checkMandatoryRecursive(ctx context.Context, localDogu *core.Dogu, currentK8sNamespace string) error {
	var problems error

	for _, dependency := range localDogu.Dependencies {
		localDependencyDoguName := dependency.Name
		objectKey := getObjectKeyForDoguAndNamespace(localDependencyDoguName, currentK8sNamespace)

		dependencyDogu, err := dc.doguLocalRegistry.Get(localDependencyDoguName)
		// "key not found" error should usually not occur here if installation was checked before
		if err != nil {
			problems = multierror.Append(problems, err)
			continue
		}

		err = dc.checkByNameAndK8sObjectKey(ctx, localDependencyDoguName, objectKey)
		if err != nil {
			problems = multierror.Append(problems, err)
			continue
		}

		err = dc.checkMandatoryRecursive(ctx, dependencyDogu, currentK8sNamespace)
		problems = multierror.Append(problems, err)
	}

	return problems
}

func (dc *doguChecker) checkOptionalRecursive(ctx context.Context, localDogu *core.Dogu, currentK8sNamespace string) error {
	const optional = true
	var problems error

	for _, dependency := range localDogu.Dependencies {
		localDependencyDoguName := dependency.Name
		objectKey := getObjectKeyForDoguAndNamespace(localDependencyDoguName, currentK8sNamespace)

		dependencyDogu, err := dc.doguLocalRegistry.Get(localDependencyDoguName)
		if err != nil {
			if optional && strings.Contains(err.Error(), "Key not found") { // if a dogu is not found an error is returned
				continue
			}
			problems = multierror.Append(problems, err)
			continue
		}

		err = dc.checkByNameAndK8sObjectKey(ctx, localDependencyDoguName, objectKey)
		if err != nil {
			problems = multierror.Append(problems, err)
			continue
		}

		err = dc.checkOptionalRecursive(ctx, dependencyDogu, currentK8sNamespace)
		problems = multierror.Append(problems, err)
	}

	return problems
}

func (dc *doguChecker) checkByNameAndK8sObjectKey(ctx context.Context, doguName string, objectKey *client.ObjectKey) error {
	println("looking at", doguName)
	deployment := &v1.Deployment{}
	err := dc.client.Get(ctx, *objectKey, deployment)
	if err != nil {
		return fmt.Errorf("failed to check if dogu %s is running: %w", doguName, err)
	}

	deploymentStatus := deployment.Status
	if deploymentStatus.ReadyReplicas == 0 {
		return NewDoguHealthError(fmt.Errorf("dogu %s appears unhealthy (desired replicas: %d, ready: %d)",
			doguName, deploymentStatus.Replicas, deploymentStatus.ReadyReplicas))
	}

	return nil
}

func getObjectKeyForDoguAndNamespace(localDogu, currentK8sNamespace string) *client.ObjectKey {
	return &client.ObjectKey{
		Namespace: currentK8sNamespace,
		Name:      localDogu,
	}
}
