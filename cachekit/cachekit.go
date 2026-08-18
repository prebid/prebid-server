// Package cachekit is a small, generic read-through fetching engine shared by
// Prebid Server subsystems (accounts today; GVL / stored data / currency later).
//
// It is intentionally higher-level than any single subsystem: a subsystem picks
// a Source (where raw bytes come from), a Transform (raw bytes -> typed value),
// and a Cache (retention policy), and cachekit wires them together. When serve-stale
// is enabled, stale entries are served immediately and revalidated in the background;
// optional single-flight coalescing collapses concurrent misses for the same key
// into one upstream call per pod.
//
// The cache stores the composed typed value V, not raw JSON. Transform runs once
// per key at insert time; a cache hit is a pure lookup with no unmarshal.
//
// The package is split by concern:
//   - cachekit.go   — the engine (Params, Fetcher, Get, load, preload).
//   - contracts.go  — the interfaces a consumer implements (Source, Cache, ...).
//   - revalidate.go — the background serve-stale revalidation mechanism.
//   - cache.go      — the LRU / no-op positive cache implementations.
//   - negative.go   — the negative (definitive-verdict) store.
package cachekit

import (
	"context"
	"fmt"
	"time"

	"github.com/benbjohnson/clock"
	"golang.org/x/sync/singleflight"
)

// Params configures a Fetcher. Source, Transform and Cache are required.
type Params[K comparable, V any] struct {
	Source            Source[K]
	Transform         TransformFunc[K, V]
	Cache             Cache[K, V]
	TTL               time.Duration
	Negatives         *NegativeStore[K] // nil disables negative caching
	Coalesce          bool              // opt-in single-flight coalescing of concurrent misses (default off)
	ServeStale        bool              // opt-in: past TTL, serve the stale value and revalidate in the background (default off = expire + synchronous reload)
	RevalidateTimeout time.Duration     // maximum duration for a background stale revalidation; <= 0 uses a safe default
	Preload           BulkSource[K]     // if set, the whole corpus is fetched once at Start to warm the cache
	Clock             clock.Clock       // nil defaults to a real clock
	Metrics           Recorder          // nil defaults to a no-op recorder
}

// Fetcher is the generic read-through engine. Construct it with New.
type Fetcher[K comparable, V any] struct {
	source       Source[K]
	transform    TransformFunc[K, V]
	cache        Cache[K, V]
	ttl          time.Duration
	negatives    *NegativeStore[K]
	coalesce     bool
	serveStale   bool
	preload      BulkSource[K]
	clock        clock.Clock
	metrics      Recorder
	group        singleflight.Group
	reval        *revalidator[K]
	revalTimeout time.Duration
}

const defaultRevalidateTimeout = 10 * time.Second

// New builds a Fetcher from the given params.
func New[K comparable, V any](p Params[K, V]) *Fetcher[K, V] {
	if p.Clock == nil {
		p.Clock = clock.New()
	}
	if p.Metrics == nil {
		p.Metrics = noopRecorder{}
	}
	if p.RevalidateTimeout <= 0 {
		p.RevalidateTimeout = defaultRevalidateTimeout
	}
	return &Fetcher[K, V]{
		source:       p.Source,
		transform:    p.Transform,
		cache:        p.Cache,
		ttl:          p.TTL,
		negatives:    p.Negatives,
		coalesce:     p.Coalesce,
		serveStale:   p.ServeStale,
		preload:      p.Preload,
		clock:        p.Clock,
		metrics:      p.Metrics,
		reval:        newRevalidator[K](p.Clock, revalidateBackoff),
		revalTimeout: p.RevalidateTimeout,
	}
}

// Start warms the cache when a Preload source is configured: it fetches the whole
// corpus once and seeds it. It is a no-op otherwise. Preload is best-effort — if the
// bulk fetch fails, the cache is left cold and fills lazily via the read path.
// Callers should invoke it once after construction.
func (f *Fetcher[K, V]) Start(ctx context.Context) {
	if f.preload == nil {
		return
	}
	start := f.clock.Now()
	raw, err := f.preload.FetchAll(ctx)
	if err != nil {
		f.metrics.BackendFetch("error", f.clock.Now().Sub(start))
		return
	}
	f.metrics.BackendFetch("ok", f.clock.Now().Sub(start))
	for key, bytes := range raw {
		v, err := f.transform(key, bytes)
		if err != nil {
			continue // skip malformed entries; they surface on demand
		}
		f.cache.Save(key, v, f.ttl)
	}
}

// Close is a no-op; background revalidations are fire-and-forget goroutines.
func (f *Fetcher[K, V]) Close() {}

// Get returns the typed value for key. On a fresh cache hit it is a pure lookup
// with no upstream call and no unmarshal. Past TTL the behaviour depends on the
// ServeStale option: when enabled, the stale value is returned immediately and
// revalidated in the background (the read path never blocks on the backend); when
// disabled (default), a stale entry is treated as expired and reloaded
// synchronously. On a miss it fetches from the source; when coalescing is enabled,
// concurrent callers for the same key collapse into a single upstream fetch. It
// returns ErrNotFound when the key does not exist, and re-serves a cached verdict
// error (not-found or malformed) when negative caching is enabled.
func (f *Fetcher[K, V]) Get(ctx context.Context, key K) (V, error) {
	var zero V

	if v, ok, stale := f.cache.Get(key); ok {
		if !stale || f.serveStale {
			f.metrics.CacheHit()
			if stale {
				f.triggerRevalidate(key)
			}
			return v, nil
		}
		// Stale with serve-stale disabled: reload synchronously (classic TTL expiry).
		f.metrics.CacheMiss()
		return f.fetch(ctx, key)
	}
	if f.negatives != nil {
		if verr, ok := f.negatives.isCached(key); ok {
			f.metrics.CacheNegative()
			return zero, verr
		}
	}
	f.metrics.CacheMiss()
	return f.fetch(ctx, key)
}

// fetch loads the key from the source, applying single-flight coalescing when it is
// enabled so concurrent callers collapse into one upstream call.
func (f *Fetcher[K, V]) fetch(ctx context.Context, key K) (V, error) {
	if !f.coalesce {
		return f.load(ctx, key)
	}
	var zero V
	res, err, _ := f.group.Do(fmt.Sprint(key), func() (interface{}, error) {
		// Another goroutine may have filled the cache (fresh) while we waited.
		if v, ok, stale := f.cache.Get(key); ok && !stale {
			return v, nil
		}
		return f.load(ctx, key)
	})
	if err != nil {
		return zero, err
	}
	return res.(V), nil
}

// load performs a blocking upstream fetch, classification and cache insert for a
// cold key. Transient failures are never cached; definitive verdicts (not-found,
// malformed) are negative-cached with the real error when negative caching is on.
func (f *Fetcher[K, V]) load(ctx context.Context, key K) (V, error) {
	var zero V

	start := f.clock.Now()
	found, err := f.source.Fetch(ctx, []K{key})
	dur := f.clock.Now().Sub(start)

	if err != nil {
		// Systemic/transient failure: never cache, never negative-cache.
		f.metrics.BackendFetch("error", dur)
		return zero, err
	}
	raw, ok := found[key]
	if !ok {
		// Definitive per-key not-found.
		f.metrics.BackendFetch("notfound", dur)
		if f.negatives != nil {
			f.negatives.mark(key, ErrNotFound)
		}
		return zero, ErrNotFound
	}

	v, err := f.transform(key, raw)
	if err != nil {
		// Malformed value: a permanent verdict. Surface the error and, when negative
		// caching is on, remember it (error-preserving) so we re-serve the same
		// malformed error without re-hitting the backend for a short window.
		f.metrics.BackendFetch("error", dur)
		if f.negatives != nil {
			f.negatives.mark(key, err)
		}
		return zero, err
	}

	f.cache.Save(key, v, f.ttl)
	f.metrics.BackendFetch("ok", dur)
	return v, nil
}
