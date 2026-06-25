package warpmenuentry

import (
	"context"
	"fmt"

	"github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	warpMenuEntry "github.com/cloudogu/k8s-warp-menu-entry-lib/api/v1"
	warpMenuEntryV1 "github.com/cloudogu/k8s-warp-menu-entry-lib/client/typed/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		return fmt.Errorf("doguResource must not be nil")
	}
	_, err := w.doguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("dogu spec cannot be retrieved : %w", err)
	}

	_, err = w.client.Get(ctx, doguResource.Name, v1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("error while getting warp menu entry : %w", err)
		}
		_, createrr := w.client.Create(ctx, w.createNewWarpMenuEntry(), v1.CreateOptions{})
		if createrr != nil {
			return fmt.Errorf("error while creating the new warp menu entry: %w", createrr)
		}
	}

	return nil
}

func (w WarpMenuEntryManager) createNewWarpMenuEntry() *warpMenuEntry.WarpMenuEntry {
	return &warpMenuEntry.WarpMenuEntry{}
}

func (w WarpMenuEntryManager) DeleteWarpMenuEntry(ctx context.Context, doguName dogu.SimpleName) error {
	//TODO implement me
	panic("implement me")
}
