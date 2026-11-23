package upgrade

import (
	"context"

	"github.com/cloudogu/dogu-lib/doguv3"
	"github.com/cloudogu/k8s-dogu-lib/v3/api/v3beta1"
)

type ChangeDetector interface {
	Detect(ctx context.Context, dogu *v3beta1.Dogu) (Change, error)
}

type DoguRegistryReader interface {
	Get(ctx context.Context, doguIdentifier doguv3.Identifier) (*doguv3.Dogu, error)
}
