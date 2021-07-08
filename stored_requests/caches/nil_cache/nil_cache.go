package nil_cache

import (
	"context"
	"encoding/json"
)

// NilCache is a no-op cache which does nothing useful.
type NilCache struct{}

func (c *NilCache) Get(ctx context.Context, ids []string) map[string]json.RawMessage {
	return make(map[string]json.RawMessage)
}

func (c *NilCache) Save(ctx context.Context, data map[string]json.RawMessage) {
	return
}

func (c *NilCache) Invalidate(ctx context.Context, ids []string) {
	return
}
