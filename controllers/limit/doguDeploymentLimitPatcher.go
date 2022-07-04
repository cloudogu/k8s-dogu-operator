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
	CpuLimitKey              = "/containers/cpu-limit"
	MemoryLimitKey           = "/containers/memory-limit"
	PodsLimitKey             = "/containers/pods-limit"
	StorageLimitKey          = "/containers/storage-limit"
	EphemeralStorageLimitKey = "/containers/ephemeral-storage-limit"
)

type doguDeploymentLimitPatcher struct {
	registry registry.Registry
}

func NewDoguDeploymentLimitPatcher(registry registry.Registry) *doguDeploymentLimitPatcher {
	return &doguDeploymentLimitPatcher{
		registry: registry,
	}
}

// RetrieveMemoryLimits reads all container keys from the dogu configuration and creates a doguLimits object.
func (d *doguDeploymentLimitPatcher) RetrieveMemoryLimits(doguResource *v12.Dogu) (doguLimits, error) {
	doguRegistry := d.registry.DoguConfig(doguResource.Name)
	doguLimitObject := doguLimits{}

	cpuLimit, err := doguRegistry.Get(CpuLimitKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return doguLimits{}, err
	} else if err == nil {
		doguLimitObject.cpuLimit = cpuLimit
	}

	memoryLimit, err := doguRegistry.Get(MemoryLimitKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return doguLimits{}, err
	} else if err == nil {
		doguLimitObject.memoryLimit = memoryLimit
	}

	podsLimit, err := doguRegistry.Get(PodsLimitKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return doguLimits{}, err
	} else if err == nil {
		doguLimitObject.podsLimit = podsLimit
	}

	storageLimit, err := doguRegistry.Get(StorageLimitKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return doguLimits{}, err
	} else if err == nil {
		doguLimitObject.storageLimit = storageLimit
	}

	ephemeralStorageLimit, err := doguRegistry.Get(EphemeralStorageLimitKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return doguLimits{}, err
	} else if err == nil {
		doguLimitObject.ephemeralStorageLimit = ephemeralStorageLimit
	}

	return doguLimitObject, nil
}

// PatchDeployment patches the given deployment with the resource limits provided.
func (d *doguDeploymentLimitPatcher) PatchDeployment(deployment *appsv1.Deployment, limits doguLimits) error {
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

func (d *doguDeploymentLimitPatcher) patchStorageEphemeralLimits(limits doguLimits, resourceRequests map[v1.ResourceName]containerresource.Quantity, resourceLimits map[v1.ResourceName]containerresource.Quantity) error {
	if limits.ephemeralStorageLimit != "" {
		storageEphemeralLimit, err := containerresource.ParseQuantity(limits.ephemeralStorageLimit)
		if err != nil {
			return fmt.Errorf("failed to parse storageEphemeral request quantity: %w", err)
		}

		resourceRequests[v1.ResourceEphemeralStorage] = storageEphemeralLimit
		resourceLimits[v1.ResourceEphemeralStorage] = storageEphemeralLimit
	}
	return nil
}

func (d *doguDeploymentLimitPatcher) patchPodsLimits(limits doguLimits, resourceRequests map[v1.ResourceName]containerresource.Quantity, resourceLimits map[v1.ResourceName]containerresource.Quantity) error {
	if limits.podsLimit != "" {
		podsLimit, err := containerresource.ParseQuantity(limits.podsLimit)
		if err != nil {
			return fmt.Errorf("failed to parse pods request quantity: %w", err)
		}

		resourceRequests[v1.ResourcePods] = podsLimit
		resourceLimits[v1.ResourcePods] = podsLimit
	}
	return nil
}

func (d *doguDeploymentLimitPatcher) patchStrorageLimits(limits doguLimits, resourceRequests map[v1.ResourceName]containerresource.Quantity, resourceLimits map[v1.ResourceName]containerresource.Quantity) error {
	if limits.storageLimit != "" {
		storageLimit, err := containerresource.ParseQuantity(limits.storageLimit)
		if err != nil {
			return fmt.Errorf("failed to parse storage request quantity: %w", err)
		}

		resourceRequests[v1.ResourceStorage] = storageLimit
		resourceLimits[v1.ResourceStorage] = storageLimit
	}
	return nil
}

func (d *doguDeploymentLimitPatcher) patchCpuLimits(limits doguLimits, resourceRequests map[v1.ResourceName]containerresource.Quantity, resourceLimits map[v1.ResourceName]containerresource.Quantity) error {
	if limits.cpuLimit != "" {
		cpuLimit, err := containerresource.ParseQuantity(limits.cpuLimit)
		if err != nil {
			return fmt.Errorf("failed to parse cpu request quantity: %w", err)
		}

		resourceRequests[v1.ResourceCPU] = cpuLimit
		resourceLimits[v1.ResourceCPU] = cpuLimit
	}
	return nil
}

func (d *doguDeploymentLimitPatcher) patchMemoryLimits(limits doguLimits, resourceRequests map[v1.ResourceName]containerresource.Quantity, resourceLimits map[v1.ResourceName]containerresource.Quantity) error {
	if limits.memoryLimit != "" {
		memLimit, err := containerresource.ParseQuantity(limits.memoryLimit)
		if err != nil {
			return fmt.Errorf("failed to parse memory request quantity: %w", err)
		}

		resourceRequests[v1.ResourceMemory] = memLimit
		resourceLimits[v1.ResourceMemory] = memLimit
	}
	return nil
}
