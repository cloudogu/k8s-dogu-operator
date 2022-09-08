package controllers

import (
	"context"
	"fmt"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	cesregistry "github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/k8s-apply-lib/apply"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NewManager is an alias mainly used for testing the main package
var NewManager = NewDoguManager

// DoguManager is a central unit in the process of handling dogu custom resources
// The DoguManager creates, updates and deletes dogus
type DoguManager struct {
	scheme         *runtime.Scheme
	installManager installManager
	deleteManager  deleteManager
	recorder       record.EventRecorder
}

type installManager interface {
	// Install installs a dogu resource.
	Install(ctx context.Context, doguResource *k8sv1.Dogu) error
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
	GetDoguDeployment(doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu, customDeployment *appsv1.Deployment) (*appsv1.Deployment, error)
	GetDoguService(doguResource *k8sv1.Dogu, imageConfig *imagev1.ConfigFile) (*corev1.Service, error)
	GetDoguPVC(doguResource *k8sv1.Dogu) (*corev1.PersistentVolumeClaim, error)
	GetDoguSecret(doguResource *k8sv1.Dogu, stringData map[string]string) (*corev1.Secret, error)
	GetDoguExposedServices(doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) ([]corev1.Service, error)
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
	RegisterDogu(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) error
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

// NewDoguManager creates a new instance of DoguManager
func NewDoguManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesregistry.Registry, eventRecorder record.EventRecorder) (*DoguManager, error) {
	err := validateKeyProvider(cesRegistry.GlobalConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to validate key provider: %w", err)
	}

	installManager, err := NewDoguInstallManager(client, operatorConfig, cesRegistry, eventRecorder)
	if err != nil {
		return nil, err
	}

	deleteManager, err := NewDoguDeleteManager(client, operatorConfig, cesRegistry)
	if err != nil {
		return nil, err
	}

	return &DoguManager{
		scheme:         client.Scheme(),
		installManager: installManager,
		deleteManager:  deleteManager,
		recorder:       eventRecorder,
	}, nil
}

func validateKeyProvider(globalConfig cesregistry.ConfigurationContext) error {
	exists, err := globalConfig.Exists("key_provider")
	if err != nil {
		return fmt.Errorf("failed to query key provider: %w", err)
	}

	if !exists {
		err = globalConfig.Set("key_provider", "pkcs1v15")
		if err != nil {
			return fmt.Errorf("failed to set default key provider: %w", err)
		}
		log.Log.Info("No key provider found. Use default pkcs1v15.")
	}

	return nil
}

func getDoguConfigMap(ctx context.Context, client client.Client, doguResource *k8sv1.Dogu) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	err := client.Get(ctx, doguResource.GetDescriptorObjectKey(), configMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get custom dogu descriptor: %w", err)
	} else {
		return configMap, nil
	}
}

func deleteDoguConfigMap(ctx context.Context, client client.Client, doguConfigMap *corev1.ConfigMap) error {
	if doguConfigMap != nil {
		err := client.Delete(ctx, doguConfigMap)
		if err != nil {
			return fmt.Errorf("failed to delete custom dogu descriptor: %w", err)
		}
	}

	return nil
}

// Install installs a dogu resource.
func (m *DoguManager) Install(ctx context.Context, doguResource *k8sv1.Dogu) error {
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Starting installation...")
	return m.installManager.Install(ctx, doguResource)
}

// Upgrade upgrades a dogu resource.
func (m *DoguManager) Upgrade(_ context.Context, _ *k8sv1.Dogu) error {
	return fmt.Errorf("currently not implemented")
}

// Delete deletes a dogu resource.
func (m *DoguManager) Delete(ctx context.Context, doguResource *k8sv1.Dogu) error {
	m.recorder.Eventf(doguResource, corev1.EventTypeNormal, DeinstallEventReason, "Starting deinstallation of the %s dogu.", doguResource.Name)
	return m.deleteManager.Delete(ctx, doguResource)
}
