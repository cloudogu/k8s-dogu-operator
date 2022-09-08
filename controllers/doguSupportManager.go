package controllers

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp-lib/registry"
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	"github.com/cloudogu/k8s-dogu-operator/controllers/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const supportModeEnvVar = "SUPPORT_MODE"

// doguSupportManager is used to handle the support mode for dogus.
type doguSupportManager struct {
	client            client.Client
	scheme            *runtime.Scheme
	doguRegistry      registry.DoguRegistry
	resourceGenerator *resource.ResourceGenerator
	eventRecorder     record.EventRecorder
}

// NewDoguSupportManager creates a new instance of doguSupportManager.
func NewDoguSupportManager(client client.Client, cesRegistry registry.Registry, eventRecorder record.EventRecorder) *doguSupportManager {
	resourceGenerator := resource.NewResourceGenerator(client.Scheme(), limit.NewDoguDeploymentLimitPatcher(cesRegistry))
	return &doguSupportManager{
		client:            client,
		scheme:            client.Scheme(),
		doguRegistry:      cesRegistry.DoguRegistry(),
		resourceGenerator: resourceGenerator,
		eventRecorder:     eventRecorder,
	}
}

// HandleSupportFlag handles the support flag in the dogu spec. If the support modes changes the method returns
// true, nil. If there was no change of the support mode it returns false, nil
func (dsm *doguSupportManager) HandleSupportFlag(ctx context.Context, doguResource *k8sv1.Dogu) (bool, error) {
	containers, err := dsm.getDoguContainers(ctx, doguResource)
	if err != nil {
		return false, err
	}

	active, err := dsm.isSupportModeActive(containers)
	if err != nil {
		return false, fmt.Errorf("failed to determine if the dogu %s is in support mode: %w", doguResource.Name, err)
	}

	if !dsm.supportModeChanged(doguResource, active) {
		return false, nil
	}

	deployment := &appsv1.Deployment{}
	err = dsm.client.Get(ctx, *doguResource.GetObjectKey(), deployment)
	if err != nil {
		return false, fmt.Errorf("failed to get deployment of the dogu %s: %w", doguResource.Name, err)
	}

	dogu, err := dsm.doguRegistry.Get(doguResource.Name)
	if err != nil {
		return false, fmt.Errorf("failed to get dogu deskriptor of the dogu %s: %w", doguResource.Name, err)
	}

	deployment.Spec.Template = dsm.resourceGenerator.GetPodTemplate(doguResource, dogu, doguResource.Spec.SupportMode)
	err = dsm.client.Update(ctx, deployment)
	if err != nil {
		return false, fmt.Errorf("failed to update dogu deployment %s: %w", doguResource.Name, err)
	}

	return true, nil
}

func (dsm *doguSupportManager) isSupportModeActive(containers []corev1.Container) (bool, error) {
	supportModeActive := false
	for _, container := range containers {
		for _, envVar := range container.Env {
			if envVar.Name == supportModeEnvVar && envVar.Value == "true" {
				supportModeActive = true
				break
			}
		}

		if supportModeActive {
			break
		}
	}

	return supportModeActive, nil
}

func (dsm *doguSupportManager) getDoguContainers(ctx context.Context, doguResource *k8sv1.Dogu) ([]corev1.Container, error) {
	podList := &corev1.PodList{}
	err := dsm.client.List(ctx, podList, client.InNamespace(doguResource.Namespace), client.HasLabels{
		fmt.Sprintf("dogu=%s", doguResource.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get list of pods from dogu %s: %w", doguResource.Name, err)
	}
	var containers []corev1.Container
	for _, pod := range podList.Items {
		for _, container := range pod.Spec.Containers {
			if strings.Contains(container.Image, fmt.Sprintf("%s:%s", doguResource.Spec.Name, doguResource.Spec.Version)) {
				containers = append(containers, container)
				continue
			}
		}
	}

	return containers, nil
}

func (dsm *doguSupportManager) supportModeChanged(doguResource *k8sv1.Dogu, active bool) bool {
	mode := doguResource.Spec.SupportMode
	if mode && active || !mode && !active {
		return false
	}

	return true
}
