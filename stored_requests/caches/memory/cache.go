package memory

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/PubMatic-OpenWrap/prebid-server/config"
	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests"
	"github.com/coocood/freecache"
	"github.com/golang/glog"
)

// NewCache returns an in-memory Cache which evicts items if:
//
// 1. They haven't been used within the TTL.
// 2. The cache is too large. This will cause the least recently used items to be evicted.
//
// For no TTL, use ttlSeconds <= 0
func NewCache(cfg *config.InMemoryCache) stored_requests.Cache {
	return &cache{
		requestDataCache: newCacheForWithLimits(cfg.RequestCacheSize, cfg.TTL, "Request"),
		impDataCache:     newCacheForWithLimits(cfg.ImpCacheSize, cfg.TTL, "Imp"),
	}
}

func newCacheForWithLimits(size int, ttl int, dataType string) mapLike {
	if ttl > 0 && size <= 0 {
		glog.Fatal("No in-memory caches defined with a finite TTL but unbounded size. Config validation should have caught this. Failing fast because something is buggy.")
	}
	if size > 0 {
		glog.Infof("Using a Stored %s in-memory cache. Max size: %d bytes. TTL: %d seconds.", dataType, size, ttl)
		return &pbsLRUCache{
			Cache:      freecache.NewCache(size),
			ttlSeconds: ttl,
		}
	} else {
		glog.Infof("Using an unbounded Stored %s in-memory cache.", dataType)
		return &pbsSyncMap{&sync.Map{}}
	}
}

type cache struct {
	requestDataCache mapLike
	impDataCache     mapLike
}

func (c *cache) Get(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage) {
	requestData = doGet(c.requestDataCache, requestIDs)
	impData = doGet(c.impDataCache, impIDs)
	return
}

func doGet(cache mapLike, ids []string) (data map[string]json.RawMessage) {
	data = make(map[string]json.RawMessage, len(ids))
	for _, id := range ids {
		if val, ok := cache.Get(id); ok {
			data[id] = val
		}
	}
	return
}

func (c *cache) Save(ctx context.Context, storedRequests map[string]json.RawMessage, storedImps map[string]json.RawMessage) {
	c.doSave(c.requestDataCache, storedRequests)
	c.doSave(c.impDataCache, storedImps)
}

func (c *cache) doSave(cache mapLike, values map[string]json.RawMessage) {
	for id, data := range values {
		cache.Set(id, data)
	}
}

func (c *cache) Invalidate(ctx context.Context, requestIDs []string, impIDs []string) {
	doInvalidate(c.requestDataCache, requestIDs)
	doInvalidate(c.impDataCache, impIDs)
}

func doInvalidate(cache mapLike, ids []string) {
	for _, id := range ids {
		cache.Delete(id)
	}
}
