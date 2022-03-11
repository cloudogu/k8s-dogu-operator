package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp/v4/core"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
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
}

// DoguRegistry is used to fetch the dogu descriptor
type DoguRegistry interface {
	GetDogu(*k8sv1.Dogu) (*core.Dogu, error)
}

// DoguResourceGenerator is used to generate kubernetes resources
type DoguResourceGenerator interface {
	GetDoguDeployment(doguResource *k8sv1.Dogu, dogu *core.Dogu) *appsv1.Deployment
	GetDoguService(doguResource *k8sv1.Dogu, imageConfig *imagev1.ConfigFile) (*corev1.Service, error)
}

// ImageRegistry is used to pull container images
type ImageRegistry interface {
	PullImageConfig(ctx context.Context, image string) (*imagev1.ConfigFile, error)
}

// NewDoguManager creates a new instance of DoguManager
func NewDoguManager(client client.Client, scheme *runtime.Scheme, resourceGenerator DoguResourceGenerator,
	doguRegistry DoguRegistry, imageRegistry ImageRegistry) *DoguManager {
	return &DoguManager{
		Client:            client,
		Scheme:            scheme,
		resourceGenerator: resourceGenerator,
		doguRegistry:      doguRegistry,
		imageRegistry:     imageRegistry,
	}
}

// Install installs a given Dogu Resource. This includes fetching the dogu.json and the container image. With the
// information Install creates a Deployment and a Service
func (m DoguManager) Install(ctx context.Context, doguResource *k8sv1.Dogu) error {
	logger := log.FromContext(ctx)

	dogu, err := m.doguRegistry.GetDogu(doguResource)
	if err != nil {
		return fmt.Errorf("failed to get dogu: %w", err)
	}

	desiredDeployment := m.resourceGenerator.GetDoguDeployment(doguResource, dogu)
	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: desiredDeployment.Name, Namespace: desiredDeployment.Namespace}}

	result, err := ctrl.CreateOrUpdate(ctx, m.Client, deployment, func() error {
		deployment.Spec = desiredDeployment.Spec
		deployment.ObjectMeta.Labels = desiredDeployment.Labels
		return ctrl.SetControllerReference(doguResource, deployment, m.Scheme)
	})

	if err != nil {
		return fmt.Errorf("failed to create dogu deployment: %w", err)
	}
	logger.Info(fmt.Sprintf("createOrUpdate deployment result: %+v", result))

	imageConfig, err := m.imageRegistry.PullImageConfig(ctx, dogu.Image+":"+dogu.Version)
	if err != nil {
		return fmt.Errorf("failed to create image config: %w", err)
	}

	desiredService, err := m.resourceGenerator.GetDoguService(doguResource, imageConfig)
	if err != nil {
		return fmt.Errorf("failed to create dogu service: %w", err)
	}

	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: desiredService.Name, Namespace: desiredService.Namespace}}
	result, err = ctrl.CreateOrUpdate(ctx, m.Client, service, func() error {
		service.ObjectMeta.Labels = desiredService.Labels
		service.Spec = desiredService.Spec
		return ctrl.SetControllerReference(doguResource, service, m.Scheme)
	})

	controllerutil.AddFinalizer(doguResource, finalizerName)
	err = m.Client.Update(ctx, doguResource)
	if err != nil {
		logger.Info(fmt.Sprintf("update doguResource: %+v", doguResource))
		return fmt.Errorf("failed to update dogu: %w", err)
	}

	if err != nil {
		return fmt.Errorf("failed to create dogu service: %w", err)
	}
	logger.Info(fmt.Sprintf("createOrUpdate service result: %+v", result))

	return nil
}

// Update TODO
func (m DoguManager) Upgrade(_ context.Context, _ *k8sv1.Dogu) error {
	return nil
}

// Delete TODO
func (m DoguManager) Delete(_ context.Context, _ *k8sv1.Dogu) error {
	return nil
}
