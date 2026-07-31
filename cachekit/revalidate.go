package cachekit

import (
	"context"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

// revalidateBackoff is how long a key waits after a failed background revalidation
// before another is attempted, so a struggling backend is not hammered.
const revalidateBackoff = 5 * time.Second

// revalState is a key's background-revalidation state: whether one is in flight
// and, if the last one failed, when (so the next attempt can back off).
type revalState struct {
	inFlight bool
	failedAt time.Time
}

// revalidator serialises background revalidations per key — at most one in flight,
// with a backoff after a failure. It owns its own lock so callers never touch it.
type revalidator[K comparable] struct {
	mu      sync.Mutex
	state   map[K]revalState
	backoff time.Duration
	clock   clock.Clock
}

func newRevalidator[K comparable](clk clock.Clock, backoff time.Duration) *revalidator[K] {
	return &revalidator[K]{state: make(map[K]revalState), backoff: backoff, clock: clk}
}

// begin reports whether the caller may start a revalidation for key now, and claims
// the in-flight slot if so. It returns false when one is already running or the last
// attempt failed within the backoff window.
func (r *revalidator[K]) begin(key K) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := r.state[key]
	if st.inFlight || (!st.failedAt.IsZero() && r.clock.Now().Before(st.failedAt.Add(r.backoff))) {
		return false
	}
	st.inFlight = true
	r.state[key] = st
	return true
}

// finish releases the in-flight slot; on failure it records the time (for backoff),
// on success it forgets the key entirely.
func (r *revalidator[K]) finish(key K, failed bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if failed {
		r.state[key] = revalState{failedAt: r.clock.Now()}
	} else {
		delete(r.state, key)
	}
}

// triggerRevalidate starts one background revalidation for a stale key. It never
// blocks the caller: the revalidator admits at most one per key at a time and backs
// off after failures, so a struggling backend is not hammered and stale values keep
// being served.
func (f *Fetcher[K, V]) triggerRevalidate(key K) {
	if !f.reval.begin(key) {
		return
	}
	go f.revalidate(context.Background(), key)
}

// revalidate reloads a stale key in the background. It never worsens availability:
// on success it replaces the value; if the key is gone upstream it is dropped (and
// negative-cached); on any error (transient or a newly-malformed value) the last
// good value keeps being served and a backoff is recorded.
func (f *Fetcher[K, V]) revalidate(ctx context.Context, key K) {
	start := f.clock.Now()
	found, err := f.source.Fetch(ctx, []K{key})
	dur := f.clock.Now().Sub(start)

	if err != nil {
		f.metrics.BackendFetch("error", dur)
		f.reval.finish(key, true)
		return
	}
	raw, ok := found[key]
	if !ok {
		// Deleted upstream: drop it so the next read reflects the deletion.
		f.metrics.BackendFetch("notfound", dur)
		f.cache.Invalidate(key)
		if f.negatives != nil {
			f.negatives.mark(key, ErrNotFound)
		}
		f.reval.finish(key, false)
		return
	}
	v, err := f.transform(key, raw)
	if err != nil {
		// Newly-malformed upstream value: keep serving the last good value.
		f.metrics.BackendFetch("error", dur)
		f.reval.finish(key, true)
		return
	}
	f.cache.Save(key, v, f.ttl)
	f.metrics.BackendFetch("ok", dur)
	f.reval.finish(key, false)
}
