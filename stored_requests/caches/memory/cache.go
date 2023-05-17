package memory

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/coocood/freecache"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/stored_requests"
)

// NewCache returns an in-memory Cache which evicts items if:
//
// 1. They haven't been used within the TTL.
// 2. The cache is too large. This will cause the least recently used items to be evicted.
//
// For no TTL, use ttlSeconds <= 0
func NewCache(size int, ttl int, dataType string) stored_requests.CacheJSON {
	if ttl > 0 && size <= 0 {
		// a positive ttl indicates "LRU" cache type, while unlimited size indicates an "unbounded" cache type
		glog.Fatalf("unbounded in-memory %s cache with TTL not allowed. Config validation should have caught this. Failing fast because something is buggy.", dataType)
	}
	if size > 0 {
		glog.Infof("Using a Stored %s in-memory cache. Max size: %d bytes. TTL: %d seconds.", dataType, size, ttl)
		return &cache{
			dataType: dataType,
			cache: &pbsLRUCache{
				Cache:      freecache.NewCache(size),
				ttlSeconds: ttl,
			},
		}
	} else {
		glog.Infof("Using an unbounded Stored %s in-memory cache.", dataType)
		return &cache{
			dataType: dataType,
			cache:    &pbsSyncMap{&sync.Map{}},
		}
	}
}

type cache struct {
	dataType string
	cache    mapLike
}

func (c *cache) Get(ctx context.Context, ids []string) (data map[string]json.RawMessage) {
	data = make(map[string]json.RawMessage, len(ids))
	for _, id := range ids {
		if val, ok := c.cache.Get(id); ok {
			data[id] = val
		}
	}
	return
}

func (c *cache) Save(ctx context.Context, data map[string]json.RawMessage) {
	for id, data := range data {
		c.cache.Set(id, data)
	}
}

func (c *cache) Invalidate(ctx context.Context, ids []string) {
	for _, id := range ids {
		c.cache.Delete(id)
	}
}
