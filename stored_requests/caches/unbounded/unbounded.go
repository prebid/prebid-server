package unbounded

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/golang/glog"
)

// An unbounded cache has infinite capacity for Stored Requests. This will offer ideal performance
// at very low-latencies, and should probably be used as long as you have enough memory to keep all your
// Stored Requests there.
//
// If you have too many Stored Requests to save in memory, see stored_requests/caches/lru/lru.go instead.

func NewUnboundedCache() *UnboundedCache {
	return &UnboundedCache{
		requestDataCache: &sync.Map{},
		impDataCache:     &sync.Map{},
	}
}

type UnboundedCache struct {
	requestDataCache *sync.Map
	impDataCache     *sync.Map
}

func (c *UnboundedCache) Get(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage) {
	requestData = doGet(c.requestDataCache, requestIDs)
	impData = doGet(c.impDataCache, impIDs)
	return
}

func doGet(data *sync.Map, ids []string) (loaded map[string]json.RawMessage) {
	if len(ids) == 0 {
		return
	}

	loaded = make(map[string]json.RawMessage, len(ids))

	for _, id := range ids {
		data.Load(id)
		if val, ok := data.Load(id); ok {
			if casted, ok := val.(json.RawMessage); ok {
				loaded[id] = casted
			} else {
				glog.Errorf("unbounded stored request cache saved something other than json.RawMessage. This shouldn't happen.")
			}
		}
	}
	return
}

func (c *UnboundedCache) Save(ctx context.Context, storedRequests map[string]json.RawMessage, storedImps map[string]json.RawMessage) {
	doSave(c.requestDataCache, storedRequests)
	doSave(c.impDataCache, storedImps)
}

func doSave(data *sync.Map, newData map[string]json.RawMessage) {
	if len(newData) == 0 {
		return
	}
	for id, val := range newData {
		data.Store(id, val)
	}
}

func (c *UnboundedCache) Invalidate(ctx context.Context, requestIDs []string, impIDs []string) {
	doDelete(c.requestDataCache, requestIDs)
	doDelete(c.impDataCache, impIDs)
}

func doDelete(data *sync.Map, ids []string) {
	if len(ids) == 0 {
		return
	}
	for _, id := range ids {
		data.Delete(id)
	}
}
