package rulesengine

import (
	"sync"
	"sync/atomic"
)

type accountID string

type cacher interface {
	Get(string) *cacheEntry
	Set(string, *cacheEntry)
	Delete(id string)
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
func (c *cache) Get(id string) *cacheEntry {
	m1 := c.m.Load().(map[accountID]*cacheEntry)
	if cachedObj, exists := m1[accountID(id)]; exists {
		return cachedObj
	}
	return nil
}

// Set stores the data parameter using the id parameter as key in a thread-safe manner. If data
// is already found under the id key, this function swaps its value with the data parameter
func (c *cache) Set(id string, data *cacheEntry) {
	c.Lock()
	defer c.Unlock()

	m1 := c.m.Load().(map[accountID]*cacheEntry)
	m2 := make(map[accountID]*cacheEntry)

	// Copy map onto another map to make it thread-safe
	for k, v := range m1 {
		// if id exists in our cache, we'll substitute v for the data param at the end
		if k == accountID(id) {
			continue
		}
		m2[k] = v
	}

	// Set new value under account id even if it already exists
	m2[accountID(id)] = data

	c.m.Store(m2)
	return
}

// Delete removes a cached object without further synchronization
func (c *cache) Delete(id string) {
	if len(id) == 0 {
		return
	}

	c.Lock()
	defer c.Unlock()

	m1 := c.m.Load().(map[accountID]*cacheEntry)
	if _, exists := m1[accountID(id)]; !exists {
		return
	}

	// Copy map onto another map to make it thread-safe
	m2 := make(map[accountID]*cacheEntry)
	for k, v := range m1 {
		// skip the element we want to delete
		if k == accountID(id) {
			continue
		}
		m2[k] = v
	}

	c.m.Store(m2)

	return
}
