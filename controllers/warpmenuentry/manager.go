package warpmenuentry

import (
	"context"

	"github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
)

type WarpMenuEntryManager struct {
}

func NewWarpMenuEntryManager() *WarpMenuEntryManager {
	return &WarpMenuEntryManager{}
}

func (w WarpMenuEntryManager) EnsureWarpMenuEntry(ctx context.Context, doguResource *v2.Dogu) error {
	//TODO implement me
	panic("implement me")
}

func (w WarpMenuEntryManager) DeleteWarpMenuEntry(ctx context.Context, doguName dogu.SimpleName) error {
	//TODO implement me
	panic("implement me")
}
