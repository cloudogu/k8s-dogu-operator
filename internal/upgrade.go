package internal

import (
	"context"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

type UpgradeExecutor interface {
	// Upgrade executes the actual dogu upgrade.
	Upgrade(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu *cesappcore.Dogu, toDogu *cesappcore.Dogu) error
}

type PremisesChecker interface {
	// Check checks if dogu premises are met before a dogu upgrade.
	Check(ctx context.Context, toDoguResource *k8sv1.Dogu, fromDogu *cesappcore.Dogu, toDogu *cesappcore.Dogu) error
}

type DependencyValidator interface {
	// ValidateDependencies is used to check if dogu dependencies are installed.
	ValidateDependencies(ctx context.Context, dogu *cesappcore.Dogu) error
}

type DoguHealthChecker interface {
	// CheckWithResource returns nil if the dogu described by the resource is up and running.
	CheckWithResource(ctx context.Context, doguResource *k8sv1.Dogu) error
}

type DoguRecursiveHealthChecker interface {
	// CheckDependenciesRecursive returns nil if the dogu's mandatory dependencies are up and running.
	CheckDependenciesRecursive(ctx context.Context, fromDogu *cesappcore.Dogu, currentK8sNamespace string) error
}
