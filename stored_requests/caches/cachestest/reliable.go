package cachestest

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/stored_requests"
)

// AssertCacheRobustness runs tests which can be used to validate any Cache that is 100% reliable.
// That is, its Save() and Invalidate() functions _alway_ work.
//
// The cacheSupplier should be a function which returns a new Cache (with no data inside) on every call.
// This will be called from separate Goroutines to make sure that different tests don't conflict.
func AssertCacheRobustness(t *testing.T, cacheSupplier func() stored_requests.Cache) {
	t.Run("TestCacheMiss", cacheMissTester(cacheSupplier()))
	t.Run("TestCacheHit", cacheHitTester(cacheSupplier()))
	t.Run("TestCacheMixed", cacheMixedTester(cacheSupplier()))
	t.Run("TestCacheOverlap", cacheOverlapTester(cacheSupplier()))
	t.Run("TestCacheSaveInvalidate", cacheSaveInvalidateTester(cacheSupplier()))
}

func cacheMissTester(cache stored_requests.Cache) func(*testing.T) {
	return func(t *testing.T) {
		storedReqs, storedImps := cache.Get(context.Background(), []string{"unknown"}, nil)
		assertMapLength(t, 0, storedReqs)
		assertMapLength(t, 0, storedImps)
	}
}

func cacheHitTester(cache stored_requests.Cache) func(*testing.T) {
	return func(t *testing.T) {
		cache.Save(context.Background(), map[string]json.RawMessage{
			"known-req": json.RawMessage(`{"req":true}`),
		}, map[string]json.RawMessage{
			"known-imp": json.RawMessage(`{"imp":true}`),
		})
		reqData, impData := cache.Get(context.Background(), []string{"known-req"}, []string{"known-imp"})
		if len(reqData) != 1 {
			t.Errorf("The cache should have returned the data.")
		}
		assertMapLength(t, 1, reqData)
		assertHasValue(t, reqData, "known-req", `{"req":true}`)

		assertMapLength(t, 1, impData)
		assertHasValue(t, impData, "known-imp", `{"imp":true}`)
	}
}

func cacheMixedTester(cache stored_requests.Cache) func(*testing.T) {
	return func(t *testing.T) {
		cache.Save(context.Background(), map[string]json.RawMessage{
			"known-req": json.RawMessage(`{"req":true}`),
		}, nil)
		reqData, impData := cache.Get(context.Background(), []string{"known-req", "unknown-req"}, nil)
		assertMapLength(t, 1, reqData)
		assertHasValue(t, reqData, "known-req", `{"req":true}`)
		assertMapLength(t, 0, impData)
	}
}

func cacheOverlapTester(cache stored_requests.Cache) func(*testing.T) {
	return func(t *testing.T) {
		cache.Save(context.Background(), map[string]json.RawMessage{
			"id": json.RawMessage(`{"req":true}`),
		}, map[string]json.RawMessage{
			"id": json.RawMessage(`{"imp":true}`),
		})
		reqData, impData := cache.Get(context.Background(), []string{"id"}, []string{"id"})
		assertMapLength(t, 1, reqData)
		assertHasValue(t, reqData, "id", `{"req":true}`)
		assertMapLength(t, 1, impData)
		assertHasValue(t, impData, "id", `{"imp":true}`)
	}
}

func cacheSaveInvalidateTester(cache stored_requests.Cache) func(*testing.T) {
	return func(t *testing.T) {
		cache.Save(context.Background(), map[string]json.RawMessage{
			"known": json.RawMessage(`{}`),
		}, map[string]json.RawMessage{
			"known": json.RawMessage(`{}`),
		})
		reqData, impData := cache.Get(context.Background(), []string{"known"}, []string{"known"})
		assertMapLength(t, 1, reqData)
		assertMapLength(t, 1, impData)

		cache.Invalidate(context.Background(), []string{"known"}, []string{"known"})
		reqData, impData = cache.Get(context.Background(), []string{"known"}, []string{"known"})
		assertMapLength(t, 0, reqData)
		assertMapLength(t, 0, impData)
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
