package controllers

import (
	"context"
	"fmt"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/retry-lib/retry"
	v1 "k8s.io/api/core/v1"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	client              client.Client
	resourceGenerator   dataSeederInitContainerGenerator
	resourceDoguFetcher resourceDoguFetcher
}

func NewDoguDataSeedManager(client client.Client, resourceGenerator dataSeederInitContainerGenerator, resourceDoguFetcher resourceDoguFetcher) *doguDataSeedManager {
	return &doguDataSeedManager{
		client:              client,
		resourceGenerator:   resourceGenerator,
		resourceDoguFetcher: resourceDoguFetcher,
	}
}

func (m *doguDataSeedManager) DataMountsChanged(ctx context.Context, doguResource *v2.Dogu) (bool, error) {
	logger := log.FromContext(ctx)
	logger.Info(fmt.Sprintf("Determine if data mounts changed from dogu resource %s", doguResource.Name))
	deployment, err := doguResource.GetDeployment(ctx, m.client)
	if err != nil {
		return false, err
	}

	initContainers := deployment.Spec.Template.Spec.InitContainers
	var actualDoguDataSeedContainer *v1.Container

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

func (m *doguDataSeedManager) createDataMountInitContainer(ctx context.Context, doguResource *v2.Dogu) (*v1.Container, error) {
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
		deployment, retryErr := doguResource.GetDeployment(ctx, m.client)
		if retryErr != nil {
			return retryErr
		}

		deployment.Spec.Template.Spec.InitContainers = append(deployment.Spec.Template.Spec.InitContainers, *container)

		return m.client.Update(ctx, deployment)
	})

	if err != nil {
		return fmt.Errorf("failed to update deployment dogu data mount for dogu %s: %w", doguResource.Name, err)
	}

	return nil
}
