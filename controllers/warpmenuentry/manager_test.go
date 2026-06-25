package warpmenuentry

import (
	"context"
	"fmt"
	"testing"

	cesappcore "github.com/cloudogu/cesapp-lib/core"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		doguDescriptor := &cesappcore.Dogu{}
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

}

func newDoguResource(doguName string) *v2.Dogu {
	return &v2.Dogu{
		Spec: v2.DoguSpec{
			Name: doguName,
		},
	}
}
