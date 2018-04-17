package unbounded

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/prebid/prebid-server/config"
)

// An unbounded cache has infinite capacity for Stored Requests. This will offer ideal performance
// at very low-latencies, and should probably be used as long as you have enough memory to keep all your
// Stored Requests there.
//
// If you have too many Stored Requests to save in memory, see stored_requests/caches/lru/lru.go instead.

func NewUnboundedCache(cfg *config.UnboundedCache) *UnboundedCache {
	return &UnboundedCache{
		requestDataCache: make(map[string]json.RawMessage, cfg.InitialStoredRequestCapacity),
		requestDataLock:  &sync.RWMutex{},
		impDataCache:     make(map[string]json.RawMessage, cfg.InitialStoredImpCapacity),
		impDataLock:      &sync.RWMutex{},
	}
}

type UnboundedCache struct {
	requestDataCache map[string]json.RawMessage
	requestDataLock  *sync.RWMutex

	impDataCache map[string]json.RawMessage
	impDataLock  *sync.RWMutex
}

func (c *UnboundedCache) Get(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage) {
	requestData = doGet(c.requestDataCache, c.requestDataLock, requestIDs)
	impData = doGet(c.impDataCache, c.impDataLock, impIDs)
	return
}

func doGet(data map[string]json.RawMessage, lock *sync.RWMutex, ids []string) (loaded map[string]json.RawMessage) {
	if len(ids) == 0 {
		return
	}

	loaded = make(map[string]json.RawMessage, len(ids))
	lock.RLock()
	defer lock.RUnlock()

	for _, id := range ids {
		if val, ok := data[id]; ok {
			loaded[id] = val
		}
	}
	return
}

func (c *UnboundedCache) Save(ctx context.Context, storedRequests map[string]json.RawMessage, storedImps map[string]json.RawMessage) {
	doSave(c.requestDataCache, c.requestDataLock, storedRequests)
	doSave(c.impDataCache, c.impDataLock, storedImps)
}

func doSave(data map[string]json.RawMessage, lock *sync.RWMutex, newData map[string]json.RawMessage) {
	if len(newData) == 0 {
		return
	}
	lock.Lock()
	defer lock.Unlock()
	for id, val := range newData {
		data[id] = val
	}
}

func (c *UnboundedCache) Invalidate(ctx context.Context, requestIDs []string, impIDs []string) {
	doDelete(c.requestDataCache, c.requestDataLock, requestIDs)
	doDelete(c.impDataCache, c.impDataLock, impIDs)
}

func doDelete(data map[string]json.RawMessage, lock *sync.RWMutex, ids []string) {
	if len(ids) == 0 {
		return
	}
	lock.Lock()
	defer lock.Unlock()
	for _, id := range ids {
		delete(data, id)
	}
}
