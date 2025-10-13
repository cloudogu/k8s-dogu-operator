package steps

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRequeueAfter(t *testing.T) {
	t.Run("should return requeue after time", func(t *testing.T) {

		result := RequeueAfter(time.Second * 3)

		assert.Equal(t, time.Second*3, result.RequeueAfter)
		assert.Equal(t, nil, result.Err)
		assert.Equal(t, false, result.Continue)
	})
}

func TestContinue(t *testing.T) {
	t.Run("should return continue true", func(t *testing.T) {

		result := Continue()

		assert.Equal(t, time.Duration(0), result.RequeueAfter)
		assert.Equal(t, nil, result.Err)
		assert.Equal(t, true, result.Continue)
	})
}

func TestAbort(t *testing.T) {
	t.Run("should return continue false", func(t *testing.T) {

		result := Abort()

		assert.Equal(t, time.Duration(0), result.RequeueAfter)
		assert.Equal(t, nil, result.Err)
		assert.Equal(t, false, result.Continue)
	})
}

func TestRequeueWithError(t *testing.T) {
	t.Run("should return error", func(t *testing.T) {
		err := errors.New("test error")
		result := RequeueWithError(err)

		assert.Equal(t, time.Duration(0), result.RequeueAfter)
		assert.Equal(t, false, result.Continue)
		assert.NotEqual(t, nil, result.Err)
		assert.ErrorContains(t, result.Err, "test error")
	})
}
