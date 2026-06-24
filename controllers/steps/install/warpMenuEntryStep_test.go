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
	testctx := context.TODO()
	doguResource := &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}

	t.Run("Should continue on success", func(t *testing.T) {
		manager := newMockWarpMenuEntryManager(t)
		manager.EXPECT().EnsureWarpMenuEntry(testctx, doguResource).Return(nil)

		step := &WarpMenuEntryStep{warpMenuEntryManager: manager}
		result := step.Run(testctx, doguResource)
		assert.NoError(t, result.Err)
		assert.True(t, result.Continue)
	})
	t.Run("should requeue if the manager returns an error", func(t *testing.T) {
		manager := newMockWarpMenuEntryManager(t)
		managerErr := errors.New("crd not found")
		manager.EXPECT().EnsureWarpMenuEntry(testctx, doguResource).Return(managerErr)

		step := &WarpMenuEntryStep{warpMenuEntryManager: manager}
		result := step.Run(testctx, doguResource)
		assert.ErrorIs(t, result.Err, managerErr)
		assert.False(t, result.Continue)

	})

}
