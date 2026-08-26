package cache

import (
	"errors"
	"sync"
	"time"

	"github.com/prebid/prebid-server/v4/util/timeutil"
)

// UnboundedCache stores every saved value until it is explicitly invalidated.
// It supports the same stale marker as LRUCache, but does not evict by size.
type UnboundedCache[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]entry[V]
	ttl  time.Duration
	time timeutil.Time
}

// NewUnboundedCache builds an unbounded cache.
func NewUnboundedCache[K comparable, V any](ttl time.Duration, t timeutil.Time) (*UnboundedCache[K, V], error) {
	if t == nil {
		return nil, errors.New("time is required")
	}
	return &UnboundedCache[K, V]{
		data: make(map[K]entry[V]),
		ttl:  ttl,
		time: t,
	}, nil
}

// Get returns the value if present, and whether it is stale (past its refresh
// time). Stale entries are still returned; the caller decides whether to trigger a
// background refresh.
func (c *UnboundedCache[K, V]) Get(key K) (V, bool, bool) {
	c.mu.RLock()
	e, ok := c.data[key]
	c.mu.RUnlock()
	if !ok {
		var zero V
		return zero, false, false
	}
	stale := !e.refreshAfter.IsZero() && c.time.Now().After(e.refreshAfter)
	return e.v, true, stale
}

// Save stores v under key. A cache ttl <= 0 means the entry never goes stale.
func (c *UnboundedCache[K, V]) Save(key K, v V) {
	var refreshAfter time.Time
	if c.ttl > 0 {
		refreshAfter = c.time.Now().Add(c.ttl)
	}
	c.mu.Lock()
	c.data[key] = entry[V]{v: v, refreshAfter: refreshAfter}
	c.mu.Unlock()
}

// Invalidate removes key from the cache if present.
func (c *UnboundedCache[K, V]) Invalidate(key K) {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
}
