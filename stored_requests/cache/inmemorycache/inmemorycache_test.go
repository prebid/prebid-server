package inmemorycache

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/stored_requests/cache/cacher"
)

func TestCacheName(t *testing.T) {
	client := cacher.Get(cacher.InMemoryCache)
	client.Configure(nil)

	if client.Name() != cacher.InMemoryCache {
		t.Fatalf("We did not get the correct cache name. We got %v and expected %v", client.Name(), cacher.InMemoryCache)
	}

	if client.Name() != "inmemorycache" {
		t.Fatalf("We did not get the correct cache name. We got %v and expected %v", client.Name(), "inmemorycache")
	}

}

func TestCacheOperations(t *testing.T) {
	client := cacher.Get(cacher.InMemoryCache)
	client.Configure(nil)

	const (
		key   = `cache`
		value = `inmemorycache`
	)

	if err := client.Set(key, value, 900); err != nil {
		t.Fatalf("We encountered an error when trying to set data in our cache: %v", err)
	}

	data, err := client.Get(key)
	if err != nil {
		t.Fatalf("We encountered the error: %v", err)
	}
	if data != value {
		t.Fatalf("We expected %v and got %v", value, data)
	}

}

func TestCacheTimeout(t *testing.T) {
	client := cacher.Get(cacher.InMemoryCache)
	client.Configure(nil)

	const (
		key   = `cache`
		value = `inmemorycache`
	)

	// set the data to expire in 5 seconds
	if err := client.Set(key, value, 5); err != nil {
		t.Fatalf("We encountered an error when trying to set data in our cache: %v", err)
	}

	time.Sleep(6 * time.Second)

	_, err := client.Get(key)
	if err != cacher.ErrDoesNotExist {
		t.Fatalf("We expected to get a 'Does Not Exist' error but got this instead: %v", err)
	}

}
