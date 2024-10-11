package upgrade

import (
	"context"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/v2/api/ecoSystem"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/resource"
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

// DoguRegistrator includes functionality to manage the registration of dogus in the local dogu registry.
type DoguRegistrator interface {
	// RegisterDoguVersion registers a new version for an existing dogu in the dogu registry.
	RegisterDoguVersion(ctx context.Context, dogu *cesappcore.Dogu) error
}

// ImageRegistry abstracts the use of a container registry and includes functionality to pull container images.
type ImageRegistry interface {
	// PullImageConfig is used to pull the given container image.
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}

// ServiceAccountCreator includes functionality to create necessary service accounts for a dogu.
type ServiceAccountCreator interface {
	// CreateAll is used to create all necessary service accounts for the given dogu.
	CreateAll(ctx context.Context, dogu *cesappcore.Dogu) error
}

type EcosystemInterface interface {
	ecoSystem.EcoSystemV1Alpha1Interface
}

type DoguInterface interface {
	ecoSystem.DoguInterface
}

type DoguRestartInterface interface {
	ecoSystem.DoguRestartInterface
}

// FileExtractor provides functionality to get the contents of files from a container.
type FileExtractor interface {
	// ExtractK8sResourcesFromContainer copies a file from stdout into map of strings.
	ExtractK8sResourcesFromContainer(ctx context.Context, k8sExecPod exec.ExecPod) (map[string]string, error)
}

// CollectApplier provides ways to collectedly apply unstructured Kubernetes resources against the API.
type CollectApplier interface {
	// CollectApply applies the given resources to the K8s cluster
	CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *k8sv2.Dogu) error
}

type EventRecorder interface {
	record.EventRecorder
}

// CommandExecutor is used to execute commands in pods and dogus
type CommandExecutor interface {
	exec.CommandExecutor
}

// ResourceUpserter includes functionality to generate and create all the necessary K8s resources for a given dogu.
type ResourceUpserter interface {
	resource.ResourceUpserter
}

// ExecPod provides methods for instantiating and removing an intermediate pod based on a Dogu container image.
type ExecPod interface {
	exec.ExecPod
}

// ExecPodFactory provides functionality to create ExecPods.
type ExecPodFactory interface {
	exec.ExecPodFactory
}
