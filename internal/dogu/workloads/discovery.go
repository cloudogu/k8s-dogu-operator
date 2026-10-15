package workloads

import (
	"context"
	"fmt"

	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Workloads contains the live workloads of a dogu.
type Workloads struct {
	Deployments  []appsv1.Deployment
	StatefulSets []appsv1.StatefulSet
	DaemonSets   []appsv1.DaemonSet
}

// IsEmpty returns true if the dogu has no workloads at all.
func (w Workloads) IsEmpty() bool {
	return len(w.Deployments) == 0 && len(w.StatefulSets) == 0 && len(w.DaemonSets) == 0
}

// Discovery finds the live resources of a dogu.
type Discovery interface {
	// Workloads returns all Deployments, StatefulSets and DaemonSets of the dogu in the given namespace.
	Workloads(ctx context.Context, namespace, doguName string) (Workloads, error)
}

type discovery struct {
	k8s client.Reader
}

// NewDiscovery creates a discovery that reads live resources with the given Kubernetes client.
func NewDiscovery(k8s client.Reader) Discovery {
	return &discovery{k8s: k8s}
}

// Workloads finds the workloads by the label k8s.cloudogu.com/dogu.name.
func (d *discovery) Workloads(ctx context.Context, namespace, doguName string) (Workloads, error) {
	opts := []client.ListOption{
		client.InNamespace(namespace),
		client.MatchingLabels{v3beta1.DoguLabelName: doguName},
	}

	deployments := &appsv1.DeploymentList{}
	if err := d.k8s.List(ctx, deployments, opts...); err != nil {
		return Workloads{}, fmt.Errorf("failed to list deployments of dogu %q: %w", doguName, err)
	}

	statefulSets := &appsv1.StatefulSetList{}
	if err := d.k8s.List(ctx, statefulSets, opts...); err != nil {
		return Workloads{}, fmt.Errorf("failed to list statefulsets of dogu %q: %w", doguName, err)
	}

	daemonSets := &appsv1.DaemonSetList{}
	if err := d.k8s.List(ctx, daemonSets, opts...); err != nil {
		return Workloads{}, fmt.Errorf("failed to list daemonsets of dogu %q: %w", doguName, err)
	}

	return Workloads{
		Deployments:  deployments.Items,
		StatefulSets: statefulSets.Items,
		DaemonSets:   daemonSets.Items,
	}, nil
}
