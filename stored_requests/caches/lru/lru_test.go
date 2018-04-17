package lru

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/caches/cachestest"
)

func TestLRURobustness(t *testing.T) {
	cachestest.AssertCacheRobustness(t, func() stored_requests.Cache {
		return NewLRUCache(&config.InMemoryCache{
			RequestCacheSize: 256 * 1024,
			ImpCacheSize:     256 * 1024,
			TTL:              -1,
		})
	})
}
