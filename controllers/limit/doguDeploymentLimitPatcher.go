package limit

import (
	"fmt"
	"github.com/cloudogu/k8s-dogu-operator/internal"

	"github.com/cloudogu/cesapp-lib/registry"
	v12 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	containerresource "k8s.io/apimachinery/pkg/api/resource"
)

const (
	cpuLimitKey              = "/pod_limit/cpu"
	memoryLimitKey           = "/pod_limit/memory"
	ephemeralStorageLimitKey = "/pod_limit/ephemeral_storage"
)

const (
	cpuLimitKind              limitKind = "cpu"
	memoryLimitKind           limitKind = "memory"
	ephemeralStorageLimitKind limitKind = "ephemeral storage"
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

// RetrievePodLimits reads all container keys from the dogu configuration and creates a doguLimits object.
func (d *doguDeploymentLimitPatcher) RetrievePodLimits(doguResource *v12.Dogu) (internal.DoguLimits, error) {
	doguRegistry := d.registry.DoguConfig(doguResource.Name)
	doguLimitObject := &doguLimits{}

	cpuLimit, err := doguRegistry.Get(cpuLimitKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return &doguLimits{}, err
	} else if err == nil {
		limit, err := parseLimit(cpuLimit, cpuLimitKind)
		if err != nil {
			return nil, err
		}
		doguLimitObject.cpuLimit = limit
	}

	memoryLimit, err := doguRegistry.Get(memoryLimitKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return &doguLimits{}, err
	} else if err == nil {
		limit, err := parseLimit(memoryLimit, memoryLimitKind)
		if err != nil {
			return nil, err
		}
		doguLimitObject.memoryLimit = limit
	}

	ephemeralStorageLimit, err := doguRegistry.Get(ephemeralStorageLimitKey)
	if err != nil && !registry.IsKeyNotFoundError(err) {
		return &doguLimits{}, err
	} else if err == nil {
		limit, err := parseLimit(ephemeralStorageLimit, ephemeralStorageLimitKind)
		if err != nil {
			return nil, err
		}
		doguLimitObject.ephemeralStorageLimit = limit
	}

	return doguLimitObject, nil
}

// PatchDeployment patches the given deployment with the resource limits provided.
func (d *doguDeploymentLimitPatcher) PatchDeployment(deployment *appsv1.Deployment, limits internal.DoguLimits) error {
	if len(deployment.Spec.Template.Spec.Containers) <= 0 {
		return fmt.Errorf("given deployment cannot be patched, no containers are defined, at least one container is required for patching")
	}

	resourceRequests := make(v1.ResourceList)
	resourceLimits := make(v1.ResourceList)

	d.patchMemoryLimits(limits, resourceRequests, resourceLimits)
	d.patchCpuLimits(limits, resourceRequests, resourceLimits)
	d.patchStorageEphemeralLimits(limits, resourceRequests, resourceLimits)

	deployment.Spec.Template.Spec.Containers[0].Resources.Requests = resourceRequests
	deployment.Spec.Template.Spec.Containers[0].Resources.Limits = resourceLimits

	return nil
}

type limitKind string

func parseLimit(raw string, kind limitKind) (*containerresource.Quantity, error) {
	if raw == "" {
		return nil, nil
	}
	limit, err := containerresource.ParseQuantity(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s limit '%s': %w", kind, raw, err)
	}
	return &limit, nil
}

func (d *doguDeploymentLimitPatcher) patchStorageEphemeralLimits(limits internal.DoguLimits, resourceRequests v1.ResourceList, resourceLimits v1.ResourceList) {
	ephemeralStorageLimit := limits.EphemeralStorageLimit()
	if ephemeralStorageLimit != nil {
		resourceRequests[v1.ResourceEphemeralStorage] = *ephemeralStorageLimit
		resourceLimits[v1.ResourceEphemeralStorage] = *ephemeralStorageLimit
	}
}

func (d *doguDeploymentLimitPatcher) patchCpuLimits(limits internal.DoguLimits, resourceRequests v1.ResourceList, resourceLimits v1.ResourceList) {
	cpuLimit := limits.CpuLimit()
	if cpuLimit != nil {
		resourceRequests[v1.ResourceCPU] = *cpuLimit
		resourceLimits[v1.ResourceCPU] = *cpuLimit
	}
}

func (d *doguDeploymentLimitPatcher) patchMemoryLimits(limits internal.DoguLimits, resourceRequests v1.ResourceList, resourceLimits v1.ResourceList) {
	memoryLimit := limits.MemoryLimit()
	if memoryLimit != nil {
		resourceRequests[v1.ResourceMemory] = *memoryLimit
		resourceLimits[v1.ResourceMemory] = *memoryLimit
	}
}
