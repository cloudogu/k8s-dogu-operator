package warpmenuentry

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	doguFormatError = "doguFetchError"
)

func TestNewWarpMenuEntryManager(t *testing.T) {
	manager := NewWarpMenuEntryManager(nil, nil)

	require.NotNil(t, manager)
	assert.Nil(t, manager.client)
	assert.Nil(t, manager.doguFetcher)

}

func TestEnsureWarpMenuEntry(t *testing.T) {
	ctx := context.Background()
	doguResource := getDogu(defaultDoguName)
	doguDescriptor := getDoguDescriptor(defaultDoguName, displayName, doguCategory, warpTag)

	t.Run("should create warp menu entry", func(t *testing.T) {
		manager := &WarpMenuEntryManager{}

		err := manager.EnsureWarpMenuEntry(ctx, nil)
		require.Error(t, err)
		assert.ErrorContains(t, err, "dogu resource must not be nil")

	})

	t.Run("should return dogu fetch error", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		doguFetchError := fmt.Errorf(doguFormatError)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(nil, doguFetchError)

		manager := &WarpMenuEntryManager{
			client:      nil,
			doguFetcher: fetcher,
		}
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to fetch dogu descriptor:")
		assert.ErrorContains(t, err, doguFormatError)

	})

	t.Run("should create warp menu entries if it does not exist", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)
		notFoundWarpMenuEntryError := &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptor, nil)
		client.EXPECT().Get(ctx, doguResource.Name, metav1.GetOptions{}).Return(nil, notFoundWarpMenuEntryError)

		warpMenuEntryToUpdate := getExpectedWarpMenuEntry(defaultDoguName, doguResource, displayName, warpMenuDisabled, path, doguCategory)
		client.EXPECT().Create(ctx, warpMenuEntryToUpdate, metav1.CreateOptions{}).Return(nil, nil)
		manager := &WarpMenuEntryManager{
			client:      client,
			doguFetcher: fetcher,
		}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)
		require.NoError(t, err)
	})

	t.Run("errorcase: should create warp menu entries if it does not exist, create fails", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)
		notFoundWarpMenuEntryError := &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptor, nil)
		client.EXPECT().Get(ctx, doguResource.Name, metav1.GetOptions{}).Return(nil, notFoundWarpMenuEntryError)

		const createWarpMenuErrorMessage = "error creating warpmenuentry"
		createErr := fmt.Errorf(createWarpMenuErrorMessage)

		warpMenuEntryToUpdate := getExpectedWarpMenuEntry(defaultDoguName, doguResource, displayName, warpMenuDisabled, path, doguCategory)
		client.EXPECT().Create(ctx, warpMenuEntryToUpdate, metav1.CreateOptions{}).Return(nil, createErr)
		manager := &WarpMenuEntryManager{
			client:      client,
			doguFetcher: fetcher,
		}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)
		require.Error(t, err)
		assert.ErrorContains(t, err, "failed to create warpmenuentry:")
		assert.ErrorContains(t, err, createWarpMenuErrorMessage)
	})

	t.Run("should create warp menu entries if it exists, and has same value", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptor, nil)

		warpMenuEntryToUpdate := getExpectedWarpMenuEntry(defaultDoguName, doguResource, displayName, warpMenuDisabled, path, doguCategory)
		client.EXPECT().Get(ctx, doguResource.Name, metav1.GetOptions{}).Return(warpMenuEntryToUpdate, nil)
		manager := &WarpMenuEntryManager{
			client:      client,
			doguFetcher: fetcher,
		}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)
		require.NoError(t, err)
	})

	t.Run("should create warp menu entries if it exists, but has a different value", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptor, nil)

		warpMenuEntryToUpdate := getExpectedWarpMenuEntry(defaultDoguName, doguResource, displayName, warpMenuDisabled, path, doguCategory)

		existingEntry := getExpectedWarpMenuEntry(defaultDoguName, doguResource, displayName, warpMenuDisabled, path+"somechange", doguCategory)
		client.EXPECT().Get(ctx, doguResource.Name, metav1.GetOptions{}).Return(existingEntry, nil)

		client.EXPECT().Update(ctx, warpMenuEntryToUpdate, metav1.UpdateOptions{}).Return(warpMenuEntryToUpdate, nil)
		manager := &WarpMenuEntryManager{
			client:      client,
			doguFetcher: fetcher,
		}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)
		require.NoError(t, err)
	})

	t.Run("Fail in should create warp menu entries if it exists, but has a different value, error while updating the warp menu entry", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptor, nil)

		warpMenuEntryToUpdate := getExpectedWarpMenuEntry(defaultDoguName, doguResource, displayName, warpMenuDisabled, path, doguCategory)

		existingEntry := getExpectedWarpMenuEntry(defaultDoguName, doguResource, displayName, warpMenuDisabled, path+"somechange", doguCategory)
		client.EXPECT().Get(ctx, doguResource.Name, metav1.GetOptions{}).Return(existingEntry, nil)

		const updateWarpMenuErrorMessage = "error updating warpmenuentry"
		updateErr := fmt.Errorf(updateWarpMenuErrorMessage)
		client.EXPECT().Update(ctx, warpMenuEntryToUpdate, metav1.UpdateOptions{}).Return(nil, updateErr)
		manager := &WarpMenuEntryManager{
			client:      client,
			doguFetcher: fetcher,
		}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)
		require.Error(t, err)
		assert.ErrorContains(t, err, "error updating warp menu entry")
		assert.ErrorContains(t, err, updateWarpMenuErrorMessage)
	})

	t.Run("should delete warp menu entries if warp tag does not exist", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)
		//notFoundWarpMenuEntryError := &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}

		existingEntry := getExpectedWarpMenuEntry(defaultDoguName, doguResource, displayName, warpMenuDisabled, path, doguCategory)
		doguDescriptor := getDoguDescriptor(defaultDoguName, displayName, doguCategory, "nonwarptag")
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptor, nil)
		client.EXPECT().Get(ctx, doguResource.Name, metav1.GetOptions{}).Return(existingEntry, nil)

		client.EXPECT().Delete(ctx, doguResource.Name, metav1.DeleteOptions{}).Return(nil)
		manager := &WarpMenuEntryManager{
			client:      client,
			doguFetcher: fetcher,
		}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)
		require.NoError(t, err)
	})

	t.Run("should do nothing to warp menu entries if warp tag does not exist and there is no existing warp menu entry", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		notFoundWarpMenuEntryError := &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}

		doguDescriptor := getDoguDescriptor(defaultDoguName, displayName, doguCategory, "nonwarptag")
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptor, nil)
		client.EXPECT().Get(ctx, doguResource.Name, metav1.GetOptions{}).Return(nil, notFoundWarpMenuEntryError)

		manager := &WarpMenuEntryManager{
			client:      client,
			doguFetcher: fetcher,
		}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)
		require.NoError(t, err)
	})

	t.Run("should return error if there is an error fetching warp menu entries if warp tag does not exist ", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		const getWarpMenuErrorMessage = "error getting warpmenuentry"
		getErr := fmt.Errorf(getWarpMenuErrorMessage)

		doguDescriptor := getDoguDescriptor(defaultDoguName, displayName, doguCategory, "nonwarptag")
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptor, nil)
		client.EXPECT().Get(ctx, doguResource.Name, metav1.GetOptions{}).Return(nil, getErr)

		manager := &WarpMenuEntryManager{
			client:      client,
			doguFetcher: fetcher,
		}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)
		require.Error(t, err)
		assert.ErrorContains(t, err, "error updating getting the menu entry")
		assert.ErrorContains(t, err, getWarpMenuErrorMessage)
	})

	t.Run("Errorcase: should delete warp menu entries if warp tag does not exist and there is an error", func(t *testing.T) {
		fetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)
		//notFoundWarpMenuEntryError := &errors.StatusError{ErrStatus: metav1.Status{Reason: metav1.StatusReasonNotFound}}

		const deleteWarpMenuErrorMessage = "error getting warpmenuentry"
		deleteErr := fmt.Errorf(deleteWarpMenuErrorMessage)

		existingEntry := getExpectedWarpMenuEntry(defaultDoguName, doguResource, displayName, warpMenuDisabled, path, doguCategory)
		doguDescriptor := getDoguDescriptor(defaultDoguName, displayName, doguCategory, "nonwarptag")
		fetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptor, nil)
		client.EXPECT().Get(ctx, doguResource.Name, metav1.GetOptions{}).Return(existingEntry, nil)

		client.EXPECT().Delete(ctx, doguResource.Name, metav1.DeleteOptions{}).Return(deleteErr)
		manager := &WarpMenuEntryManager{
			client:      client,
			doguFetcher: fetcher,
		}

		err := manager.EnsureWarpMenuEntry(ctx, doguResource)

		require.Error(t, err)
		assert.ErrorContains(t, err, "error deleting the menu entry")
		assert.ErrorContains(t, err, deleteWarpMenuErrorMessage)
	})
}
