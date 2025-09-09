package upgrade

import (
	"context"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"k8s.io/apimachinery/pkg/types"
)

// PremisesChecker includes functionality to check if the premises for an upgrade are met.
type PremisesChecker interface {
	// Check checks if dogu premises are met before a dogu upgrade.
	Check(ctx context.Context, toDoguResource *k8sv2.Dogu, fromDogu *cesappcore.Dogu, toDogu *cesappcore.Dogu) error
}

// DependencyValidator checks if all necessary dependencies for an upgrade are installed.
type DependencyValidator interface {
	// ValidateDependencies is used to check if dogu dependencies are installed.
	ValidateDependencies(ctx context.Context, dogu *cesappcore.Dogu) error
}

type securityValidator interface {
	// ValidateSecurity verifies the security fields of dogu descriptor and resource for correctness.
	ValidateSecurity(doguDescriptor *cesappcore.Dogu, doguResource *k8sv2.Dogu) error
}

type doguAdditionalMountsValidator interface {
	ValidateAdditionalMounts(ctx context.Context, doguDescriptor *cesappcore.Dogu, doguResource *k8sv2.Dogu) error
}

// doguHealthChecker includes functionality to check if the dogu described by the resource is up and running.
type doguHealthChecker interface {
	// CheckByName returns nil if the dogu described by the resource is up and running.
	CheckByName(ctx context.Context, doguName types.NamespacedName) error
}

// doguRecursiveHealthChecker includes functionality to check if a dogus dependencies are up and running.
type doguRecursiveHealthChecker interface {
	// CheckDependenciesRecursive returns nil if the dogu's mandatory dependencies are up and running.
	CheckDependenciesRecursive(ctx context.Context, fromDogu *cesappcore.Dogu, namespace string) error
}
