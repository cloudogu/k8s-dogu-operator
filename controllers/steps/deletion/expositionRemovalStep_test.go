package deletion

import (
	"context"
	"testing"

	cescommons "github.com/cloudogu/ces-commons-lib/dogu"
	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/cloudogu/k8s-dogu-operator/v3/controllers/steps"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewExpositionRemoverStep(t *testing.T) {
	step := NewExpositionRemoverStep(newMockExpositionManager(t))
	assert.NotNil(t, step)
}

func TestExpositionRemoverStep_Run(t *testing.T) {
	testCtx := context.TODO()
	doguResource := &v2.Dogu{ObjectMeta: metav1.ObjectMeta{Name: "test"}}

	t.Run("should requeue on manager error", func(t *testing.T) {
		manager := newMockExpositionManager(t)
		manager.On("RemoveExposition", testCtx, cescommons.SimpleName("test")).Return(assert.AnError)
		step := &ExpositionRemoverStep{expositionManager: manager}
		assert.Equal(t, steps.RequeueWithError(assert.AnError), step.Run(testCtx, doguResource))
	})

	t.Run("should continue on success", func(t *testing.T) {
		manager := newMockExpositionManager(t)
		manager.On("RemoveExposition", testCtx, cescommons.SimpleName("test")).Return(nil)
		step := &ExpositionRemoverStep{expositionManager: manager}
		assert.Equal(t, steps.Continue(), step.Run(testCtx, doguResource))
	})
}
