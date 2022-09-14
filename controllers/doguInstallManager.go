package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	cesregistry "github.com/cloudogu/cesapp-lib/registry"
	cesremote "github.com/cloudogu/cesapp-lib/remote"
	"github.com/cloudogu/k8s-apply-lib/apply"
	reg "github.com/cloudogu/k8s-dogu-operator/controllers/cesregistry"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependency"
	"github.com/cloudogu/k8s-dogu-operator/controllers/imageregistry"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/controllers/serviceaccount"

	"github.com/go-logr/logr"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const k8sDoguOperatorFieldManagerName = "k8s-dogu-operator"

// doguInstallManager is a central unit in the process of handling the installation process of a custom dogu resource.
type doguInstallManager struct {
	client                client.Client
	recorder              record.EventRecorder
	doguRemoteRegistry    cesremote.Registry
	doguLocalRegistry     cesregistry.DoguRegistry
	imageRegistry         imageRegistry
	doguRegistrator       doguRegistrator
	dependencyValidator   dependencyValidator
	serviceAccountCreator serviceAccountCreator
	doguSecretHandler     doguSecretHandler
	fileExtractor         fileExtractor
	collectApplier        collectApplier
	resourceUpserter      resourceUpserter
}

// NewDoguInstallManager creates a new instance of doguInstallManager.
func NewDoguInstallManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry cesregistry.Registry, eventRecorder record.EventRecorder) (*doguInstallManager, error) {
	doguRemoteRegistry, err := cesremote.New(operatorConfig.GetRemoteConfiguration(), operatorConfig.GetRemoteCredentials())
	if err != nil {
		return nil, fmt.Errorf("failed to create new remote dogu registry: %w", err)
	}

	restConfig := ctrl.GetConfigOrDie()
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to find cluster config: %w", err)
	}

	imageRegistry := imageregistry.NewCraneContainerImageRegistry(operatorConfig.DockerRegistry.Username, operatorConfig.DockerRegistry.Password)
	limitPatcher := limit.NewDoguDeploymentLimitPatcher(cesRegistry)
	resourceGenerator := resource.NewResourceGenerator(client.Scheme(), limit.NewDoguDeploymentLimitPatcher(cesRegistry))
	upserter := resource.NewUpserter(client, limitPatcher)

	fileExtract := newPodFileExtractor(client, restConfig, clientSet)
	applier, scheme, err := apply.New(restConfig, k8sDoguOperatorFieldManagerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create K8s applier: %w", err)
	}
	err = k8sv1.AddToScheme(scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to add applier scheme to dogu CRD scheme handling: %w", err)
	}

	doguRegistrator := reg.NewCESDoguRegistrator(client, cesRegistry, resourceGenerator)
	dependencyValidator := dependency.NewCompositeDependencyValidator(operatorConfig.Version, cesRegistry.DoguRegistry())

	executor := resource.NewCommandExecutor(clientSet, clientSet.CoreV1().RESTClient())
	serviceAccountCreator := serviceaccount.NewCreator(cesRegistry, executor)
	collectApplier := resource.NewCollectApplier(applier)

	return &doguInstallManager{
		client:                client,
		recorder:              eventRecorder,
		doguRemoteRegistry:    doguRemoteRegistry,
		doguLocalRegistry:     cesRegistry.DoguRegistry(),
		imageRegistry:         imageRegistry,
		doguRegistrator:       doguRegistrator,
		dependencyValidator:   dependencyValidator,
		serviceAccountCreator: serviceAccountCreator,
		doguSecretHandler:     resource.NewDoguSecretsWriter(client, cesRegistry),
		fileExtractor:         fileExtract,
		collectApplier:        collectApplier,
		resourceUpserter:      upserter,
	}, nil
}

// Install installs a given Dogu Resource. This includes fetching the dogu.json and the container image. With the
// information Install creates a Deployment and a Service
func (m *doguInstallManager) Install(ctx context.Context, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)

	doguResource.Status = k8sv1.DoguStatus{RequeueTime: doguResource.Status.RequeueTime, Status: k8sv1.DoguStatusInstalling, StatusMessages: []string{}}
	err := doguResource.Update(ctx, m.client)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	// Set the finalizer at the beginning of the install procedure.
	// This is required because an error during installation would leave a dogu resource with its
	// k8s resources in the cluster. A delete would tidy up those resources but would not start the
	// delete procedure from the controller.
	logger.Info("Add dogu finalizer...")
	controllerutil.AddFinalizer(doguResource, finalizerName)
	err = m.client.Update(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to update dogu: %w", err)
	}

	// we need to retrieve the config map with the custom descriptor to delete it after ending the installation
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
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Checking dependencies...")
	err = m.dependencyValidator.ValidateDependencies(dogu)
	if err != nil {
		return err
	}

	logger.Info("Register dogu...")
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Registering in the local dogu registry...")
	err = m.doguRegistrator.RegisterNewDogu(ctx, doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to register dogu: %w", err)
	}

	logger.Info("Write dogu secrets from setup...")
	err = m.doguSecretHandler.WriteDoguSecretsToRegistry(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to write dogu secrets from setup: %w", err)
	}

	logger.Info("Create service accounts...")
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Creating required service accounts...")
	err = m.serviceAccountCreator.CreateAll(ctx, doguResource.Namespace, dogu)
	if err != nil {
		return fmt.Errorf("failed to create service accounts: %w", err)
	}

	logger.Info("Pull image config...")
	m.recorder.Eventf(doguResource, corev1.EventTypeNormal, InstallEventReason, "Pulling dogu image %s...", dogu.Image+":"+dogu.Version)
	imageConfig, err := m.imageRegistry.PullImageConfig(ctx, dogu.Image+":"+dogu.Version)
	if err != nil {
		return fmt.Errorf("failed to pull image config: %w", err)
	}

	customK8sResources, err := m.fileExtractor.ExtractK8sResourcesFromContainer(ctx, doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to pull customK8sResources: %w", err)
	}

	customDeployment, err := m.applyCustomK8sResources(logger, customK8sResources, doguResource)
	if err != nil {
		return err
	}

	logger.Info("Create dogu resources...")
	m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Creating kubernetes resources...")
	err = m.createDoguResources(ctx, doguResource, dogu, imageConfig, customDeployment)
	if err != nil {
		return fmt.Errorf("failed to create dogu resources: %w", err)
	}

	doguResource.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalled, StatusMessages: []string{}}
	err = doguResource.Update(ctx, m.client)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	err = deleteDoguConfigMap(ctx, m.client, doguConfigMap)
	if err != nil {
		return err
	}

	return nil
}

func (m *doguInstallManager) applyCustomK8sResources(logger logr.Logger, customK8sResources map[string]string, doguResource *k8sv1.Dogu) (*appsv1.Deployment, error) {
	return m.collectApplier.CollectApply(logger, customK8sResources, doguResource)
}

func (m *doguInstallManager) getDoguDescriptorFromConfigMap(doguConfigMap *corev1.ConfigMap) (*cesappcore.Dogu, error) {
	jsonStr := doguConfigMap.Data["dogu.json"]
	dogu := &cesappcore.Dogu{}
	err := json.Unmarshal([]byte(jsonStr), dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal custom dogu descriptor: %w", err)
	}

	return dogu, nil
}

func (m *doguInstallManager) getDoguDescriptorFromRemoteRegistry(doguResource *k8sv1.Dogu) (*cesappcore.Dogu, error) {
	ctrl.Log.Info(doguResource.Spec.Name)
	dogu, err := m.doguRemoteRegistry.Get(doguResource.Spec.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu from remote dogu registry: %w", err)
	}

	return dogu, nil
}

func (m *doguInstallManager) getDoguConfigMap(ctx context.Context, doguResource *k8sv1.Dogu) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{}
	err := m.client.Get(ctx, doguResource.GetDescriptorObjectKey(), configMap)
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

func (m *doguInstallManager) getDoguDescriptorWithConfigMap(ctx context.Context, doguResource *k8sv1.Dogu, doguConfigMap *corev1.ConfigMap) (*cesappcore.Dogu, error) {
	logger := log.FromContext(ctx)

	if doguConfigMap != nil {
		logger.Info("Fetching dogu from custom configmap...")
		m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor using custom configmap...")
		return m.getDoguDescriptorFromConfigMap(doguConfigMap)
	} else {
		logger.Info("Fetching dogu from dogu registry...")
		m.recorder.Event(doguResource, corev1.EventTypeNormal, InstallEventReason, "Fetching dogu descriptor from dogu registry...")
		return m.getDoguDescriptorFromRemoteRegistry(doguResource)
	}
}

func (m *doguInstallManager) getDoguDescriptor(ctx context.Context, doguResource *k8sv1.Dogu) (*cesappcore.Dogu, error) {
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

func (m *doguInstallManager) createDoguResources(ctx context.Context, doguResource *k8sv1.Dogu, dogu *cesappcore.Dogu, imageConfig *imagev1.ConfigFile, patchingDeployment *appsv1.Deployment) error {
	err := m.resourceUpserter.ApplyDoguResource(ctx, doguResource, dogu, imageConfig, patchingDeployment)
	if err != nil {
		return fmt.Errorf("failed to create resource(s) for dogu %s: %w", dogu.Name, err)
	}

	return nil
}
