package controllers

import (
	"context"
	"fmt"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
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
	deploymentInterface deploymentInterface
	resourceGenerator   dataSeederInitContainerGenerator
	resourceDoguFetcher resourceDoguFetcher
}

func NewDoguDataSeedManager(deploymentInterface deploymentInterface, resourceGenerator dataSeederInitContainerGenerator, resourceDoguFetcher resourceDoguFetcher) *doguDataSeedManager {
	return &doguDataSeedManager{
		deploymentInterface: deploymentInterface,
		resourceGenerator:   resourceGenerator,
		resourceDoguFetcher: resourceDoguFetcher,
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

	data := doguResource.Spec.Data
	// If either data or container is missing, check if they're in different states => changed
	if len(data) == 0 || actualDoguDataSeedContainer == nil {
		return (len(data) == 0) != (actualDoguDataSeedContainer == nil), nil
	}

	// Recreate init container and check for equality
	container, err := m.createDataMountInitContainer(ctx, doguResource)
	if err != nil {
		return false, err
	}

	return !reflect.DeepEqual(container, actualDoguDataSeedContainer), nil
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

	// TODO Image
	container, err := m.resourceGenerator.GetDataSeederContainer(dogu, doguResource, "")
	if err != nil {
		return nil, fmt.Errorf("failed to generate data seeder init container while diff calculation: %w", err)
	}

	return container, nil
}

func (m *doguDataSeedManager) UpdateDataMounts(ctx context.Context, doguResource *v2.Dogu) error {
	container, err := m.createDataMountInitContainer(ctx, doguResource)
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

		_, retryErr = m.deploymentInterface.Update(ctx, deployment, v1.UpdateOptions{})

		return retryErr
	})

	if err != nil {
		return fmt.Errorf("failed to update deployment dogu data mount for dogu %s: %w", doguResource.Name, err)
	}

	return nil
}
