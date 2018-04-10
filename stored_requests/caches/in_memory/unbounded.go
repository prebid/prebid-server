package in_memory

import (
	"context"
	"encoding/json"
	"sync"
)

func NewUnboundedCache() {

}

type unboundedCache struct {
	requestDataCache sync.Map
	impDataCache     sync.Map
}

func (c *unboundedCache) Get(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage) {

}

func (c *unboundedCache) Save(ctx context.Context, storedRequests map[string]json.RawMessage, storedImps map[string]json.RawMessage) {

}

func (c *cache) Invalidate(ctx context.Context, requestIDs []string, impIDs []string) {

}
