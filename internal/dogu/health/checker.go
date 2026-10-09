// Package health derives the health of a v3 dogu from its live workloads.
package health

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/workloads"
	appsv1 "k8s.io/api/apps/v1"
)

// State is the health of a dogu in terms of the Healthy condition of the dogu resource.
type State struct {
	Healthy bool
	Reason  string
	Message string
}

// workloadDiscovery finds the live workloads of a dogu.
type workloadDiscovery interface {
	Workloads(ctx context.Context, namespace, doguName string) (workloads.Workloads, error)
}

// Checker derives the health of a dogu from its workloads.
type Checker struct {
	discovery workloadDiscovery
}

func NewChecker(discovery workloadDiscovery) *Checker {
	return &Checker{discovery: discovery}
}

// Check returns the health of the dogu:
//   - a stopped dogu is never healthy, even if all of its workloads are formally ready with zero replicas
//   - a dogu without workloads or without any desired replica is not healthy
//   - a dogu is healthy if all of its workloads are ready
func (c *Checker) Check(ctx context.Context, dogu *v3beta1.Dogu) (State, error) {
	if dogu.Spec.Stopped {
		return State{Healthy: false, Reason: v3beta1.ReasonStopped, Message: "the dogu is stopped"}, nil
	}

	found, err := c.discovery.Workloads(ctx, dogu.Namespace, dogu.Spec.Name)
	if err != nil {
		return State{}, fmt.Errorf("failed to get workloads of dogu %q: %w", dogu.Spec.Name, err)
	}

	desiredReplicas := int32(0)
	var notReady []string

	for _, deployment := range found.Deployments {
		desired := replicasOrDefault(deployment.Spec.Replicas)
		desiredReplicas += desired
		if !isDeploymentReady(deployment, desired) {
			notReady = append(notReady, describe("Deployment", deployment.Name, "available", deployment.Status.AvailableReplicas, deployment.Status.UpdatedReplicas, desired))
		}
	}

	for _, statefulSet := range found.StatefulSets {
		desired := replicasOrDefault(statefulSet.Spec.Replicas)
		desiredReplicas += desired
		if !isStatefulSetReady(statefulSet, desired) {
			notReady = append(notReady, describe("StatefulSet", statefulSet.Name, "ready", statefulSet.Status.ReadyReplicas, statefulSet.Status.UpdatedReplicas, desired))
		}
	}

	for _, daemonSet := range found.DaemonSets {
		desired := daemonSet.Status.DesiredNumberScheduled
		desiredReplicas += desired
		if !isDaemonSetReady(daemonSet) {
			notReady = append(notReady, describe("DaemonSet", daemonSet.Name, "ready", daemonSet.Status.NumberReady, daemonSet.Status.UpdatedNumberScheduled, desired))
		}
	}

	if desiredReplicas == 0 {
		return State{Healthy: false, Reason: v3beta1.ReasonWorkloadsNotReady, Message: "the dogu has no running workloads"}, nil
	}

	if len(notReady) > 0 {
		msg := fmt.Sprintf("workloads not ready: %s", strings.Join(notReady, ", "))
		return State{Healthy: false, Reason: v3beta1.ReasonWorkloadsNotReady, Message: msg}, nil
	}

	return State{Healthy: true, Reason: v3beta1.ReasonSucceeded, Message: "all workloads are ready"}, nil
}

// describe returns a short description of a workload that is not ready, e.g. "StatefulSet/nexus (0/1 ready, 1/1 updated)".
func describe(kind, name, readyLabel string, ready, updated, desired int32) string {
	return fmt.Sprintf("%s/%s (%d/%d %s, %d/%d updated)", kind, name, ready, desired, readyLabel, updated, desired)
}

// replicasOrDefault returns the desired replicas of a Deployment or StatefulSet; Kubernetes defaults them to 1.
func replicasOrDefault(replicas *int32) int32 {
	if replicas == nil {
		return 1
	}
	return *replicas
}

func isDeploymentReady(deployment appsv1.Deployment, desired int32) bool {
	return deployment.Status.ObservedGeneration >= deployment.Generation &&
		deployment.Status.UpdatedReplicas == desired &&
		deployment.Status.AvailableReplicas == desired
}

func isStatefulSetReady(statefulSet appsv1.StatefulSet, desired int32) bool {
	return statefulSet.Status.ObservedGeneration >= statefulSet.Generation &&
		statefulSet.Status.UpdatedReplicas == desired &&
		statefulSet.Status.ReadyReplicas == desired
}

func isDaemonSetReady(daemonSet appsv1.DaemonSet) bool {
	desired := daemonSet.Status.DesiredNumberScheduled
	return daemonSet.Status.ObservedGeneration >= daemonSet.Generation &&
		daemonSet.Status.UpdatedNumberScheduled == desired &&
		daemonSet.Status.NumberReady == desired
}
