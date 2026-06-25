package warpmenuentry

import (
	"context"
	"testing"

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
		assert.ErrorContains(t, err, "doguresource must not be nil")
	})
}
