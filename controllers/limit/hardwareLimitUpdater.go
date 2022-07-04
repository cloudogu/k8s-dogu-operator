package limit

import (
	"context"
	"fmt"
	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/cesapp-lib/registry"
	coreosclient "github.com/coreos/etcd/client"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	triggerSyncEtcdKeyFullPath = "/config/_global/trigger-container-limit-sync"
)

// hardwareLimitUpdater is responsible to create a cluster wide ingress class in the cluster.
type hardwareLimitUpdater struct {
	client   client.Client
	registry registry.Registry
}

type doguLimits struct {
	cpuLimit              string
	memoryLimit           string
	storageLimit          string
	podsLimit             string
	ephemeralStorageLimit string
}

// NewHardwareLimitUpdater creates a new runnable responsible to detect changes in the container configuration of dogus.
func NewHardwareLimitUpdater(client client.Client, namespace string) (*hardwareLimitUpdater, error) {
	endpoint := fmt.Sprintf("http://etcd.%s.svc.cluster.local:4001", namespace)
	reg, err := registry.New(core.Registry{
		Type:      "etcd",
		Endpoints: []string{endpoint},
	})
	if err != nil {
		return nil, err
	}

	return &hardwareLimitUpdater{
		client:   client,
		registry: reg,
	}, nil
}

func (hlu *hardwareLimitUpdater) Start(ctx context.Context) error {
	return hlu.startEtcdWatch(ctx)
}

func (hlu *hardwareLimitUpdater) startEtcdWatch(ctx context.Context) error {
	ctrl.LoggerFrom(ctx).Info(fmt.Sprintf("Start etcd watcher on certificate key [%s]", triggerSyncEtcdKeyFullPath))

	triggerChannel := make(chan *coreosclient.Response)
	go func() {
		hlu.registry.RootConfig().Watch(ctx, triggerSyncEtcdKeyFullPath, false, triggerChannel)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-triggerChannel:
			err := hlu.triggerSync(ctx)
			if err != nil {
				return err
			}
		}
	}
}

func (hlu *hardwareLimitUpdater) triggerSync(ctx context.Context) error {
	ctrl.LoggerFrom(ctx).Info("Trigger for updating dogu hardware limits detected in registry. Updating deployment for all dogus...")

	return nil
}
