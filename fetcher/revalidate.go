package fetcher

import (
	"context"
	"sync"
	"time"

	"github.com/prebid/prebid-server/v4/util/timeutil"
)

// backgroundRefreshState is a key's background-refresh state: whether one is running
// and, if the last one failed, when (so the next attempt can back off).
type backgroundRefreshState struct {
	inFlight bool
	failedAt time.Time
}

// backgroundRefreshCoordinator coordinates background refreshes per key. For a given key, it
// allows only one refresh goroutine to run at a time. If that refresh fails, the
// same key waits for the configured backoff before another refresh can start.
// It owns its own lock so callers never touch the state map directly.
type backgroundRefreshCoordinator[K comparable] struct {
	mu      sync.Mutex
	state   map[K]backgroundRefreshState
	backoff time.Duration
	time    timeutil.Time
}

func newBackgroundRefreshCoordinator[K comparable](t timeutil.Time, backoff time.Duration) *backgroundRefreshCoordinator[K] {
	return &backgroundRefreshCoordinator[K]{state: make(map[K]backgroundRefreshState), backoff: backoff, time: t}
}

// begin reports whether a background refresh may start for key now. It returns
// true only if no refresh is already running for that key and the key is not
// waiting after a recent failed refresh.
func (r *backgroundRefreshCoordinator[K]) begin(key K) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.time.Now()
	r.pruneExpiredFailures(now)
	st := r.state[key]
	if st.inFlight || (!st.failedAt.IsZero() && r.time.Now().Before(st.failedAt.Add(r.backoff))) {
		return false
	}
	st.inFlight = true
	r.state[key] = st
	return true
}

func (r *backgroundRefreshCoordinator[K]) pruneExpiredFailures(now time.Time) {
	for key, st := range r.state {
		if !st.inFlight && !st.failedAt.IsZero() && !now.Before(st.failedAt.Add(r.backoff)) {
			delete(r.state, key)
		}
	}
}

// finish records the result of a background refresh for key. On failure it keeps
// the failure time so the key backs off before retrying; on success it removes
// the key from the revalidation state.
func (r *backgroundRefreshCoordinator[K]) finish(key K, failed bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if failed {
		r.state[key] = backgroundRefreshState{failedAt: r.time.Now()}
	} else {
		delete(r.state, key)
	}
}

// triggerRevalidate starts one background refresh for a stale key. It never
// blocks the caller: the coordinator admits at most one per key at a time and backs
// off after failures, so a struggling backend is not hammered and stale values keep
// being served.
func (f *Fetcher[K, V]) triggerRevalidate(key K) {
	if !f.backgroundRefresh.begin(key) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), f.refreshTimeout)
	go f.revalidate(ctx, cancel, key)
}

// revalidate reloads a stale key in the background. It never worsens availability:
// on success it replaces the value; if the key is gone upstream it is dropped (and
// negative-cached); on any error (transient or a newly-malformed value) the last
// good value keeps being served and a backoff is recorded.
func (f *Fetcher[K, V]) revalidate(ctx context.Context, cancel context.CancelFunc, key K) {
	defer cancel()
	start := f.time.Now()
	raw, found, err := f.source.Fetch(ctx, key)
	dur := f.time.Now().Sub(start)

	if err != nil {
		f.metrics.BackendFetch("background_refresh", "error", dur)
		f.backgroundRefresh.finish(key, true)
		return
	}
	if !found {
		// Deleted upstream: drop it so the next read reflects the deletion.
		err := NotFoundError{Key: key}
		f.metrics.BackendFetch("background_refresh", "notfound", dur)
		f.cache.Invalidate(key)
		if f.negatives != nil {
			f.negatives.mark(key, err)
		}
		f.backgroundRefresh.finish(key, false)
		return
	}
	v, err := f.transform(key, raw)
	if err != nil {
		// Newly-malformed upstream value: keep serving the last good value.
		f.metrics.BackendFetch("background_refresh", "error", dur)
		f.backgroundRefresh.finish(key, true)
		return
	}
	f.cache.Save(key, v)
	f.metrics.BackendFetch("background_refresh", "ok", dur)
	f.backgroundRefresh.finish(key, false)
}
