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

	t.Run("should create warp menu entries", func(t *testing.T) {
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

}
