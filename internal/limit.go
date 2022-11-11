package internal

import (
	k8sv1 "github.com/cloudogu/k8s-dogu-operator/api/v1"
	"github.com/cloudogu/k8s-dogu-operator/controllers/limit"
	v1 "k8s.io/api/apps/v1"
)

type LimitPatcher interface {
	// RetrievePodLimits reads all container keys from the dogu configuration and creates a DoguLimits object.
	RetrievePodLimits(doguResource *k8sv1.Dogu) (limit.DoguLimits, error)
	// PatchDeployment patches the given deployment with the resource limits provided.
	PatchDeployment(deployment *v1.Deployment, limits limit.DoguLimits) error
}
