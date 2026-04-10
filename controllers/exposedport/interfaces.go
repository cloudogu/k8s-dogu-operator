package exposedport

import (
	"context"

	"github.com/cloudogu/cesapp-lib/core"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type configMapInterface interface {
	v1.ConfigMapInterface
}

type ExposedPortsManager interface {
	AddPorts(ctx context.Context, ports []core.ExposedPort) (*coreV1.ConfigMap, error)
	DeletePorts(ctx context.Context, ports []core.ExposedPort) (*coreV1.ConfigMap, error)
}
