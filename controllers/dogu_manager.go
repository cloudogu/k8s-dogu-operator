package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	cesregistry "github.com/cloudogu/cesapp/v4/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/controllers/registry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const finalizerName = "dogu-finalizer"

// NewManager is an alias mainly used for testing the main package
var NewManager = NewDoguManager

// DoguManager is a central unit in the process of handling dogu custom resources
// The DoguManager creates, updates and deletes dogus
type DoguManager struct {
	client.Client
	Scheme                *runtime.Scheme
	ResourceGenerator     doguResourceGenerator
	DoguRegistry          doguRegistry
	ImageRegistry         imageRegistry
	DoguRegistrator       doguRegistrator
	DependencyValidator   dependencyValidator
	ServiceAccountCreator serviceAccountCreator
	ServiceAccountRemover serviceAccountRemover
}

// doguRegistry is used to fetch the dogu descriptor
type doguRegistry interface {
	GetDogu(*k8sv1.Dogu) (*core.Dogu, error)
}

// doguResourceGenerator is used to generate kubernetes resources
type doguResourceGenerator interface {
	GetDoguDeployment(doguResource *k8sv1.Dogu, dogu *core.Dogu) (*appsv1.Deployment, error)
	GetDoguService(doguResource *k8sv1.Dogu, imageConfig *imagev1.ConfigFile) (*corev1.Service, error)
	GetDoguPVC(doguResource *k8sv1.Dogu) (*corev1.PersistentVolumeClaim, error)
	GetDoguSecret(doguResource *k8sv1.Dogu, stringData map[string]string) (*corev1.Secret, error)
	GetDoguExposedServices(doguResource *k8sv1.Dogu, dogu *core.Dogu) ([]corev1.Service, error)
}

// imageRegistry is used to pull container images
type imageRegistry interface {
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}

// doguRegistrator is used to register dogus
type doguRegistrator interface {
	RegisterDogu(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error
	UnregisterDogu(dogu string) error
}

// dependencyValidator is used to check if dogu dependencies are installed
type dependencyValidator interface {
	ValidateDependencies(dogu *core.Dogu) error
}

// serviceAccountCreator is used to create service accounts for a given dogu
type serviceAccountCreator interface {
	CreateAll(ctx context.Context, namespace string, dogu *core.Dogu) error
}

// serviceAccountRemover is used to remove service accounts for a given dogu
type serviceAccountRemover interface {
	RemoveAll(ctx context.Context, namespace string, dogu *core.Dogu) error
}

// DoguSecretsHandler is used to write the encrypted secrets from the setup to the dogu config
type DoguSecretsHandler interface {
	WriteDoguSecretsToRegistry(ctx context.Context, doguResource *k8sv1.Dogu) error
}

// NewDoguManager creates a new instance of DoguManager
func NewDoguManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesregistry.Registry) (*DoguManager, error) {
	doguRegistry := registry.New(operatorConfig.DoguRegistry.Username, operatorConfig.DoguRegistry.Password, operatorConfig.DoguRegistry.Endpoint)
	imageRegistry := registry.NewCraneContainerImageRegistry(operatorConfig.DockerRegistry.Username, operatorConfig.DockerRegistry.Password)
	resourceGenerator := resource.NewResourceGenerator(client.Scheme())

	err := validateKeyProvider(cesRegistry.GlobalConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to validate key provider: %w", err)
	}

	doguRegistrator := NewCESDoguRegistrator(client, cesRegistry, resourceGenerator)
	dependencyValidator := dependency.NewCompositeDependencyValidator(operatorConfig.Version, cesRegistry.DoguRegistry())

	clientSet, err := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		return nil, fmt.Errorf("failed to create clientSet: %w", err)
	}

	executor := resource.NewCommandExecutor(clientSet, clientSet.CoreV1().RESTClient())
	serviceAccountCreator := serviceaccount.NewCreator(cesRegistry, executor)
	serviceAccountRemover := serviceaccount.NewRemover(cesRegistry, executor)

	doguSecretsHandler := resource.NewDoguSecretsWriter(client, cesRegistry)

	return &DoguManager{
		Client:                client,
		Scheme:                client.Scheme(),
		ResourceGenerator:     resourceGenerator,
		DoguRegistry:          doguRegistry,
		ImageRegistry:         imageRegistry,
		DoguRegistrator:       doguRegistrator,
		DependencyValidator:   dependencyValidator,
		ServiceAccountCreator: serviceAccountCreator,
		DoguSecretHandler:     doguSecretsHandler,
		ServiceAccountRemover: serviceAccountRemover,
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
func (m DoguManager) Install(ctx context.Context, doguResource *k8sv1.Dogu) error {
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

	logger.Info("Create dogu resources...")
	err = m.createDoguResources(ctx, doguResource, dogu, imageConfig)
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

func (m DoguManager) getDoguDescriptorFromConfigMap(doguConfigMap *corev1.ConfigMap) (*core.Dogu, error) {
	jsonStr := doguConfigMap.Data["dogu.json"]
	dogu := &core.Dogu{}
	err := json.Unmarshal([]byte(jsonStr), dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarschal custom dogu descriptor: %w", err)
	}

	return dogu, nil
}

func (m DoguManager) getDoguDescriptorFromRegistry(doguResource *k8sv1.Dogu) (*core.Dogu, error) {
	dogu, err := m.DoguRegistry.GetDogu(doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu from dogu registry: %w", err)
	}

	return dogu, nil
}

func (m DoguManager) getDoguConfigMap(ctx context.Context, doguResource *k8sv1.Dogu) (*corev1.ConfigMap, error) {
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

func (m DoguManager) getDoguDescriptorWithConfigMap(ctx context.Context, doguResource *k8sv1.Dogu, doguConfigMap *corev1.ConfigMap) (*core.Dogu, error) {
	logger := log.FromContext(ctx)

	if doguConfigMap != nil {
		logger.Info("Fetching dogu from custom configmap...")
		return m.getDoguDescriptorFromConfigMap(doguConfigMap)
	} else {
		logger.Info("Fetching dogu from dogu registry...")
		return m.getDoguDescriptorFromRegistry(doguResource)
	}
}

func (m DoguManager) getDoguDescriptor(ctx context.Context, doguResource *k8sv1.Dogu) (*core.Dogu, error) {
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

func (m DoguManager) createDoguResources(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu, imageConfig *imagev1.ConfigFile) error {
	err := m.createVolumes(ctx, doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to create volumes for dogu %s: %w", dogu.Name, err)
	}

	err = m.createDeployment(ctx, doguResource, dogu)
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

func (m DoguManager) createVolumes(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
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

func (m DoguManager) createDeployment(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	desiredDeployment, err := m.ResourceGenerator.GetDoguDeployment(doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to generate dogu deployment: %w", err)
	}

	err = m.Client.Create(ctx, desiredDeployment)
	if err != nil {
		return fmt.Errorf("failed to create dogu deployment: %w", err)
	}

	logger.Info(fmt.Sprintf("Deployment %s/%s has been : %s", desiredDeployment.Namespace, desiredDeployment.Name, controllerutil.OperationResultCreated))
	return nil
}

func (m DoguManager) createService(ctx context.Context, doguResource *k8sv1.Dogu, imageConfig *imagev1.ConfigFile) error {
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

func (m DoguManager) createExposedServices(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
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

func (m DoguManager) Delete(ctx context.Context, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)
	doguResource.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusDeleting, StatusMessages: []string{}}
	err := doguResource.Update(ctx, m.Client)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	logger.Info("Fetching dogu...")
	dogu, err := m.getDoguDescriptor(ctx, doguResource)
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
func (m DoguManager) Upgrade(_ context.Context, _ *k8sv1.Dogu) error {
	return nil
}
