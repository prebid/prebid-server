package in_memory

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/config"
)

func TestCacheMiss(t *testing.T) {
	cache := NewLRUCache(&config.InMemoryCache{
		RequestCacheSize: 256 * 1024,
		ImpCacheSize:     256 * 1024,
		TTL:              -1,
	})
	storedReqs, storedImps := cache.Get(context.Background(), []string{"unknown"}, nil)
	assertMapLength(t, 0, storedReqs)
	assertMapLength(t, 0, storedImps)
}

func TestCacheHit(t *testing.T) {
	cache := NewLRUCache(&config.InMemoryCache{
		RequestCacheSize: 256 * 1024,
		ImpCacheSize:     256 * 1024,
		TTL:              -1,
	})
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

func TestCacheMixed(t *testing.T) {
	cache := NewLRUCache(&config.InMemoryCache{
		RequestCacheSize: 256 * 1024,
		ImpCacheSize:     256 * 1024,
		TTL:              -1,
	})
	cache.Save(context.Background(), map[string]json.RawMessage{
		"known-req": json.RawMessage(`{"req":true}`),
	}, nil)
	reqData, impData := cache.Get(context.Background(), []string{"known-req", "unknown-req"}, nil)
	assertMapLength(t, 1, reqData)
	assertHasValue(t, reqData, "known-req", `{"req":true}`)
	assertMapLength(t, 0, impData)
}

func TestCacheOverlap(t *testing.T) {
	cache := NewLRUCache(&config.InMemoryCache{
		RequestCacheSize: 256 * 1024,
		ImpCacheSize:     256 * 1024,
		TTL:              -1,
	})
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

func TestCacheSaveInvalidate(t *testing.T) {
	cache := NewLRUCache(&config.InMemoryCache{
		RequestCacheSize: 256 * 1024,
		ImpCacheSize:     256 * 1024,
		TTL:              -1,
	})

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
