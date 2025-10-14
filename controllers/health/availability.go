package health

import (
	appsv1 "k8s.io/api/apps/v1"
)

func NewAvailabilityChecker() *AvailabilityChecker {
	return &AvailabilityChecker{}
}

type AvailabilityChecker struct{}

// IsAvailable checks whether the deployment has reached its desired state and is available.
func (ac *AvailabilityChecker) IsAvailable(deployment *appsv1.Deployment) bool {
	// if replicas is nil, it is defaulted to 1
	status := deployment.Status
	if deployment.Spec.Replicas == nil && status.UpdatedReplicas < 1 {
		return false
	}
	if deployment.Spec.Replicas != nil && status.UpdatedReplicas < *deployment.Spec.Replicas {
		return false
	}
	if deployment.Spec.Replicas != nil && *deployment.Spec.Replicas == 0 {
		return false
	}
	if status.Replicas > status.UpdatedReplicas {
		return false
	}
	if status.AvailableReplicas < status.UpdatedReplicas {
		return false
	}

	return true
}
