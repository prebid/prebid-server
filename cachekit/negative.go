package cachekit

import (
	"time"

	"github.com/benbjohnson/clock"
	lru "github.com/hashicorp/golang-lru/v2"
)

// negativeEntry is a cached definitive verdict: the error to re-serve and its
// expiry. The error is preserved (rather than collapsed to a generic marker) so
// callers see the real verdict — e.g. ErrNotFound vs a malformed-value error.
type negativeEntry struct {
	err     error
	expires time.Time
}

// NegativeStore is a small, bounded store of definitive verdicts (not-found or
// permanent malformed). It is kept separate from the positive cache with its own
// capacity and short TTL so an unknown-key flood can never evict real data. Only
// permanent verdicts belong here; transient failures are never stored. Safe for
// concurrent use.
type NegativeStore[K comparable] struct {
	lru   *lru.Cache[K, negativeEntry]
	ttl   time.Duration
	clock clock.Clock
}

// NewNegativeStore builds a negative store holding up to maxEntries verdicts, each
// retained for ttl.
func NewNegativeStore[K comparable](maxEntries int, ttl time.Duration, clk clock.Clock) (*NegativeStore[K], error) {
	if clk == nil {
		clk = clock.New()
	}
	l, err := lru.New[K, negativeEntry](maxEntries)
	if err != nil {
		return nil, err
	}
	return &NegativeStore[K]{lru: l, ttl: ttl, clock: clk}, nil
}

// isCached reports whether key has a live negative verdict, returning the verdict
// error to re-serve (e.g. ErrNotFound or a malformed-value error).
func (n *NegativeStore[K]) isCached(key K) (error, bool) {
	e, ok := n.lru.Get(key)
	if !ok {
		return nil, false
	}
	if n.clock.Now().After(e.expires) {
		n.lru.Remove(key)
		return nil, false
	}
	return e.err, true
}

// mark records a definitive verdict error for key, retained for the store TTL.
func (n *NegativeStore[K]) mark(key K, err error) {
	n.lru.Add(key, negativeEntry{err: err, expires: n.clock.Now().Add(n.ttl)})
}
