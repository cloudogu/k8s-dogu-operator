package initfx

import (
	"go.uber.org/fx"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

//nolint:unused
//goland:noinspection GoUnusedType
type k8sManager interface {
	manager.Manager
}

type fxLifecycle interface {
	fx.Lifecycle
}

//nolint:unused
//goland:noinspection GoUnusedType
type configMapInterface interface {
	v1.ConfigMapInterface
}
