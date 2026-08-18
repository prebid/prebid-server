package cachekit

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ErrNotFound is returned by Get when the key does not exist at the source.
// It is a definitive, per-key verdict (as opposed to a systemic/backend error).
var ErrNotFound = errors.New("cachekit: not found")

// Source pulls raw, undecoded bytes for a batch of keys. A single-key lookup is
// a one-element slice. By convention, a key that is absent from the returned map
// is treated as a definitive "not found" for that key; a non-nil error is a
// systemic failure (never cached, never negative-cached).
type Source[K comparable] interface {
	Fetch(ctx context.Context, keys []K) (map[K]json.RawMessage, error)
}

// BulkSource is an optional capability a Source may implement to return the entire
// corpus in a single call. It is what the engine's preload warm-up uses to fill the
// cache at startup; sources that cannot enumerate everything simply do not implement it.
type BulkSource[K comparable] interface {
	FetchAll(ctx context.Context) (map[K]json.RawMessage, error)
}

// TransformFunc converts raw bytes into the typed value V. It runs exactly once
// per key, at cache insert time. The key is provided so transforms that need it
// (e.g. filling an ID) can do so without mutating the shared cached value later.
type TransformFunc[K comparable, V any] func(key K, raw json.RawMessage) (V, error)

// Cache is a keyed store of composed typed values. Implementations must be safe
// for concurrent use.
type Cache[K comparable, V any] interface {
	// Get returns the value if present, and whether it is stale (past its refresh
	// time). Stale values are still returned so the read path never blocks on the
	// backend; the engine revalidates them in the background.
	Get(key K) (v V, ok bool, stale bool)
	Save(key K, v V, ttl time.Duration)
	// Invalidate drops a key so the next Get is a miss. Used when a background
	// revalidation finds the key was deleted upstream.
	Invalidate(key K)
}

// Recorder receives low-cardinality telemetry. The subsystem label is applied by
// the implementation, not passed per call, to keep cardinality bounded.
type Recorder interface {
	CacheHit()
	CacheMiss()
	CacheNegative()
	// BackendFetch reports one upstream call. result is "ok", "notfound" or "error".
	BackendFetch(result string, d time.Duration)
}

// noopRecorder is the default Recorder.
type noopRecorder struct{}

func (noopRecorder) CacheHit()                          {}
func (noopRecorder) CacheMiss()                         {}
func (noopRecorder) CacheNegative()                     {}
func (noopRecorder) BackendFetch(string, time.Duration) {}
