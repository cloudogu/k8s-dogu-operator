package exposedport

import (
	"context"

	"github.com/cloudogu/cesapp-lib/core"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type configMapInterface interface {
	v1.ConfigMapInterface
}

// localDoguFetcher includes functionality to search the local dogu registry for a dogu.
type localDoguFetcher interface {
	cesregistry.LocalDoguFetcher
}

type ExposedPortsManager interface {
	AddPorts(ctx context.Context, ports []core.ExposedPort) (*coreV1.ConfigMap, error)
	DeletePorts(ctx context.Context, ports []core.ExposedPort) (*coreV1.ConfigMap, error)
}
