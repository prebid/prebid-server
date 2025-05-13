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

// Set stores the data parameter using the id parameter as key in a thread-safe manner. If data
// is already found under the id key, this function swaps its value with the data parameter
func (c *cache) Set(id accountID, data *cacheEntry) {
	if len(id) == 0 {
		return
	}

	c.Lock()
	defer c.Unlock()

	m1 := c.m.Load().(map[accountID]*cacheEntry)
	m2 := make(map[accountID]*cacheEntry)

	// Copy all values that are not under key id (if id exists in this map)
	for k, v := range m1 {
		if k != id {
			m2[k] = v
		}
	}

	// Set new value under account id even if it already exists
	m2[id] = data

	c.m.Store(m2)
	return
}

// Delete removes a cached object without further synchronization
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

	// Copy map onto another map to make it thread-safe
	m2 := make(map[accountID]*cacheEntry)
	for k, v := range m1 {
		// skip the element we want to delete
		if k != id {
			m2[k] = v
		}
	}

	c.m.Store(m2)

	return
}
