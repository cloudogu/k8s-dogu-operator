package warpmenuentry

import (
	"context"
	"fmt"
	"slices"

	"github.com/cloudogu/ces-commons-lib/dogu"
	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	warpMenuEntry "github.com/cloudogu/k8s-warp-menu-entry-lib/api/v1"
	warpMenuEntryV1 "github.com/cloudogu/k8s-warp-menu-entry-lib/client/typed/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	warpTag = "warp"
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
	doguDescriptor, err := w.doguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("dogu spec cannot be retrieved : %w", err)
	}

	entry, err := w.client.Get(ctx, doguResource.Name, v1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("error while getting warp menu entry : %w", err)
		}
		if hasWarpMenuTag(doguDescriptor) {
			_, createrr := w.client.Create(ctx, w.createNewWarpMenuEntry(doguResource.Name, doguDescriptor), v1.CreateOptions{})
			if createrr != nil {
				return fmt.Errorf("error while creating the new warp menu entry: %w", createrr)
			}
		}
	} else {
		if !hasWarpMenuTag(doguDescriptor) {
			err = w.client.Delete(ctx, entry.Name, v1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("eerror while deleting existing warp menu entry: %w", err)
			}
		}
	}

	return nil
}

func hasWarpMenuTag(descriptor *core.Dogu) bool {
	return slices.Contains(descriptor.Tags, warpTag)
}

func (w WarpMenuEntryManager) createNewWarpMenuEntry(doguName string, doguDescriptor *core.Dogu) *warpMenuEntry.WarpMenuEntry {

	return &warpMenuEntry.WarpMenuEntry{
		ObjectMeta: v1.ObjectMeta{
			Name: doguName,
		},
		Spec: warpMenuEntry.WarpMenuEntrySpec{
			DisplayName: warpMenuEntry.DisplayName{
				DE: doguDescriptor.DisplayName,
				EN: doguDescriptor.DisplayName,
			},
			Category: doguDescriptor.Category,
			Path:     "/" + doguDescriptor.Name,
			Disabled: false,
		},
	}
}

func (w WarpMenuEntryManager) DeleteWarpMenuEntry(ctx context.Context, doguName dogu.SimpleName) error {
	//TODO implement me
	panic("implement me")
}
