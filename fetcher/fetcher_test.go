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
	"github.com/prebid/prebid-server/v4/metrics"
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

type secondLookupHitCache struct {
	gets  atomic.Int32
	value string
}

func (c *secondLookupHitCache) Get(string) (string, bool, bool) {
	if c.gets.Add(1) == 1 {
		return "", false, false
	}
	return c.value, true, false
}

func (c *secondLookupHitCache) Save(string, string) {}

func (c *secondLookupHitCache) Invalidate(string) {}

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

func TestStartPreloadsValuesIntoCache(t *testing.T) {
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
		Metrics: NilRecorder{},
	})
	require.NoError(t, err)

	require.NoError(t, f.Start(context.Background()))

	v, err := f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", v)
}

func TestStartReturnsPreloadTransformErrors(t *testing.T) {
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
		Metrics: NilRecorder{},
	})
	require.NoError(t, err)

	err = f.Start(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "preload transform failed for key bad")
	assert.ErrorIs(t, err, transformErr)
}

func TestStartReturnsPreloadSourceError(t *testing.T) {
	preloadErr := errors.New("preload failed")
	f, err := New(Params[string, string]{
		Source: bulkStub{
			data: map[string]json.RawMessage{},
			err:  preloadErr,
		},
		Transform: identityTransform,
		Config: Config{
			Cache:   CacheConfig{Type: "lru", MaxEntries: 100, TTL: time.Hour},
			Refresh: RefreshConfig{Mode: "preload"},
		},
		Time:    newFakeTime(),
		Metrics: NilRecorder{},
	})
	require.NoError(t, err)

	err = f.Start(context.Background())

	assert.ErrorIs(t, err, preloadErr)
}

func TestNewRejectsMissingTimeOrMetricsRecorder(t *testing.T) {
	src := &stubSource{data: map[string]json.RawMessage{}}
	params := Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Config:    Config{Cache: CacheConfig{Type: "none"}},
		Metrics:   NilRecorder{},
	}

	_, err := New(params)
	require.EqualError(t, err, "time is required")

	params.Time = newFakeTime()
	params.Metrics = nil
	_, err = New(params)
	require.EqualError(t, err, "metrics recorder is required")
}

func newLRUFetcher(t *testing.T, src Source[string], clk *fakeTime, ttl time.Duration, negatives *NegativeStore[string]) *Fetcher[string, string] {
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
		Metrics:   NilRecorder{},
	})
	require.NoError(t, err)
	if negatives != nil {
		f.negatives = negatives
	}
	return f
}

func newServeStaleFetcher(t *testing.T, src Source[string], clk *fakeTime, ttl time.Duration) *Fetcher[string, string] {
	t.Helper()
	f, err := New(Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Config:    Config{Cache: CacheConfig{Type: "lru", MaxEntries: 100, TTL: ttl}, Refresh: RefreshConfig{ServeStale: true}},
		Time:      clk,
		Metrics:   NilRecorder{},
	})
	require.NoError(t, err)
	return f
}

// TestGetExpiresAndReloadsByDefault verifies the default (serve-stale off): past
// TTL the entry is treated as expired and reloaded synchronously on the next read.
func TestGetReloadsExpiredValueWhenStaleServingIsDisabled(t *testing.T) {
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

func TestGetCachesSuccessfulBackendResult(t *testing.T) {
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

func TestGetUsesNegativeCacheForRepeatedMissingKey(t *testing.T) {
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

func TestNegativeStoreInvalidatesStaleVerdict(t *testing.T) {
	clk := newFakeTime()
	negativeCache, err := fetchercache.NewLRUCache[string, error](10, time.Minute, clk)
	require.NoError(t, err)
	negatives, err := NewNegativeStore[string](negativeCache)
	require.NoError(t, err)
	verdict := NotFoundError{Key: "missing"}
	negatives.mark("missing", verdict)

	got, found := negatives.isCached("missing")
	require.True(t, found)
	assert.ErrorIs(t, got, ErrNotFound)

	clk.Add(2 * time.Minute)
	got, found = negatives.isCached("missing")
	assert.False(t, found)
	assert.NoError(t, got)
	_, found, _ = negativeCache.Get("missing")
	assert.False(t, found, "stale negative verdict should be removed from the cache")
}

func TestGetRefetchesRepeatedMissingKeyWhenNegativeCacheIsDisabled(t *testing.T) {
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
		Metrics:   NilRecorder{},
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

func TestGetUsesValueCachedBeforeSingleflightLoad(t *testing.T) {
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`backend`)}}
	f, err := New(Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Config:    Config{Cache: CacheConfig{Type: "none"}, CoalesceRequests: true},
		Time:      newFakeTime(),
		Metrics:   NilRecorder{},
	})
	require.NoError(t, err)
	cache := &secondLookupHitCache{value: "cached"}
	f.cache = cache

	value, err := f.Get(context.Background(), "a")

	require.NoError(t, err)
	assert.Equal(t, "cached", value)
	assert.Equal(t, int32(2), cache.gets.Load())
	assert.Equal(t, 0, src.callCount(), "the singleflight cache recheck should avoid a backend fetch")
}

func TestGetReturnsCoalescedBackendError(t *testing.T) {
	backendErr := errors.New("backend error")
	src := &stubSource{err: backendErr}
	f, err := New(Params[string, string]{
		Source:    src,
		Transform: identityTransform,
		Config:    Config{Cache: CacheConfig{Type: "none"}, CoalesceRequests: true},
		Time:      newFakeTime(),
		Metrics:   NilRecorder{},
	})
	require.NoError(t, err)

	value, err := f.Get(context.Background(), "a")

	assert.Empty(t, value)
	assert.ErrorIs(t, err, backendErr)
	assert.Equal(t, 1, src.callCount())
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

func TestBackgroundRevalidationInvalidatesValueDeletedBySource(t *testing.T) {
	clk := newFakeTime()
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	negativeCache, err := fetchercache.NewLRUCache[string, error](10, time.Minute, clk)
	require.NoError(t, err)
	negatives, err := NewNegativeStore[string](negativeCache)
	require.NoError(t, err)
	f := newServeStaleFetcher(t, src, clk, time.Hour)
	f.negatives = negatives

	value, err := f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", value)

	delete(src.data, "a")
	clk.Add(2 * time.Hour)
	value, err = f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", value)

	assert.Eventually(t, func() bool {
		verdict, found := negatives.isCached("a")
		return found && errors.Is(verdict, ErrNotFound)
	}, time.Second, 5*time.Millisecond)

	_, err = f.Get(context.Background(), "a")
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, 2, src.callCount(), "the negative verdict should prevent another backend fetch")
}

func TestBackgroundRevalidationKeepsStaleValueOnTransformError(t *testing.T) {
	clk := newFakeTime()
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	transformErr := errors.New("bad refreshed value")
	var failTransform atomic.Bool
	f, err := New(Params[string, string]{
		Source: src,
		Transform: func(_ string, raw json.RawMessage) (string, error) {
			if failTransform.Load() {
				return "", transformErr
			}
			return string(raw), nil
		},
		Config: Config{
			Cache:   CacheConfig{Type: "lru", MaxEntries: 100, TTL: time.Hour},
			Refresh: RefreshConfig{ServeStale: true},
		},
		Time:    clk,
		Metrics: NilRecorder{},
	})
	require.NoError(t, err)

	value, err := f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", value)

	failTransform.Store(true)
	clk.Add(2 * time.Hour)
	value, err = f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", value)

	assert.Eventually(t, func() bool {
		f.backgroundRefresh.mu.Lock()
		defer f.backgroundRefresh.mu.Unlock()
		state, found := f.backgroundRefresh.state["a"]
		return found && !state.inFlight && !state.failedAt.IsZero()
	}, time.Second, 5*time.Millisecond)

	calls := src.callCount()
	value, err = f.Get(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, "v1", value)
	assert.Equal(t, calls, src.callCount(), "failed refreshes should back off while the stale value remains available")
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
		Metrics: NilRecorder{},
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
		Metrics:   NilRecorder{},
	})
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		v, err := f.Get(context.Background(), "a")
		require.NoError(t, err)
		assert.Equal(t, "v1", v)
	}
	assert.Equal(t, 3, src.callCount())
}

func TestGetRetriesBackendErrors(t *testing.T) {
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

func TestGetRetriesTransformErrorWhenNegativeCacheIsDisabled(t *testing.T) {
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	transformErr := errors.New("bad value")
	f, err := New(Params[string, string]{
		Source:    src,
		Transform: func(string, json.RawMessage) (string, error) { return "", transformErr },
		Config:    Config{Cache: CacheConfig{Type: "lru", MaxEntries: 100, TTL: time.Hour}},
		Time:      newFakeTime(),
		Metrics:   NilRecorder{},
	})
	require.NoError(t, err)

	_, err = f.Get(context.Background(), "a")
	assert.ErrorIs(t, err, transformErr)
	_, err = f.Get(context.Background(), "a")
	assert.ErrorIs(t, err, transformErr)
	assert.Equal(t, 2, src.callCount(), "malformed values must not be cached")
}

func TestGetCachesTransformErrorWhenNegativeCacheIsEnabled(t *testing.T) {
	src := &stubSource{data: map[string]json.RawMessage{"a": json.RawMessage(`v1`)}}
	transformErr := errors.New("bad value")
	f, err := New(Params[string, string]{
		Source:    src,
		Transform: func(string, json.RawMessage) (string, error) { return "", transformErr },
		Config: Config{
			Cache: CacheConfig{Type: "lru", MaxEntries: 100, TTL: time.Hour},
			Negative: NegativeConfig{
				Enabled:    true,
				Type:       "lru",
				MaxEntries: 10,
				TTL:        time.Minute,
			},
		},
		Time:    newFakeTime(),
		Metrics: NilRecorder{},
	})
	require.NoError(t, err)

	_, err = f.Get(context.Background(), "a")
	assert.ErrorIs(t, err, transformErr)
	_, err = f.Get(context.Background(), "a")
	assert.ErrorIs(t, err, transformErr)
	assert.Equal(t, 1, src.callCount(), "the cached transform error should prevent another backend fetch")
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
func (r *countingRecorder) BackendFetch(operation metrics.FetcherOperation, result metrics.FetcherBackendResult, _ time.Duration) {
	r.backend[string(operation)+":"+string(result)]++
}

func TestFetcherRecordsCacheAndBackendOutcomes(t *testing.T) {
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
