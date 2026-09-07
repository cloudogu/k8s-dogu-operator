package install

import (
	"context"

	"github.com/cloudogu/dogu-lib/doguv3"
)

type DoguRegistryReader interface {
	Get(ctx context.Context, doguIdentifier doguv3.Identifier) (*doguv3.Dogu, error)
}
