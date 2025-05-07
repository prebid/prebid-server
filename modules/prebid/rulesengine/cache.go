package rulesengine

import (
	"sync"
)

// Is sync.Map the best choice for our use case? Would it better to use a go map with mutex?
// https://pkg.go.dev/sync/atomic#Pointer

type cacher interface {
	Get(string) *cacheEntry
	Set(string, cacheEntry)
	Delete(id string)
}

type cache struct {
	*sync.Map
}

func (c *cache) Get(id string) (data *cacheEntry) {
	if val, ok := c.Map.Load(id); ok {
		return val.(*cacheEntry)
	}
	return nil
}

func (c *cache) Set(id string, data cacheEntry) {
	c.Map.Store(id, data)
}

func (c *cache) Delete(id string) {
	c.Map.Delete(id)
}
