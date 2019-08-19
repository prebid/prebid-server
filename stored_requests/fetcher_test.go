package stored_requests

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/prebid/prebid-server/pbsmetrics"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupFetcherWithCacheDeps() (*mockCache, *mockFetcher, AllFetcher, *pbsmetrics.MetricsEngineMock) {
	cache := &mockCache{}
	metricsEngine := &pbsmetrics.MetricsEngineMock{}
	fetcher := &mockFetcher{}
	afetcherWithCache := WithCache(fetcher, cache, metricsEngine)

	return cache, fetcher, afetcherWithCache, metricsEngine
}

func TestPerfectCache(t *testing.T) {
	cache, fetcher, aFetcherWithCache, metricsEngine := setupFetcherWithCacheDeps()
	impIDs := []string{"known"}
	reqIDs := []string{"req-id"}
	ctx := context.Background()

	cache.On("Get", ctx, reqIDs, impIDs).Return(
		map[string]json.RawMessage{
			"req-id": json.RawMessage(`{"req":true}`),
		},
		map[string]json.RawMessage{
			"known": json.RawMessage(`{}`),
		})
	metricsEngine.On("RecordStoredReqCacheResult", pbsmetrics.CacheHit, 1)
	metricsEngine.On("RecordStoredReqCacheResult", pbsmetrics.CacheMiss, 0)
	metricsEngine.On("RecordStoredImpCacheResult", pbsmetrics.CacheHit, 1)
	metricsEngine.On("RecordStoredImpCacheResult", pbsmetrics.CacheMiss, 0)

	reqData, impData, errs := aFetcherWithCache.FetchRequests(ctx, reqIDs, impIDs)

	cache.AssertExpectations(t)
	fetcher.AssertExpectations(t)
	metricsEngine.AssertExpectations(t)
	assert.JSONEq(t, `{"req":true}`, string(reqData["req-id"]), "Fetch requests should fetch the right request data")
	assert.JSONEq(t, `{}`, string(impData["known"]), "FetchRequests should fetch the right imp data")
	assert.Len(t, errs, 0, "FetchRequest shouldn't return any errors")
}

func TestImperfectCache(t *testing.T) {
	cache, fetcher, aFetcherWithCache, metricsEngine := setupFetcherWithCacheDeps()
	impIDs := []string{"cached", "uncached"}
	ctx := context.Background()

	cache.On("Get", ctx, []string(nil), impIDs).Return(
		map[string]json.RawMessage{},
		map[string]json.RawMessage{
			"cached": json.RawMessage(`true`),
		})

	fetcher.On("FetchRequests", ctx, []string{}, []string{"uncached"}).Return(
		map[string]json.RawMessage{},
		map[string]json.RawMessage{
			"uncached": json.RawMessage(`false`),
		},
		[]error{},
	)
	cache.On("Save", ctx,
		map[string]json.RawMessage{},
		map[string]json.RawMessage{
			"uncached": json.RawMessage(`false`),
		})
	metricsEngine.On("RecordStoredReqCacheResult", pbsmetrics.CacheHit, 0)
	metricsEngine.On("RecordStoredReqCacheResult", pbsmetrics.CacheMiss, 0)
	metricsEngine.On("RecordStoredImpCacheResult", pbsmetrics.CacheHit, 1)
	metricsEngine.On("RecordStoredImpCacheResult", pbsmetrics.CacheMiss, 1)

	reqData, impData, errs := aFetcherWithCache.FetchRequests(ctx, nil, impIDs)

	cache.AssertExpectations(t)
	fetcher.AssertExpectations(t)
	metricsEngine.AssertExpectations(t)
	assert.Len(t, reqData, 0, "Fetch requests should return nil if no request IDs were passed")
	assert.JSONEq(t, `true`, string(impData["cached"]), "FetchRequests should fetch the right imp data")
	assert.JSONEq(t, `false`, string(impData["uncached"]), "FetchRequests should fetch the right imp data")
	assert.Len(t, errs, 0, "FetchRequest shouldn't return any errors")
}

func TestMissingData(t *testing.T) {
	cache, fetcher, aFetcherWithCache, metricsEngine := setupFetcherWithCacheDeps()
	impIDs := []string{"unknown"}
	ctx := context.Background()

	cache.On("Get", ctx, []string(nil), impIDs).Return(
		map[string]json.RawMessage{},
		map[string]json.RawMessage{},
	)
	fetcher.On("FetchRequests", ctx, []string{}, impIDs).Return(
		map[string]json.RawMessage{},
		map[string]json.RawMessage{},
		[]error{
			errors.New("Data not found"),
		},
	)
	cache.On("Save", ctx,
		map[string]json.RawMessage{},
		map[string]json.RawMessage{},
	)
	metricsEngine.On("RecordStoredReqCacheResult", pbsmetrics.CacheHit, 0)
	metricsEngine.On("RecordStoredReqCacheResult", pbsmetrics.CacheMiss, 0)
	metricsEngine.On("RecordStoredImpCacheResult", pbsmetrics.CacheHit, 0)
	metricsEngine.On("RecordStoredImpCacheResult", pbsmetrics.CacheMiss, 1)

	reqData, impData, errs := aFetcherWithCache.FetchRequests(ctx, nil, impIDs)

	cache.AssertExpectations(t)
	fetcher.AssertExpectations(t)
	metricsEngine.AssertExpectations(t)
	assert.Len(t, errs, 1, "FetchRequests for missing data should return an error")
	assert.Len(t, reqData, 0, "FetchRequests for missing data shouldn't return anything")
	assert.Len(t, impData, 0, "FetchRequests for missing data shouldn't return anything")
}

// Prevents #311
func TestCacheSaves(t *testing.T) {
	cache, fetcher, aFetcherWithCache, metricsEngine := setupFetcherWithCacheDeps()
	impIDs := []string{"abc", "abc"}
	ctx := context.Background()

	cache.On("Get", ctx, []string(nil), impIDs).Return(
		map[string]json.RawMessage{},
		map[string]json.RawMessage{
			"abc": json.RawMessage(`{}`),
		})
	metricsEngine.On("RecordStoredReqCacheResult", pbsmetrics.CacheHit, 0)
	metricsEngine.On("RecordStoredReqCacheResult", pbsmetrics.CacheMiss, 0)
	metricsEngine.On("RecordStoredImpCacheResult", pbsmetrics.CacheHit, 2)
	metricsEngine.On("RecordStoredImpCacheResult", pbsmetrics.CacheMiss, 0)

	_, impData, errs := aFetcherWithCache.FetchRequests(ctx, nil, []string{"abc", "abc"})

	cache.AssertExpectations(t)
	fetcher.AssertExpectations(t)
	metricsEngine.AssertExpectations(t)
	assert.Len(t, impData, 1, "FetchRequests should return data only once for duplicate requests")
	assert.JSONEq(t, `{}`, string(impData["abc"]), "FetchRequests should fetch the right imp data")
	assert.Len(t, errs, 0, "FetchRequests with duplicate IDs shouldn't return an error")
}

func TestComposedCache(t *testing.T) {
	c1 := &mockCache{}
	c2 := &mockCache{}
	c3 := &mockCache{}
	c4 := &mockCache{}
	cache := ComposedCache{c1, c2, c3, c4}
	metricsEngine := &pbsmetrics.MetricsEngineMock{}
	fetcher := &mockFetcher{}
	aFetcherWithCache := WithCache(fetcher, cache, metricsEngine)
	impIDs := []string{"1", "2", "3"}
	reqIDs := []string{"1", "2", "3"}
	ctx := context.Background()

	c1.On("Get", ctx, reqIDs, impIDs).Return(
		map[string]json.RawMessage{
			"1": json.RawMessage(`{"id": "1"}`),
		},
		map[string]json.RawMessage{
			"1": json.RawMessage(`{"id": "1"}`),
		})
	c2.On("Get", ctx, []string{"2", "3"}, []string{"2", "3"}).Return(
		map[string]json.RawMessage{
			"2": json.RawMessage(`{"id": "2"}`),
		},
		map[string]json.RawMessage{
			"2": json.RawMessage(`{"id": "2"}`),
		})
	c3.On("Get", ctx, []string{"3"}, []string{"3"}).Return(
		map[string]json.RawMessage{
			"3": json.RawMessage(`{"id": "3"}`),
		},
		map[string]json.RawMessage{
			"3": json.RawMessage(`{"id": "3"}`),
		})
	metricsEngine.On("RecordStoredReqCacheResult", pbsmetrics.CacheHit, 3)
	metricsEngine.On("RecordStoredReqCacheResult", pbsmetrics.CacheMiss, 0)
	metricsEngine.On("RecordStoredImpCacheResult", pbsmetrics.CacheHit, 3)
	metricsEngine.On("RecordStoredImpCacheResult", pbsmetrics.CacheMiss, 0)

	reqData, impData, errs := aFetcherWithCache.FetchRequests(ctx, reqIDs, impIDs)

	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
	c3.AssertExpectations(t)
	fetcher.AssertExpectations(t)
	metricsEngine.AssertExpectations(t)
	assert.Len(t, reqData, len(reqIDs), "FetchRequests should be able to return all request data from a composed cache")
	assert.Len(t, impData, len(impIDs), "FetchRequests should be able to return all imp data from a composed cache")
	assert.Len(t, errs, 0, "FetchRequests shouldn't return an error when trying to use a composed cache")
	assert.JSONEq(t, `{"id": "1"}`, string(impData["1"]), "FetchRequests should fetch the right imp data")
	assert.JSONEq(t, `{"id": "2"}`, string(impData["2"]), "FetchRequests should fetch the right imp data")
	assert.JSONEq(t, `{"id": "3"}`, string(impData["3"]), "FetchRequests should fetch the right imp data")
	assert.JSONEq(t, `{"id": "1"}`, string(reqData["1"]), "FetchRequests should fetch the right req data")
	assert.JSONEq(t, `{"id": "2"}`, string(reqData["2"]), "FetchRequests should fetch the right req data")
	assert.JSONEq(t, `{"id": "3"}`, string(reqData["3"]), "FetchRequests should fetch the right req data")
}

type mockFetcher struct {
	mock.Mock
}

func (f *mockFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (map[string]json.RawMessage, map[string]json.RawMessage, []error) {
	args := f.Called(ctx, requestIDs, impIDs)
	return args.Get(0).(map[string]json.RawMessage), args.Get(1).(map[string]json.RawMessage), args.Get(2).([]error)
}

func (f *mockFetcher) FetchCategories(ctx context.Context, primaryAdServer, publisherId, iabCategory string) (string, error) {
	return "", nil
}

type mockCache struct {
	mock.Mock
}

func (c *mockCache) Get(ctx context.Context, requestIDs []string, impIDs []string) (map[string]json.RawMessage, map[string]json.RawMessage) {
	args := c.Called(ctx, requestIDs, impIDs)
	return args.Get(0).(map[string]json.RawMessage), args.Get(1).(map[string]json.RawMessage)
}

func (c *mockCache) Save(ctx context.Context, storedRequests map[string]json.RawMessage, storedImps map[string]json.RawMessage) {
	c.Called(ctx, storedRequests, storedImps)
}

func (c *mockCache) Invalidate(ctx context.Context, requestIDs []string, impIDs []string) {
	c.Called(ctx, requestIDs, impIDs)
}
