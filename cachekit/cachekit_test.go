package cachekit

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubSource is a test Source. It counts calls, can block (to exercise
// coalescing), and returns a fixed error when set.
type stubSource struct {
	calls int32
	data  map[string]json.RawMessage
	err   error
	block chan struct{}
}

func (s *stubSource) Fetch(_ context.Context, keys []string) (map[string]json.RawMessage, error) {
	atomic.AddInt32(&s.calls, 1)
	if s.block != nil {
		<-s.block
	}
	if s.err != nil {
		return nil, s.err
	}
	out := make(map[string]json.RawMessage, len(keys))
	for _, k := range keys {
		if v, ok := s.data[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}

func (s *stubSource) callCount() int { return int(atomic.LoadInt32(&s.calls)) }

type timeoutOnceSource struct {
	calls int32
	data  map[string]json.RawMessage
}

func (s *timeoutOnceSource) Fetch(ctx context.Context, keys []string) (map[string]json.RawMessage, error) {
	call := atomic.AddInt32(&s.calls, 1)
	if call == 2 {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	out := make(map[string]json.RawMessage, len(keys))
	for _, k := range keys {
		if v, ok := s.data[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}

func (s *timeoutOnceSource) callCount() int { return int(atomic.LoadInt32(&s.calls)) }

func identityTransform(_ string, raw json.RawMessage) (string, error) {
	return string(raw), nil
}

// bulkStub is a test BulkSource.
type bulkStub struct {
	data map[string]json.RawMessage
	err  error
}

func (b bulkStub) FetchAll(_ context.Context) (map[string]json.RawMessage, error) {
	return b.data, b.err
}

func TestPreloadSeedsCache(t *testing.T) {
	clk := clock.NewMock()
	cache, err := NewLRUCache[string, string](100, clk)
	require.NoError(t, err)
	readSrc := &stubSource{data: map[string]json.RawMessage{}} // read path source is empty
	f := New(Params[string, string]{
		Source:    readSrc,
		Transform: identityTransform,
		Cache:     cache,
		TTL:       time.Hour,
		Preload:   bulkStub{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}},
		Clock:     clk,
	})

	f.Start(context.Background())

	v, err := f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)
	assert.Equal(t, 0, readSrc.callCount(), "preloaded key should be served without hitting the read source")
}

func newLRUFetcher(t *testing.T, src Source[string], clk clock.Clock, ttl time.Duration, negatives *NegativeStore[string]) *Fetcher[string, string] {
	t.Helper()
	cache, err := NewLRUCache[string, string](100, clk)
	require.NoError(t, err)
	return New(Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Cache:     cache,
		TTL:       ttl,
		Negatives: negatives,
		Clock:     clk,
	})
}

func newServeStaleFetcher(t *testing.T, src Source[string], clk clock.Clock, ttl time.Duration) *Fetcher[string, string] {
	t.Helper()
	cache, err := NewLRUCache[string, string](100, clk)
	require.NoError(t, err)
	return New(Params[string, string]{
		Source:     src,
		Transform:  identityTransform,
		Cache:      cache,
		TTL:        ttl,
		ServeStale: true,
		Clock:      clk,
	})
}

// TestGetExpiresAndReloadsByDefault verifies the default (serve-stale off): past
// TTL the entry is treated as expired and reloaded synchronously on the next read.
func TestGetExpiresAndReloadsByDefault(t *testing.T) {
	clk := clock.NewMock()
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	f := newLRUFetcher(t, src, clk, time.Hour, nil)

	_, err := f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, 1, src.callCount())

	// Still fresh.
	clk.Add(30 * time.Minute)
	_, err = f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, 1, src.callCount())

	// Past TTL with serve-stale off: the read reloads synchronously and returns fresh.
	clk.Add(2 * time.Hour)
	src.data["a"] = json.RawMessage(`v2`)
	v, err := f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v2", v, "a stale entry must be reloaded synchronously, returning the fresh value")
	assert.Equal(t, 2, src.callCount())
}

func TestGetHitAfterMiss(t *testing.T) {
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	f := newLRUFetcher(t, src, clock.NewMock(), time.Hour, nil)

	v, err := f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)

	v, err = f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)

	assert.Equal(t, 1, src.callCount(), "second Get should be served from cache")
}

func TestGetNotFoundWithNegativeCache(t *testing.T) {
	clk := clock.NewMock()
	neg, err := NewNegativeStore[string](10, time.Minute, clk)
	require.NoError(t, err)
	src := &stubSource{data: map[string]json.RawMessage{}}
	f := newLRUFetcher(t, src, clk, time.Hour, neg)

	_, err = f.Get(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)

	_, err = f.Get(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)

	assert.Equal(t, 1, src.callCount(), "negative cache should prevent a second backend call")
}

func TestGetNotFoundWithoutNegativeCache(t *testing.T) {
	src := &stubSource{data: map[string]json.RawMessage{}}
	f := newLRUFetcher(t, src, clock.NewMock(), time.Hour, nil)

	_, err := f.Get(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
	_, err = f.Get(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)

	assert.Equal(t, 2, src.callCount(), "without negative cache each miss hits the backend")
}

func TestGetCoalescesConcurrentMisses(t *testing.T) {
	src := &stubSource{
		data:  map[string]json.RawMessage{"a": json.RawMessage(`v1`)},
		block: make(chan struct{}),
	}
	cache, err := NewLRUCache[string, string](100, clock.NewMock())
	require.NoError(t, err)
	f := New(Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Cache:     cache,
		TTL:       time.Hour,
		Coalesce:  true,
	})

	const n = 8
	var wg sync.WaitGroup
	results := make([]string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			v, err := f.Get(context.Background(), "a")
			assert.NoError(t, err)
			results[idx] = v
		}(i)
	}

	// Give the goroutines time to converge on the single in-flight call, then release it.
	time.Sleep(50 * time.Millisecond)
	close(src.block)
	wg.Wait()

	for _, r := range results {
		assert.Equal(t, "v1", r)
	}
	assert.Equal(t, 1, src.callCount(), "concurrent misses should collapse into a single backend call")
}

func TestGetServesStaleAndRefreshesInBackground(t *testing.T) {
	clk := clock.NewMock()
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	f := newServeStaleFetcher(t, src, clk, time.Hour)

	// Cold load.
	v, err := f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)
	assert.Equal(t, 1, src.callCount())

	// Still fresh: no extra fetch.
	clk.Add(30 * time.Minute)
	_, err = f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, 1, src.callCount())

	// Past TTL: the read returns the stale value immediately and refreshes in the
	// background (never blocks). The backend value changes so we can observe it.
	clk.Add(2 * time.Hour)
	src.data["a"] = json.RawMessage(`v2`)
	v, err = f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", v, "stale read must return the last good value, not block")

	// The background refresh eventually re-fetches and updates the cache.
	assert.Eventually(t, func() bool { return src.callCount() == 2 }, time.Second, 5*time.Millisecond,
		"a stale read should trigger exactly one background refresh")
	assert.Eventually(t, func() bool {
		got, _ := f.Get(context.Background(), "a")
		return got == "v2"
	}, time.Second, 5*time.Millisecond, "the refreshed value should become visible")
}

func TestGetServesStaleWhileBackendDown(t *testing.T) {
	clk := clock.NewMock()
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	f := newServeStaleFetcher(t, src, clk, time.Hour)

	// Warm the cache.
	_, err := f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, 1, src.callCount())

	// Backend goes down and the entry goes stale.
	src.err = errors.New("backend down")
	clk.Add(2 * time.Hour)

	// Reads keep returning the last good value; failed refresh backs off so the
	// backend is not hammered.
	for i := 0; i < 5; i++ {
		v, gErr := f.Get(context.Background(), "a")
		require.NoError(t, gErr)
		assert.Equal(t, "v1", v)
	}
	assert.Eventually(t, func() bool { return src.callCount() >= 2 }, time.Second, 5*time.Millisecond)
	assert.LessOrEqual(t, src.callCount(), 2, "failed refreshes must back off, not storm the backend")
}

func TestBackgroundRevalidationTimeoutReleasesSlot(t *testing.T) {
	clk := clock.NewMock()
	src := &timeoutOnceSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	cache, err := NewLRUCache[string, string](100, clk)
	require.NoError(t, err)
	f := New(Params[string, string]{
		Source:            src,
		Transform:         identityTransform,
		Cache:             cache,
		TTL:               time.Hour,
		ServeStale:        true,
		RevalidateTimeout: 10 * time.Millisecond,
		Clock:             clk,
	})

	v, err := f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)

	clk.Add(2 * time.Hour)
	src.data["a"] = json.RawMessage(`v2`)
	v, err = f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)
	assert.Eventually(t, func() bool { return src.callCount() == 2 }, time.Second, 5*time.Millisecond)
	assert.Eventually(t, func() bool {
		f.reval.mu.Lock()
		defer f.reval.mu.Unlock()
		st := f.reval.state["a"]
		return !st.inFlight && !st.failedAt.IsZero()
	}, time.Second, 5*time.Millisecond)

	clk.Add(revalidateBackoff + time.Second)
	v, err = f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)
	assert.Eventually(t, func() bool { return src.callCount() == 3 }, time.Second, 5*time.Millisecond)
	assert.Eventually(t, func() bool {
		got, _ := f.Get(context.Background(), "a")
		return got == "v2"
	}, time.Second, 5*time.Millisecond)
}

func TestRevalidatorPrunesExpiredFailures(t *testing.T) {
	clk := clock.NewMock()
	r := newRevalidator[string](clk, revalidateBackoff)
	r.finish("old", true)
	require.Contains(t, r.state, "old")

	clk.Add(revalidateBackoff + time.Second)
	assert.True(t, r.begin("new"))
	assert.NotContains(t, r.state, "old")
	assert.True(t, r.state["new"].inFlight)
}

func TestGetNoCacheAlwaysFetches(t *testing.T) {
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	f := New(Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Cache:     NoCache[string, string]{},
		TTL:       time.Hour,
	})

	for i := 0; i < 3; i++ {
		v, err := f.Get(context.Background(), "a")
		require.NoError(t, err)
		assert.Equal(t, "v1", v)
	}
	assert.Equal(t, 3, src.callCount())
}

func TestGetSourceErrorNotCached(t *testing.T) {
	boom := errors.New("boom")
	src := &stubSource{err: boom}
	f := newLRUFetcher(t, src, clock.NewMock(), time.Hour, nil)

	_, err := f.Get(context.Background(), "a")
	assert.ErrorIs(t, err, boom)
	assert.NotErrorIs(t, err, ErrNotFound)

	_, err = f.Get(context.Background(), "a")
	assert.ErrorIs(t, err, boom)
	assert.Equal(t, 2, src.callCount(), "systemic errors must not be cached")
}

func TestGetTransformErrorNotCached(t *testing.T) {
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	cache, err := NewLRUCache[string, string](100, clock.NewMock())
	require.NoError(t, err)
	transformErr := errors.New("bad value")
	f := New(Params[string, string]{
		Source:    src,
		Transform: func(string, json.RawMessage) (string, error) { return "", transformErr },
		Cache:     cache,
		TTL:       time.Hour,
	})

	_, err = f.Get(context.Background(), "a")
	assert.ErrorIs(t, err, transformErr)
	_, err = f.Get(context.Background(), "a")
	assert.ErrorIs(t, err, transformErr)
	assert.Equal(t, 2, src.callCount(), "malformed values must not be cached")
}

// countingRecorder verifies the engine emits the expected telemetry.
type countingRecorder struct {
	hits, misses, negatives int
	backend                 map[string]int
}

func newCountingRecorder() *countingRecorder {
	return &countingRecorder{backend: map[string]int{}}
}

func (r *countingRecorder) CacheHit()                                   { r.hits++ }
func (r *countingRecorder) CacheMiss()                                  { r.misses++ }
func (r *countingRecorder) CacheNegative()                              { r.negatives++ }
func (r *countingRecorder) BackendFetch(result string, _ time.Duration) { r.backend[result]++ }

func TestRecorderSignals(t *testing.T) {
	clk := clock.NewMock()
	neg, err := NewNegativeStore[string](10, time.Minute, clk)
	require.NoError(t, err)
	cache, err := NewLRUCache[string, string](100, clk)
	require.NoError(t, err)
	rec := newCountingRecorder()
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	f := New(Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Cache:     cache,
		TTL:       time.Hour,
		Negatives: neg,
		Clock:     clk,
		Metrics:   rec,
	})

	// miss -> ok, then hit.
	_, _ = f.Get(context.Background(), "a")
	_, _ = f.Get(context.Background(), "a")
	// miss -> notfound, then negative.
	_, _ = f.Get(context.Background(), "missing")
	_, _ = f.Get(context.Background(), "missing")

	assert.Equal(t, 1, rec.hits)
	assert.Equal(t, 2, rec.misses)
	assert.Equal(t, 1, rec.negatives)
	assert.Equal(t, 1, rec.backend["ok"])
	assert.Equal(t, 1, rec.backend["notfound"])
}
