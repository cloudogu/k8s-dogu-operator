package controllers

import (
	"context"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-apply-lib/apply"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/go-logr/logr"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type installManager interface {
	// Install installs a dogu resource.
	Install(ctx context.Context, doguResource *k8sv1.Dogu) error
}

type upgradeManager interface {
	// Upgrade upgrades a dogu resource.
	Upgrade(ctx context.Context, doguResource *k8sv1.Dogu) error
}
type deleteManager interface {
	// Delete deletes a dogu resource.
	Delete(ctx context.Context, doguResource *k8sv1.Dogu) error
}

type fileExtractor interface {
	// ExtractK8sResourcesFromContainer copies a file from stdout into map of strings
	ExtractK8sResourcesFromContainer(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) (map[string]string, error)
}

// doguResourceGenerator is used to generate kubernetes resources
type doguResourceGenerator interface {
	CreateDoguDeployment(doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu, customDeployment *appsv1.Deployment) (*appsv1.Deployment, error)
	CreateDoguService(doguResource *k8sv1.Dogu, imageConfig *imagev1.ConfigFile) (*corev1.Service, error)
	CreateDoguPVC(doguResource *k8sv1.Dogu) (*corev1.PersistentVolumeClaim, error)
	CreateDoguSecret(doguResource *k8sv1.Dogu, stringData map[string]string) (*corev1.Secret, error)
	CreateDoguExposedServices(doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) ([]corev1.Service, error)
}

// doguSecretHandler is used to write potential secret from the setup.json registryConfigEncrypted
type doguSecretHandler interface {
	WriteDoguSecretsToRegistry(ctx context.Context, doguResource *k8sv1.Dogu) error
}

// imageRegistry is used to pull container images
type imageRegistry interface {
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}

// doguRegistrator is used to register dogus
type doguRegistrator interface {
	RegisterNewDogu(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) error
	RegisterDoguVersion(dogu *cesappcore.Dogu) error
	UnregisterDogu(dogu string) error
}

// dependencyValidator is used to check if dogu dependencies are installed
type dependencyValidator interface {
	ValidateDependencies(dogu *cesappcore.Dogu) error
}

// serviceAccountCreator is used to create service accounts for a given dogu
type serviceAccountCreator interface {
	CreateAll(ctx context.Context, namespace string, dogu *cesappcore.Dogu) error
}

// serviceAccountRemover is used to remove service accounts for a given dogu
type serviceAccountRemover interface {
	RemoveAll(ctx context.Context, namespace string, dogu *cesappcore.Dogu) error
}

// DoguSecretsHandler is used to write the encrypted secrets from the setup to the dogu config
type DoguSecretsHandler interface {
	WriteDoguSecretsToRegistry(ctx context.Context, doguResource *k8sv1.Dogu) error
}

// applier provides ways to apply unstructured Kubernetes resources against the API.
type applier interface {
	// ApplyWithOwner provides a testable method for applying generic, unstructured K8s resources to the API
	ApplyWithOwner(doc apply.YamlDocument, namespace string, resource metav1.Object) error
}

type collectApplier interface {
	// CollectApply applies the given resources to the K8s cluster but filters and collects deployments.
	CollectApply(logger logr.Logger, customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*appsv1.Deployment, error)
}
