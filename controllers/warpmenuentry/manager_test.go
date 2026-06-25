package warpmenuentry

import (
	"context"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	t.Run("Should Create new test manager", func(t *testing.T) {
		var manager Manager
		manager = *NewWarpMenuEntryManager()
		require.NotNil(t, manager)
	})
}

func TestEnsureWarpMenuEntry(t *testing.T) {
	ctx := context.Background()

	t.Run("Should return error when doguresource is nil", func(t *testing.T) {
		manager := NewWarpMenuEntryManager()
		err := manager.EnsureWarpMenuEntry(ctx, nil)
		require.Error(t, err)
		assert.ErrorContains(t, err, "doguResource must not be nil")
	})

	t.Run("Should return error when dogu spec cannot be retrieved", func(t *testing.T) {
		manager := NewWarpMenuEntryManager()
		err := manager.EnsureWarpMenuEntry(ctx, newDoguResource("unknown"))
		require.Error(t, err)
		assert.ErrorContains(t, err, "dogu spec cannot be retrieved")
	})

}

func newDoguResource(doguName string) *v2.Dogu {
	return &v2.Dogu{
		Spec: v2.DoguSpec{
			Name: doguName,
		},
	}
}
