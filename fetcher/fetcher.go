// Package fetcher provides a generic read-through fetching engine.
package fetcher

import (
	"context"
	"errors"
	"fmt"
	"time"

	fetchersource "github.com/prebid/prebid-server/v4/fetcher/source"
	"github.com/prebid/prebid-server/v4/util/timeutil"
	"golang.org/x/sync/singleflight"
)

// Params configures a Fetcher. Source and Transform are required.
type Params[K comparable, V any] struct {
	Source    fetchersource.Source[K]
	Transform TransformFunc[K, V]
	Config    Config
	Time      timeutil.Time
	Metrics   Recorder
}

// Fetcher is the generic read-through engine. Construct it with New.
type Fetcher[K comparable, V any] struct {
	source              fetchersource.Source[K]
	transform           TransformFunc[K, V]
	cache               Cache[K, V]
	negatives           *NegativeStore[K]
	coalesceRequests    bool
	refreshInBackground bool
	preload             fetchersource.BulkSource[K]
	time                timeutil.Time
	metrics             Recorder
	requestCoalescer    singleflight.Group
	backgroundRefresh   *backgroundRefreshCoordinator[K]
	refreshTimeout      time.Duration
}

const defaultBackgroundRefreshTimeout = 10 * time.Second
const defaultBackgroundRefreshBackoff = 5 * time.Second

// New builds a Fetcher from the given params.
func New[K comparable, V any](p Params[K, V]) (*Fetcher[K, V], error) {
	if p.Time == nil {
		return nil, errors.New("time is required")
	}
	if p.Metrics == nil {
		return nil, errors.New("metrics recorder is required")
	}
	cache, err := buildCache[K, V](effectiveCacheConfig(p.Config), p.Time)
	if err != nil {
		return nil, err
	}
	negatives, err := buildNegativeStore[K](p.Config.Negative, p.Time)
	if err != nil {
		return nil, err
	}
	refreshInBackground, preload, err := applyRefreshConfig(p.Config.Refresh, p.Source)
	if err != nil {
		return nil, err
	}
	if p.Config.Refresh.BackgroundRefreshTimeout <= 0 {
		p.Config.Refresh.BackgroundRefreshTimeout = defaultBackgroundRefreshTimeout
	}
	if p.Config.Refresh.BackgroundRefreshBackoff <= 0 {
		p.Config.Refresh.BackgroundRefreshBackoff = defaultBackgroundRefreshBackoff
	}

	return &Fetcher[K, V]{
		source:              p.Source,
		transform:           p.Transform,
		cache:               cache,
		negatives:           negatives,
		coalesceRequests:    p.Config.CoalesceRequests,
		refreshInBackground: refreshInBackground,
		preload:             preload,
		time:                p.Time,
		metrics:             p.Metrics,
		backgroundRefresh:   newBackgroundRefreshCoordinator[K](p.Time, p.Config.Refresh.BackgroundRefreshBackoff),
		refreshTimeout:      p.Config.Refresh.BackgroundRefreshTimeout,
	}, nil
}

// Start warms the cache when a Preload source is configured: it fetches the
// whole corpus once and seeds it. It is a no-op otherwise. A preload fetch or
// transform error is returned so startup can surface bad source data immediately.
func (f *Fetcher[K, V]) Start(ctx context.Context) error {
	if f.preload == nil {
		return nil
	}
	start := f.time.Now()
	raw, err := f.preload.FetchAll(ctx)
	if err != nil {
		f.metrics.BackendFetch("start", "error", f.time.Now().Sub(start))
		return err
	}
	f.metrics.BackendFetch("start", "ok", f.time.Now().Sub(start))
	var errs []error
	for key, bytes := range raw {
		v, err := f.transform(key, bytes)
		if err != nil {
			errs = append(errs, fmt.Errorf("preload transform failed for key %v: %w", key, err))
			continue
		}
		f.cache.Save(key, v)
	}
	return errors.Join(errs...)
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
// returns a NotFoundError when the key does not exist, and re-serves a cached
// verdict error (not-found or malformed) when negative caching is enabled.
func (f *Fetcher[K, V]) Get(ctx context.Context, key K) (V, error) {
	var zero V

	if v, ok, stale := f.cache.Get(key); ok {
		if !stale || f.refreshInBackground {
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
	if !f.coalesceRequests {
		return f.load(ctx, key)
	}
	var zero V
	res, err, _ := f.requestCoalescer.Do(fmt.Sprint(key), func() (interface{}, error) {
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

	start := f.time.Now()
	raw, found, err := f.source.Fetch(ctx, key)
	dur := f.time.Now().Sub(start)

	if err != nil {
		// Systemic/transient failure: never cache, never negative-cache.
		f.metrics.BackendFetch("get", "error", dur)
		return zero, err
	}
	if !found {
		// Definitive per-key not-found.
		err := NotFoundError{Key: key}
		f.metrics.BackendFetch("get", "notfound", dur)
		if f.negatives != nil {
			f.negatives.mark(key, err)
		}
		return zero, err
	}

	v, err := f.transform(key, raw)
	if err != nil {
		// Malformed value: a permanent verdict. Surface the error and, when negative
		// caching is on, remember it (error-preserving) so we re-serve the same
		// malformed error without re-hitting the backend for a short window.
		f.metrics.BackendFetch("get", "error", dur)
		if f.negatives != nil {
			f.negatives.mark(key, err)
		}
		return zero, err
	}

	f.cache.Save(key, v)
	f.metrics.BackendFetch("get", "ok", dur)
	return v, nil
}
