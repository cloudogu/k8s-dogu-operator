package manager

import (
	"context"
	"fmt"
	"reflect"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	doguClient "github.com/cloudogu/k8s-dogu-lib/v2/client"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/retry-lib/retry"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	additionalMountsInitContainerName = "dogu-additional-mounts-init"
	// ChangeDoguAdditionalMountsEventReason is the reason string for firing events for changing additional mounts in the dogu cr.
	ChangeDoguAdditionalMountsEventReason = "ChangeDoguAdditionalMounts"
	// ErrorOnChangeDoguAdditionalMountsEventReason is the error string for firing change dogu additional mounts.
	ErrorOnChangeDoguAdditionalMountsEventReason = "ErrChangeDoguAdditionalMounts"
)

type doguAdditionalMountManager struct {
	deploymentInterface          deploymentInterface
	resourceGenerator            additionalMountsInitContainerGenerator
	localDoguFetcher             localDoguFetcher
	requirementsGenerator        requirementsGenerator
	doguAdditionalMountValidator doguAdditionalMountsValidator
	doguInterface                doguClient.DoguInterface
	image                        string
}

func NewDoguAdditionalMountManager(deploymentInterface deploymentInterface, mgrSet *util.ManagerSet, doguInterface doguClient.DoguInterface) *doguAdditionalMountManager {
	return &doguAdditionalMountManager{
		deploymentInterface:          deploymentInterface,
		resourceGenerator:            mgrSet.DoguAdditionalMountsInitContainerGenerator,
		localDoguFetcher:             mgrSet.LocalDoguFetcher,
		requirementsGenerator:        mgrSet.RequirementsGenerator,
		doguAdditionalMountValidator: mgrSet.DoguAdditionalMountValidator,
		image:                        mgrSet.AdditionalImages[config.AdditionalMountsInitContainerImageConfigmapNameKey],
		doguInterface:                doguInterface,
	}
}

func (m *doguAdditionalMountManager) AdditionalMountsChanged(ctx context.Context, doguResource *v2.Dogu) (bool, error) {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("Determine if additional mounts changed from dogu resource %s", doguResource.Name))
	deployment, err := m.getDoguDeployment(ctx, doguResource)
	if err != nil {
		return false, err
	}

	initContainers := deployment.Spec.Template.Spec.InitContainers
	var actualAdditionalMountContainer *corev1.Container

	// find init container
	for _, container := range initContainers {
		if container.Name == additionalMountsInitContainerName {
			actualAdditionalMountContainer = &container
			break
		}
	}

	data := doguResource.Spec.AdditionalMounts
	// If either data or container is missing, check if they're in different states => changed
	noAdditionalMountsExists := actualAdditionalMountContainer == nil || len(actualAdditionalMountContainer.Args) <= 1
	if len(data) == 0 || noAdditionalMountsExists {
		return (len(data) == 0) != (noAdditionalMountsExists), nil
	}

	// Recreate init container and check for equality
	container, err := m.createAdditionalMountInitContainer(ctx, doguResource)
	if err != nil {
		return false, err
	}

	argsEqual := reflect.DeepEqual(container.Args, actualAdditionalMountContainer.Args)
	imageEqual := actualAdditionalMountContainer.Image == m.image

	return !argsEqual || !imageEqual, nil
}

func (m *doguAdditionalMountManager) getDoguDeployment(ctx context.Context, doguResource *v2.Dogu) (*appsv1.Deployment, error) {
	list, err := m.deploymentInterface.List(ctx, v1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v2.DoguLabelName, doguResource.GetObjectKey().Name)})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment for dogu %s: %w", doguResource.Name, err)
	}
	if len(list.Items) == 1 {
		return &list.Items[0], nil
	}

	return nil, fmt.Errorf("dogu %s has more than one or zero deployments", doguResource.GetObjectKey().Name)
}

func (m *doguAdditionalMountManager) createAdditionalMountInitContainer(ctx context.Context, doguResource *v2.Dogu) (*corev1.Container, error) {
	// We have to fetch the dogu.json from the local dogu registry because the actual dogu.json is already registered.
	// If we fetched the dogu.json from remote or dev configmap the dev configmap dogu.json will be deleted at this time.
	dogu, err := m.localDoguFetcher.FetchInstalled(ctx, cescommons.SimpleName(doguResource.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu descriptor for dogu %s: %w", doguResource.Name, err)
	}

	requirements, err := m.requirementsGenerator.Generate(ctx, dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to generate requirements for dogu %s: %w", doguResource.Name, err)
	}

	container, err := m.resourceGenerator.BuildAdditionalMountInitContainer(ctx, dogu, doguResource, m.image, requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to generate dogu additional mounts init container while diff calculation: %w", err)
	}

	return container, nil
}

func (m *doguAdditionalMountManager) UpdateAdditionalMounts(ctx context.Context, doguResource *v2.Dogu) error {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("Update additional mounts for dogu resource %s...", doguResource.Name))
	dogu, err := m.localDoguFetcher.FetchInstalled(ctx, cescommons.SimpleName(doguResource.Name))
	if err != nil {
		return fmt.Errorf("failed to get dogu descriptor for dogu %s: %w", doguResource.Name, err)
	}
	err = m.doguAdditionalMountValidator.ValidateAdditionalMounts(ctx, dogu, doguResource)
	if err != nil {
		return fmt.Errorf("additional mounts are not valid for dogu %s: %w", doguResource.Name, err)
	}

	container, err := m.createAdditionalMountInitContainer(ctx, doguResource)
	if err != nil {
		return err
	}

	volumes, err := resource.CreateVolumes(doguResource, dogu, doguResource.Spec.ExportMode)
	if err != nil {
		return err
	}

	err = retry.OnConflict(func() error {
		deployment, retryErr := m.getDoguDeployment(ctx, doguResource)
		if retryErr != nil {
			return retryErr
		}

		var updatedInitContainers []corev1.Container
		for _, c := range deployment.Spec.Template.Spec.InitContainers {
			if c.Name != additionalMountsInitContainerName {
				updatedInitContainers = append(updatedInitContainers, c)
			}
		}
		updatedInitContainers = append(updatedInitContainers, *container)
		deployment.Spec.Template.Spec.InitContainers = updatedInitContainers
		deployment.Spec.Template.Spec.Volumes = volumes

		_, retryErr = m.deploymentInterface.Update(ctx, deployment, v1.UpdateOptions{})

		return retryErr
	})

	if err != nil {
		return fmt.Errorf("failed to update deployment additional mounts for dogu %s: %w", doguResource.Name, err)
	}

	logger.Info(fmt.Sprintf("Successfully updated additional mounts for dogu resource %s", doguResource.Name))

	installedStatus := v2.DoguStatusInstalled
	_, err = m.doguInterface.UpdateStatusWithRetry(ctx, doguResource, func(status v2.DoguStatus) v2.DoguStatus {
		doguResource.Status.Status = installedStatus
		return doguResource.Status
	}, v1.UpdateOptions{})

	if err != nil {
		return fmt.Errorf("failed to update status of dogu %s to %s", doguResource.Name, installedStatus)
	}

	return err
}
