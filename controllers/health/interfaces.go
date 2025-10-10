package health

import (
	"context"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

type DeploymentAvailabilityChecker interface {
	// IsAvailable checks whether the deployment has reached its desired state and is available.
	IsAvailable(deployment *appsv1.Deployment) bool
}

type DoguHealthStatusUpdater interface {
	UpdateHealthConfigMap(ctx context.Context, deployment *appsv1.Deployment, doguJson *cesappcore.Dogu) error
	DeleteDoguOutOfHealthConfigMap(ctx context.Context, dogu *v2.Dogu) error
}

type DoguHealthChecker interface {
	// CheckByName returns nil if the dogu described by the resource is up and running.
	CheckByName(ctx context.Context, doguName types.NamespacedName) error
	// CheckDependenciesRecursive returns nil if the dogu's mandatory dependencies are up and running.
	CheckDependenciesRecursive(ctx context.Context, fromDogu *cesappcore.Dogu, namespace string) error
}

// localDoguFetcher includes functionality to search the local dogu registry for a dogu.
type localDoguFetcher interface {
	cesregistry.LocalDoguFetcher
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemInterface interface {
	doguClient.EcoSystemV2Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type ctrlManager interface {
	manager.Manager
}

//nolint:unused
//goland:noinspection GoUnusedType
type doguInterface interface {
	doguClient.DoguInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type doguRestartInterface interface {
	doguClient.DoguRestartInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type eventRecorder interface {
	record.EventRecorder
}

//nolint:unused
//goland:noinspection GoUnusedType
type configMapInterface interface {
	v1.ConfigMapInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type podInterface interface {
	v1.PodInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type coreV1Interface interface {
	v1.CoreV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type deploymentInterface interface {
	appsv1client.DeploymentInterface
}

// HealthShutdownHandler is responsible for setting health states to unknown on shutdown of the operator.
type HealthShutdownHandler interface {
	// Handle waits for the context to be cancelled and then sets health states to unknown.
	Handle(ctx context.Context) error
}
