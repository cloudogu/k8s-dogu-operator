package warpmenuentry

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	warpMenuEntry "github.com/cloudogu/k8s-warp-menu-entry-lib/api/v1"
	warpMenuEntryV1 "github.com/cloudogu/k8s-warp-menu-entry-lib/client/typed/api/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	warpTag  = "warp"
	kindDogu = "Dogu"
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
			_, createErr := w.client.Create(ctx, w.createNewWarpMenuEntry(doguResource, doguDescriptor), v1.CreateOptions{})
			if createErr != nil {
				return fmt.Errorf("error while creating the new warp menu entry: %w", createErr)
			}
		}
	} else {
		if hasWarpMenuTag(doguDescriptor) {
			desiredEntry := w.createNewWarpMenuEntry(doguResource, doguDescriptor)
			if reflect.DeepEqual(entry.Spec, desiredEntry.Spec) && reflect.DeepEqual(entry.OwnerReferences, desiredEntry.OwnerReferences) {
				return nil
			} else {
				_, updateErr := w.client.Update(ctx, desiredEntry, v1.UpdateOptions{})
				if updateErr != nil {
					return fmt.Errorf("error while updating the warp menu entry: %w", updateErr)
				}
			}
		} else {
			err = w.client.Delete(ctx, entry.Name, v1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("error while deleting existing warp menu entry: %w", err)
			}
		}
	}

	return nil
}

func hasWarpMenuTag(descriptor *core.Dogu) bool {
	return slices.Contains(descriptor.Tags, warpTag)
}

func (w WarpMenuEntryManager) createNewWarpMenuEntry(dogu *v2.Dogu, doguDescriptor *core.Dogu) *warpMenuEntry.WarpMenuEntry {
	displayName := doguDescriptor.DisplayName
	if displayName == "" {
		displayName = doguDescriptor.Name
	}

	return &warpMenuEntry.WarpMenuEntry{
		ObjectMeta: v1.ObjectMeta{
			Name: dogu.Name,
			OwnerReferences: []v1.OwnerReference{
				*v1.NewControllerRef(dogu, v2.GroupVersion.WithKind(kindDogu)),
			},
		},
		Spec: warpMenuEntry.WarpMenuEntrySpec{
			DisplayName: warpMenuEntry.DisplayName{
				DE: displayName,
				EN: displayName,
			},
			Category: doguDescriptor.Category,
			Path:     "/" + dogu.GetSimpleDoguName().String(),
			Disabled: false,
		},
	}
}
