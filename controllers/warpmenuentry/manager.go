package warpmenuentry

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/resource"
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
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error while getting warp menu entry : %w", err)
	}
	entryExists := err == nil

	// Without the warp tag there must be no warp menu entry.
	if !hasWarpMenuTag(doguDescriptor) {
		if !entryExists {
			return nil
		}
		return w.deleteEntry(ctx, entry)
	}

	// With the warp tag the warp menu entry must exist and be up to date.
	desiredEntry := w.buildNewWarpMenuEntry(doguResource, doguDescriptor)
	if !entryExists {
		return w.createEntry(ctx, desiredEntry)
	}
	return w.updateEntryIfChanged(ctx, entry, desiredEntry)
}

func (w WarpMenuEntryManager) createEntry(ctx context.Context, entry *warpMenuEntry.WarpMenuEntry) error {
	if _, err := w.client.Create(ctx, entry, v1.CreateOptions{}); err != nil {
		return fmt.Errorf("error while creating the new warp menu entry: %w", err)
	}
	return nil
}

func (w WarpMenuEntryManager) updateEntryIfChanged(ctx context.Context, current, desired *warpMenuEntry.WarpMenuEntry) error {
	if reflect.DeepEqual(current.Spec, desired.Spec) && reflect.DeepEqual(current.OwnerReferences, desired.OwnerReferences) {
		return nil
	}
	current.Spec = desired.Spec
	current.OwnerReferences = desired.OwnerReferences
	if _, err := w.client.Update(ctx, current, v1.UpdateOptions{}); err != nil {
		return fmt.Errorf("error while updating the warp menu entry: %w", err)
	}
	return nil
}

func (w WarpMenuEntryManager) deleteEntry(ctx context.Context, entry *warpMenuEntry.WarpMenuEntry) error {
	if err := w.client.Delete(ctx, entry.Name, v1.DeleteOptions{}); err != nil {
		return fmt.Errorf("error while deleting existing warp menu entry: %w", err)
	}
	return nil
}

func hasWarpMenuTag(descriptor *core.Dogu) bool {
	return slices.Contains(descriptor.Tags, warpTag)
}

func (w WarpMenuEntryManager) buildNewWarpMenuEntry(dogu *v2.Dogu, doguDescriptor *core.Dogu) *warpMenuEntry.WarpMenuEntry {
	displayName := doguDescriptor.DisplayName
	if displayName == "" {
		displayName = doguDescriptor.GetSimpleName()
	}

	return &warpMenuEntry.WarpMenuEntry{
		ObjectMeta: v1.ObjectMeta{
			Name: dogu.Name,
			OwnerReferences: []v1.OwnerReference{
				*v1.NewControllerRef(dogu, v2.GroupVersion.WithKind(kindDogu)),
			},
			Labels: resource.GetAppLabel().Add(dogu.GetDoguNameLabel()),
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
