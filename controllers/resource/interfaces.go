package resource

import (
	"context"
	"github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-apply-lib/apply"
	k8sv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-registry-lib/config"
	"github.com/cloudogu/k8s-registry-lib/repository"
	image "github.com/google/go-containerregistry/pkg/v1"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type globalConfigurationWatcher interface {
	// Watch watches for changes of the provided config-key and sends the event through the channel
	Watch(ctx context.Context, filters ...config.WatchFilter) (<-chan repository.GlobalConfigWatchResult, error)
}

// RequirementsGenerator handles resource requirements (limits and requests) for dogu deployments.
type requirementsGenerator interface {
	Generate(ctx context.Context, dogu *cesappcore.Dogu) (v1.ResourceRequirements, error)
}

// hostAliasGenerator creates host aliases from fqdn, internal ip and additional host configuration.
type hostAliasGenerator interface {
	Generate(context.Context) (hostAliases []v1.HostAlias, err error)
}

type doguConfigGetter interface {
	Get(ctx context.Context, name dogu.SimpleName) (config.DoguConfig, error)
}

type securityContextGenerator interface {
	// Generate creates a k8s security context for the pod and containers of a dogu.
	Generate(ctx context.Context, dogu *cesappcore.Dogu, doguResource *k8sv2.Dogu) (*v1.PodSecurityContext, *v1.SecurityContext)
}

// ResourceUpserter includes functionality to generate and create all the necessary K8s resources for a given dogu.
type ResourceUpserter interface {
	// UpsertDoguDeployment generates a deployment for a given dogu and applies it to the cluster.
	// All parameters are mandatory except deploymentPatch which may be nil.
	// The deploymentPatch can be used to arbitrarily alter the deployment after resource generation.
	UpsertDoguDeployment(ctx context.Context, doguResource *k8sv2.Dogu, dogu *cesappcore.Dogu, deploymentPatch func(*apps.Deployment)) (*apps.Deployment, error)
	// UpsertDoguService generates a service for a given dogu and applies it to the cluster.
	UpsertDoguService(ctx context.Context, doguResource *k8sv2.Dogu, dogu *cesappcore.Dogu, image *image.ConfigFile) (*v1.Service, error)
	// UpsertDoguPVCs generates a persistent volume claim for a given dogu and applies it to the cluster.
	UpsertDoguPVCs(ctx context.Context, doguResource *k8sv2.Dogu, dogu *cesappcore.Dogu) (*v1.PersistentVolumeClaim, error)
	UpsertDoguNetworkPolicies(ctx context.Context, doguResource *k8sv2.Dogu, dogu *cesappcore.Dogu) error
}

// doguSecretHandler includes functionality to associate secrets from setup with a dogu.
//
//nolint:unused
//goland:noinspection GoUnusedType
type doguSecretHandler interface {
	// WriteDoguSecretsToRegistry is used to write potential secret from the setup.json registryConfigEncrypted to the
	// respective dogu configurations.
	WriteDoguSecretsToRegistry(ctx context.Context, doguResource *k8sv2.Dogu) error
}

// Applier provides ways to apply unstructured Kubernetes resources against the API.
type Applier interface {
	// ApplyWithOwner provides a testable method for applying generic, unstructured K8s resources to the API
	ApplyWithOwner(doc apply.YamlDocument, namespace string, resource metav1.Object) error
}

// CollectApplier provides ways to collectedly apply unstructured Kubernetes resources against the API.
type CollectApplier interface {
	// CollectApply applies the given resources to the K8s cluster
	CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *k8sv2.Dogu) error
}

// podTemplateResourceGenerator is used to generate pod templates.
type podTemplateResourceGenerator interface {
	// GetPodTemplate returns a pod template for the given dogu.
	GetPodTemplate(ctx context.Context, doguResource *k8sv2.Dogu, dogu *cesappcore.Dogu) (*v1.PodTemplateSpec, error)
}

// DoguResourceGenerator is used to generate kubernetes resources for the dogu.
type DoguResourceGenerator interface {
	podTemplateResourceGenerator

	// CreateDoguDeployment creates a new instance of a deployment with a given dogu.json and dogu custom resource.
	CreateDoguDeployment(ctx context.Context, doguResource *k8sv2.Dogu, dogu *cesappcore.Dogu) (*apps.Deployment, error)
	// CreateDoguService creates a new instance of a service with the given dogu custom resource and container image.
	// The container image is used to extract the exposed ports. The created service is rather meant for cluster-internal
	// apps and dogus (f. e. postgresql) which do not need external access. The given container image config provides
	// the service ports to the created service.
	CreateDoguService(doguResource *k8sv2.Dogu, dogu *cesappcore.Dogu, imageConfig *image.ConfigFile) (*v1.Service, error)
	// CreateDoguPVC creates a persistent volume claim with a 5Gi storage for the given dogu.
	CreateDoguPVC(doguResource *k8sv2.Dogu) (*v1.PersistentVolumeClaim, error)
}

// resourceRequirementsGenerator handles resource requirements (limits and requests) for dogu deployments.
//
//nolint:unused
//goland:noinspection GoUnusedType
type resourceRequirementsGenerator interface {
	// Generate creates resource limits and requests for dogu deployments.
	// It tries to retrieve them from the dogu config registry. If not set, defaults from the dogu.json are used.
	// If there is no default, the requirement will be omitted.
	Generate(dogu *cesappcore.Dogu) (v1.ResourceRequirements, error)
}

// localDoguFetcher includes functionality to search the local dogu registry for a dogu.
type localDoguFetcher interface {
	// FetchInstalled fetches the dogu from the local registry and returns it with patched dogu dependencies (which
	// otherwise might be incompatible with K8s CES).
	FetchInstalled(ctx context.Context, doguName dogu.SimpleName) (installedDogu *cesappcore.Dogu, err error)
}

type k8sClient interface {
	client.Client
}

type doguClientInterface interface {
	doguClient.DoguInterface
}
