package install

import (
	"context"
	"errors"
	"testing"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewExpositionStep(t *testing.T) {
	step := NewExpositionStep(newMockExpositionManager(t), newMockServiceInterface(t), &config.OperatorConfig{ExpositionEnabled: true})
	assert.NotNil(t, step)
}

func TestExpositionStep_Run(t *testing.T) {
	testCtx := context.TODO()
	doguResource := &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}
	doguService := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test"}}

	t.Run("should requeue if manager returns an error", func(t *testing.T) {
		manager := newMockExpositionManager(t)
		serviceInterface := newMockServiceInterface(t)
		managerErr := errors.New("exposition not ready yet")
		serviceInterface.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(doguService, nil)
		manager.EXPECT().EnsureExposition(testCtx, doguResource, doguService).Return(managerErr)

		step := &ExpositionStep{expositionManager: manager, serviceInterface: serviceInterface, expositionEnabled: true}
		result := step.Run(testCtx, doguResource)
		assert.ErrorIs(t, result.Err, managerErr)
		assert.False(t, result.Continue)
	})

	t.Run("should continue on success", func(t *testing.T) {
		manager := newMockExpositionManager(t)
		serviceInterface := newMockServiceInterface(t)
		serviceInterface.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(doguService, nil)
		manager.EXPECT().EnsureExposition(testCtx, doguResource, doguService).Return(nil)

		step := &ExpositionStep{expositionManager: manager, serviceInterface: serviceInterface, expositionEnabled: true}
		result := step.Run(testCtx, doguResource)
		assert.NoError(t, result.Err)
		assert.True(t, result.Continue)
	})

	t.Run("should requeue if service lookup fails", func(t *testing.T) {
		manager := newMockExpositionManager(t)
		serviceInterface := newMockServiceInterface(t)
		serviceErr := errors.New("service missing")
		serviceInterface.EXPECT().Get(testCtx, "test", metav1.GetOptions{}).Return(nil, serviceErr)

		step := &ExpositionStep{expositionManager: manager, serviceInterface: serviceInterface, expositionEnabled: true}
		result := step.Run(testCtx, doguResource)
		require.Error(t, result.Err)
		assert.ErrorContains(t, result.Err, `failed to get dogu service for "test"`)
		assert.ErrorIs(t, result.Err, serviceErr)
		assert.False(t, result.Continue)
	})

	t.Run("should continue if exposition is disabled", func(t *testing.T) {
		step := &ExpositionStep{expositionManager: newMockExpositionManager(t), serviceInterface: newMockServiceInterface(t), expositionEnabled: false}
		result := step.Run(testCtx, doguResource)
		assert.NoError(t, result.Err)
		assert.True(t, result.Continue)
	})
}
