package in_memory

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/prebid/prebid-server/config"
	"testing"
)

func TestCacheMiss(t *testing.T) {
	cache := NewLRUCache(&config.InMemoryCache{
		Size: 512 * 1024,
		TTL:  -1,
	})
	data := cache.GetRequests(context.Background(), []string{"unknown"})
	if len(data) > 0 {
		t.Errorf("An empty cache should not return any data on unknown IDs.")
	}
}

func TestCacheHit(t *testing.T) {
	cache := NewLRUCache(&config.InMemoryCache{
		Size: 512 * 1024,
		TTL:  -1,
	})
	cache.SaveRequests(context.Background(), map[string]json.RawMessage{
		"known": json.RawMessage(`{}`),
	})
	data := cache.GetRequests(context.Background(), []string{"known"})
	if len(data) != 1 {
		t.Errorf("The cache should have returned the data.")
	}
	if value, ok := data["known"]; ok {
		if !bytes.Equal(value, []byte("{}")) {
			t.Errorf("Cache returned bad data. Expected {}, got %s", value)
		}
	} else {
		t.Errorf(`Missing expected data with key: "known"`)
	}
}

func TestCacheMixed(t *testing.T) {
	cache := NewLRUCache(&config.InMemoryCache{
		Size: 512 * 1024,
		TTL:  -1,
	})
	cache.SaveRequests(context.Background(), map[string]json.RawMessage{
		"known": json.RawMessage(`{}`),
	})
	data := cache.GetRequests(context.Background(), []string{"known", "unknown"})
	if len(data) != 1 {
		t.Errorf("The cache should have returned the available data.")
	}
	if value, ok := data["known"]; ok {
		if !bytes.Equal(value, []byte("{}")) {
			t.Errorf("Cache returned bad data. Expected {}, got %s", value)
		}
	} else {
		t.Errorf(`Missing expected data with key: "known"`)
	}
}
