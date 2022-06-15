// Package lru contains a simple implementation of a LRU cache.
// It is inspired by the Cache type at https://github.com/hashicorp/golang-lru.
package lru

import (
	"sync"

	"go.lepak.sg/playground/lmap"
)

const (
	DefaultCacheMax = 100
)

// Cache is a LRU cache. It is safe for concurrent use.
type Cache[K comparable, V any] struct {
	l   *lmap.LinkedMap[K, V]
	mu  sync.Mutex
	max int
}

// New creates a new Cache ready for use.
// If max > 0, the cache will only be allowed to contain max number of entries.
// Otherwise a default maximum number will be used.
func New[K comparable, V any](max int) *Cache[K, V] {
	if max <= 0 {
		max = DefaultCacheMax
	}

	return &Cache[K, V]{
		l:   lmap.New[K, V](),
		max: max,
	}
}

// Add adds a key-value pair to the cache.
// If a cache entry is evicted as a result, Add returns true.
// If the type of key is inconsistent with previous cache additions,
// Add panics.
func (c *Cache[K, V]) Add(key K, value V) (evicted bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, ok := c.l.Get(key, false)

	if !ok && c.max > 0 && c.l.Len() >= c.max {
		evicted = true
		_, _, _ = c.l.Head(true)
	}

	c.l.Set(key, value, true)
	return
}

// Get reads a value from the cache.
// If the key was not found, ok will be false.
func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.l.Get(key, true)
}

// Peek is like Get, but it will not affect the recency of the key.
func (c *Cache[K, V]) Peek(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.l.Get(key, false)
}

// Len returns the number of elements in the cache.
func (c *Cache[_, _]) Len() int {
	c.mu.Lock()
	l := c.l.Len()
	c.mu.Unlock()
	return l
}

// Trim removes least recently used elements from the cache
// so that its size is at most max elements.
func (c *Cache[_, _]) Trim(max int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for c.l.Len() > max {
		_, _, _ = c.l.Head(true)
	}
}

// Keys returns the keys in the cache, ordered by increasing recency.
func (c *Cache[K, V]) Keys() []K {
	c.mu.Lock()
	defer c.mu.Unlock()

	ks := make([]K, 0, c.l.Len())
	i := c.l.Iterator()
	for i.Next() {
		k, _ := i.Entry()
		ks = append(ks, k)
	}

	return ks
}
