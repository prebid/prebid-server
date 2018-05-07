package nil_cache

import (
	"context"
	"encoding/json"
)

// NilCache is a no-op cache which does nothing useful.
type NilCache struct{}

func (c *NilCache) Get(ctx context.Context, requestIDs []string, impIDs []string) (map[string]json.RawMessage, map[string]json.RawMessage) {
	return nil, nil
}
func (c *NilCache) Save(ctx context.Context, storedRequests map[string]json.RawMessage, storedImps map[string]json.RawMessage) {
	return
}

func (c *NilCache) Invalidate(ctx context.Context, requestIDs []string, impIDs []string) {
	return
}
