package cloudogu

import (
	"context"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
)

// UpgradeExecutor applies upgrades the upgrade from an earlier dogu version to a newer version.
type UpgradeExecutor interface {
	// Upgrade executes the actual dogu upgrade.
	Upgrade(ctx context.Context, toDoguResource *k8sv2.Dogu, fromDogu *cesappcore.Dogu, toDogu *cesappcore.Dogu) error
}

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
