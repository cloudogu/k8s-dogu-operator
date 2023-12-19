package health

import (
	appsv1 "k8s.io/api/apps/v1"
)

type AvailabilityChecker struct{}

// IsAvailable checks whether the deployment has reached its desired state and is available.
func (ac *AvailabilityChecker) IsAvailable(deployment *appsv1.Deployment) bool {
	// if replicas is nil, it is defaulted to 1
	if deployment.Spec.Replicas == nil && deployment.Status.UpdatedReplicas < 1 {
		return false
	}
	if deployment.Spec.Replicas != nil && deployment.Status.UpdatedReplicas < *deployment.Spec.Replicas {
		return false
	}
	if deployment.Status.Replicas > deployment.Status.UpdatedReplicas {
		return false
	}
	if deployment.Status.AvailableReplicas < deployment.Status.UpdatedReplicas {
		return false
	}

	return true
}
