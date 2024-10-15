package health

import (
	"context"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/v2/api/ecoSystem"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
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

// LocalDoguFetcher includes functionality to search the local dogu registry for a dogu.
type LocalDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(ctx context.Context, doguName string) (installedDogu *cesappcore.Dogu, err error)
}

type EcosystemInterface interface {
	ecoSystem.EcoSystemV2Interface
}

type DoguInterface interface {
	ecoSystem.DoguInterface
}

type DoguRestartInterface interface {
	ecoSystem.DoguRestartInterface
}

type EventRecorder interface {
	record.EventRecorder
}

type ClientSet interface {
	kubernetes.Interface
}

type ConfigMapInterface interface {
	v1.ConfigMapInterface
}

type PodInterface interface {
	v1.PodInterface
}

type CoreV1Interface interface {
	v1.CoreV1Interface
}

type DeploymentInterface interface {
	appsv1client.DeploymentInterface
}
