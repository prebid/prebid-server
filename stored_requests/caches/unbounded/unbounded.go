package in_memory

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/prebid/prebid-server/stored_requests"
)

func NewUnboundedCache() stored_requests.Cache {
	return &unboundedCache{
		requestDataCache: &sync.Map{},
		impDataCache:     &sync.Map{},
	}
}

type unboundedCache struct {
	requestDataCache *sync.Map
	impDataCache     *sync.Map
}

func (c *unboundedCache) Get(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage) {
	requestData = doGet(c.requestDataCache, requestIDs)
	impData = doGet(c.impDataCache, impIDs)
}

func doGet(data *sync.Map, ids []string) (loaded map[string]json.RawMessage) {
	if len(ids) == 0 {
		return
	}

	loaded = make(map[string]json.RawMessage, len(ids))
	for _, id := range ids {
		if val, ok := data.Load(id); ok {
			loaded[id] = val.(json.RawMessage)
		}
	}
	return
}

func (c *unboundedCache) Save(ctx context.Context, storedRequests map[string]json.RawMessage, storedImps map[string]json.RawMessage) {
	doSave(c.requestDataCache, storedRequests)
	doSave(c.impDataCache, storedImps)
}

func doSave(data *sync.Map, newData map[string]json.RawMessage) {
	for id, val := range newData {
		data.Store(id, val)
	}
}

func (c *unboundedCache) Invalidate(ctx context.Context, requestIDs []string, impIDs []string) {
	doDelete(c.requestDataCache, requestIDs)
	doDelete(c.impDataCache, impIDs)
}

func doDelete(data *sync.Map, ids []string) {
	for _, id := range ids {
		data.Delete(id)
	}
}
