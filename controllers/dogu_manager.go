package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	cesregistry "github.com/cloudogu/cesapp-lib/registry"
	"github.com/cloudogu/k8s-apply-lib/apply"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/controllers/registry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"
	"github.com/go-logr/logr"
	"sigs.k8s.io/yaml"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const finalizerName = "dogu-finalizer"
const k8sDoguOperatorFieldManagerName = "k8s-dogu-operator"

// NewManager is an alias mainly used for testing the main package
var NewManager = NewDoguManager

// DoguManager is a central unit in the process of handling dogu custom resources
// The DoguManager creates, updates and deletes dogus
type DoguManager struct {
	client.Client
	Scheme                *runtime.Scheme
	ResourceGenerator     doguResourceGenerator
	DoguRemoteRegistry    doguRegistry
	DoguLocalRegistry     cesregistry.DoguRegistry
	ImageRegistry         imageRegistry
	DoguRegistrator       doguRegistrator
	DependencyValidator   dependencyValidator
	ServiceAccountCreator serviceAccountCreator
	ServiceAccountRemover serviceAccountRemover
	DoguSecretHandler     doguSecretHandler
	FileExtractor         fileExtractor
	Applier               Applier
}

type fileExtractor interface {
	// ExtractK8sResourcesFromContainer copies a file from stdout into map of strings
	ExtractK8sResourcesFromContainer(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) (map[string]string, error)
}

// doguRegistry is used to fetch the dogu descriptor
type doguRegistry interface {
	GetDogu(*k8sv1.Dogu) (*cesappcore.Dogu, error)
}

// doguResourceGenerator is used to generate kubernetes resources
type doguResourceGenerator interface {
	GetDoguDeployment(doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) (*appsv1.Deployment, error)
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

// Applier provides ways to apply unstructured Kubernetes resources against the API.
type Applier interface {
	// ApplyWithOwner provides a testable method for applying generic, unstructured K8s resources to the API
	ApplyWithOwner(doc apply.YamlDocument, namespace string, resource metav1.Object) error
}

// NewDoguManager creates a new instance of DoguManager
func NewDoguManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesregistry.Registry) (*DoguManager, error) {
	doguRemoteRegistry := registry.New(operatorConfig.DoguRegistry.Username, operatorConfig.DoguRegistry.Password, operatorConfig.DoguRegistry.Endpoint)
	imageRegistry := registry.NewCraneContainerImageRegistry(operatorConfig.DockerRegistry.Username, operatorConfig.DockerRegistry.Password)
	resourceGenerator := resource.NewResourceGenerator(client.Scheme())
	restConfig := ctrl.GetConfigOrDie()
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed find cluster config: %w", err)
	}

	fileExtract := newPodFileExtractor(client, restConfig, clientSet)
	applier, scheme, err := apply.New(restConfig, k8sDoguOperatorFieldManagerName)
	if err != nil {
		return nil, fmt.Errorf("failed create K8s Applier: %w", err)
	}
	err = k8sv1.AddToScheme(scheme)
	if err != nil {
		return nil, fmt.Errorf("failed add applier scheme to dogu CRD scheme handling: %w", err)
	}

	err = validateKeyProvider(cesRegistry.GlobalConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to validate key provider: %w", err)
	}

	doguRegistrator := NewCESDoguRegistrator(client, cesRegistry, resourceGenerator)
	dependencyValidator := dependency.NewCompositeDependencyValidator(operatorConfig.Version, cesRegistry.DoguRegistry())

	executor := resource.NewCommandExecutor(clientSet, clientSet.CoreV1().RESTClient())
	serviceAccountCreator := serviceaccount.NewCreator(cesRegistry, executor)
	serviceAccountRemover := serviceaccount.NewRemover(cesRegistry, executor)

	doguSecretHandler := resource.NewDoguSecretsWriter(client, cesRegistry)

	return &DoguManager{
		Client:                client,
		Scheme:                client.Scheme(),
		ResourceGenerator:     resourceGenerator,
		DoguRemoteRegistry:    doguRemoteRegistry,
		DoguLocalRegistry:     cesRegistry.DoguRegistry(),
		ImageRegistry:         imageRegistry,
		DoguRegistrator:       doguRegistrator,
		DependencyValidator:   dependencyValidator,
		ServiceAccountCreator: serviceAccountCreator,
		DoguSecretHandler:     doguSecretHandler,
		ServiceAccountRemover: serviceAccountRemover,
		FileExtractor:         fileExtract,
		Applier:               applier,
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

// Install installs a given Dogu Resource. This includes fetching the dogu.json and the container image. With the
// information Install creates a Deployment and a Service
func (m *DoguManager) Install(ctx context.Context, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)

	doguResource.Status = k8sv1.DoguStatus{RequeueTime: doguResource.Status.RequeueTime, Status: k8sv1.DoguStatusInstalling, StatusMessages: []string{}}
	err := doguResource.Update(ctx, m.Client)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	// Set the finalizer at the beginning of the install procedure.
	// This is required because an error during installation would leave a dogu resource with its
	// k8s resources in the cluster. A delete would tidy up those resources but would not start the
	// delete procedure from the controller.
	logger.Info("Add dogu finalizer...")
	controllerutil.AddFinalizer(doguResource, finalizerName)
	err = m.Client.Update(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to update dogu: %w", err)
	}

	doguConfigMap, err := m.getDoguConfigMap(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to get dogu config map: %w", err)
	}

	logger.Info("Fetching dogu...")
	dogu, err := m.getDoguDescriptor(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to get dogu: %w", err)
	}

	logger.Info("Check dogu dependencies...")
	err = m.DependencyValidator.ValidateDependencies(dogu)
	if err != nil {
		return err
	}

	logger.Info("Register dogu...")
	err = m.DoguRegistrator.RegisterDogu(ctx, doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to register dogu: %w", err)
	}

	logger.Info("Write dogu secrets from setup...")
	err = m.DoguSecretHandler.WriteDoguSecretsToRegistry(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to write dogu secrets from setup: %w", err)
	}

	logger.Info("Create service accounts...")
	err = m.ServiceAccountCreator.CreateAll(ctx, doguResource.Namespace, dogu)
	if err != nil {
		return fmt.Errorf("failed to create service accounts: %w", err)
	}

	logger.Info("Pull image config...")
	imageConfig, err := m.ImageRegistry.PullImageConfig(ctx, dogu.Image+":"+dogu.Version)
	if err != nil {
		return fmt.Errorf("failed to pull image config: %w", err)
	}

	customK8sResources, err := m.FileExtractor.ExtractK8sResourcesFromContainer(ctx, doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to pull customK8sResources: %w", err)
	}

	serviceAccount, err := m.applyCustomK8sResources(logger, customK8sResources, doguResource)
	if err != nil {
		return err
	}

	logger.Info("Create dogu resources...")
	err = m.createDoguResources(ctx, doguResource, dogu, imageConfig, serviceAccount)
	if err != nil {
		return fmt.Errorf("failed to create dogu resources: %w", err)
	}

	doguResource.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalled, StatusMessages: []string{}}
	err = doguResource.Update(ctx, m.Client)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	if doguConfigMap != nil {
		err = m.Client.Delete(ctx, doguConfigMap)
		if err != nil {
			return fmt.Errorf("failed to delete custom dogu descriptor: %w", err)
		}
	}

	return nil
}

func (m *DoguManager) applyCustomK8sResources(logger logr.Logger, customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*corev1.ServiceAccount, error) {
	if len(customK8sResources) == 0 {
		logger.Info("No custom K8s resources found")
		return nil, nil
	}

	targetNamespace := doguResource.ObjectMeta.Namespace

	namespaceTemplate := struct {
		Namespace string
	}{
		Namespace: targetNamespace,
	}

	saCollector := &serviceAccountCollector{collected: []*corev1.ServiceAccount{}}

	for file, yamlDocs := range customK8sResources {
		logger.Info(fmt.Sprintf("Applying custom K8s resources from file %s", file))

		err := apply.NewBuilder(m.Applier).
			WithNamespace(targetNamespace).
			WithOwner(doguResource).
			WithTemplate(file, namespaceTemplate).
			WithCollector(saCollector).
			WithYamlResource(file, []byte(yamlDocs)).
			ExecuteApply()

		if err != nil {
			return nil, err
		}
	}

	if len(saCollector.collected) > 1 {
		return nil, fmt.Errorf("expected exactly one ServiceAccount but found %d - not sure how to continue", len(saCollector.collected))
	}
	if len(saCollector.collected) == 1 {
		return saCollector.collected[0], nil
	}

	return nil, nil
}

type serviceAccountCollector struct {
	collected []*corev1.ServiceAccount
}

func (sac *serviceAccountCollector) Predicate(doc apply.YamlDocument) (bool, error) {
	var serviceAccount = &corev1.ServiceAccount{}

	err := yaml.Unmarshal(doc, serviceAccount)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal object [%s] into service account: %w", string(doc), err)
	}

	return serviceAccount.Kind == "ServiceAccount", nil
}

func (sac *serviceAccountCollector) Collect(doc apply.YamlDocument) {
	var serviceAccount = &corev1.ServiceAccount{}

	// ignore error because it has already been parsed in Predicate()
	_ = yaml.Unmarshal(doc, serviceAccount)

	sac.collected = append(sac.collected, serviceAccount)
}

func (m *DoguManager) getDoguDescriptorFromConfigMap(doguConfigMap *corev1.ConfigMap) (*cesappcore.Dogu, error) {
	jsonStr := doguConfigMap.Data["dogu.json"]
	dogu := &cesappcore.Dogu{}
	err := json.Unmarshal([]byte(jsonStr), dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal custom dogu descriptor: %w", err)
	}

	return dogu, nil
}

func (m *DoguManager) getDoguDescriptorFromRemoteRegistry(doguResource *k8sv1.Dogu) (*cesappcore.Dogu, error) {
	dogu, err := m.DoguRemoteRegistry.GetDogu(doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu from remote dogu registry: %w", err)
	}

	return dogu, nil
}

func (m *DoguManager) getDoguDescriptorFromLocalRegistry(doguResource *k8sv1.Dogu) (*cesappcore.Dogu, error) {
	dogu, err := m.DoguLocalRegistry.Get(doguResource.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu from local dogu registry: %w", err)
	}

	return dogu, nil
}

func (m *DoguManager) getDoguConfigMap(ctx context.Context, doguResource *k8sv1.Dogu) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	err := m.Client.Get(ctx, doguResource.GetDescriptorObjectKey(), configMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		} else {
			return nil, fmt.Errorf("failed to get custom dogu descriptor: %w", err)
		}
	} else {
		return configMap, nil
	}
}

func (m *DoguManager) getDoguDescriptorWithConfigMap(ctx context.Context, doguResource *k8sv1.Dogu, doguConfigMap *corev1.ConfigMap) (*cesappcore.Dogu, error) {
	logger := log.FromContext(ctx)

	if doguConfigMap != nil {
		logger.Info("Fetching dogu from custom configmap...")
		return m.getDoguDescriptorFromConfigMap(doguConfigMap)
	} else {
		logger.Info("Fetching dogu from dogu registry...")
		return m.getDoguDescriptorFromRemoteRegistry(doguResource)
	}
}

func (m *DoguManager) getDoguDescriptor(ctx context.Context, doguResource *k8sv1.Dogu) (*cesappcore.Dogu, error) {
	doguConfigMap, err := m.getDoguConfigMap(ctx, doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu config map: %w", err)
	}

	dogu, err := m.getDoguDescriptorWithConfigMap(ctx, doguResource, doguConfigMap)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu: %w", err)
	}

	return dogu, nil
}

func (m *DoguManager) createDoguResources(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu, imageConfig *imagev1.ConfigFile, serviceAccount *corev1.ServiceAccount) error {
	err := m.createVolumes(ctx, doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to create volumes for dogu %s: %w", dogu.Name, err)
	}

	err = m.createDeployment(ctx, doguResource, dogu, serviceAccount)
	if err != nil {
		return fmt.Errorf("failed to create deployment for dogu %s: %w", dogu.Name, err)
	}

	err = m.createService(ctx, doguResource, imageConfig)
	if err != nil {
		return fmt.Errorf("failed to create service for dogu %s: %w", dogu.Name, err)
	}

	err = m.createExposedServices(ctx, doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to create exposed services for dogu %s: %w", dogu.Name, err)
	}

	return nil
}

func (m *DoguManager) createVolumes(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) error {
	logger := log.FromContext(ctx)

	if len(dogu.Volumes) > 0 {
		desiredPvc, err := m.ResourceGenerator.GetDoguPVC(doguResource)
		if err != nil {
			return fmt.Errorf("failed to generate pvc: %w", err)
		}

		err = m.Client.Create(ctx, desiredPvc)
		if err != nil {
			return fmt.Errorf("failed to create pvc: %w", err)
		}

		logger.Info(fmt.Sprintf("PersistentVolumeClaim %s/%s has been : %s", desiredPvc.Namespace, desiredPvc.Name, controllerutil.OperationResultCreated))
	}

	return nil
}

func (m *DoguManager) createDeployment(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu, serviceAccount *corev1.ServiceAccount) error {
	logger := log.FromContext(ctx)

	desiredDeployment, err := m.ResourceGenerator.GetDoguDeployment(doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to generate dogu deployment: %w", err)
	}

	if serviceAccount != nil {
		logger.Info("Found service account in k8s folder... injecting into deployment")
		desiredDeployment.Spec.Template.Spec.ServiceAccountName = serviceAccount.GetName()
	}

	err = m.Client.Create(ctx, desiredDeployment)
	if err != nil {
		return fmt.Errorf("failed to create dogu deployment: %w", err)
	}

	logger.Info(fmt.Sprintf("Deployment %s/%s has been : %s", desiredDeployment.Namespace, desiredDeployment.Name, controllerutil.OperationResultCreated))
	return nil
}

func (m *DoguManager) createService(ctx context.Context, doguResource *k8sv1.Dogu, imageConfig *imagev1.ConfigFile) error {
	logger := log.FromContext(ctx)

	desiredService, err := m.ResourceGenerator.GetDoguService(doguResource, imageConfig)
	if err != nil {
		return fmt.Errorf("failed to generate dogu service: %w", err)
	}

	err = m.Client.Create(ctx, desiredService)
	if err != nil {
		return fmt.Errorf("failed to create dogu service: %w", err)
	}

	logger.Info(fmt.Sprintf("Service %s/%s has been : %s", desiredService.Namespace, desiredService.Name, controllerutil.OperationResultCreated))
	return nil
}

func (m *DoguManager) createExposedServices(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu) error {
	logger := log.FromContext(ctx)

	exposedServices, err := m.ResourceGenerator.GetDoguExposedServices(doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to generate exposed services: %w", err)
	}

	for _, service := range exposedServices {
		err = m.Client.Create(ctx, &service)
		if err != nil {
			return fmt.Errorf("failed to create exposed service: %w", err)
		}

		logger.Info(fmt.Sprintf("Exposed Service %s/%s have been : %s", service.Namespace, service.Name, controllerutil.OperationResultCreated))
	}
	return nil
}

func (m *DoguManager) Delete(ctx context.Context, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)
	doguResource.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusDeleting, StatusMessages: []string{}}
	err := doguResource.Update(ctx, m.Client)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	logger.Info("Fetching dogu...")
	dogu, err := m.getDoguDescriptorFromLocalRegistry(doguResource)
	if err != nil {
		return fmt.Errorf("failed to get dogu: %w", err)
	}

	logger.Info("Delete service accounts...")
	err = m.ServiceAccountRemover.RemoveAll(ctx, doguResource.Namespace, dogu)
	if err != nil {
		logger.Error(err, "failed to remove service accounts")
	}

	logger.Info("Unregister dogu...")
	err = m.DoguRegistrator.UnregisterDogu(doguResource.Name)
	if err != nil {
		logger.Error(err, "failed to unregister dogu")
	}

	logger.Info("Remove finalizer...")
	controllerutil.RemoveFinalizer(doguResource, finalizerName)
	err = m.Client.Update(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to update dogu: %w", err)
	}
	logger.Info(fmt.Sprintf("Dogu %s/%s has been : %s", doguResource.Namespace, doguResource.Name, controllerutil.OperationResultUpdated))

	return nil
}

// TODO: Implement Upgrade
func (m *DoguManager) Upgrade(_ context.Context, _ *k8sv1.Dogu) error {
	return nil
}
