package internal

import (
	"context"
	"github.com/cloudogu/k8s-apply-lib/apply"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	image "github.com/google/go-containerregistry/pkg/v1"
	apps "k8s.io/api/apps/v1"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// ResourceUpserter includes functionality to generate and create all the necessary K8s resources for a given dogu.
type ResourceUpserter interface {
	// UpsertDoguDeployment generates a deployment for a given dogu and applies it to the cluster.
	// All parameters are mandatory except customDeployment which may be nil.
	UpsertDoguDeployment(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu, customDeployment *apps.Deployment) (*apps.Deployment, error)
	// UpsertDoguService generates a service for a given dogu and applies it to the cluster.
	UpsertDoguService(ctx context.Context, doguResource *k8sv1.Dogu, image *image.ConfigFile) (*v1.Service, error)
	// UpsertDoguExposedServices creates exposed services based on the given dogu. If an error occurs during creating
	// several exposed services, this method tries to apply as many exposed services as possible and returns then
	// an error collection.
	UpsertDoguExposedServices(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) ([]*v1.Service, error)
	// UpsertDoguPVCs generates a persitent volume claim for a given dogu and applies it to the cluster.
	UpsertDoguPVCs(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) (*v1.PersistentVolumeClaim, error)
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
	// CollectApply applies the given resources to the K8s cluster but filters and collects deployments.
	CollectApply(ctx context.Context, customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*apps.Deployment, error)
}

// DoguResourceGenerator is used to generate kubernetes resources for the dogu.
type DoguResourceGenerator interface {
	CreateDoguDeployment(doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu, customDeployment *apps.Deployment) (*apps.Deployment, error)
	CreateDoguService(doguResource *k8sv1.Dogu, imageConfig *image.ConfigFile) (*v1.Service, error)
	CreateDoguPVC(doguResource *k8sv1.Dogu) (*v1.PersistentVolumeClaim, error)
	CreateReservedPVC(doguResource *k8sv1.Dogu) (*v1.PersistentVolumeClaim, error)
	CreateDoguExposedServices(doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) ([]*v1.Service, error)
}

// ResourceValidator provides functionality to validate resources for a given dogu.
type ResourceValidator interface {
	// Validate checks that a resource contains all necessary data to be used in a dogu.
	Validate(ctx context.Context, doguName string, obj client.Object) error
}
