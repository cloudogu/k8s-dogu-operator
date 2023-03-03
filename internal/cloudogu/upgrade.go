package cloudogu

import (
	"context"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// UpgradeExecutor applies upgrades the upgrade from an earlier dogu version to a newer version.
type UpgradeExecutor interface {
	// Upgrade executes the actual dogu upgrade.
	Upgrade(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu *cesappcore.Dogu, toDogu *cesappcore.Dogu) error
}

// PremisesChecker includes functionality to check if the premises for an upgrade are met.
type PremisesChecker interface {
	// Check checks if dogu premises are met before a dogu upgrade.
	Check(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu *cesappcore.Dogu, toDogu *cesappcore.Dogu) error
}

// DependencyValidator checks if all necessary dependencies for an upgrade are installed.
type DependencyValidator interface {
	// ValidateDependencies is used to check if dogu dependencies are installed.
	ValidateDependencies(ctx context.Context, dogu *cesappcore.Dogu) error
}

// DoguHealthChecker includes functionality to check if the dogu described by the resource is up and running.
type DoguHealthChecker interface {
	// CheckWithResource returns nil if the dogu described by the resource is up and running.
	CheckWithResource(ctx context.Context, doguResource *k8sv1.Dogu) error
}

// DoguRecursiveHealthChecker includes functionality to check if a dogus dependencies are up and running.
type DoguRecursiveHealthChecker interface {
	// CheckDependenciesRecursive returns nil if the dogu's mandatory dependencies are up and running.
	CheckDependenciesRecursive(ctx context.Context, fromDogu *cesappcore.Dogu, currentK8sNamespace string) error
}
