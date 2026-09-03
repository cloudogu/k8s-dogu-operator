package warpmenuentry

import (
	"context"
	"fmt"
	"testing"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v3/api/v2"
	warpMenuEntryV1 "github.com/cloudogu/k8s-warp-menu-entry-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	doguName           = "testdogu"
	doguDescriptorName = "testing/" + doguName
	displayName        = "testdoguDisplayName"
	category           = "category"
	path               = "/" + doguName
)

func TestNewManager(t *testing.T) {
	t.Run("Should Create new test manager", func(t *testing.T) {
		var manager Manager

		manager = *NewWarpMenuEntryManager(nil, nil)
		require.NotNil(t, manager)
	})
}

func TestEnsureWarpMenuEntry(t *testing.T) {
	ctx := context.Background()
	doguResource := newDoguResource()
	doguDescriptorWithWarp := newDoguDescriptor(true)
	warpMenuEntryWithWarp := newWarpMenuEntry(doguResource)
	doguDescriptorWithoutWarp := newDoguDescriptor(false)
	warpMenuEntryWithoutWarp := newWarpMenuEntry(doguResource)

	t.Run("Should return error when doguresource is nil", func(t *testing.T) {
		//given
		manager := &WarpMenuEntryManager{doguFetcher: nil}
		//When
		err := manager.EnsureWarpMenuEntry(ctx, nil)
		//then
		require.Error(t, err)
		assert.ErrorContains(t, err, "doguResource must not be nil")
	})

	t.Run("Should return error when dogu spec cannot be retrieved", func(t *testing.T) {
		//given
		const fetchError = "fetch error"
		doguFetcher := newMockLocalDoguFetcher(t)
		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(nil, fmt.Errorf(fetchError))
		manager := &WarpMenuEntryManager{doguFetcher: doguFetcher}
		//when
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)
		//then
		require.Error(t, err)
		assert.ErrorContains(t, err, "dogu spec cannot be retrieved")
		assert.ErrorContains(t, err, fetchError)
	})

	t.Run("Should return error when warp menu entry get returns an error other than not found", func(t *testing.T) {
		//given
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		const getWarpMenuEntryError = "get warp menu entry error"

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptorWithWarp, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(nil, fmt.Errorf(getWarpMenuEntryError))
		manager := &WarpMenuEntryManager{doguFetcher: doguFetcher, client: client}

		//when
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		//then
		require.Error(t, err)
		assert.ErrorContains(t, err, "error while getting warp menu entry")
		assert.ErrorContains(t, err, getWarpMenuEntryError)
	})

	t.Run("Should return error when creating a new warp menu entry returns an error", func(t *testing.T) {
		//given
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		const createWarpMenuEntryError = "create warp menu entry error"

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptorWithWarp, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(nil, newNotFoundError())
		client.EXPECT().Create(ctx, warpMenuEntryWithWarp, v1.CreateOptions{}).Return(nil, fmt.Errorf(createWarpMenuEntryError))
		manager := &WarpMenuEntryManager{doguFetcher: doguFetcher, client: client}

		//when
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		//then
		require.Error(t, err)
		assert.ErrorContains(t, err, "error while creating the new warp menu entry")
		assert.ErrorContains(t, err, createWarpMenuEntryError)
	})

	t.Run("Should create a new warp menu entry if warp menu entry does not exist", func(t *testing.T) {
		//given
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptorWithWarp, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(nil, newNotFoundError())
		client.EXPECT().Create(ctx, warpMenuEntryWithWarp, v1.CreateOptions{}).Return(warpMenuEntryWithWarp, nil)
		manager := &WarpMenuEntryManager{doguFetcher: doguFetcher, client: client}

		//when
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		//then
		require.NoError(t, err)
	})

	t.Run("Should not create a new warp menu entry if warpmenu entry is not present and dogu has no warp tag", func(t *testing.T) {
		//given
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptorWithoutWarp, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(nil, newNotFoundError())
		manager := &WarpMenuEntryManager{doguFetcher: doguFetcher, client: client}

		//when
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		//then
		require.NoError(t, err)
	})

	t.Run("Should delete existing warp menu entry if dogu has no warp tag", func(t *testing.T) {
		//given
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptorWithoutWarp, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(warpMenuEntryWithWarp, nil)
		client.EXPECT().Delete(ctx, doguResource.Name, v1.DeleteOptions{}).Return(nil)
		manager := &WarpMenuEntryManager{doguFetcher: doguFetcher, client: client}

		//when
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		//then
		require.NoError(t, err)
	})

	t.Run("Should return error when there is an error deleting an existing warp menu entry if dogu has no warp tag", func(t *testing.T) {
		//given
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptorWithoutWarp, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(warpMenuEntryWithoutWarp, nil)
		const deleteError = "deleteError"
		client.EXPECT().Delete(ctx, doguResource.Name, v1.DeleteOptions{}).Return(fmt.Errorf(deleteError))
		manager := &WarpMenuEntryManager{doguFetcher: doguFetcher, client: client}

		//when
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		//then
		require.Error(t, err)
		assert.ErrorContains(t, err, "error while deleting existing warp menu entry")
		assert.ErrorContains(t, err, deleteError)
	})

	t.Run("Should  do nothing if there is an existing warp menu entry and there are no changes", func(t *testing.T) {
		//given
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptorWithWarp, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(warpMenuEntryWithWarp, nil)
		manager := &WarpMenuEntryManager{doguFetcher: doguFetcher, client: client}

		//when
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		//then
		require.NoError(t, err)
	})

	t.Run("Should update existing warp menu entry preserving its resourceVersion if the spec differs", func(t *testing.T) {
		//given
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		existingEntry := newWarpMenuEntry(doguResource)
		existingEntry.ResourceVersion = "57900"
		existingEntry.Spec.Category = "outdated-category"

		expectedUpdate := newWarpMenuEntry(doguResource)
		expectedUpdate.ResourceVersion = "57900"

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptorWithWarp, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(existingEntry, nil)
		client.EXPECT().Update(ctx, expectedUpdate, v1.UpdateOptions{}).Return(expectedUpdate, nil)
		manager := &WarpMenuEntryManager{doguFetcher: doguFetcher, client: client}

		//when
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		//then
		require.NoError(t, err)
	})

	t.Run("Should update existing warp menu entry preserving its resourceVersion if the owner references differ", func(t *testing.T) {
		//given
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		existingEntry := newWarpMenuEntry(doguResource)
		existingEntry.ResourceVersion = "57900"
		existingEntry.SetOwnerReferences([]v1.OwnerReference{
			*v1.NewControllerRef(doguResource, v2.GroupVersion.WithKind("Dogu1")),
		})

		expectedUpdate := newWarpMenuEntry(doguResource)
		expectedUpdate.ResourceVersion = "57900"

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptorWithWarp, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(existingEntry, nil)
		client.EXPECT().Update(ctx, expectedUpdate, v1.UpdateOptions{}).Return(expectedUpdate, nil)
		manager := &WarpMenuEntryManager{doguFetcher: doguFetcher, client: client}

		//when
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		//then
		require.NoError(t, err)
	})

	t.Run("Should return error when updating an existing warp menu entry returns an error", func(t *testing.T) {
		//given
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		existingEntry := newWarpMenuEntry(doguResource)
		existingEntry.Spec.Category = "outdated-category"

		const updateError = "update warp menu entry error"

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptorWithWarp, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(existingEntry, nil)
		client.EXPECT().Update(ctx, warpMenuEntryWithWarp, v1.UpdateOptions{}).Return(nil, fmt.Errorf(updateError))
		manager := &WarpMenuEntryManager{doguFetcher: doguFetcher, client: client}

		//when
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		//then
		require.Error(t, err)
		assert.ErrorContains(t, err, "error while updating the warp menu entry")
		assert.ErrorContains(t, err, updateError)
	})
}

func newNotFoundError() *errors2.StatusError {
	return errors2.NewNotFound(schema.GroupResource{Group: "k8s", Resource: "warpmenuentry"}, doguName)
}

func newDoguResource() *v2.Dogu {
	return &v2.Dogu{
		ObjectMeta: v1.ObjectMeta{
			Name: doguName,
		},
	}
}

func newDoguDescriptor(withWarpTag bool) *cesappcore.Dogu {
	var tags []string
	if withWarpTag {
		tags = append(tags, warpTag)
	}
	return &cesappcore.Dogu{
		Name:        doguDescriptorName,
		DisplayName: displayName,
		Category:    category,
		Tags:        tags,
	}
}

func newWarpMenuEntry(doguResource *v2.Dogu) *warpMenuEntryV1.WarpMenuEntry {

	return &warpMenuEntryV1.WarpMenuEntry{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{
				*v1.NewControllerRef(doguResource, v2.GroupVersion.WithKind("Dogu")),
			},
			Name: doguName,
			Labels: map[string]string{
				"app":       "ces",
				"dogu.name": doguName,
			},
		},
		Spec: warpMenuEntryV1.WarpMenuEntrySpec{
			DisplayName: warpMenuEntryV1.DisplayName{
				DE: displayName,
				EN: displayName,
			},
			Category: category,
			Path:     path,
			Disabled: false,
		},
	}
}
