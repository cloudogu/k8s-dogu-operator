package controllers

import (
	"context"
	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-apply-lib/apply"
	"github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/exec"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
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

// installManager includes functionality to install dogus in the cluster.
type installManager interface {
	// Install installs a dogu resource.
	Install(ctx context.Context, doguResource *v2.Dogu) error
}

// upgradeManager includes functionality to upgrade dogus in the cluster.
type upgradeManager interface {
	// Upgrade upgrades a dogu resource.
	Upgrade(ctx context.Context, doguResource *v2.Dogu) error
}

// deleteManager includes functionality to delete dogus from the cluster.
type deleteManager interface {
	// Delete deletes a dogu resource.
	Delete(ctx context.Context, doguResource *v2.Dogu) error
}

// supportManager includes functionality to handle the support flag for dogus in the cluster.
type supportManager interface {
	// HandleSupportMode handles the support flag in the dogu spec.
	HandleSupportMode(ctx context.Context, doguResource *v2.Dogu) (bool, error)
}

// exportManager includes functionality to handle the export flag for dogus in the cluster.
type exportManager interface {
	// UpdateExportMode activates/deactivates the export mode for the dogu
	UpdateExportMode(ctx context.Context, doguResource *v2.Dogu) error
}

// volumeManager includes functionality to edit volumes for dogus in the cluster.
type volumeManager interface {
	// SetDoguDataVolumeSize sets the volume size for the given dogu.
	SetDoguDataVolumeSize(ctx context.Context, doguResource *v2.Dogu) error
}

// additionalIngressAnnotationsManager includes functionality to edit additional ingress annotations for dogus in the cluster.
type additionalIngressAnnotationsManager interface {
	// SetDoguAdditionalIngressAnnotations edits the additional ingress annotations in the given dogu's service.
	SetDoguAdditionalIngressAnnotations(ctx context.Context, doguResource *v2.Dogu) error
}

type securityContextManager interface {
	// UpdateDeploymentWithSecurityContext regenerates the security context of a dogu deployment.
	UpdateDeploymentWithSecurityContext(ctx context.Context, doguResource *v2.Dogu) error
}

type additionalMountsManager interface {
	AdditionalMountsChanged(ctx context.Context, doguResource *v2.Dogu) (bool, error)
	UpdateAdditionalMounts(ctx context.Context, doguResource *v2.Dogu) error
}

type additionalMountsInitContainerGenerator interface {
	BuildAdditionalMountInitContainer(ctx context.Context, dogu *cesappcore.Dogu, doguResource *v2.Dogu, image string, requirements coreV1.ResourceRequirements) (*coreV1.Container, error)
}

type startStopManager interface {
	StartStopDogu(ctx context.Context, doguResource *v2.Dogu) error
}

// CombinedDoguManager abstracts the simple dogu operations in a k8s CES.
type CombinedDoguManager interface {
	installManager
	upgradeManager
	deleteManager
	volumeManager
	additionalIngressAnnotationsManager
	exportManager
	supportManager
	startStopManager
	securityContextManager
	additionalMountsManager
}

// requeueHandler abstracts the process to decide whether a requeue process should be done based on received errors.
type requeueHandler interface {
	// Handle takes an error and handles the requeue process for the current dogu operation.
	Handle(ctx context.Context, contextMessage string, doguResource *v2.Dogu, err error, onRequeue func(dogu *v2.Dogu) error) (result ctrl.Result, requeueErr error)
}

type securityValidator interface {
	ValidateSecurity(doguDescriptor *cesappcore.Dogu, doguResource *v2.Dogu) error
}

type doguAdditionalMountsValidator interface {
	ValidateAdditionalMounts(ctx context.Context, doguDescriptor *cesappcore.Dogu, doguResource *v2.Dogu) error
}

// requirementsGenerator handles resource requirements (limits and requests) for dogu deployments.
//
//nolint:unused
//goland:noinspection GoUnusedType
type requirementsGenerator interface {
	Generate(ctx context.Context, dogu *cesappcore.Dogu) (coreV1.ResourceRequirements, error)
}

// DoguResourceGenerator handles resource generation for dogus.
//
//nolint:unused
//goland:noinspection GoUnusedType
type DoguResourceGenerator interface {
	podTemplateResourceGenerator

	// CreateDoguDeployment creates a new instance of a deployment with a given dogu.json and dogu custom resource.
	CreateDoguDeployment(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) (*appsv1.Deployment, error)
	// CreateDoguService creates a new instance of a service with the given dogu custom resource and container image.
	// The container image is used to extract the exposed ports. The created service is rather meant for cluster-internal
	// apps and dogus (f. e. postgresql) which do not need external access. The given container image config provides
	// the service ports to the created service.
	CreateDoguService(doguResource *v2.Dogu, dogu *cesappcore.Dogu, imageConfig *imagev1.ConfigFile) (*coreV1.Service, error)
	// CreateDoguPVC creates a persistent volume claim with a 5Gi storage for the given dogu.
	CreateDoguPVC(doguResource *v2.Dogu) (*coreV1.PersistentVolumeClaim, error)
}

// localDoguFetcher includes functionality to search the local dogu registry for a dogu.
type localDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(ctx context.Context, doguName cescommons.SimpleName) (installedDogu *cesappcore.Dogu, err error)
	// Enabled checks is the given dogu is enabled.
	// Returns false (without error), when the dogu is not installed
	Enabled(ctx context.Context, doguName cescommons.SimpleName) (bool, error)
}

// doguRegistrator includes functionality to manage the registration of dogus in the local dogu registry.
type doguRegistrator interface {
	// RegisterNewDogu registers a new dogu in the local dogu registry.
	RegisterNewDogu(ctx context.Context, doguResource *v2.Dogu, dogu *cesappcore.Dogu) error
	// RegisterDoguVersion registers a new version for an existing dogu in the dogu registry.
	RegisterDoguVersion(ctx context.Context, dogu *cesappcore.Dogu) error
	// UnregisterDogu removes a registration of a dogu from the local dogu registry.
	UnregisterDogu(ctx context.Context, dogu string) error
}

// resourceDoguFetcher includes functionality to get a dogu either from the remote dogu registry or from a local development dogu map.
type resourceDoguFetcher interface {
	// FetchWithResource fetches the dogu either from the remote dogu registry or from a local development dogu map and
	// returns it with patched dogu dependencies (which otherwise might be incompatible with K8s CES).
	FetchWithResource(ctx context.Context, doguResource *v2.Dogu) (*cesappcore.Dogu, *v2.DevelopmentDoguMap, error)
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

// serviceAccountRemover includes functionality to remove existing service accounts for a dogu.
//
//nolint:unused
//goland:noinspection GoUnusedType
type serviceAccountRemover interface {
	// RemoveAll is used to remove all existing service accounts for the given dogu.
	RemoveAll(ctx context.Context, dogu *cesappcore.Dogu) error
}

type deploymentInterface interface {
	appsv1client.DeploymentInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type appsV1Interface interface {
	appsv1client.AppsV1Interface
}

type ClientSet interface {
	kubernetes.Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type coreV1Interface interface {
	v1.CoreV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type pvcInterface interface {
	v1.PersistentVolumeClaimInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type deploymentAvailabilityChecker interface {
	// IsAvailable checks whether the deployment has reached its desired state and is available.
	IsAvailable(deployment *appsv1.Deployment) bool
}

//nolint:unused
//goland:noinspection GoUnusedType
type doguHealthStatusUpdater interface {
	// UpdateStatus sets the health status of the dogu according to whether if it's available or not.
	UpdateStatus(ctx context.Context, doguName types.NamespacedName, available bool) error
	UpdateHealthConfigMap(ctx context.Context, deployment *appsv1.Deployment, doguJson *cesappcore.Dogu) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type controllerManager interface {
	manager.Manager
}

//nolint:unused
//goland:noinspection GoUnusedType
type remoteDoguDescriptorRepository interface {
	cescommons.RemoteDoguDescriptorRepository
}

// commandExecutor is used to execute commands in pods and dogus
//
//nolint:unused
//goland:noinspection GoUnusedType
type commandExecutor interface {
	exec.CommandExecutor
}

type K8sClient interface {
	client.Client
}

// hostAliasGenerator creates host aliases from fqdn, internal ip and additional host configuration.
//
//nolint:unused
//goland:noinspection GoUnusedType
type hostAliasGenerator interface {
	Generate(context.Context) (hostAliases []coreV1.HostAlias, err error)
}

// fileExtractor provides functionality to get the contents of files from a container.
//
//nolint:unused
//goland:noinspection GoUnusedType
type fileExtractor interface {
	// ExtractK8sResourcesFromContainer copies a file from stdout into map of strings.
	ExtractK8sResourcesFromContainer(ctx context.Context, k8sExecPod exec.ExecPod) (map[string]string, error)
}

// applier provides ways to apply unstructured Kubernetes resources against the API.
//
//nolint:unused
//goland:noinspection GoUnusedType
type applier interface {
	// ApplyWithOwner provides a testable method for applying generic, unstructured K8s resources to the API
	ApplyWithOwner(doc apply.YamlDocument, namespace string, resource metav1.Object) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type eventRecorder interface {
	record.EventRecorder
}

type doguConfigRepository interface {
	Get(ctx context.Context, name cescommons.SimpleName) (config.DoguConfig, error)
	Create(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	Update(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	SaveOrMerge(ctx context.Context, doguConfig config.DoguConfig) (config.DoguConfig, error)
	Delete(ctx context.Context, name cescommons.SimpleName) error
	Watch(ctx context.Context, dName cescommons.SimpleName, filters ...config.WatchFilter) (<-chan repository.DoguConfigWatchResult, error)
}

// dependencyValidator checks if all necessary dependencies for an upgrade are installed.
//
//nolint:unused
//goland:noinspection GoUnusedType
type dependencyValidator interface {
	// ValidateDependencies is used to check if dogu dependencies are installed.
	ValidateDependencies(ctx context.Context, dogu *cesappcore.Dogu) error
}

// resourceUpserter includes functionality to generate and create all the necessary K8s resources for a given dogu.
//
//nolint:unused
//goland:noinspection GoUnusedType
type resourceUpserter interface {
	resource.ResourceUpserter
}

// execPod provides methods for instantiating and removing an intermediate pod based on a Dogu container image.
//
//nolint:unused
//goland:noinspection GoUnusedType
type execPod interface {
	exec.ExecPod
}

// execPodFactory provides functionality to create ExecPods.
//
//nolint:unused
//goland:noinspection GoUnusedType
type execPodFactory interface {
	exec.ExecPodFactory
}

type podInterface interface {
	v1.PodInterface
}

// premisesChecker includes functionality to check if the premises for an upgrade are met.
//
//nolint:unused
//goland:noinspection GoUnusedType
type premisesChecker interface {
	// Check checks if dogu premises are met before a dogu upgrade.
	Check(ctx context.Context, toDoguResource *v2.Dogu, fromDogu *cesappcore.Dogu, toDogu *cesappcore.Dogu) error
}

// upgradeExecutor applies upgrades the upgrade from an earlier dogu version to a newer version.
//
//nolint:unused
//goland:noinspection GoUnusedType
type upgradeExecutor interface {
	// Upgrade executes the actual dogu upgrade.
	Upgrade(ctx context.Context, toDoguResource *v2.Dogu, fromDogu *cesappcore.Dogu, toDogu *cesappcore.Dogu) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type k8sSubResourceWriter interface {
	client.SubResourceWriter
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
