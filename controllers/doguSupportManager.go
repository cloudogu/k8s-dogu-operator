package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// podTemplateResourceGenerator is used to generate pod templates.
type podTemplateResourceGenerator interface {
	GetPodTemplate(*k8sv1.Dogu, *core.Dogu, bool) *corev1.PodTemplateSpec
}

// doguSupportManager is used to handle the support mode for dogus.
type doguSupportManager struct {
	client            client.Client
	scheme            *runtime.Scheme
	doguRegistry      registry.DoguRegistry
	resourceGenerator podTemplateResourceGenerator
	eventRecorder     record.EventRecorder
}

// NewDoguSupportManager creates a new instance of doguSupportManager.
func NewDoguSupportManager(client client.Client, cesRegistry registry.Registry, eventRecorder record.EventRecorder) *doguSupportManager {
	resourceGenerator := resource.NewResourceGenerator(client.Scheme(), limit.NewDoguDeploymentLimitPatcher(cesRegistry))
	return &doguSupportManager{
		client:            client,
		doguRegistry:      cesRegistry.DoguRegistry(),
		resourceGenerator: resourceGenerator,
		eventRecorder:     eventRecorder,
	}
}

// HandleSupportMode handles the support flag in the dogu spec and returns whether the support modes changed during the
// last operation. If any action failed a non-requeue-able error will be returned.
func (dsm *doguSupportManager) HandleSupportMode(ctx context.Context, doguResource *k8sv1.Dogu) (bool, error) {
	logger := log.FromContext(ctx)

	deployment := &appsv1.Deployment{}
	err := dsm.client.Get(ctx, doguResource.GetObjectKey(), deployment)
	if err != nil {
		return false, fmt.Errorf("failed to get deployment of dogu %s: %w", doguResource.Name, err)
	}

	logger.Info(fmt.Sprintf("Check if support mode is currently active for dogu %s...", doguResource.Name))
	active := isDeploymentInSupportMode(deployment)
	if !supportModeChanged(doguResource, active) {
		dsm.eventRecorder.Event(doguResource, corev1.EventTypeNormal, SupportEventReason, "Support flag did not change. Do nothing.")
		return false, nil
	}

	logger.Info(fmt.Sprintf("Update deployment for dogu %s...", doguResource.Name))
	err = dsm.updateDeployment(ctx, doguResource, deployment)
	if err != nil {
		return false, err
	}
	dsm.eventRecorder.Eventf(doguResource, corev1.EventTypeNormal, SupportEventReason, "Support flag changed to %t. Deployment updated.", doguResource.Spec.SupportMode)

	return true, nil
}

func (dsm *doguSupportManager) updateDeployment(ctx context.Context, doguResource *k8sv1.Dogu, deployment *appsv1.Deployment) error {
	dogu, err := dsm.doguRegistry.Get(doguResource.Name)
	if err != nil {
		return fmt.Errorf("failed to get dogu descriptor of dogu %s: %w", doguResource.Name, err)
	}

	deployment.Spec.Template = *dsm.resourceGenerator.GetPodTemplate(doguResource, dogu, doguResource.Spec.SupportMode)
	err = dsm.client.Update(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to update dogu deployment %s: %w", doguResource.Name, err)
	}

	return nil
}

func isDeploymentInSupportMode(deployment *appsv1.Deployment) bool {
	for _, container := range deployment.Spec.Template.Spec.Containers {
		envVars := container.Env
		for _, env := range envVars {
			if env.Name == resource.SupportModeEnvVar && env.Value == "true" {
				return true
			}
		}
	}

	return false
}

func supportModeChanged(doguResource *k8sv1.Dogu, active bool) bool {
	mode := doguResource.Spec.SupportMode
	if mode && active || !mode && !active {
		return false
	}

	return true
}
