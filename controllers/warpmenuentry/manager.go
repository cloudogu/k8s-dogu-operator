package warpmenuentry

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"unicode/utf8"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	cesappcore "github.com/cloudogu/cesapp-lib/core"
	doguv2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/cesregistry"
	warpv1 "github.com/cloudogu/k8s-warp-menu-entry-lib/api/v1"
	warpClientV1 "github.com/cloudogu/k8s-warp-menu-entry-lib/client/typed/api/v1"
	k8sErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const warpTag = "warp"
const maxFieldLength = 50

type WarpMenuEntryManager struct {
	client      warpMenuEntryClient
	doguFetcher localDoguFetcher
}

func NewManager(client warpClientV1.WarpMenuEntryInterface, doguFetcher cesregistry.LocalDoguFetcher) *WarpMenuEntryManager {
	return &WarpMenuEntryManager{
		client:      client,
		doguFetcher: doguFetcher,
	}
}

func (m *WarpMenuEntryManager) EnsureWarpMenuEntry(ctx context.Context, doguResource *doguv2.Dogu) error {
	if doguResource == nil {
		return fmt.Errorf("dogu resource must not be nil")
	}

	doguDescriptor, err := m.doguFetcher.FetchForResource(ctx, doguResource)
	if err != nil {
		return fmt.Errorf("failed to fetch dogu descriptor: %w", err)
	}

	// Remove a previously created entry if the dogu no longer carries the warp tag.
	if !hasWarpTag(doguDescriptor) {
		return m.RemoveWarpMenuEntry(ctx, doguResource.GetSimpleDoguName())
	}

	spec, err := buildSpec(doguResource, doguDescriptor)
	if err != nil {
		return fmt.Errorf("failed to build WarpMenuEntry spec: %w", err)
	}

	desired := &warpv1.WarpMenuEntry{
		ObjectMeta: metav1.ObjectMeta{
			Name: doguResource.GetSimpleDoguName().String(),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(doguResource, doguv2.GroupVersion.WithKind("Dogu")),
			},
		},
		Spec: spec,
	}

	_, err = m.ensureWarpMenuEntry(ctx, desired)
	return err
}

func (m *WarpMenuEntryManager) RemoveWarpMenuEntry(ctx context.Context, doguName cescommons.SimpleName) error {
	err := m.client.Delete(ctx, doguName.String(), metav1.DeleteOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to delete WarpMenuEntry: %w", err)
	}

	return nil
}

func (m *WarpMenuEntryManager) ensureWarpMenuEntry(ctx context.Context, desired *warpv1.WarpMenuEntry) (*warpv1.WarpMenuEntry, error) {
	current, err := m.client.Get(ctx, desired.Name, metav1.GetOptions{})
	if err != nil {
		if k8sErr.IsNotFound(err) {
			created, createErr := m.client.Create(ctx, desired, metav1.CreateOptions{})
			if createErr != nil {
				return nil, fmt.Errorf("failed to create WarpMenuEntry: %w", createErr)
			}
			return created, nil
		}

		return nil, fmt.Errorf("failed to get WarpMenuEntry: %w", err)
	}

	if reflect.DeepEqual(current.Spec, desired.Spec) && reflect.DeepEqual(current.OwnerReferences, desired.OwnerReferences) {
		return current, nil
	}

	current.Spec = desired.Spec
	current.OwnerReferences = desired.OwnerReferences

	updated, err := m.client.Update(ctx, current, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update WarpMenuEntry: %w", err)
	}

	return updated, nil
}

func hasWarpTag(doguDescriptor *cesappcore.Dogu) bool {
	return slices.Contains(doguDescriptor.Tags, warpTag)
}

// buildSpec derives the desired WarpMenuEntry spec from the dogu resource and its descriptor
func buildSpec(doguResource *doguv2.Dogu, doguDescriptor *cesappcore.Dogu) (warpv1.WarpMenuEntrySpec, error) {
	simpleName := doguResource.GetSimpleDoguName().String()

	// v2 dogus only carry a single, non-localized display name.
	// Fall back to the descriptor name when empty, so the CRD's mandatory de/en fields are satisfied.
	displayName := doguDescriptor.DisplayName
	if displayName == "" {
		displayName = doguDescriptor.Name
	}
	if utf8.RuneCountInString(displayName) > maxFieldLength {
		return warpv1.WarpMenuEntrySpec{}, fmt.Errorf("display name %q of dogu %q exceeds maximum length of %d characters", displayName, simpleName, maxFieldLength)
	}
	if doguDescriptor.Category == "" {
		return warpv1.WarpMenuEntrySpec{}, fmt.Errorf("category of dogu %q must not be empty", simpleName)
	}
	if utf8.RuneCountInString(doguDescriptor.Category) > maxFieldLength {
		return warpv1.WarpMenuEntrySpec{}, fmt.Errorf("category %q of dogu %q exceeds maximum length of %d characters", doguDescriptor.Category, simpleName, maxFieldLength)
	}

	return warpv1.WarpMenuEntrySpec{
		DisplayName: warpv1.DisplayName{
			DE: displayName,
			EN: displayName,
		},
		Category: doguDescriptor.Category,
		Path:     "/" + simpleName,
		Disabled: false,
	}, nil
}
