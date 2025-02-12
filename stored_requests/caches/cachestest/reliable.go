package cachestest

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/stored_requests"
)

const (
	reqCacheKey = "known-req"
	reqCacheVal = `{"req":true}`
)

// AssertCacheRobustness runs tests which can be used to validate any Cache that is 100% reliable.
// That is, its Save() and Invalidate() functions _alway_ work.
//
// The cacheSupplier should be a function which returns a new Cache (with no data inside) on every call.
// This will be called from separate Goroutines to make sure that different tests don't conflict.
func AssertCacheRobustness(t *testing.T, cacheSupplier func() stored_requests.CacheJSON) {
	t.Run("TestCacheMiss", cacheMissTester(cacheSupplier()))
	t.Run("TestCacheHit", cacheHitTester(cacheSupplier()))
	t.Run("TestCacheSaveInvalidate", cacheSaveInvalidateTester(cacheSupplier()))
}

func cacheMissTester(cache stored_requests.CacheJSON) func(*testing.T) {
	return func(t *testing.T) {
		storedData := cache.Get(context.Background(), []string{"unknown"})
		assertMapLength(t, 0, storedData)
	}
}

func cacheHitTester(cache stored_requests.CacheJSON) func(*testing.T) {
	return func(t *testing.T) {
		cache.Save(context.Background(), map[string]json.RawMessage{
			reqCacheKey: json.RawMessage(reqCacheVal),
		})
		reqData := cache.Get(context.Background(), []string{reqCacheKey})
		assertMapLength(t, 1, reqData)
		assertHasValue(t, reqData, reqCacheKey, reqCacheVal)
	}
}

func cacheSaveInvalidateTester(cache stored_requests.CacheJSON) func(*testing.T) {
	return func(t *testing.T) {
		cache.Save(context.Background(), map[string]json.RawMessage{
			reqCacheKey: json.RawMessage(reqCacheVal),
		})
		reqData := cache.Get(context.Background(), []string{reqCacheKey})
		assertMapLength(t, 1, reqData)

		cache.Invalidate(context.Background(), []string{reqCacheKey})
		reqData = cache.Get(context.Background(), []string{reqCacheKey})
		assertMapLength(t, 0, reqData)
	}
}

func assertMapLength(t *testing.T, expectedLen int, theMap map[string]json.RawMessage) {
	t.Helper()
	if len(theMap) != expectedLen {
		t.Errorf("Wrong map length. Expected %d, Got %d.", expectedLen, len(theMap))
	}
}

func assertHasValue(t *testing.T, m map[string]json.RawMessage, key string, val string) {
	t.Helper()
	realVal, ok := m[key]
	if !ok {
		t.Errorf("Map missing required key: %s", key)
	}
	if val != string(realVal) {
		t.Errorf("Unexpected value at key %s. Expected %s, Got %s", key, val, string(realVal))
	}
}
