package optimization

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/prebid/prebid-server/v3/modules/prebid/optimization/rulesengine"
)

// Is sync.Map the best choice for our use case? Would it better to use a go map with mutex?
// https://pkg.go.dev/sync/atomic#Pointer

// TTL expiration check every 5 min
// When TTL expires, perform raw JSON hash diff to determine if tree rebuild is needed

type hash string
type stage string
type accountID string

type cacheObject struct {
	timestamp    time.Time
	hashedConfig hash
	ruleSets     map[stage][]cacheRuleSet
}
type cacheRuleSet struct {
	name        string
	modelGroups []cacheModelGroup
}
type cacheModelGroup struct {
	weight       int
	version      string
	analyticsKey string
	defaults     []rulesengine.ResultFunction
	root         rulesengine.Node
}

func NewCache() *cache {
	var atomicMap atomic.Value
	atomicMap.Store(make(map[accountID]*cacheObject))

	var mu sync.Mutex

	return &cache{
		m:  atomicMap,
		mu: mu,
	}
}

type cacher interface {
	Get(string) *cacheObject
	Set(string, *cacheObject)
	Delete(id string)
}

type cache struct {
	m  atomic.Value
	mu sync.Mutex
}

// Get has been implemented to read from the cache without further synchronization
func (c *cache) Get(id string) *cacheObject {
	m1 := c.m.Load().(map[accountID]*cacheObject)
	if cachedObj, exists := m1[accountID(id)]; exists {
		return cachedObj
	}
	return nil
}

// Set stores the data parameter using the id parameter as key in a thread-safe manner. If data
// is already found under the id key, this function swaps its value with the data parameter
func (c *cache) Set(id string, data *cacheObject) {
	c.mu.Lock()
	defer c.mu.Unlock()

	m1 := c.m.Load().(map[accountID]*cacheObject)
	m2 := make(map[accountID]*cacheObject)
	for k, v := range m1 {
		// if id exists in our cache, we'll substitute v for the data param at the end
		if k == accountID(id) {
			continue
		}
		m2[k] = v
	}
	m2[accountID(id)] = data
	c.m.Store(m2)
	return
}

// Delete removes a cached object without further synchronization
func (c *cache) Delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	m1 := c.m.Load().(map[accountID]*cacheObject)
	m2 := make(map[accountID]*cacheObject)
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
