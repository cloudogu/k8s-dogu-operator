package warpmenuentry

import (
	"context"

	"github.com/cloudogu/ces-commons-lib/dogu"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

type Manager interface {
	EnsureWarpMenuEntry(ctx context.Context, doguResource *doguv2.Dogu) error
	DeleteWarpMenuEntry(ctx context.Context, doguName dogu.SimpleName) error
}
