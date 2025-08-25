package upgrade

import (
	"context"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
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

// doguRegistrator includes functionality to manage the registration of dogus in the local dogu registry.
type doguRegistrator interface {
	// RegisterDoguVersion registers a new version for an existing dogu in the dogu registry.
	RegisterDoguVersion(ctx context.Context, dogu *cesappcore.Dogu) error
}

// imageRegistry abstracts the use of a container registry and includes functionality to pull container images.
type imageRegistry interface {
	// PullImageConfig is used to pull the given container image.
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}

// serviceAccountCreator includes functionality to create necessary service accounts for a dogu.
type serviceAccountCreator interface {
	// CreateAll is used to create all necessary service accounts for the given dogu.
	CreateAll(ctx context.Context, dogu *cesappcore.Dogu) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type ecosystemInterface interface {
	doguClient.EcoSystemV2Interface
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

// fileExtractor provides functionality to get the contents of files from a container.
//
//nolint:unused
//goland:noinspection GoUnusedType
type fileExtractor interface {
	// ExtractK8sResourcesFromExecPod copies files from a dogu's exec pod into map of strings.
	ExtractK8sResourcesFromExecPod(ctx context.Context, doguResource *k8sv2.Dogu, dogu *cesappcore.Dogu) (map[string]string, error)
}

// collectApplier provides ways to collectedly apply unstructured Kubernetes resources against the API.
//
//nolint:unused
//goland:noinspection GoUnusedType
type collectApplier interface {
	// CollectApply applies the given resources to the K8s cluster
	CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *k8sv2.Dogu) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type eventRecorder interface {
	record.EventRecorder
}

// commandExecutor is used to execute commands in pods and dogus
//
//nolint:unused
//goland:noinspection GoUnusedType
type commandExecutor interface {
	exec.CommandExecutor
}

// resourceUpserter includes functionality to generate and create all the necessary K8s resources for a given dogu.
//
//nolint:unused
//goland:noinspection GoUnusedType
type resourceUpserter interface {
	resource.ResourceUpserter
}

// execPodFactory provides functionality to create ExecPods.
//
//nolint:unused
//goland:noinspection GoUnusedType
type execPodFactory interface {
	exec.ExecPodFactory
}
