package cachekit

import (
	"time"

	"github.com/benbjohnson/clock"
	lru "github.com/hashicorp/golang-lru/v2"
)

// entry is the positive-store value: the composed typed value plus the time after
// which it is considered stale and should be refreshed in the background. A zero
// refreshAfter means the entry never goes stale (load-once / mirror modes).
type entry[V any] struct {
	v            V
	refreshAfter time.Time
}

// LRUCache is a bounded, serve-stale read-through cache backed by
// hashicorp/golang-lru/v2. Entries are never evicted by time: past refreshAfter
// they are still returned (flagged stale) so the read path never blocks on the
// backend, and the engine refreshes them in the background. LRU capacity is the
// only eviction. It is safe for concurrent use.
type LRUCache[K comparable, V any] struct {
	lru   *lru.Cache[K, entry[V]]
	clock clock.Clock
}

// NewLRUCache builds an LRU cache holding up to maxEntries values.
func NewLRUCache[K comparable, V any](maxEntries int, clk clock.Clock) (*LRUCache[K, V], error) {
	if clk == nil {
		clk = clock.New()
	}
	l, err := lru.New[K, entry[V]](maxEntries)
	if err != nil {
		return nil, err
	}
	return &LRUCache[K, V]{lru: l, clock: clk}, nil
}

// Get returns the value if present, and whether it is stale (past its refresh
// time). Stale entries are still returned; the caller decides whether to trigger a
// background refresh.
func (c *LRUCache[K, V]) Get(key K) (V, bool, bool) {
	e, ok := c.lru.Get(key)
	if !ok {
		var zero V
		return zero, false, false
	}
	stale := !e.refreshAfter.IsZero() && c.clock.Now().After(e.refreshAfter)
	return e.v, true, stale
}

// Save stores v under key. ttl <= 0 means the entry never goes stale.
func (c *LRUCache[K, V]) Save(key K, v V, ttl time.Duration) {
	var refreshAfter time.Time
	if ttl > 0 {
		refreshAfter = c.clock.Now().Add(ttl)
	}
	c.lru.Add(key, entry[V]{v: v, refreshAfter: refreshAfter})
}

// Invalidate removes key from the cache if present.
func (c *LRUCache[K, V]) Invalidate(key K) {
	c.lru.Remove(key)
}

// NoCache is a pass-through cache: every Get is a miss and Save is a no-op. Paired
// with the engine's single-flight coalescing it yields "always fetch, still
// deduplicate" behaviour for direct-source / live tenants.
type NoCache[K comparable, V any] struct{}

func (NoCache[K, V]) Get(K) (V, bool, bool)    { var zero V; return zero, false, false }
func (NoCache[K, V]) Save(K, V, time.Duration) {}
func (NoCache[K, V]) Invalidate(K)             {}
