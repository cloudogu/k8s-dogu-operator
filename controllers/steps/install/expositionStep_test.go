package install

import (
	"context"
	"errors"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewExpositionStep(t *testing.T) {
	step := NewExpositionStep(newMockExpositionManager(t), &config.OperatorConfig{ExpositionEnabled: true})
	assert.NotNil(t, step)
}

func TestExpositionStep_Run(t *testing.T) {
	testCtx := context.TODO()
	doguResource := &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}

	t.Run("should requeue if manager returns an error", func(t *testing.T) {
		manager := newMockExpositionManager(t)
		managerErr := errors.New("exposition not ready yet")
		manager.On("EnsureExposition", testCtx, doguResource).Return(managerErr)

		step := &ExpositionStep{expositionManager: manager, expositionEnabled: true}
		result := step.Run(testCtx, doguResource)
		assert.ErrorIs(t, result.Err, managerErr)
		assert.False(t, result.Continue)
	})

	t.Run("should continue on success", func(t *testing.T) {
		manager := newMockExpositionManager(t)
		manager.On("EnsureExposition", testCtx, doguResource).Return(nil)

		step := &ExpositionStep{expositionManager: manager, expositionEnabled: true}
		result := step.Run(testCtx, doguResource)
		assert.NoError(t, result.Err)
		assert.True(t, result.Continue)
	})

	t.Run("should continue if exposition is disabled", func(t *testing.T) {
		step := &ExpositionStep{expositionManager: newMockExpositionManager(t), expositionEnabled: false}
		result := step.Run(testCtx, doguResource)
		assert.NoError(t, result.Err)
		assert.True(t, result.Continue)
	})
}
