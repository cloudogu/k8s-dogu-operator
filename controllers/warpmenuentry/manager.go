package warpmenuentry

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	warpMenuEntryV1 "github.com/cloudogu/k8s-warp-menu-entry-lib/client/typed/api/v1"
)

type WarpMenuEntryManager struct {
	client      warpmenuentryClient
	doguFetcher localDoguFetcher
}

func NewWarpMenuEntryManager(
	client warpMenuEntryV1.WarpMenuEntryInterface,
	localDoguFetcher cesregistry.LocalDoguFetcher) *WarpMenuEntryManager {
	return &WarpMenuEntryManager{
		client:      client,
		doguFetcher: localDoguFetcher,
	}
}

func (w WarpMenuEntryManager) EnsureWarpMenuEntry(ctx context.Context, doguResource *v2.Dogu) error {
	if doguResource == nil {
		return errors.New("doguResource must not be nil")
	}
	_, err := w.doguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("dogu spec cannot be retrieved : %w", err)
	}

	return nil
}

func (w WarpMenuEntryManager) DeleteWarpMenuEntry(ctx context.Context, doguName dogu.SimpleName) error {
	//TODO implement me
	panic("implement me")
}
