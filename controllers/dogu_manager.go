package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	"github.com/cloudogu/cesapp/v4/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/dependencies"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const finalizerName = "dogu-finalizer"

// DoguManager is a central unit in the process of handling dogu custom resources
// The DoguManager creates, updates and deletes dogus
type DoguManager struct {
	client.Client
	Scheme              *runtime.Scheme
	resourceGenerator   DoguResourceGenerator
	doguRegistry        DoguRegistry
	imageRegistry       ImageRegistry
	doguRegistrator     DoguRegistrator
	dependencyValidator DependencyValidator
}

// DoguRegistry is used to fetch the dogu descriptor
type DoguRegistry interface {
	GetDogu(*k8sv1.Dogu) (*core.Dogu, error)
}

// DoguResourceGenerator is used to generate kubernetes resources
type DoguResourceGenerator interface {
	GetDoguDeployment(doguResource *k8sv1.Dogu, dogu *core.Dogu) (*appsv1.Deployment, error)
	GetDoguService(doguResource *k8sv1.Dogu, imageConfig *imagev1.ConfigFile) (*corev1.Service, error)
	GetDoguPVC(doguResource *k8sv1.Dogu) (*corev1.PersistentVolumeClaim, error)
	GetDoguSecret(doguResource *k8sv1.Dogu, stringData map[string]string) (*corev1.Secret, error)
}

// ImageRegistry is used to pull container images
type ImageRegistry interface {
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}

// DoguRegistrator is used to register dogus
type DoguRegistrator interface {
	RegisterDogu(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error
	UnregisterDogu(dogu string) error
}

// DependencyValidator is used to check if dogu dependencies are installed
type DependencyValidator interface {
	ValidateDependencies(dogu *core.Dogu) error
}

// NewDoguManager creates a new instance of DoguManager
func NewDoguManager(version *core.Version, client client.Client, scheme *runtime.Scheme, resourceGenerator DoguResourceGenerator,
	doguRegistry DoguRegistry, imageRegistry ImageRegistry, doguRegistrator DoguRegistrator, registry registry.DoguRegistry) *DoguManager {
	dependencyValidator := dependencies.NewDependencyChecker(version, registry)

	return &DoguManager{
		Client:              client,
		Scheme:              scheme,
		resourceGenerator:   resourceGenerator,
		doguRegistry:        doguRegistry,
		imageRegistry:       imageRegistry,
		doguRegistrator:     doguRegistrator,
		dependencyValidator: dependencyValidator,
	}
}

// Install installs a given Dogu Resource. This includes fetching the dogu.json and the container image. With the
// information Install creates a Deployment and a Service
func (m DoguManager) Install(ctx context.Context, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)
	doguResource.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalling, StatusMessages: []string{}}
	err := m.Client.Status().Update(ctx, doguResource)
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
	dogu, err := m.getDoguDescriptor(ctx, doguResource, doguConfigMap)
	if err != nil {
		return fmt.Errorf("failed to get dogu: %w", err)
	}

	logger.Info("Check dogu dependencies...")
	err = m.dependencyValidator.ValidateDependencies(dogu)
	if err != nil {
		// No wrap needed because err is a multierror
		return err
	}

	logger.Info("Register dogu...")
	err = m.doguRegistrator.RegisterDogu(ctx, doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to register dogu: %w", err)
	}

	logger.Info("Pull image config...")
	imageConfig, err := m.imageRegistry.PullImageConfig(ctx, dogu.Image+":"+dogu.Version)
	if err != nil {
		return fmt.Errorf("failed to pull image config: %w", err)
	}

	logger.Info("Create dogu resources...")
	err = m.createDoguResources(ctx, doguResource, dogu, imageConfig)
	if err != nil {
		return fmt.Errorf("failed to create dogu resources: %w", err)
	}

	doguResource.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusInstalled, StatusMessages: []string{}}
	err = m.Client.Status().Update(ctx, doguResource)
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
	dogu, err := m.doguRegistry.GetDogu(doguResource)
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

func (m DoguManager) getDoguDescriptor(ctx context.Context, doguResource *k8sv1.Dogu, doguConfigMap *corev1.ConfigMap) (*core.Dogu, error) {
	logger := log.FromContext(ctx)

	if doguConfigMap != nil {
		logger.Info("Fetching dogu from custom configmap...")
		return m.getDoguDescriptorFromConfigMap(doguConfigMap)
	} else {
		logger.Info("Fetching dogu from dogu registry...")
		return m.getDoguDescriptorFromRegistry(doguResource)
	}
}

func (m DoguManager) createDoguResources(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu, imageConfig *imagev1.ConfigFile) error {
	logger := log.FromContext(ctx)

	if len(dogu.Volumes) > 0 {
		desiredPvc, err := m.resourceGenerator.GetDoguPVC(doguResource)
		if err != nil {
			return fmt.Errorf("failed to generate pvc: %w", err)
		}
		err = m.Client.Create(ctx, desiredPvc)
		if err != nil {
			return fmt.Errorf("failed to create pvc: %w", err)
		}
		logger.Info(fmt.Sprintf("PersistentVolumeClaim %s/%s has been : %s", desiredPvc.Namespace, desiredPvc.Name, controllerutil.OperationResultCreated))
	}

	desiredDeployment, err := m.resourceGenerator.GetDoguDeployment(doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to generate dogu deployment: %w", err)
	}

	err = m.Client.Create(ctx, desiredDeployment)
	if err != nil {
		return fmt.Errorf("failed to create dogu deployment: %w", err)
	}
	logger.Info(fmt.Sprintf("Deployment %s/%s has been : %s", desiredDeployment.Namespace, desiredDeployment.Name, controllerutil.OperationResultCreated))

	desiredService, err := m.resourceGenerator.GetDoguService(doguResource, imageConfig)
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

func (m DoguManager) Delete(ctx context.Context, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)
	doguResource.Status = k8sv1.DoguStatus{Status: k8sv1.DoguStatusDeleting, StatusMessages: []string{}}
	err := m.Client.Status().Update(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to update dogu status: %w", err)
	}

	logger.Info("Unregister dogu...")
	err = m.doguRegistrator.UnregisterDogu(doguResource.Name)
	if err != nil {
		return fmt.Errorf("failed to unregister dogu: %w", err)
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
