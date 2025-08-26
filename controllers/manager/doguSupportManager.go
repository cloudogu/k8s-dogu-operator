package manager

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

const (
	SupportEventReason = "Support"
)
const SupportModeEnvVar = "SUPPORT_MODE"

// podTemplateResourceGenerator is used to generate pod templates.
type podTemplateResourceGenerator interface {
	GetPodTemplate(ctx context.Context, doguResource *doguv2.Dogu, dogu *core.Dogu) (*corev1.PodTemplateSpec, error)
}

// doguSupportManager is used to handle the support mode for dogus.
type doguSupportManager struct {
	client                       client.Client
	doguFetcher                  localDoguFetcher
	podTemplateResourceGenerator podTemplateResourceGenerator
	eventRecorder                record.EventRecorder
}

// NewDoguSupportManager creates a new instance of doguSupportManager.
func NewDoguSupportManager(client client.Client, mgrSet *util.ManagerSet, eventRecorder record.EventRecorder) *doguSupportManager {
	return &doguSupportManager{
		client:                       client,
		doguFetcher:                  mgrSet.LocalDoguFetcher,
		podTemplateResourceGenerator: mgrSet.DoguResourceGenerator,
		eventRecorder:                eventRecorder,
	}
}

// HandleSupportMode handles the support flag in the dogu spec and returns whether the support modes changed during the
// last operation. If any action failed a non-requeue-able error will be returned.
func (dsm *doguSupportManager) HandleSupportMode(ctx context.Context, doguResource *doguv2.Dogu) (bool, error) {
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

	err = dsm.updateDeployment(ctx, doguResource, deployment)
	if err != nil {
		return false, err
	}
	dsm.eventRecorder.Eventf(doguResource, corev1.EventTypeNormal, SupportEventReason, "Support flag changed to %t. Deployment updated.", doguResource.Spec.SupportMode)

	return true, nil
}

func (dsm *doguSupportManager) updateDeployment(ctx context.Context, doguResource *doguv2.Dogu, deployment *appsv1.Deployment) error {
	logger := log.FromContext(ctx)

	dogu, err := dsm.doguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return fmt.Errorf("failed to get dogu descriptor of dogu %s: %w", doguResource.Name, err)
	}

	podTemplate, err := dsm.podTemplateResourceGenerator.GetPodTemplate(ctx, doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to get pod template for dogu %s in support action: %w", doguResource.Name, err)
	}

	if doguResource.Spec.SupportMode {
		setDoguPodTemplateInSupportMode(doguResource, podTemplate)
	}

	deployment.Spec.Template = *podTemplate
	logger.Info(fmt.Sprintf("Update deployment for dogu %s...", doguResource.Name))
	err = dsm.client.Update(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to update dogu deployment %s: %w", doguResource.Name, err)
	}

	return nil
}

func setDoguPodTemplateInSupportMode(doguResource *doguv2.Dogu, template *corev1.PodTemplateSpec) *corev1.PodTemplateSpec {
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

func supportModeChanged(doguResource *doguv2.Dogu, active bool) bool {
	mode := doguResource.Spec.SupportMode
	if mode && active || !mode && !active {
		return false
	}

	return true
}
