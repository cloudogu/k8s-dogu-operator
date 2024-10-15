package controllers

import (
	"context"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/remote"
	"github.com/cloudogu/k8s-apply-lib/apply"
	"github.com/cloudogu/k8s-dogu-operator/v2/api/ecoSystem"
	"github.com/cloudogu/k8s-dogu-operator/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v2/controllers/resource"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/cloudogu/k8s-registry-lib/repository"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	appsv1client "k8s.io/client-go/kubernetes/typed/apps/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// InstallManager includes functionality to install dogus in the cluster.
type InstallManager interface {
	// Install installs a dogu resource.
	Install(ctx context.Context, doguResource *v2.Dogu) error
}

// UpgradeManager includes functionality to upgrade dogus in the cluster.
type UpgradeManager interface {
	// Upgrade upgrades a dogu resource.
	Upgrade(ctx context.Context, doguResource *v2.Dogu) error
}

// DeleteManager includes functionality to delete dogus from the cluster.
type DeleteManager interface {
	// Delete deletes a dogu resource.
	Delete(ctx context.Context, doguResource *v2.Dogu) error
}

// SupportManager includes functionality to handle the support flag for dogus in the cluster.
type SupportManager interface {
	// HandleSupportMode handles the support flag in the dogu spec.
	HandleSupportMode(ctx context.Context, doguResource *v2.Dogu) (bool, error)
}

// VolumeManager includes functionality to edit volumes for dogus in the cluster.
type VolumeManager interface {
	// SetDoguDataVolumeSize sets the volume size for the given dogu.
	SetDoguDataVolumeSize(ctx context.Context, doguResource *v2.Dogu) error
}

// AdditionalIngressAnnotationsManager includes functionality to edit additional ingress annotations for dogus in the cluster.
type AdditionalIngressAnnotationsManager interface {
	// SetDoguAdditionalIngressAnnotations edits the additional ingress annotations in the given dogu's service.
	SetDoguAdditionalIngressAnnotations(ctx context.Context, doguResource *v2.Dogu) error
}

// StartDoguManager includes functionality to start (stopped) dogus.
type StartDoguManager interface {
	// StartDogu scales up a dogu to 1.
	StartDogu(ctx context.Context, doguResource *v2.Dogu) error
	// CheckStarted checks if the dogu has been successfully scaled to 1.
	CheckStarted(ctx context.Context, doguResource *v2.Dogu) error
}

// StopDoguManager includes functionality to stop running dogus.
type StopDoguManager interface {
	// StopDogu scales down a dogu to 0.
	StopDogu(ctx context.Context, doguResource *v2.Dogu) error
	// CheckStopped checks if the dogu has been successfully scaled to 0.
	CheckStopped(ctx context.Context, doguResource *v2.Dogu) error
}

// DoguStartStopManager includes functionality to start and stop dogus.
type DoguStartStopManager interface {
	StartDoguManager
	StopDoguManager
}

// CombinedDoguManager abstracts the simple dogu operations in a k8s CES.
type CombinedDoguManager interface {
	InstallManager
	UpgradeManager
	DeleteManager
	VolumeManager
	AdditionalIngressAnnotationsManager
	SupportManager
	StartDoguManager
	StopDoguManager
}

// RequeueHandler abstracts the process to decide whether a requeue process should be done based on received errors.
type RequeueHandler interface {
	// Handle takes an error and handles the requeue process for the current dogu operation.
	Handle(ctx context.Context, contextMessage string, doguResource *v2.Dogu, err error, onRequeue func(dogu *v2.Dogu) error) (result ctrl.Result, requeueErr error)
}

// RequirementsGenerator handles resource requirements (limits and requests) for dogu deployments.
type RequirementsGenerator interface {
	Generate(ctx context.Context, dogu *cesappcore.Dogu) (coreV1.ResourceRequirements, error)
}

// LocalDoguFetcher includes functionality to search the local dogu registry for a dogu.
type LocalDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(ctx context.Context, doguName string) (installedDogu *cesappcore.Dogu, err error)
}

// DoguRegistrator includes functionality to manage the registration of dogus in the local dogu registry.
type DoguRegistrator interface {
	// RegisterNewDogu registers a new dogu in the local dogu registry.
	RegisterNewDogu(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) error
	// RegisterDoguVersion registers a new version for an existing dogu in the dogu registry.
	RegisterDoguVersion(ctx context.Context, dogu *cesappcore.Dogu) error
	// UnregisterDogu removes a registration of a dogu from the local dogu registry.
	UnregisterDogu(ctx context.Context, dogu string) error
}

// ResourceDoguFetcher includes functionality to get a dogu either from the remote dogu registry or from a local development dogu map.
type ResourceDoguFetcher interface {
	// FetchWithResource fetches the dogu either from the remote dogu registry or from a local development dogu map and
	// returns it with patched dogu dependencies (which otherwise might be incompatible with K8s CES).
	FetchWithResource(ctx context.Context, doguResource *v2.Dogu) (*cesappcore.Dogu, *v2.DevelopmentDoguMap, error)
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

// ServiceAccountRemover includes functionality to remove existing service accounts for a dogu.
type ServiceAccountRemover interface {
	// RemoveAll is used to remove all existing service accounts for the given dogu.
	RemoveAll(ctx context.Context, dogu *cesappcore.Dogu) error
}

type DeploymentInterface interface {
	appsv1client.DeploymentInterface
}

type AppsV1Interface interface {
	appsv1client.AppsV1Interface
}

type ClientSet interface {
	kubernetes.Interface
}

type DeploymentAvailabilityChecker interface {
	// IsAvailable checks whether the deployment has reached its desired state and is available.
	IsAvailable(deployment *appsv1.Deployment) bool
}

type DoguHealthStatusUpdater interface {
	// UpdateStatus sets the health status of the dogu according to whether if it's available or not.
	UpdateStatus(ctx context.Context, doguName types.NamespacedName, available bool) error
	UpdateHealthConfigMap(ctx context.Context, deployment *appsv1.Deployment, doguJson *cesappcore.Dogu) error
}

type ControllerManager interface {
	manager.Manager
}

// RemoteRegistry is able to manage the remote dogu registry.
type RemoteRegistry interface {
	remote.Registry
}

// CommandExecutor is used to execute commands in pods and dogus
type CommandExecutor interface {
	exec.CommandExecutor
}

type K8sClient interface {
	client.Client
}

// HostAliasGenerator creates host aliases from fqdn, internal ip and additional host configuration.
type HostAliasGenerator interface {
	Generate(context.Context) (hostAliases []coreV1.HostAlias, err error)
}

// FileExtractor provides functionality to get the contents of files from a container.
type FileExtractor interface {
	// ExtractK8sResourcesFromContainer copies a file from stdout into map of strings.
	ExtractK8sResourcesFromContainer(ctx context.Context, k8sExecPod exec.ExecPod) (map[string]string, error)
}

// Applier provides ways to apply unstructured Kubernetes resources against the API.
type Applier interface {
	// ApplyWithOwner provides a testable method for applying generic, unstructured K8s resources to the API
	ApplyWithOwner(doc apply.YamlDocument, namespace string, resource metav1.Object) error
}

type EventRecorder interface {
	record.EventRecorder
}

// ExposePortRemover is used to delete the exposure of the exposed services from the dogu.
type ExposePortRemover interface {
	// RemoveExposedPorts deletes the exposure of the exposed services from the dogu.
	RemoveExposedPorts(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) error
}

type DoguConfigRepository interface {
	Get(ctx context.Context, name config.SimpleDoguName) (config.DoguConfig, error)
	Create(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	Update(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	SaveOrMerge(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	Delete(ctx context.Context, name config.SimpleDoguName) error
	Watch(ctx context.Context, dName config.SimpleDoguName, filters ...config.WatchFilter) (<-chan repository.DoguConfigWatchResult, error)
}

// DependencyValidator checks if all necessary dependencies for an upgrade are installed.
type DependencyValidator interface {
	// ValidateDependencies is used to check if dogu dependencies are installed.
	ValidateDependencies(ctx context.Context, dogu *cesappcore.Dogu) error
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

type PodInterface interface {
	v1.PodInterface
}

// PremisesChecker includes functionality to check if the premises for an upgrade are met.
type PremisesChecker interface {
	// Check checks if dogu premises are met before a dogu upgrade.
	Check(ctx context.Context, toDoguResource *v2.Dogu, fromDogu *cesappcore.Dogu, toDogu *cesappcore.Dogu) error
}

// UpgradeExecutor applies upgrades the upgrade from an earlier dogu version to a newer version.
type UpgradeExecutor interface {
	// Upgrade executes the actual dogu upgrade.
	Upgrade(ctx context.Context, toDoguResource *v2.Dogu, fromDogu *cesappcore.Dogu, toDogu *cesappcore.Dogu) error
}

type K8sSubResourceWriter interface {
	client.SubResourceWriter
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
