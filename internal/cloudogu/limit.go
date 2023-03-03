package cloudogu

import (
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
)

// LimitPatcher includes functionality to read and patch the physical resource limits of a dogu.
type LimitPatcher interface {
	// RetrievePodLimits reads all container keys from the dogu configuration and creates a DoguLimits object.
	RetrievePodLimits(doguResource *k8sv1.Dogu) (DoguLimits, error)
	// PatchDeployment patches the given deployment with the resource limits provided.
	PatchDeployment(deployment *v1.Deployment, limits DoguLimits) error
}

// DoguLimits provides physical resource limits for a dogu.
type DoguLimits interface {
	// CpuLimit returns the cpu requests and limit values for the dogu deployment. For more information about resource management in Kubernetes see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/.
	CpuLimit() *resource.Quantity
	// MemoryLimit returns the memory requests and limit values for the dogu deployment. For more information about resource management in Kubernetes see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/.
	MemoryLimit() *resource.Quantity
	// EphemeralStorageLimit returns the ephemeral storage requests and limit values for the dogu deployment. For more information about resource management in Kubernetes see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/.
	EphemeralStorageLimit() *resource.Quantity
}
