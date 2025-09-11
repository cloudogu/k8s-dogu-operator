package install

import (
	"testing"
	"time"

	v2 "github.com/cloudogu/k8s-dogu-lib/v2/api/v2"
	"github.com/stretchr/testify/assert"
)

func TestNewFinalizerExistsStep(t *testing.T) {
	t.Run("Successfully created step", func(t *testing.T) {
		step := NewFinalizerExistsStep()

		assert.NotNil(t, step)
	})
}

func TestFinalizerExistsStep_Run(t *testing.T) {
	t.Run("Successfully added finalizer", func(t *testing.T) {
		sut := NewFinalizerExistsStep()
		doguResource := &v2.Dogu{}

		result := sut.Run(testCtx, doguResource)

		assert.NotNil(t, sut)
		assert.Equal(t, true, result.Continue)
		assert.Equal(t, time.Duration(0), result.RequeueAfter)
		assert.Equal(t, nil, result.Err)
	})
}
