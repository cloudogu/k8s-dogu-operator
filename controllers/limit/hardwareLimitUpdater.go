package limit

import (
	"context"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// hardwareLimitUpdater is responsible to create a cluster wide ingress class in the cluster.
type hardwareLimitUpdater struct {
	Client client.Client `json:"client"`
}

// NewHardwareLimitUpdater creates a new runnable responsible to detect changes in the container configuration of dogus.
func NewHardwareLimitUpdater(client client.Client) *hardwareLimitUpdater {
	return &hardwareLimitUpdater{
		Client: client,
	}
}

func (icc *hardwareLimitUpdater) Start(ctx context.Context) error {
	logger := ctrl.LoggerFrom(ctx)
	logger.Info("Starting hardware limiter updater listening on global trigger /config/_global/trigger-container-limit-sync ")

	return nil
}
