package fetcher

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	fetchercache "github.com/prebid/prebid-server/v4/fetcher/cache"
	fetchersource "github.com/prebid/prebid-server/v4/fetcher/source"
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

type fakeTime struct {
	now time.Time
}

func newFakeTime() *fakeTime {
	return &fakeTime{now: time.Unix(1000, 0)}
}

func (f *fakeTime) Now() time.Time {
	return f.now
}

func (f *fakeTime) Add(d time.Duration) {
	f.now = f.now.Add(d)
}

func (s *stubSource) Fetch(_ context.Context, key string) (json.RawMessage, bool, error) {
	atomic.AddInt32(&s.calls, 1)
	if s.block != nil {
		<-s.block
	}
	if s.err != nil {
		return nil, false, s.err
	}
	v, ok := s.data[key]
	if !ok {
		return nil, false, nil
	}
	return v, true, nil
}

func (s *stubSource) callCount() int { return int(atomic.LoadInt32(&s.calls)) }

type timeoutOnceSource struct {
	calls int32
	data  map[string]json.RawMessage
}

func (s *timeoutOnceSource) Fetch(ctx context.Context, key string) (json.RawMessage, bool, error) {
	call := atomic.AddInt32(&s.calls, 1)
	if call == 2 {
		<-ctx.Done()
		return nil, false, ctx.Err()
	}
	v, ok := s.data[key]
	if !ok {
		return nil, false, nil
	}
	return v, true, nil
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

func (b bulkStub) Fetch(_ context.Context, key string) (json.RawMessage, bool, error) {
	if b.err != nil {
		return nil, false, b.err
	}
	raw, ok := b.data[key]
	return raw, ok, nil
}

func TestPreloadSeedsCache(t *testing.T) {
	clk := newFakeTime()
	src := bulkStub{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	f, err := New(Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Config: Config{
			Cache:   CacheConfig{Type: "lru", MaxEntries: 100, TTL: time.Hour},
			Refresh: RefreshConfig{Mode: "preload"},
		},
		Time:    clk,
		Metrics: NoopRecorder{},
	})
	require.NoError(t, err)

	require.NoError(t, f.Start(context.Background()))

	v, err := f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)
}

func TestPreloadTransformErrorSurfaces(t *testing.T) {
	src := bulkStub{data: map[string]json.RawMessage{"bad": json.RawMessage(`v1`)}}
	transformErr := errors.New("bad value")
	f, err := New(Params[string, string]{
		Source:    src,
		Transform: func(string, json.RawMessage) (string, error) { return "", transformErr },
		Config: Config{
			Cache:   CacheConfig{Type: "lru", MaxEntries: 100, TTL: time.Hour},
			Refresh: RefreshConfig{Mode: "preload"},
		},
		Time:    newFakeTime(),
		Metrics: NoopRecorder{},
	})
	require.NoError(t, err)

	err = f.Start(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "preload transform failed for key bad")
	assert.ErrorIs(t, err, transformErr)
}

func TestNewRequiresTimeAndMetrics(t *testing.T) {
	src := &stubSource{data: map[string]json.RawMessage{}}
	params := Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Config:    Config{Cache: CacheConfig{Type: "none"}},
		Metrics:   NoopRecorder{},
	}

	_, err := New(params)
	require.EqualError(t, err, "time is required")

	params.Time = newFakeTime()
	params.Metrics = nil
	_, err = New(params)
	require.EqualError(t, err, "metrics recorder is required")
}

func newLRUFetcher(t *testing.T, src fetchersource.Source[string], clk *fakeTime, ttl time.Duration, negatives *NegativeStore[string]) *Fetcher[string, string] {
	t.Helper()
	cfg := Config{
		Cache: CacheConfig{Type: "lru", MaxEntries: 100, TTL: ttl},
	}
	if negatives != nil {
		cfg.Negative.Enabled = true
		cfg.Negative.Type = "lru"
		cfg.Negative.MaxEntries = 10
		cfg.Negative.TTL = time.Minute
	}
	f, err := New(Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Config:    cfg,
		Time:      clk,
		Metrics:   NoopRecorder{},
	})
	require.NoError(t, err)
	if negatives != nil {
		f.negatives = negatives
	}
	return f
}

func newServeStaleFetcher(t *testing.T, src fetchersource.Source[string], clk *fakeTime, ttl time.Duration) *Fetcher[string, string] {
	t.Helper()
	f, err := New(Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Config:    Config{Cache: CacheConfig{Type: "lru", MaxEntries: 100, TTL: ttl}, Refresh: RefreshConfig{ServeStale: true}},
		Time:      clk,
		Metrics:   NoopRecorder{},
	})
	require.NoError(t, err)
	return f
}

// TestGetExpiresAndReloadsByDefault verifies the default (serve-stale off): past
// TTL the entry is treated as expired and reloaded synchronously on the next read.
func TestGetExpiresAndReloadsByDefault(t *testing.T) {
	clk := newFakeTime()
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
	f := newLRUFetcher(t, src, newFakeTime(), time.Hour, nil)

	v, err := f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)

	v, err = f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)

	assert.Equal(t, 1, src.callCount(), "second Get should be served from cache")
}

func TestGetNotFoundWithNegativeCache(t *testing.T) {
	clk := newFakeTime()
	negativeCache, err := fetchercache.NewLRUCache[string, error](10, time.Minute, clk)
	require.NoError(t, err)
	neg, err := NewNegativeStore[string](negativeCache)
	require.NoError(t, err)
	src := &stubSource{data: map[string]json.RawMessage{}}
	f := newLRUFetcher(t, src, clk, time.Hour, neg)

	_, err = f.Get(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.EqualError(t, err, "fetcher: key missing not found")

	_, err = f.Get(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.EqualError(t, err, "fetcher: key missing not found")

	assert.Equal(t, 1, src.callCount(), "negative cache should prevent a second backend call")
}

func TestGetNotFoundWithoutNegativeCache(t *testing.T) {
	src := &stubSource{data: map[string]json.RawMessage{}}
	f := newLRUFetcher(t, src, newFakeTime(), time.Hour, nil)

	_, err := f.Get(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.EqualError(t, err, "fetcher: key missing not found")
	_, err = f.Get(context.Background(), "missing")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.EqualError(t, err, "fetcher: key missing not found")

	assert.Equal(t, 2, src.callCount(), "without negative cache each miss hits the backend")
}

func TestGetCoalescesConcurrentMisses(t *testing.T) {
	src := &stubSource{
		data:  map[string]json.RawMessage{"a": json.RawMessage(`v1`)},
		block: make(chan struct{}),
	}
	f, err := New(Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Config:    Config{Cache: CacheConfig{Type: "lru", MaxEntries: 100, TTL: time.Hour}, CoalesceRequests: true},
		Time:      newFakeTime(),
		Metrics:   NoopRecorder{},
	})
	require.NoError(t, err)

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
	clk := newFakeTime()
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
	clk := newFakeTime()
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
	clk := newFakeTime()
	src := &timeoutOnceSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	f, err := New(Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Config: Config{
			Cache:   CacheConfig{Type: "lru", MaxEntries: 100, TTL: time.Hour},
			Refresh: RefreshConfig{ServeStale: true, BackgroundRefreshTimeout: 10 * time.Millisecond},
		},
		Time:    clk,
		Metrics: NoopRecorder{},
	})
	require.NoError(t, err)

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
		f.backgroundRefresh.mu.Lock()
		defer f.backgroundRefresh.mu.Unlock()
		st := f.backgroundRefresh.state["a"]
		return !st.inFlight && !st.failedAt.IsZero()
	}, time.Second, 5*time.Millisecond)

	clk.Add(defaultBackgroundRefreshBackoff + time.Second)
	v, err = f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)
	assert.Eventually(t, func() bool { return src.callCount() == 3 }, time.Second, 5*time.Millisecond)
	assert.Eventually(t, func() bool {
		got, _ := f.Get(context.Background(), "a")
		return got == "v2"
	}, time.Second, 5*time.Millisecond)
}

func TestBackgroundRefreshCoordinatorPrunesExpiredFailures(t *testing.T) {
	clk := newFakeTime()
	r := newBackgroundRefreshCoordinator[string](clk, defaultBackgroundRefreshBackoff)
	r.finish("old", true)
	require.Contains(t, r.state, "old")

	clk.Add(defaultBackgroundRefreshBackoff + time.Second)
	assert.True(t, r.begin("new"))
	assert.NotContains(t, r.state, "old")
	assert.True(t, r.state["new"].inFlight)
}

func TestGetNilCacheAlwaysFetches(t *testing.T) {
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	f, err := New(Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Config:    Config{Cache: CacheConfig{Type: "none", TTL: time.Hour}},
		Time:      newFakeTime(),
		Metrics:   NoopRecorder{},
	})
	require.NoError(t, err)

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
	f := newLRUFetcher(t, src, newFakeTime(), time.Hour, nil)

	_, err := f.Get(context.Background(), "a")
	assert.ErrorIs(t, err, boom)
	assert.NotErrorIs(t, err, ErrNotFound)

	_, err = f.Get(context.Background(), "a")
	assert.ErrorIs(t, err, boom)
	assert.Equal(t, 2, src.callCount(), "systemic errors must not be cached")
}

func TestGetTransformErrorNotCached(t *testing.T) {
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	transformErr := errors.New("bad value")
	f, err := New(Params[string, string]{
		Source:    src,
		Transform: func(string, json.RawMessage) (string, error) { return "", transformErr },
		Config:    Config{Cache: CacheConfig{Type: "lru", MaxEntries: 100, TTL: time.Hour}},
		Time:      newFakeTime(),
		Metrics:   NoopRecorder{},
	})
	require.NoError(t, err)

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

func (r *countingRecorder) CacheHit()      { r.hits++ }
func (r *countingRecorder) CacheMiss()     { r.misses++ }
func (r *countingRecorder) CacheNegative() { r.negatives++ }
func (r *countingRecorder) BackendFetch(operation string, result string, _ time.Duration) {
	r.backend[operation+":"+result]++
}

func TestRecorderSignals(t *testing.T) {
	clk := newFakeTime()
	negativeCache, err := fetchercache.NewLRUCache[string, error](10, time.Minute, clk)
	require.NoError(t, err)
	neg, err := NewNegativeStore[string](negativeCache)
	require.NoError(t, err)
	rec := newCountingRecorder()
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	f, err := New(Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Config:    Config{Cache: CacheConfig{Type: "lru", MaxEntries: 100, TTL: time.Hour}},
		Time:      clk,
		Metrics:   rec,
	})
	require.NoError(t, err)
	f.negatives = neg

	// miss -> ok, then hit.
	_, _ = f.Get(context.Background(), "a")
	_, _ = f.Get(context.Background(), "a")
	// miss -> notfound, then negative.
	_, _ = f.Get(context.Background(), "missing")
	_, _ = f.Get(context.Background(), "missing")

	assert.Equal(t, 1, rec.hits)
	assert.Equal(t, 2, rec.misses)
	assert.Equal(t, 1, rec.negatives)
	assert.Equal(t, 1, rec.backend["get:ok"])
	assert.Equal(t, 1, rec.backend["get:notfound"])
}
