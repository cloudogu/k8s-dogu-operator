package imageregistry

import (
	"container/list"
	"sync"

	imagev1 "github.com/google/go-containerregistry/pkg/v1"
)

// lruImageConfigCache is a small, thread-safe, fixed-capacity LRU cache for image config files keyed by the
// image reference ("name:tag").
//
// It exists to avoid pulling the same image config from the registry on every reconcile (e.g. while a dogu is
// stuck retrying, or because several reconcile steps inspect the same image config). An image reference is
// immutable by convention - released dogu versions are never re-pushed, and development builds publish a unique
// tag per build - so entries never need time-based invalidation. The cache is bounded to keep the operator's
// memory footprint small. The least-recently-used entry is evicted once the capacity is exceeded, which keeps an
// actively-reconciled image resident while stale old versions age out.
//
// The cached *imagev1.ConfigFile must be treated as read-only by callers: it is shared across all callers of a
// cache hit and mutating it would corrupt the cache and other callers' view.
type lruImageConfigCache struct {
	mu       sync.Mutex
	capacity int
	// order holds *cacheEntry values with the most-recently-used entry at the front.
	order *list.List
	items map[string]*list.Element
}

type cacheEntry struct {
	key   string
	value *imagev1.ConfigFile
}

// newLRUImageConfigCache creates an LRU cache that holds at most capacity entries. The caller must ensure capacity > 0.
func newLRUImageConfigCache(capacity int) *lruImageConfigCache {
	return &lruImageConfigCache{
		capacity: capacity,
		order:    list.New(),
		items:    make(map[string]*list.Element, capacity),
	}
}

// Get returns the cached image config for key and whether the key was present, marking the entry as most-recently-used.
func (c *lruImageConfigCache) Get(key string) (value *imagev1.ConfigFile, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.items[key]
	if !ok {
		return nil, false
	}

	c.order.MoveToFront(element)

	entry, _ := element.Value.(*cacheEntry)
	return entry.value, true
}

// Add stores value under key as the most-recently-used entry and reports whether an entry was evicted to stay within
// capacity. Adding an existing key updates its value and refreshes its recency.
func (c *lruImageConfigCache) Add(key string, value *imagev1.ConfigFile) (evicted bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.items[key]; ok {
		entry, _ := element.Value.(*cacheEntry)
		entry.value = value
		c.order.MoveToFront(element)

		return false
	}

	c.items[key] = c.order.PushFront(&cacheEntry{key: key, value: value})

	if c.order.Len() > c.capacity {
		c.removeOldest()
		return true
	}

	return false
}

// removeOldest evicts the least-recently-used entry. The caller must hold c.mu.
func (c *lruImageConfigCache) removeOldest() {
	oldest := c.order.Back()
	if oldest == nil {
		return
	}

	c.order.Remove(oldest)
	entry, _ := oldest.Value.(*cacheEntry)

	delete(c.items, entry.key)
}
