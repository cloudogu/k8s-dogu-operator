package doguv3

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRequeueAfter(t *testing.T) {
	t.Run("should return requeue after time", func(t *testing.T) {

		result := RequeueAfter(time.Second*3, "reason", "message")

		assert.Equal(t, time.Second*3, result.RequeueAfter)
		assert.Equal(t, "reason", result.ReadyReason)
		assert.Equal(t, "message", result.ReadyMessage)
		assert.NoError(t, result.Err)
		assert.False(t, result.Continue)
	})
}

func TestContinue(t *testing.T) {
	t.Run("should return continue true", func(t *testing.T) {

		result := Continue()

		assert.Equal(t, time.Duration(0), result.RequeueAfter)
		assert.NoError(t, result.Err)
		assert.True(t, result.Continue)
	})
}

func TestAbort(t *testing.T) {
	t.Run("should return continue false", func(t *testing.T) {

		result := Abort("reason", "message")

		assert.Equal(t, time.Duration(0), result.RequeueAfter)
		assert.Equal(t, "reason", result.ReadyReason)
		assert.Equal(t, "message", result.ReadyMessage)
		assert.NoError(t, result.Err)
		assert.False(t, result.Continue)
	})
}

func TestRequeueWithError(t *testing.T) {
	t.Run("should return error", func(t *testing.T) {
		result := RequeueWithError(assert.AnError, "reason")

		assert.Equal(t, time.Duration(0), result.RequeueAfter)
		assert.Equal(t, "reason", result.ReadyReason)
		assert.Equal(t, assert.AnError.Error(), result.ReadyMessage)
		assert.False(t, result.Continue)
		assert.ErrorContains(t, result.Err, assert.AnError.Error())
	})
}
