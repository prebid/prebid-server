package rulesengine

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/prebid/prebid-server/v4/util/timeutil"
)

type accountID = string

type cacher interface {
	Get(string) *cacheEntry
	Set(string, *cacheEntry)
	Delete(id accountID)
	Expired(time.Time) bool
}

// defaultMaxEntries is the number of accounts the cache holds when the module
// config doesn't set "maxcacheentries".
const defaultMaxEntries = 10000

type cache struct {
	sync.Mutex
	m                atomic.Value
	maxEntries       int
	clock            atomic.Uint64 // logical clock, ticks on every use
	lastUsed         sync.Map      // accountID -> *atomic.Uint64, clock value at the last use
	refreshFrequency time.Duration
	t                timeutil.Time
}

func NewCache(refreshRateSeconds int) *cache {
	var atomicMap atomic.Value
	atomicMap.Store(make(map[accountID]*cacheEntry))

	return &cache{
		m:                atomicMap,
		maxEntries:       defaultMaxEntries,
		refreshFrequency: time.Duration(refreshRateSeconds) * time.Second,
		t:                &timeutil.RealTime{},
	}
}

func (c *cache) Expired(coTimestamp time.Time) bool {
	if c.refreshFrequency <= 0 {
		return false
	}
	currentTime := c.t.Now()
	delta := currentTime.Sub(coTimestamp)
	return delta.Seconds() > c.refreshFrequency.Seconds()
}

// Get has been implemented to read from the cache without further synchronization
func (c *cache) Get(id accountID) *cacheEntry {
	m1 := c.m.Load().(map[accountID]*cacheEntry)
	if cachedObj, exists := m1[id]; exists {
		c.touch(id)
		return cachedObj
	}
	return nil
}

// touch records that the account was just used. It only updates a counter that Set already
// created, so it takes no lock and adds no entries for ids that are not cached.
func (c *cache) touch(id accountID) {
	if v, ok := c.lastUsed.Load(id); ok {
		v.(*atomic.Uint64).Store(c.clock.Add(1))
	}
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

	// Account IDs come from the request, so the number of keys must be bounded. When a new
	// key would exceed the limit, drop the entry that was used the longest ago. Entries
	// are refreshed on every Get, so accounts that keep getting auctions stay cached.
	if _, exists := m1[id]; !exists && c.maxEntries > 0 && len(m2) >= c.maxEntries {
		var oldestID accountID
		var oldest uint64
		first := true
		for k := range m2 {
			var used uint64
			if v, ok := c.lastUsed.Load(k); ok {
				used = v.(*atomic.Uint64).Load()
			}
			if first || used < oldest {
				oldestID, oldest, first = k, used, false
			}
		}
		delete(m2, oldestID)
		c.lastUsed.Delete(oldestID)
	}

	// A write counts as a use, so a fresh entry is not the first one evicted.
	stamp, _ := c.lastUsed.LoadOrStore(id, new(atomic.Uint64))
	stamp.(*atomic.Uint64).Store(c.clock.Add(1))

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
	c.lastUsed.Delete(id)

	return
}
