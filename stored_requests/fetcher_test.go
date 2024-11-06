package stored_requests

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/stored_requests/caches/nil_cache"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupFetcherWithCacheDeps() (*mockCache, *mockCache, *mockCache, *mockFetcher, AllFetcher, *metrics.MetricsEngineMock) {
	reqCache := &mockCache{}
	impCache := &mockCache{}
	respCache := &mockCache{}
	metricsEngine := &metrics.MetricsEngineMock{}
	fetcher := &mockFetcher{}
	afetcherWithCache := WithCache(fetcher, Cache{reqCache, impCache, respCache, &nil_cache.NilCache{}}, metricsEngine)

	return reqCache, impCache, respCache, fetcher, afetcherWithCache, metricsEngine
}

func TestPerfectCache(t *testing.T) {
	reqCache, impCache, respCache, fetcher, aFetcherWithCache, metricsEngine := setupFetcherWithCacheDeps()
	impIDs := []string{"known"}
	reqIDs := []string{"req-id"}
	respIDs := []string{"resp-id"}
	ctx := context.Background()

	reqCache.On("Get", ctx, reqIDs).Return(
		map[string]json.RawMessage{
			"req-id": json.RawMessage(`{"req":true}`),
		})
	impCache.On("Get", ctx, impIDs).Return(
		map[string]json.RawMessage{
			"known": json.RawMessage(`{}`),
		})
	respCache.On("Get", ctx, respIDs).Return(
		map[string]json.RawMessage{
			"resp-id": json.RawMessage(`{"req":true}`),
		})
	metricsEngine.On("RecordStoredReqCacheResult", metrics.CacheHit, 1)
	metricsEngine.On("RecordStoredReqCacheResult", metrics.CacheMiss, 0)
	metricsEngine.On("RecordStoredImpCacheResult", metrics.CacheHit, 1)
	metricsEngine.On("RecordStoredImpCacheResult", metrics.CacheMiss, 0)

	reqData, impData, errs := aFetcherWithCache.FetchRequests(ctx, reqIDs, impIDs)
	respData, fetchRespErrs := aFetcherWithCache.FetchResponses(ctx, respIDs)

	reqCache.AssertExpectations(t)
	impCache.AssertExpectations(t)
	respCache.AssertExpectations(t)
	fetcher.AssertExpectations(t)
	metricsEngine.AssertExpectations(t)
	assert.JSONEq(t, `{"req":true}`, string(reqData["req-id"]), "Fetch requests should fetch the right request data")
	assert.JSONEq(t, `{"req":true}`, string(respData["resp-id"]), "Fetch responses should fetch the right response data")
	assert.JSONEq(t, `{}`, string(impData["known"]), "FetchRequests should fetch the right imp data")
	assert.Len(t, errs, 0, "FetchRequest shouldn't return any errors")
	assert.Len(t, fetchRespErrs, 0, "FetchResponses shouldn't return any errors")
}

func TestImperfectCache(t *testing.T) {
	reqCache, impCache, respCache, fetcher, aFetcherWithCache, metricsEngine := setupFetcherWithCacheDeps()
	impIDs := []string{"cached", "uncached"}
	respIDs := []string{"cached", "uncached"}
	ctx := context.Background()

	impCache.On("Get", ctx, impIDs).Return(
		map[string]json.RawMessage{
			"cached": json.RawMessage(`true`),
		})
	reqCache.On("Get", ctx, []string(nil)).Return(
		map[string]json.RawMessage{})
	respCache.On("Get", ctx, respIDs).Return(
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
	impCache.On("Save", ctx,
		map[string]json.RawMessage{
			"uncached": json.RawMessage(`false`),
		})

	fetcher.On("FetchResponses", ctx, []string{"uncached"}).Return(
		map[string]json.RawMessage{
			"uncached": json.RawMessage(`false`),
		},
		[]error{},
	)
	respCache.On("Save", ctx,
		map[string]json.RawMessage{
			"uncached": json.RawMessage(`false`),
		})

	reqCache.On("Save", ctx, map[string]json.RawMessage{})

	metricsEngine.On("RecordStoredReqCacheResult", metrics.CacheHit, 0)
	metricsEngine.On("RecordStoredReqCacheResult", metrics.CacheMiss, 0)
	metricsEngine.On("RecordStoredImpCacheResult", metrics.CacheHit, 1)
	metricsEngine.On("RecordStoredImpCacheResult", metrics.CacheMiss, 1)

	reqData, impData, errs := aFetcherWithCache.FetchRequests(ctx, nil, impIDs)
	respData, fetchRespErrs := aFetcherWithCache.FetchResponses(ctx, respIDs)

	impCache.AssertExpectations(t)
	respCache.AssertExpectations(t)
	fetcher.AssertExpectations(t)
	metricsEngine.AssertExpectations(t)
	assert.Len(t, reqData, 0, "Fetch requests should return nil if no request IDs were passed")
	assert.JSONEq(t, `true`, string(impData["cached"]), "FetchRequests should fetch the right imp data")
	assert.JSONEq(t, `false`, string(impData["uncached"]), "FetchRequests should fetch the right imp data")
	assert.JSONEq(t, `true`, string(respData["cached"]), "FetchResponses should fetch the right resp data")
	assert.JSONEq(t, `false`, string(respData["uncached"]), "FetchResponses should fetch the right resp data")
	assert.Len(t, errs, 0, "FetchRequest shouldn't return any errors")
	assert.Len(t, fetchRespErrs, 0, "FetchResponses shouldn't return any errors")
}

func TestMissingData(t *testing.T) {
	reqCache, impCache, respCache, fetcher, aFetcherWithCache, metricsEngine := setupFetcherWithCacheDeps()
	impIDs := []string{"unknown"}
	respIDs := []string{"unknown"}
	ctx := context.Background()

	impCache.On("Get", ctx, impIDs).Return(
		map[string]json.RawMessage{},
	)
	reqCache.On("Get", ctx, []string(nil)).Return(
		map[string]json.RawMessage{})

	respCache.On("Get", ctx, []string(respIDs)).Return(
		map[string]json.RawMessage{})

	fetcher.On("FetchRequests", ctx, []string{}, impIDs).Return(
		map[string]json.RawMessage{},
		map[string]json.RawMessage{},
		[]error{
			errors.New("Data not found"),
		},
	)

	fetcher.On("FetchResponses", ctx, respIDs).Return(
		map[string]json.RawMessage{},
		[]error{
			errors.New("Data not found"),
		},
	)

	impCache.On("Save", ctx,
		map[string]json.RawMessage{},
	)
	reqCache.On("Save", ctx,
		map[string]json.RawMessage{},
	)

	respCache.On("Save", ctx,
		map[string]json.RawMessage{},
	)
	metricsEngine.On("RecordStoredReqCacheResult", metrics.CacheHit, 0)
	metricsEngine.On("RecordStoredReqCacheResult", metrics.CacheMiss, 0)
	metricsEngine.On("RecordStoredImpCacheResult", metrics.CacheHit, 0)
	metricsEngine.On("RecordStoredImpCacheResult", metrics.CacheMiss, 1)

	reqData, impData, errs := aFetcherWithCache.FetchRequests(ctx, nil, impIDs)
	respData, fetchRespErrs := aFetcherWithCache.FetchResponses(ctx, respIDs)

	reqCache.AssertExpectations(t)
	impCache.AssertExpectations(t)
	respCache.AssertExpectations(t)
	fetcher.AssertExpectations(t)
	metricsEngine.AssertExpectations(t)
	assert.Len(t, errs, 1, "FetchRequests for missing data should return an error")
	assert.Len(t, fetchRespErrs, 1, "FetchResponses for missing data should return an error")
	assert.Len(t, reqData, 0, "FetchRequests for missing data shouldn't return anything")
	assert.Len(t, impData, 0, "FetchRequests for missing data shouldn't return anything")
	assert.Len(t, respData, 0, "FetchRequests for missing data shouldn't return anything")
}

// Prevents #311
func TestCacheSaves(t *testing.T) {
	reqCache, impCache, respCache, fetcher, aFetcherWithCache, metricsEngine := setupFetcherWithCacheDeps()
	impIDs := []string{"abc", "abc"}
	respIDs := []string{"abc", "abc"}
	ctx := context.Background()

	impCache.On("Get", ctx, impIDs).Return(
		map[string]json.RawMessage{
			"abc": json.RawMessage(`{}`),
		})
	respCache.On("Get", ctx, respIDs).Return(
		map[string]json.RawMessage{
			"abc": json.RawMessage(`{}`),
		})
	reqCache.On("Get", ctx, []string(nil)).Return(
		map[string]json.RawMessage{})
	metricsEngine.On("RecordStoredReqCacheResult", metrics.CacheHit, 0)
	metricsEngine.On("RecordStoredReqCacheResult", metrics.CacheMiss, 0)
	metricsEngine.On("RecordStoredImpCacheResult", metrics.CacheHit, 2)
	metricsEngine.On("RecordStoredImpCacheResult", metrics.CacheMiss, 0)

	_, impData, errs := aFetcherWithCache.FetchRequests(ctx, nil, []string{"abc", "abc"})
	respData, fetchRespErrs := aFetcherWithCache.FetchResponses(ctx, respIDs)

	impCache.AssertExpectations(t)
	respCache.AssertExpectations(t)
	fetcher.AssertExpectations(t)
	metricsEngine.AssertExpectations(t)
	assert.Len(t, impData, 1, "FetchRequests should return data only once for duplicate requests")
	assert.JSONEq(t, `{}`, string(impData["abc"]), "FetchRequests should fetch the right imp data")
	assert.JSONEq(t, `{}`, string(respData["abc"]), "FetchResponses should fetch the right resp data")
	assert.Len(t, errs, 0, "FetchRequests with duplicate IDs shouldn't return an error")
	assert.Len(t, fetchRespErrs, 0, "FetchResponses with duplicate IDs shouldn't return an error")
}

func setupAccountFetcherWithCacheDeps() (*mockCache, *mockFetcher, AllFetcher, *metrics.MetricsEngineMock) {
	accCache := &mockCache{}
	metricsEngine := &metrics.MetricsEngineMock{}
	fetcher := &mockFetcher{}
	afetcherWithCache := WithCache(fetcher, Cache{&nil_cache.NilCache{}, &nil_cache.NilCache{}, &nil_cache.NilCache{}, accCache}, metricsEngine)

	return accCache, fetcher, afetcherWithCache, metricsEngine
}

func TestAccountCacheHit(t *testing.T) {
	accCache, fetcher, aFetcherWithCache, metricsEngine := setupAccountFetcherWithCacheDeps()
	cachedAccounts := []string{"known"}
	ctx := context.Background()

	// Test read from cache
	accCache.On("Get", ctx, cachedAccounts).Return(
		map[string]json.RawMessage{
			"known": json.RawMessage(`true`),
		})

	metricsEngine.On("RecordAccountCacheResult", metrics.CacheHit, 1)
	account, errs := aFetcherWithCache.FetchAccount(ctx, json.RawMessage("{}"), "known")

	accCache.AssertExpectations(t)
	fetcher.AssertExpectations(t)
	metricsEngine.AssertExpectations(t)
	assert.JSONEq(t, `true`, string(account), "FetchAccount should fetch the right account data")
	assert.Len(t, errs, 0, "FetchAccount shouldn't return any errors")
}

func TestAccountCacheMiss(t *testing.T) {
	accCache, fetcher, aFetcherWithCache, metricsEngine := setupAccountFetcherWithCacheDeps()
	uncachedAccounts := []string{"uncached"}
	uncachedAccountsData := map[string]json.RawMessage{
		"uncached": json.RawMessage(`true`),
	}
	ctx := context.Background()

	// Test read from cache
	accCache.On("Get", ctx, uncachedAccounts).Return(map[string]json.RawMessage{})
	accCache.On("Save", ctx, uncachedAccountsData)
	fetcher.On("FetchAccount", ctx, json.RawMessage("{}"), "uncached").Return(uncachedAccountsData["uncached"], []error{})
	metricsEngine.On("RecordAccountCacheResult", metrics.CacheMiss, 1)

	account, errs := aFetcherWithCache.FetchAccount(ctx, json.RawMessage("{}"), "uncached")

	accCache.AssertExpectations(t)
	fetcher.AssertExpectations(t)
	metricsEngine.AssertExpectations(t)
	assert.JSONEq(t, `true`, string(account), "FetchAccount should fetch the right account data")
	assert.Len(t, errs, 0, "FetchAccount shouldn't return any errors")
}
func TestComposedCache(t *testing.T) {
	c1 := &mockCache{}
	c2 := &mockCache{}
	c3 := &mockCache{}
	c4 := &mockCache{}
	impCache := &mockCache{}
	cache := Cache{
		Requests:  ComposedCache{c1, c2, c3, c4},
		Imps:      impCache,
		Responses: ComposedCache{c1, c2, c3, c4},
	}
	metricsEngine := &metrics.MetricsEngineMock{}
	fetcher := &mockFetcher{}
	aFetcherWithCache := WithCache(fetcher, cache, metricsEngine)
	reqIDs := []string{"1", "2", "3"}
	impIDs := []string{}
	ctx := context.Background()

	c1.On("Get", ctx, reqIDs).Return(
		map[string]json.RawMessage{
			"1": json.RawMessage(`{"id": "1"}`),
		})
	c2.On("Get", ctx, []string{"2", "3"}).Return(
		map[string]json.RawMessage{
			"2": json.RawMessage(`{"id": "2"}`),
		})
	c3.On("Get", ctx, []string{"3"}).Return(
		map[string]json.RawMessage{
			"3": json.RawMessage(`{"id": "3"}`),
		})
	impCache.On("Get", ctx, []string{}).Return(map[string]json.RawMessage{})

	metricsEngine.On("RecordStoredReqCacheResult", metrics.CacheHit, 3)
	metricsEngine.On("RecordStoredReqCacheResult", metrics.CacheMiss, 0)
	metricsEngine.On("RecordStoredImpCacheResult", metrics.CacheHit, 0)
	metricsEngine.On("RecordStoredImpCacheResult", metrics.CacheMiss, 0)

	reqData, impData, errs := aFetcherWithCache.FetchRequests(ctx, reqIDs, impIDs)
	respData, fetchRespErrs := aFetcherWithCache.FetchResponses(ctx, reqIDs)

	c1.AssertExpectations(t)
	c2.AssertExpectations(t)
	c3.AssertExpectations(t)
	impCache.AssertExpectations(t)
	fetcher.AssertExpectations(t)
	metricsEngine.AssertExpectations(t)
	assert.Len(t, reqData, len(reqIDs), "FetchRequests should be able to return all request data from a composed cache")
	assert.Len(t, respData, len(reqIDs), "FetchResponses should be able to return all response data from a composed cache")
	assert.Len(t, impData, len(impIDs), "FetchRequests should be able to return all imp data from a composed cache")
	assert.Len(t, errs, 0, "FetchRequests shouldn't return an error when trying to use a composed cache")
	assert.Len(t, fetchRespErrs, 0, "FetchResponses shouldn't return an error when trying to use a composed cache")
	assert.JSONEq(t, `{"id": "1"}`, string(reqData["1"]), "FetchRequests should fetch the right req data")
	assert.JSONEq(t, `{"id": "2"}`, string(reqData["2"]), "FetchRequests should fetch the right req data")
	assert.JSONEq(t, `{"id": "3"}`, string(reqData["3"]), "FetchRequests should fetch the right req data")
	assert.JSONEq(t, `{"id": "1"}`, string(respData["1"]), "FetchResponses should fetch the right resp data")
	assert.JSONEq(t, `{"id": "2"}`, string(respData["2"]), "FetchResponses should fetch the right resp data")
	assert.JSONEq(t, `{"id": "3"}`, string(respData["3"]), "FetchResponses should fetch the right resp data")
}

type mockFetcher struct {
	mock.Mock
}

func (f *mockFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (map[string]json.RawMessage, map[string]json.RawMessage, []error) {
	args := f.Called(ctx, requestIDs, impIDs)
	return args.Get(0).(map[string]json.RawMessage), args.Get(1).(map[string]json.RawMessage), args.Get(2).([]error)
}

func (f *mockFetcher) FetchResponses(ctx context.Context, ids []string) (data map[string]json.RawMessage, errs []error) {
	args := f.Called(ctx, ids)
	return args.Get(0).(map[string]json.RawMessage), args.Get(1).([]error)
}

func (a *mockFetcher) FetchAccount(ctx context.Context, defaultAccountsJSON json.RawMessage, accountID string) (json.RawMessage, []error) {
	args := a.Called(ctx, defaultAccountsJSON, accountID)
	return args.Get(0).(json.RawMessage), args.Get(1).([]error)
}

func (f *mockFetcher) FetchCategories(ctx context.Context, primaryAdServer, publisherId, iabCategory string) (string, error) {
	return "", nil
}

type mockCache struct {
	mock.Mock
}

func (c *mockCache) Get(ctx context.Context, ids []string) map[string]json.RawMessage {
	args := c.Called(ctx, ids)
	return args.Get(0).(map[string]json.RawMessage)
}

func (c *mockCache) Save(ctx context.Context, data map[string]json.RawMessage) {
	c.Called(ctx, data)
}

func (c *mockCache) Invalidate(ctx context.Context, ids []string) {
	c.Called(ctx, ids)
}
