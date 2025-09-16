package initfx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_noSpamKey(t *testing.T) {
	t.Run("should not return the same key", func(t *testing.T) {
		// when
		first := noSpamKey(nil)
		second := noSpamKey(nil)

		// then
		assert.NotEqual(t, first, second)
	})
}

func Test_noAggregationKey(t *testing.T) {
	t.Run("should not return the same keys", func(t *testing.T) {
		// when
		firstAggregateKey, firstLocalKey := noAggregationKey(nil)
		secondAggregateKey, secondLocalKey := noAggregationKey(nil)

		// then
		assert.NotEqual(t, firstAggregateKey, secondAggregateKey)
		assert.NotEqual(t, firstLocalKey, secondLocalKey)
	})
}
