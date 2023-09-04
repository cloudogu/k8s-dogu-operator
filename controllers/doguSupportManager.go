package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	cesremote "github.com/cloudogu/cesapp-lib/remote"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/config"
)

const SupportModeEnvVar = "SUPPORT_MODE"

// podTemplateResourceGenerator is used to generate pod templates.
type podTemplateResourceGenerator interface {
	GetPodTemplate(doguResource *k8sv1.Dogu, dogu *core.Dogu, chownInitImage string) (*corev1.PodTemplateSpec, error)
}

// doguSupportManager is used to handle the support mode for dogus.
type doguSupportManager struct {
	client                       client.Client
	scheme                       *runtime.Scheme
	doguRegistry                 registry.DoguRegistry
	podTemplateResourceGenerator podTemplateResourceGenerator
	eventRecorder                record.EventRecorder
}

// NewDoguSupportManager creates a new instance of doguSupportManager.
func NewDoguSupportManager(client client.Client, operatorConfig *config.OperatorConfig, cesRegistry registry.Registry, eventRecorder record.EventRecorder) (*doguSupportManager, error) {
	doguRemoteRegistry, err := cesremote.New(operatorConfig.GetRemoteConfiguration(), operatorConfig.GetRemoteCredentials())
	if err != nil {
		return nil, fmt.Errorf("failed to create new remote dogu registry: %w", err)
	}
	_, _, _, _, _, _, _, _, resourceGenerator, err := initManagerObjects(client, operatorConfig, cesRegistry, doguRemoteRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dogu support manager objects: %w", err)
	}

	return &doguSupportManager{
		client:                       client,
		doguRegistry:                 cesRegistry.DoguRegistry(),
		podTemplateResourceGenerator: resourceGenerator,
		eventRecorder:                eventRecorder,
	}, nil
}

// HandleSupportMode handles the support flag in the dogu spec and returns whether the support modes changed during the
// last operation. If any action failed a non-requeue-able error will be returned.
func (dsm *doguSupportManager) HandleSupportMode(ctx context.Context, doguResource *k8sv1.Dogu) (bool, error) {
	logger := log.FromContext(ctx)

	deployment := &appsv1.Deployment{}
	err := dsm.client.Get(ctx, doguResource.GetObjectKey(), deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			dsm.eventRecorder.Eventf(doguResource, corev1.EventTypeWarning, SupportEventReason, "No deployment found for dogu %s when checking support handler", doguResource.Name)
			return false, nil
		}
		return false, fmt.Errorf("failed to get deployment of dogu %s: %w", doguResource.Name, err)
	}

	logger.Info(fmt.Sprintf("Check if support mode is currently active for dogu %s...", doguResource.Name))
	active := isDeploymentInSupportMode(deployment)
	if !supportModeChanged(doguResource, active) {
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

	podTemplate, err := dsm.podTemplateResourceGenerator.GetPodTemplate(doguResource, dogu, "")
	if err != nil {
		return err
	}
	if doguResource.Spec.SupportMode {
		setDoguPodTemplateInSupportMode(doguResource, podTemplate)
	}

	deployment.Spec.Template = *podTemplate
	err = dsm.client.Update(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to update dogu deployment %s: %w", doguResource.Name, err)
	}

	return nil
}

func setDoguPodTemplateInSupportMode(doguResource *k8sv1.Dogu, template *corev1.PodTemplateSpec) *corev1.PodTemplateSpec {
	for index := range template.Spec.Containers {
		container := &template.Spec.Containers[index]
		if container.Name == doguResource.Name {
			container.LivenessProbe = nil
			container.ReadinessProbe = nil
			container.StartupProbe = nil
			container.Command = []string{"/bin/bash", "-c", "--"}
			container.Args = []string{"while true; do sleep 5; done;"}
			container.Env = append(container.Env, corev1.EnvVar{Name: SupportModeEnvVar, Value: "true"})
		}
	}

	return template
}

func isDeploymentInSupportMode(deployment *appsv1.Deployment) bool {
	for _, container := range deployment.Spec.Template.Spec.Containers {
		envVars := container.Env
		for _, env := range envVars {
			if env.Name == SupportModeEnvVar && env.Value == "true" {
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
