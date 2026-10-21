package workloads

import appsv1 "k8s.io/api/apps/v1"

// ReplicasOrDefault returns the desired replicas of a Deployment or StatefulSet; Kubernetes defaults them to 1.
func ReplicasOrDefault(replicas *int32) int32 {
	if replicas == nil {
		return 1
	}
	return *replicas
}

// IsDeploymentReady returns true if the latest spec of the Deployment is rolled out and all desired replicas are available.
func IsDeploymentReady(deployment appsv1.Deployment) bool {
	desired := ReplicasOrDefault(deployment.Spec.Replicas)
	return deployment.Status.ObservedGeneration >= deployment.Generation &&
		deployment.Status.UpdatedReplicas == desired &&
		deployment.Status.AvailableReplicas == desired
}

// IsStatefulSetReady returns true if the latest spec of the StatefulSet is rolled out and all desired replicas are ready.
func IsStatefulSetReady(statefulSet appsv1.StatefulSet) bool {
	desired := ReplicasOrDefault(statefulSet.Spec.Replicas)
	return statefulSet.Status.ObservedGeneration >= statefulSet.Generation &&
		statefulSet.Status.UpdatedReplicas == desired &&
		statefulSet.Status.ReadyReplicas == desired
}

// IsDaemonSetReady returns true if the latest spec of the DaemonSet is rolled out and all scheduled pods are ready.
func IsDaemonSetReady(daemonSet appsv1.DaemonSet) bool {
	desired := daemonSet.Status.DesiredNumberScheduled
	return daemonSet.Status.ObservedGeneration >= daemonSet.Generation &&
		daemonSet.Status.UpdatedNumberScheduled == desired &&
		daemonSet.Status.NumberReady == desired
}
