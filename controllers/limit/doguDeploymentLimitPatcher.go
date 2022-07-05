package limit

import (
	"fmt"
	"github.com/cloudogu/cesapp-lib/registry"
	v12 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	containerresource "k8s.io/apimachinery/pkg/api/resource"
)

const (
	cpuLimitKey              = "/containers/cpu-limit"
	memoryLimitKey           = "/containers/memory-limit"
	podsLimitKey             = "/containers/pods-limit"
	storageLimitKey          = "/containers/storage-limit"
	ephemeralStorageLimitKey = "/containers/ephemeral-storage-limit"
)

type doguDeploymentLimitPatcher struct {
	registry registry.Registry
}

// NewDoguDeploymentLimitPatcher creates a patcher with the ability to update the dogu deployments with their configured hardware limits
func NewDoguDeploymentLimitPatcher(registry registry.Registry) *doguDeploymentLimitPatcher {
	return &doguDeploymentLimitPatcher{
		registry: registry,
	}
}

// RetrieveMemoryLimits reads all container keys from the dogu configuration and creates a DoguLimits object.
func (d *doguDeploymentLimitPatcher) RetrieveMemoryLimits(doguResource *v12.Dogu) (DoguLimits, error) {
	doguRegistry := d.registry.DoguConfig(doguResource.Name)
	doguLimitObject := DoguLimits{}

	cpuLimit, err := doguRegistry.Get(cpuLimitKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return DoguLimits{}, err
	} else if err == nil {
		doguLimitObject.CpuLimit = cpuLimit
	}

	memoryLimit, err := doguRegistry.Get(memoryLimitKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return DoguLimits{}, err
	} else if err == nil {
		doguLimitObject.MemoryLimit = memoryLimit
	}

	podsLimit, err := doguRegistry.Get(podsLimitKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return DoguLimits{}, err
	} else if err == nil {
		doguLimitObject.PodsLimit = podsLimit
	}

	storageLimit, err := doguRegistry.Get(storageLimitKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return DoguLimits{}, err
	} else if err == nil {
		doguLimitObject.StorageLimit = storageLimit
	}

	ephemeralStorageLimit, err := doguRegistry.Get(ephemeralStorageLimitKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return DoguLimits{}, err
	} else if err == nil {
		doguLimitObject.EphemeralStorageLimit = ephemeralStorageLimit
	}

	return doguLimitObject, nil
}

// PatchDeployment patches the given deployment with the resource limits provided.
func (d *doguDeploymentLimitPatcher) PatchDeployment(deployment *appsv1.Deployment, limits DoguLimits) error {
	if len(deployment.Spec.Template.Spec.Containers) <= 0 {
		return fmt.Errorf("given deployment cannot be patched, no containers are defined, at least one container is required for patching")
	}

	resourceRequests := make(map[v1.ResourceName]containerresource.Quantity)
	resourceLimits := make(map[v1.ResourceName]containerresource.Quantity)

	err := d.patchMemoryLimits(limits, resourceRequests, resourceLimits)
	if err != nil {
		return err
	}

	err = d.patchCpuLimits(limits, resourceRequests, resourceLimits)
	if err != nil {
		return err
	}

	err = d.patchStrorageLimits(limits, resourceRequests, resourceLimits)
	if err != nil {
		return err
	}

	err = d.patchPodsLimits(limits, resourceRequests, resourceLimits)
	if err != nil {
		return err
	}

	err = d.patchStorageEphemeralLimits(limits, resourceRequests, resourceLimits)
	if err != nil {
		return err
	}

	deployment.Spec.Template.Spec.Containers[0].Resources.Requests = resourceRequests
	deployment.Spec.Template.Spec.Containers[0].Resources.Limits = resourceLimits

	return nil
}

func (d *doguDeploymentLimitPatcher) patchStorageEphemeralLimits(limits DoguLimits, resourceRequests map[v1.ResourceName]containerresource.Quantity, resourceLimits map[v1.ResourceName]containerresource.Quantity) error {
	if limits.EphemeralStorageLimit != "" {
		storageEphemeralLimit, err := containerresource.ParseQuantity(limits.EphemeralStorageLimit)
		if err != nil {
			return fmt.Errorf("failed to parse storageEphemeral request quantity: %w", err)
		}

		resourceRequests[v1.ResourceEphemeralStorage] = storageEphemeralLimit
		resourceLimits[v1.ResourceEphemeralStorage] = storageEphemeralLimit
	}
	return nil
}

func (d *doguDeploymentLimitPatcher) patchPodsLimits(limits DoguLimits, resourceRequests map[v1.ResourceName]containerresource.Quantity, resourceLimits map[v1.ResourceName]containerresource.Quantity) error {
	if limits.PodsLimit != "" {
		podsLimit, err := containerresource.ParseQuantity(limits.PodsLimit)
		if err != nil {
			return fmt.Errorf("failed to parse pods request quantity: %w", err)
		}

		resourceRequests[v1.ResourcePods] = podsLimit
		resourceLimits[v1.ResourcePods] = podsLimit
	}
	return nil
}

func (d *doguDeploymentLimitPatcher) patchStrorageLimits(limits DoguLimits, resourceRequests map[v1.ResourceName]containerresource.Quantity, resourceLimits map[v1.ResourceName]containerresource.Quantity) error {
	if limits.StorageLimit != "" {
		storageLimit, err := containerresource.ParseQuantity(limits.StorageLimit)
		if err != nil {
			return fmt.Errorf("failed to parse storage request quantity: %w", err)
		}

		resourceRequests[v1.ResourceStorage] = storageLimit
		resourceLimits[v1.ResourceStorage] = storageLimit
	}
	return nil
}

func (d *doguDeploymentLimitPatcher) patchCpuLimits(limits DoguLimits, resourceRequests map[v1.ResourceName]containerresource.Quantity, resourceLimits map[v1.ResourceName]containerresource.Quantity) error {
	if limits.CpuLimit != "" {
		cpuLimit, err := containerresource.ParseQuantity(limits.CpuLimit)
		if err != nil {
			return fmt.Errorf("failed to parse cpu request quantity: %w", err)
		}

		resourceRequests[v1.ResourceCPU] = cpuLimit
		resourceLimits[v1.ResourceCPU] = cpuLimit
	}
	return nil
}

func (d *doguDeploymentLimitPatcher) patchMemoryLimits(limits DoguLimits, resourceRequests map[v1.ResourceName]containerresource.Quantity, resourceLimits map[v1.ResourceName]containerresource.Quantity) error {
	if limits.MemoryLimit != "" {
		memLimit, err := containerresource.ParseQuantity(limits.MemoryLimit)
		if err != nil {
			return fmt.Errorf("failed to parse memory request quantity: %w", err)
		}

		resourceRequests[v1.ResourceMemory] = memLimit
		resourceLimits[v1.ResourceMemory] = memLimit
	}
	return nil
}
