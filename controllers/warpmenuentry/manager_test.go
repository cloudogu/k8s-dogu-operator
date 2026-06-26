package warpmenuentry

import (
	"context"
	"fmt"
	"testing"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	warpMenuEntryV1 "github.com/cloudogu/k8s-warp-menu-entry-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	doguName                 = "testdogu"
	doguDescriptorSimpleName = "testdogudescriptorname"
	doguDescriptorName       = "testing/" + doguDescriptorSimpleName
	displayName              = "testdoguDisplayName"
	category                 = "category"
	path                     = "/testdogudescriptorname"
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
	doguResource := newDoguResource(doguName)
	doguDescriptorWithWarp := newDoguDescriptor(true)
	warpMenuEntryWithWarp := newWarpMenuEntry(doguResource, doguDescriptorWithWarp)
	doguDescriptorWithoutWarp := newDoguDescriptor(false)
	warpMenuEntryWithoutWarp := newWarpMenuEntry(doguResource, doguDescriptorWithoutWarp)

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

	t.Run("Should return error when warp menu entry get returns an error", func(t *testing.T) {
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

	t.Run("Should update existing warp menu entry if it differs from the dogu", func(t *testing.T) {
		//given
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		existingEntry := newWarpMenuEntry(doguResource, doguDescriptorWithWarp)
		existingEntry.Spec.Category = "outdated-category"

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptorWithWarp, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(existingEntry, nil)
		client.EXPECT().Update(ctx, warpMenuEntryWithWarp, v1.UpdateOptions{}).Return(warpMenuEntryWithWarp, nil)
		manager := &WarpMenuEntryManager{doguFetcher: doguFetcher, client: client}

		//when
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		//then
		require.NoError(t, err)
	})

	t.Run("Should update existing warp menu entry if it differs from the dogu", func(t *testing.T) {
		//given
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		existingEntry := newWarpMenuEntry(doguResource, doguDescriptorWithWarp)

		existingEntry.SetOwnerReferences([]v1.OwnerReference{
			*v1.NewControllerRef(doguResource, v2.GroupVersion.WithKind("Dogu1")),
		})

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptorWithWarp, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(existingEntry, nil)
		client.EXPECT().Update(ctx, warpMenuEntryWithWarp, v1.UpdateOptions{}).Return(warpMenuEntryWithWarp, nil)
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

		existingEntry := newWarpMenuEntry(doguResource, doguDescriptorWithWarp)
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

func TestDeleteWarpMenuEntry(t *testing.T) {
	ctx := context.Background()

	t.Run("Should not give an error if no warp menu entry exists for the given name", func(t *testing.T) {
		//given
		client := newMockWarpmenuentryClient(t)
		client.EXPECT().Delete(ctx, doguName, v1.DeleteOptions{}).Return(newNotFoundError())
		manager := &WarpMenuEntryManager{client: client}

		//when
		err := manager.DeleteWarpMenuEntry(ctx, doguName)

		//then
		require.NoError(t, err)
	})

	t.Run("Should return an error when deleting warp menu entry fails", func(t *testing.T) {
		//given
		client := newMockWarpmenuentryClient(t)
		const deleteError = "delete warpmenu entry fail"
		client.EXPECT().Delete(ctx, doguName, v1.DeleteOptions{}).Return(fmt.Errorf(deleteError))
		manager := &WarpMenuEntryManager{client: client}

		//when
		err := manager.DeleteWarpMenuEntry(ctx, doguName)

		//then
		require.Error(t, err)
		assert.ErrorContains(t, err, "error deleting existing warp menu entry")
		assert.ErrorContains(t, err, deleteError)
	})

	t.Run("Should delete existing warp menu entry for the given dogu name", func(t *testing.T) {
		//given
		client := newMockWarpmenuentryClient(t)
		client.EXPECT().Delete(ctx, doguName, v1.DeleteOptions{}).Return(nil)
		manager := &WarpMenuEntryManager{client: client}

		//when
		err := manager.DeleteWarpMenuEntry(ctx, doguName)

		//then
		require.NoError(t, err)
	})
}

func newNotFoundError() *errors2.StatusError {
	return errors2.NewNotFound(schema.GroupResource{Group: "k8s", Resource: "warpmenuentry"}, doguName)
}

func newDoguResource(doguName string) *v2.Dogu {
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

func newWarpMenuEntry(doguResource *v2.Dogu, doguDescriptor *cesappcore.Dogu) *warpMenuEntryV1.WarpMenuEntry {

	return &warpMenuEntryV1.WarpMenuEntry{
		ObjectMeta: v1.ObjectMeta{
			OwnerReferences: []v1.OwnerReference{
				*v1.NewControllerRef(doguResource, v2.GroupVersion.WithKind("Dogu")),
			},
			Name: doguResource.Name,
		},
		Spec: warpMenuEntryV1.WarpMenuEntrySpec{
			DisplayName: warpMenuEntryV1.DisplayName{
				DE: doguDescriptor.DisplayName,
				EN: doguDescriptor.DisplayName,
			},
			Category: doguDescriptor.Category,
			Path:     path,
			Disabled: false,
		},
	}
}
