package cacher_test

import (
	"testing"

	"github.com/prebid/prebid-server/stored_requests/cache/cacher"
	_ "github.com/prebid/prebid-server/stored_requests/cache/inmemorycache"
	_ "github.com/prebid/prebid-server/stored_requests/cache/rediscache"
)

func TestRegisteredClients(t *testing.T) {

	if exists := cacher.Get(cacher.InMemoryCache); exists == nil {
		t.Fatal("We did not get a cache driver for inmemory")
	}

	if exists := cacher.Get(cacher.RedisCache); exists == nil {
		t.Fatal("We did not get a cache driver for redis")
	}

}

func TestMissingClient(t *testing.T) {

	defer func() {
		if err := recover(); err != nil {
			// if get a panic it's because the invald cache caused a panic (this is expected)
		}
	}()

	cacher.Get("random-cache-client")

}
