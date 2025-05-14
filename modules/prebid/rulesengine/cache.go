package rulesengine

import (
	"sync"
	"sync/atomic"
)

type accountID = string

type cacher interface {
	Get(string) *cacheEntry
	Set(string, *cacheEntry)
	Delete(id accountID)
}

type cache struct {
	sync.Mutex
	m atomic.Value
}

func NewCache() *cache {
	var atomicMap atomic.Value
	atomicMap.Store(make(map[accountID]*cacheEntry))

	return &cache{
		m: atomicMap,
	}
}

// Get has been implemented to read from the cache without further synchronization
func (c *cache) Get(id accountID) *cacheEntry {
	m1 := c.m.Load().(map[accountID]*cacheEntry)
	if cachedObj, exists := m1[id]; exists {
		return cachedObj
	}
	return nil
}

// Set stores the data parameter in the data store using the id parameter as key in a
// thread-safe manner by copying the cache-stored map onto a new one and updating in
// an atomic operation. Allows for rewrites: if the key is alreay found in the data
// store, its contents get updated with the data param.
func (c *cache) Set(id accountID, data *cacheEntry) {
	if len(id) == 0 {
		return
	}

	c.Lock()
	defer c.Unlock()

	m1 := c.m.Load().(map[accountID]*cacheEntry)
	m2 := make(map[accountID]*cacheEntry)

	for k, v := range m1 {
		if k != id {
			m2[k] = v
		}
	}

	m2[id] = data

	c.m.Store(m2)
	return
}

// Delete removes a cached object if the id parameter is found as key to stored data in
// the data store. To do this in a thread-safe manner, we copy the cache-stored map onto
// a new map and simply skip the object whose key is equal to our id.
func (c *cache) Delete(id accountID) {
	if len(id) == 0 {
		return
	}

	c.Lock()
	defer c.Unlock()

	m1 := c.m.Load().(map[accountID]*cacheEntry)
	if _, exists := m1[id]; !exists {
		return
	}

	m2 := make(map[accountID]*cacheEntry)
	for k, v := range m1 {
		if k != id {
			m2[k] = v
		}
	}

	c.m.Store(m2)

	return
}
