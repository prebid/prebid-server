package in_memory

import (
	"context"
	"encoding/json"

	"github.com/coocood/freecache"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests"
)

// NewLRUCache returns an in-memory Cache which evicts items if:
//
// 1. They haven't been used within the TTL.
// 2. The cache is too large. This will cause the least recently used items to be evicted.
//
// For no TTL, use ttlSeconds <= 0
func NewLRUCache(cfg *config.InMemoryCache) stored_requests.Cache {
	return &cache{
		requestDataCache: freecache.NewCache(cfg.RequestCacheSize),
		impDataCache:     freecache.NewCache(cfg.ImpCacheSize),
		ttlSeconds:       cfg.TTL,
	}
}

type cache struct {
	requestDataCache *freecache.Cache
	impDataCache     *freecache.Cache
	ttlSeconds       int
}

func (c *cache) Get(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage) {
	requestData = doGet(c.requestDataCache, requestIDs)
	impData = doGet(c.impDataCache, impIDs)
	return
}

func doGet(cache *freecache.Cache, ids []string) (data map[string]json.RawMessage) {
	data = make(map[string]json.RawMessage, len(ids))
	for _, id := range ids {
		if bytes, err := cache.Get([]byte(id)); err == nil {
			data[id] = bytes
		} else if err != freecache.ErrNotFound {
			glog.Errorf("unexpected error from freecache: %v", err)
		}
	}
	return
}

func (c *cache) Save(ctx context.Context, storedRequests map[string]json.RawMessage, storedImps map[string]json.RawMessage) {
	c.doSave(c.requestDataCache, storedRequests)
	c.doSave(c.impDataCache, storedImps)
}

func (c *cache) doSave(cache *freecache.Cache, values map[string]json.RawMessage) {
	for id, data := range values {
		if err := cache.Set([]byte(id), data, c.ttlSeconds); err != nil {
			glog.Errorf("error saving value in freecache: %v", err)
		}
	}
}

func (c *cache) Invalidate(ctx context.Context, requestIDs []string, impIDs []string) {
	doInvalidate(c.requestDataCache, requestIDs)
	doInvalidate(c.impDataCache, impIDs)
}

func doInvalidate(cache *freecache.Cache, ids []string) {
	for _, id := range ids {
		cache.Del([]byte(id))
	}
}

func (c *cache) Update(ctx context.Context, storedRequests map[string]json.RawMessage, storedImps map[string]json.RawMessage) {
	c.doUpdate(c.requestDataCache, storedRequests)
	c.doUpdate(c.impDataCache, storedImps)
}

func (c *cache) doUpdate(cache *freecache.Cache, values map[string]json.RawMessage) {
	toSave := make(map[string]json.RawMessage, len(values))
	for id, data := range values {
		if _, err := cache.Get([]byte(id)); err == nil {
			toSave[id] = data
		} else if err != freecache.ErrNotFound {
			glog.Errorf("unexpected error from freecache: %v", err)
		}
	}

	if len(toSave) > 0 {
		c.doSave(cache, toSave)
	}
}
