package imageregistry

import (
	"fmt"
	"sync"
	"testing"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
)

func configFor(id string) *imagev1.ConfigFile {
	return &imagev1.ConfigFile{Author: id}
}

func TestLruImageConfigCache_Add(t *testing.T) {
	t.Run("should store without evicting", func(t *testing.T) {
		sut := newLRUImageConfigCache(2)

		evicted := sut.Add("a", configFor("config-a"))
		assert.False(t, evicted)
		assert.Len(t, sut.items, 1)
		assert.Equal(t, 1, sut.order.Len())

		item, ok := sut.items["a"]
		assert.True(t, ok)

		entry, ok := item.Value.(*cacheEntry)
		assert.True(t, ok)
		assert.Equal(t, "config-a", entry.value.Author)

		assert.Equal(t, item, sut.order.Front())
	})

	t.Run("should evict the least-recently-used entry when capacity is exceeded", func(t *testing.T) {
		sut := newLRUImageConfigCache(2)
		sut.Add("a", configFor("config-a"))
		sut.Add("b", configFor("config-b"))

		// adding a third entry must evict "a" (the least recently used)
		evicted := sut.Add("c", configFor("config-c"))
		assert.True(t, evicted)

		_, okA := sut.items["a"]
		assert.False(t, okA)
		assert.Len(t, sut.items, 2)
		assert.Equal(t, 2, sut.order.Len())

		itemB, okB := sut.items["b"]
		assert.True(t, okB)
		itemC, okC := sut.items["c"]
		assert.True(t, okC)

		assert.Equal(t, itemC, sut.order.Front())
		assert.Equal(t, itemB, sut.order.Back())
	})

	t.Run("should keep a recently-read entry resident on eviction", func(t *testing.T) {
		sut := newLRUImageConfigCache(2)
		sut.Add("a", configFor("config-a"))
		sut.Add("b", configFor("config-b"))

		// reading "a" makes it most-recently-used, so adding "c" must evict "b"
		_, _ = sut.Get("a")
		sut.Add("c", configFor("config-c"))

		_, okA := sut.items["a"]
		assert.True(t, okA)
		_, okB := sut.items["b"]
		assert.False(t, okB)
	})
}

func TestLruImageConfigCache_Get(t *testing.T) {
	t.Run("should return not-ok for missing key", func(t *testing.T) {
		sut := newLRUImageConfigCache(2)

		value, ok := sut.Get("missing")

		assert.False(t, ok)
		assert.Nil(t, value)
	})

	t.Run("should store and return a value", func(t *testing.T) {
		sut := newLRUImageConfigCache(2)

		expected := configFor("config-a")
		evicted := sut.Add("a", expected)

		value, ok := sut.Get("a")
		assert.False(t, evicted)
		assert.True(t, ok)
		assert.Same(t, expected, value)
	})

	t.Run("should update the value and recency of an existing key without growing", func(t *testing.T) {
		sut := newLRUImageConfigCache(2)
		sut.Add("a", configFor("old"))

		updated := configFor("new")
		evicted := sut.Add("a", updated)

		value, ok := sut.Get("a")
		assert.False(t, evicted)
		assert.True(t, ok)
		assert.Same(t, updated, value)
		assert.Equal(t, 1, sut.order.Len())
	})
}

// run with -race to detect data races on the shared cache
func TestLruImageConfigCache_Concurrent(t *testing.T) {
	sut := newLRUImageConfigCache(50)

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("image:%d", n%25)
			sut.Add(key, configFor(key))
			_, _ = sut.Get(key)
		}(i)
	}
	wg.Wait()

	assert.LessOrEqual(t, sut.order.Len(), 50)
	assert.Equal(t, sut.order.Len(), len(sut.items))
}
