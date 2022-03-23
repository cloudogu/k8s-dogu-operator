package controllers

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DoguManager is a central unit in the process of handling dogu custom resources
// The DoguManager creates, updates and deletes dogus
type DoguManager struct {
	client.Client
	Scheme            *runtime.Scheme
	resourceGenerator DoguResourceGenerator
	doguRegistry      DoguRegistry
	imageRegistry     ImageRegistry
	doguRegistrator   DoguRegistrator
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
	GetDoguExposedService(doguResource *k8sv1.Dogu, dogu *core.Dogu) (*corev1.Service, error)
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

// NewDoguManager creates a new instance of DoguManager
func NewDoguManager(client client.Client, scheme *runtime.Scheme, resourceGenerator DoguResourceGenerator,
	doguRegistry DoguRegistry, imageRegistry ImageRegistry, doguRegistrator DoguRegistrator) *DoguManager {
	return &DoguManager{
		Client:            client,
		Scheme:            scheme,
		resourceGenerator: resourceGenerator,
		doguRegistry:      doguRegistry,
		imageRegistry:     imageRegistry,
		doguRegistrator:   doguRegistrator,
	}
}

// Install installs a given Dogu Resource. This includes fetching the dogu.json and the container image. With the
// information Install creates a Deployment and a Service
func (m DoguManager) Install(ctx context.Context, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)

	logger.Info("Fetching dogu...")
	dogu, err := m.doguRegistry.GetDogu(doguResource)
	if err != nil {
		return fmt.Errorf("failed to get dogu: %w", err)
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

	logger.Info("Add dogu finalizer...")
	controllerutil.AddFinalizer(doguResource, finalizerName)
	err = m.Client.Update(ctx, doguResource)
	if err != nil {
		logger.Info(fmt.Sprintf("Dogu %s/%s has been : %s", doguResource.Namespace, doguResource.Name, controllerutil.OperationResultUpdated))
		return fmt.Errorf("failed to update dogu: %w", err)
	}

	return nil
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

	return nil
}

func (m DoguManager) createDeployment(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	desiredDeployment, err := m.resourceGenerator.GetDoguDeployment(doguResource, dogu)
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

func (m DoguManager) createExposedServices(ctx context.Context, doguResource *k8sv1.Dogu, dogu *core.Dogu) error {
	logger := log.FromContext(ctx)

	if len(dogu.ExposedPorts) < 1 {
		return nil
	}

	exposedService, err := m.resourceGenerator.GetDoguExposedService(doguResource, dogu)
	if errors.Is(err, ErrorExposedServiceNoPorts) {
		logger.Info("Skipping the creation of an exposed service as dogu [%s] does not contain any exposed ports", dogu.Name)
	} else if err != nil {
		return fmt.Errorf("failed to generate exposed service: %w", err)
	}

	err = m.Client.Create(ctx, exposedService)
	if err != nil {
		return fmt.Errorf("failed to create exposed service: %w", err)
	}

	logger.Info(fmt.Sprintf("Exposed Service %s/%s have been : %s", exposedService.Namespace, exposedService.Name, controllerutil.OperationResultCreated))

	return nil
}

func (m DoguManager) Delete(ctx context.Context, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)

	logger.Info("Unregister dogu...")
	err := m.doguRegistrator.UnregisterDogu(doguResource.Name)
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
