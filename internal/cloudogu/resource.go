package cloudogu

import (
	"context"
	"github.com/cloudogu/k8s-apply-lib/apply"
	image "github.com/google/go-containerregistry/pkg/v1"
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// ResourceUpserter includes functionality to generate and create all the necessary K8s resources for a given dogu.
type ResourceUpserter interface {
	// UpsertDoguDeployment generates a deployment for a given dogu and applies it to the cluster.
	// All parameters are mandatory except deploymentPatch which may be nil.
	// The deploymentPatch can be used to arbitrarily alter the deployment after resource generation.
	UpsertDoguDeployment(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu, deploymentPatch func(*apps.Deployment)) (*apps.Deployment, error)
	// UpsertDoguService generates a service for a given dogu and applies it to the cluster.
	UpsertDoguService(ctx context.Context, doguResource *k8sv1.Dogu, image *image.ConfigFile) (*v1.Service, error)
	// UpsertDoguPVCs generates a persistent volume claim for a given dogu and applies it to the cluster.
	UpsertDoguPVCs(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) (*v1.PersistentVolumeClaim, error)
	// UpsertDoguExposedService creates oder updates the exposed service with the given dogu.
	UpsertDoguExposedService(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) (*v1.Service, error)
}

// DoguSecretHandler includes functionality to associate secrets from setup with a dogu.
type DoguSecretHandler interface {
	// WriteDoguSecretsToRegistry is used to write potential secret from the setup.json registryConfigEncrypted to the
	// respective dogu configurations.
	WriteDoguSecretsToRegistry(ctx context.Context, doguResource *k8sv1.Dogu) error
}

// Applier provides ways to apply unstructured Kubernetes resources against the API.
type Applier interface {
	// ApplyWithOwner provides a testable method for applying generic, unstructured K8s resources to the API
	ApplyWithOwner(doc apply.YamlDocument, namespace string, resource metav1.Object) error
}

// CollectApplier provides ways to collectedly apply unstructured Kubernetes resources against the API.
type CollectApplier interface {
	// CollectApply applies the given resources to the K8s cluster
	CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *k8sv1.Dogu) error
}

// DoguResourceGenerator is used to generate kubernetes resources for the dogu.
type DoguResourceGenerator interface {
	// CreateDoguDeployment creates a new instance of a deployment with a given dogu.json and dogu custom resource.
	CreateDoguDeployment(doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) (*apps.Deployment, error)
	// CreateDoguService creates a new instance of a service with the given dogu custom resource and container image.
	// The container image is used to extract the exposed ports. The created service is rather meant for cluster-internal
	// apps and dogus (f. e. postgresql) which do not need external access. The given container image config provides
	// the service ports to the created service.
	CreateDoguService(doguResource *k8sv1.Dogu, imageConfig *image.ConfigFile) (*v1.Service, error)
	// CreateDoguPVC creates a persistent volume claim with a 5Gi storage for the given dogu.
	CreateDoguPVC(doguResource *k8sv1.Dogu) (*v1.PersistentVolumeClaim, error)
	// CreateReservedPVC creates a persistent volume claim with a 10Mi storage for the given dogu.
	// Used for example for upgrade operations.
	CreateReservedPVC(doguResource *k8sv1.Dogu) (*v1.PersistentVolumeClaim, error)
}

// ExposePortAdder is used to expose exposed services from the dogu.
type ExposePortAdder interface {
	// CreateOrUpdateCesLoadbalancerService deletes the exposure of the exposed services from the dogu.
	CreateOrUpdateCesLoadbalancerService(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) (*v1.Service, error)
}

// ExposePortRemover is used to delete the exposure of the exposed services from the dogu.
type ExposePortRemover interface {
	// RemoveExposedPorts deletes the exposure of the exposed services from the dogu.
	RemoveExposedPorts(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) error
}

// TcpUpdServiceExposer is used to expose non http services.
type TcpUpdServiceExposer interface {
	// ExposeOrUpdateDoguServices adds or updates the exposing of the exposed ports in the dogu from the cluster. These are typically
	// entries in a configmap.
	ExposeOrUpdateDoguServices(ctx context.Context, namespace string, dogu *cesappcore.Dogu) error
	// DeleteDoguServices removes the exposing of the exposed ports in the dogu from the cluster. These are typically
	// entries in a configmap.
	DeleteDoguServices(ctx context.Context, namespace string, dogu *cesappcore.Dogu) error
}
