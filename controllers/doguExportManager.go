package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/cloudogu/cesapp-lib/core"
	k8sv2 "github.com/cloudogu/k8s-dogu-operator/v3/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
)

const ExportModeEnvVar = "EXPORT_MODE"

type doguExportManager struct {
	client                       client.Client
	doguFetcher                  localDoguFetcher
	podTemplateResourceGenerator podTemplateResourceGenerator
	eventRecorder                record.EventRecorder
}

func NewDoguExportManager(client client.Client, mgrSet *util.ManagerSet, eventRecorder record.EventRecorder) *doguExportManager {
	return &doguExportManager{
		client:                       client,
		doguFetcher:                  mgrSet.LocalDoguFetcher,
		podTemplateResourceGenerator: mgrSet.DoguResourceGenerator,
		eventRecorder:                eventRecorder,
	}
}
func (dem *doguExportManager) HandleExportMode(ctx context.Context, doguResource *k8sv2.Dogu) (bool, error) {
	logger := log.FromContext(ctx)

	deployment := &appsv1.Deployment{}
	err := dem.client.Get(ctx, doguResource.GetObjectKey(), deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			dem.eventRecorder.Eventf(doguResource, corev1.EventTypeWarning, ExportEventReason, "No deployment found for dogu %s when checking export handler", doguResource.Name)
			return false, nil
		}
		return false, fmt.Errorf("failed to get deployment of dogu %s: %w", doguResource.Name, err)
	}

	logger.Info(fmt.Sprintf("Check if export mode is currently active for dogu %s...", doguResource.Name))
	active := isDeploymentInExportMode(deployment)
	if !exportModeChanged(doguResource, active) {
		return false, nil
	}

	err = dem.updateDeployment(ctx, doguResource, deployment)
	if err != nil {
		return false, err
	}
	dem.eventRecorder.Eventf(doguResource, corev1.EventTypeNormal, SupportEventReason, "Export flag changed to %t. Deployment updated.", doguResource.Spec.ExportMode)

	return true, nil
}

func setDoguPodTemplateInExportMode(doguResource *k8sv2.Dogu, template *corev1.PodTemplateSpec) *corev1.PodTemplateSpec {
	exportContainer := template.Spec.Containers[0]
	exportContainer.Name = fmt.Sprintf("%s-sidecar", doguResource.GetSimpleDoguName())
	exportContainer.Image = config.ExporterImageConfigmapNameKey
	exportContainer.StartupProbe = nil
	exportContainer.LivenessProbe = nil
	exportContainer.Resources = corev1.ResourceRequirements{}
	exportContainer.SecurityContext.Capabilities.Add = append(exportContainer.SecurityContext.Capabilities.Add, core.SysChroot)

	var newVolumes []corev1.VolumeMount

	newVolumes = append(newVolumes, corev1.VolumeMount{
		Name:      fmt.Sprintf("%s-data", doguResource.GetSimpleDoguName()),
		MountPath: "/storage",
	})

	log.Log.Error(fmt.Errorf("created volume mount for %s", doguResource.GetSimpleDoguName()), "+++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")

	template.Spec.Containers = append(template.Spec.Containers, exportContainer)

	return template
}

func isDeploymentInExportMode(deployment *appsv1.Deployment) bool {
	for _, container := range deployment.Spec.Template.Spec.Containers {
		envVars := container.Env
		for _, env := range envVars {
			if env.Name == ExportModeEnvVar && env.Value == "true" {
				return true
			}
		}
	}

	return false
}

func (dem *doguExportManager) updateDeployment(ctx context.Context, doguResource *k8sv2.Dogu, deployment *appsv1.Deployment) error {
	logger := log.FromContext(ctx)

	dogu, err := dem.doguFetcher.FetchInstalled(ctx, doguResource.GetSimpleDoguName())
	if err != nil {
		return fmt.Errorf("failed to get dogu descriptor of dogu %s: %w", doguResource.Name, err)
	}

	podTemplate, err := dem.podTemplateResourceGenerator.GetPodTemplate(ctx, doguResource, dogu)
	if err != nil {
		return fmt.Errorf("failed to get pod template for dogu %s in export action: %w", doguResource.Name, err)
	}

	if doguResource.Spec.ExportMode {
		setDoguPodTemplateInExportMode(doguResource, podTemplate)
	}

	deployment.Spec.Template = *podTemplate
	logger.Info(fmt.Sprintf("Update deployment for dogu %s...", doguResource.Name))
	err = dem.client.Update(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to update dogu deployment %s: %w", doguResource.Name, err)
	}

	return nil
}

func exportModeChanged(doguResource *k8sv2.Dogu, active bool) bool {
	mode := doguResource.Spec.ExportMode
	if mode && active || !mode && !active {
		return false
	}

	return true
}
