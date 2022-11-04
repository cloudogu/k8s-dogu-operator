package v1

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPodForLabels returns a pod for the given dogu labels. An error is returned if either no pod or more than one pod is found.
func GetPodForLabels(ctx context.Context, cli client.Client, doguLabels client.MatchingLabels) (*v1.Pod, error) {
	// note for future improvement:
	// this pod's selection must be revised if dogus are horizontally scalable by adding more pods with the same image.
	pods := &v1.PodList{}
	err := cli.List(ctx, pods, doguLabels)
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
