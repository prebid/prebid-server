package fetcher

import "errors"

// NegativeStore wraps a cache of definitive verdicts (not-found or permanent
// malformed). It is kept separate from the positive cache so an unknown-key
// flood can never evict real data. Only permanent verdicts belong here;
// transient failures are never stored. Safe for concurrent use when the wrapped
// cache is safe for concurrent use.
type NegativeStore[K comparable] struct {
	cache Cache[K, error]
}

// NewNegativeStore builds a negative store around the supplied cache.
func NewNegativeStore[K comparable](cache Cache[K, error]) (*NegativeStore[K], error) {
	if cache == nil {
		return nil, errors.New("negative cache is required")
	}
	return &NegativeStore[K]{cache: cache}, nil
}

// isCached reports whether key has a live negative verdict, returning the verdict
// error to re-serve (e.g. NotFoundError with key or a malformed-value error).
func (n *NegativeStore[K]) isCached(key K) (error, bool) {
	err, ok, stale := n.cache.Get(key)
	if !ok {
		return nil, false
	}
	if stale {
		n.cache.Invalidate(key)
		return nil, false
	}
	return err, true
}

// mark records a definitive verdict error for key.
func (n *NegativeStore[K]) mark(key K, err error) {
	n.cache.Save(key, err)
}
