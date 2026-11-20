package upgrade

import (
	"context"

	"github.com/cloudogu/dogu-lib/doguv3"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
	"github.com/cloudogu/k8s-dogu-operator/v3/internal/dogu/upgrade"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type UpgradeDetector interface {
	Detect(ctx context.Context, dogu *v3beta1.Dogu) (upgrade.Change, error)
}

type DoguRegistryReader interface {
	Get(ctx context.Context, doguIdentifier doguv3.Identifier) (*doguv3.Dogu, error)
}

type EventRecorder interface {
	Event(object runtime.Object, eventtype, reason, message string)
}

type K8sClient interface {
	client.Client
}
