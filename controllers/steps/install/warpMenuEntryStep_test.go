package install

import (
	"context"
	"errors"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewWarpMenuEntryStep(t *testing.T) {
	step := NewWarpMenuEntryStep(newMockWarpMenuEntryManager(t))
	assert.NotNil(t, step)
}

func TestWarpMenuEntryStep_Run(t *testing.T) {
	testCtx := context.TODO()
	doguResource := &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}

	t.Run("Should requeue if manager returns an error", func(t *testing.T) {
		manager := newMockWarpMenuEntryManager(t)
		managerErr := errors.New("warpMenuEntry not ready yet")
		manager.EXPECT().EnsureWarpMenuEntry(testCtx, doguResource).Return(managerErr)

		step := &WarpMenuEntryStep{warpmenuEntryManager: manager}
		result := step.Run(testCtx, doguResource)
		assert.ErrorIs(t, result.Err, managerErr)
		assert.False(t, result.Continue)
	})

	t.Run("should continue on success", func(t *testing.T) {
		manager := newMockWarpMenuEntryManager(t)
		manager.EXPECT().EnsureWarpMenuEntry(testCtx, doguResource).Return(nil)

		step := &WarpMenuEntryStep{warpmenuEntryManager: manager}
		result := step.Run(testCtx, doguResource)
		assert.NoError(t, result.Err)
		assert.True(t, result.Continue)
	})
}
