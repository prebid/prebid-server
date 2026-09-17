package doohqty

import (
	"encoding/json"

	"github.com/coocood/freecache"
)

type cachedImpressionValue struct {
	Found bool            `json:"found"`
	Value impressionValue `json:"value,omitempty"`
}

type valueCache struct {
	cache     *freecache.Cache
	marshal   func(v any) ([]byte, error)
	unmarshal func(data []byte, v any) error
}

func newValueCache(sizeBytes int) *valueCache {
	return &valueCache{
		cache:     freecache.NewCache(sizeBytes),
		marshal:   json.Marshal,
		unmarshal: json.Unmarshal,
	}
}

func (c *valueCache) get(key lookupKey) (impressionValue, bool, bool) {
	if c == nil || c.cache == nil {
		return impressionValue{}, false, false
	}

	data, err := c.cache.Get([]byte(key.cacheKey()))
	if err != nil {
		return impressionValue{}, false, false
	}

	var entry cachedImpressionValue
	if err := c.unmarshal(data, &entry); err != nil {
		return impressionValue{}, false, false
	}

	return entry.Value, entry.Found, true
}

func (c *valueCache) setValueWithTTL(key lookupKey, value impressionValue, ttl int) {
	c.set(key, cachedImpressionValue{Found: true, Value: value}, ttl)
}

func (c *valueCache) setMissWithTTL(key lookupKey, ttl int) {
	c.set(key, cachedImpressionValue{Found: false}, ttl)
}

func (c *valueCache) set(key lookupKey, entry cachedImpressionValue, ttl int) {
	if c == nil || c.cache == nil || ttl <= 0 {
		return
	}

	data, err := c.marshal(entry)
	if err != nil {
		return
	}

	_ = c.cache.Set([]byte(key.cacheKey()), data, ttl)
}
