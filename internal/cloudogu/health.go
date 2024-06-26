package cloudogu

import (
	"context"
	"k8s.io/apimachinery/pkg/types"

	cesappcore "github.com/cloudogu/cesapp-lib/core"

	appsv1 "k8s.io/api/apps/v1"
)

type DeploymentAvailabilityChecker interface {
	// IsAvailable checks whether the deployment has reached its desired state and is available.
	IsAvailable(deployment *appsv1.Deployment) bool
}

type DoguHealthStatusUpdater interface {
	// UpdateStatus sets the health status of the dogu according to whether if it's available or not.
	UpdateStatus(ctx context.Context, doguName types.NamespacedName, available bool) error
	UpdateHealthConfigMap(ctx context.Context, deployment *appsv1.Deployment, doguJson *cesappcore.Dogu) error
}

// DoguHealthChecker includes functionality to check if the dogu described by the resource is up and running.
type DoguHealthChecker interface {
	// CheckByName returns nil if the dogu described by the resource is up and running.
	CheckByName(ctx context.Context, doguName types.NamespacedName) error
}

// DoguRecursiveHealthChecker includes functionality to check if a dogus dependencies are up and running.
type DoguRecursiveHealthChecker interface {
	// CheckDependenciesRecursive returns nil if the dogu's mandatory dependencies are up and running.
	CheckDependenciesRecursive(ctx context.Context, fromDogu *cesappcore.Dogu, namespace string) error
}
