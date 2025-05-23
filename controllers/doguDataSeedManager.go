package controllers

import (
	"context"
	"fmt"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/util"
	"github.com/cloudogu/retry-lib/retry"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	dataSeedInitContainerName = "dogu-data-seeder-init"
	// ChangeDataMountsEventReason is the reason string for firing events for changing custom data mounts in the dogu cr.
	ChangeDataMountsEventReason = "ChangeDoguDataMounts"
	// ErrorOnChangeDataMountsEventReason is the error string for firing change dogu data mounts.
	ErrorOnChangeDataMountsEventReason = "ErrChangeDoguDataMounts"
)

type doguDataSeedManager struct {
	deploymentInterface   deploymentInterface
	resourceGenerator     dataSeederInitContainerGenerator
	resourceDoguFetcher   resourceDoguFetcher
	requirementsGenerator requirementsGenerator
	dataSeedValidator     doguDataSeedValidator
	image                 string
}

func NewDoguDataSeedManager(deploymentInterface deploymentInterface, mgrSet *util.ManagerSet) *doguDataSeedManager {
	return &doguDataSeedManager{
		deploymentInterface:   deploymentInterface,
		resourceGenerator:     mgrSet.DoguDataSeedContainerGenerator,
		resourceDoguFetcher:   mgrSet.ResourceDoguFetcher,
		requirementsGenerator: mgrSet.RequirementsGenerator,
		dataSeedValidator:     mgrSet.DoguDataSeedValidator,
		image:                 mgrSet.AdditionalImages[config.DataSeederImageConfigmapNameKey],
	}
}

func (m *doguDataSeedManager) DataMountsChanged(ctx context.Context, doguResource *v2.Dogu) (bool, error) {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("Determine if data mounts changed from dogu resource %s", doguResource.Name))
	deployment, err := m.getDoguDeployment(ctx, doguResource)
	if err != nil {
		return false, err
	}

	initContainers := deployment.Spec.Template.Spec.InitContainers
	var actualDoguDataSeedContainer *corev1.Container

	// find init container
	for _, container := range initContainers {
		if container.Name == dataSeedInitContainerName {
			actualDoguDataSeedContainer = &container
			break
		}
	}

	data := doguResource.Spec.AdditionalMounts
	// If either data or container is missing, check if they're in different states => changed
	isNoCopyDataSeeder := actualDoguDataSeedContainer == nil || len(actualDoguDataSeedContainer.Args) <= 1
	if len(data) == 0 || isNoCopyDataSeeder {
		return (len(data) == 0) != (isNoCopyDataSeeder), nil
	}

	// Recreate init container and check for equality
	container, err := m.createDataMountInitContainer(ctx, doguResource)
	if err != nil {
		return false, err
	}

	argsEqual := reflect.DeepEqual(container.Args, actualDoguDataSeedContainer.Args)
	imageEqual := actualDoguDataSeedContainer.Image == m.image

	return !argsEqual || !imageEqual, nil
}

func (m *doguDataSeedManager) getDoguDeployment(ctx context.Context, doguResource *v2.Dogu) (*appsv1.Deployment, error) {
	list, err := m.deploymentInterface.List(ctx, v1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v2.DoguLabelName, doguResource.GetObjectKey().Name)})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment for dogu %s: %w", doguResource.Name, err)
	}
	if len(list.Items) == 1 {
		return &list.Items[0], nil
	}

	return nil, fmt.Errorf("dogu %s has more than one or zero deployments", doguResource.GetObjectKey().Name)
}

func (m *doguDataSeedManager) createDataMountInitContainer(ctx context.Context, doguResource *v2.Dogu) (*corev1.Container, error) {
	dogu, _, err := m.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return nil, fmt.Errorf("failed to get dogu descriptor for dogu %s: %w", doguResource.Name, err)
	}

	requirements, err := m.requirementsGenerator.Generate(ctx, dogu)
	if err != nil {
		return nil, fmt.Errorf("failed to generate requirements for dogu %s: %w", doguResource.Name, err)
	}

	container, err := m.resourceGenerator.BuildDataSeederContainer(dogu, doguResource, m.image, requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to generate data seeder init container while diff calculation: %w", err)
	}

	return container, nil
}

func (m *doguDataSeedManager) UpdateDataMounts(ctx context.Context, doguResource *v2.Dogu) error {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("Update data mounts for dogu resource %s...", doguResource.Name))
	dogu, _, err := m.resourceDoguFetcher.FetchWithResource(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to get dogu descriptor for dogu %s: %w", doguResource.Name, err)
	}
	err = m.dataSeedValidator.ValidateDataSeeds(ctx, dogu, doguResource)
	if err != nil {
		return fmt.Errorf("additinal data mounts are not valid dogu %s: %w", doguResource.Name, err)
	}

	container, err := m.createDataMountInitContainer(ctx, doguResource)
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
			if c.Name != dataSeedInitContainerName {
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
		return fmt.Errorf("failed to update deployment dogu data mount for dogu %s: %w", doguResource.Name, err)
	}

	logger.Info(fmt.Sprintf("Successfully updated data mounts for dogu resource %s", doguResource.Name))
	return nil
}
