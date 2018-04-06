package in_memory

import (
	"context"
	"encoding/json"

	"github.com/coocood/freecache"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests"
)

// These prefixes are used when generating cache keys so that there's no chance
// of conflict between the two types.
const (
	reqPrefix = "r"
	impPrefix = "i"
)

// NewLRUCache returns an in-memory Cache which evicts items if:
//
// 1. They haven't been used within the TTL.
// 2. The cache is too large. This will cause the least recently used items to be evicted.
//
// For no TTL, use ttlSeconds <= 0
func NewLRUCache(cfg *config.InMemoryCache) stored_requests.Cache {
	return &cache{
		lru:        freecache.NewCache(cfg.Size),
		ttlSeconds: cfg.TTL,
	}
}

type cache struct {
	lru        *freecache.Cache
	ttlSeconds int
}

func (c *cache) GetRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage) {
	requestData = c.doGet(reqPrefix, requestIDs)
	impData = c.doGet(impPrefix, impIDs)
	return
}

func (c *cache) SaveRequests(ctx context.Context, storedRequests map[string]json.RawMessage, storedImps map[string]json.RawMessage) {
	c.doSave(reqPrefix, storedRequests)
	c.doSave(impPrefix, storedImps)
}

func (c *cache) doGet(prefix string, ids []string) (data map[string]json.RawMessage) {
	data = make(map[string]json.RawMessage, len(ids))
	for _, id := range ids {
		if bytes, err := c.lru.Get([]byte(prefix + id)); err == nil {
			data[id] = bytes
		} else if err != freecache.ErrNotFound {
			glog.Errorf("unexpected error from freecache: %v", err)
		}
	}
	return
}

func (c *cache) doSave(prefix string, values map[string]json.RawMessage) {
	for id, data := range values {
		if err := c.lru.Set([]byte(prefix+id), data, c.ttlSeconds); err != nil {
			glog.Errorf("error saving value in freecache: %v", err)
		}
	}
}
