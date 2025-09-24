package initfx

import (
	"go.uber.org/fx"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type k8sManager interface {
	manager.Manager
}

type fxLifecycle interface {
	fx.Lifecycle
}
