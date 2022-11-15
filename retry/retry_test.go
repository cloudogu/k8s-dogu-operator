package retry

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_OnErrorRetry(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		maxTries := 2
		fn := func() error {
			println(fmt.Sprintf("Current time: %s", time.Now()))
			return nil
		}

		// when
		err := OnError(maxTries, AlwaysRetryFunc, fn)

		// then
		require.NoError(t, err)
	})
	t.Run("should fail", func(t *testing.T) {
		// given
		maxTries := 2
		fn := func() error {
			println(fmt.Sprintf("Current time: %s", time.Now()))
			return assert.AnError
		}

		// when
		err := OnError(maxTries, AlwaysRetryFunc, fn)

		// then
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestTestableRetrierError(t *testing.T) {
	sut := new(TestableRetrierError)
	sut.Err = assert.AnError
	require.Error(t, sut)
	assert.ErrorContains(t, sut, assert.AnError.Error())
}

func Test_TestableRetryFunc(t *testing.T) {
	assert.False(t, TestableRetryFunc(nil))
	assert.False(t, TestableRetryFunc(assert.AnError))
	retrierErr := new(TestableRetrierError)
	retrierErr.Err = assert.AnError
	assert.True(t, TestableRetryFunc(retrierErr))
}
