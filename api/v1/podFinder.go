package v1

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPodForLabels returns a pod for the given dogu labels. An error is returned if either no pod or more than one pod is found.
func GetPodForLabels(ctx context.Context, cli client.Client, doguLabels CesMatchingLabels) (*v1.Pod, error) {
	// note for future improvement:
	// this pod's selection must be revised if dogus are horizontally scalable by adding more pods with the same image.
	pods := &v1.PodList{}
	err := cli.List(ctx, pods, client.MatchingLabels(doguLabels))
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("found no pods for labels %s", doguLabels)
	}
	if len(pods.Items) > 1 {
		return nil, fmt.Errorf("found more than one pod (%s) for labels %s", pods, doguLabels)
	}

	return &pods.Items[0], nil
}

// GetServiceForLabels returns a service for the given dogu labels. An error is returned if either no service or more than one service is found.
func GetServiceForLabels(ctx context.Context, cli client.Client, labels CesMatchingLabels) (*v1.Service, error) {
	services := &v1.ServiceList{}
	err := cli.List(ctx, services, client.MatchingLabels(labels))
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	if len(services.Items) == 0 {
		return nil, fmt.Errorf("found no services for labels %s", labels)
	}
	if len(services.Items) > 1 {
		return nil, fmt.Errorf("found more than one service (%s) for labels %s", services, labels)
	}

	return &services.Items[0], nil
}
