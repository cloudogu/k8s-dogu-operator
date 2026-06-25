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
	doguName = "testdogu"
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

	t.Run("Should return error when doguresource is nil", func(t *testing.T) {

		manager := &WarpMenuEntryManager{doguFetcher: nil}

		err := manager.EnsureWarpMenuEntry(ctx, nil)
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
		doguDescriptor := newDoguDescriptor(true)
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		const getWarpMenuEntryError = "get warp menu entry error"

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptor, nil)
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
		doguDescriptor := newDoguDescriptor(true)
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		const createWarpMenuEntryError = "create warp menu entry error"

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptor, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(nil, newNotFoundError())
		client.EXPECT().Create(ctx, newWarpMenuEntry(), v1.CreateOptions{}).Return(nil, fmt.Errorf(createWarpMenuEntryError))
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
		doguDescriptor := newDoguDescriptor(true)
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptor, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(nil, newNotFoundError())
		client.EXPECT().Create(ctx, newWarpMenuEntry(), v1.CreateOptions{}).Return(newWarpMenuEntry(), nil)
		manager := &WarpMenuEntryManager{doguFetcher: doguFetcher, client: client}

		//when
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)
		//then
		require.NoError(t, err)
	})

	t.Run("Should not create a new warp menu entry if dogu has no warp tag", func(t *testing.T) {
		//given
		doguDescriptor := newDoguDescriptor(false)
		doguFetcher := newMockLocalDoguFetcher(t)
		client := newMockWarpmenuentryClient(t)

		doguFetcher.EXPECT().FetchForResource(ctx, doguResource).Return(doguDescriptor, nil)
		client.EXPECT().Get(ctx, doguName, v1.GetOptions{}).Return(nil, newNotFoundError())
		manager := &WarpMenuEntryManager{doguFetcher: doguFetcher, client: client}

		//when
		err := manager.EnsureWarpMenuEntry(ctx, doguResource)
		//then
		require.NoError(t, err)
	})
}

func newDoguDescriptor(withWarpTag bool) *cesappcore.Dogu {
	var tags []string
	if withWarpTag {
		tags = append(tags, "warp")
	}
	return &cesappcore.Dogu{
		Name: doguName,
		Tags: tags,
	}
}

func newWarpMenuEntry() *warpMenuEntryV1.WarpMenuEntry {
	return &warpMenuEntryV1.WarpMenuEntry{}
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
