package doohimpressionvalue

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
	ttl       int
	negTTL    int
	marshal   func(v any) ([]byte, error)
	unmarshal func(data []byte, v any) error
}

func newValueCache(sizeBytes, ttlSeconds, negativeTTLSeconds int) *valueCache {
	return &valueCache{
		cache:     freecache.NewCache(sizeBytes),
		ttl:       ttlSeconds,
		negTTL:    negativeTTLSeconds,
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

func (c *valueCache) setValue(key lookupKey, value impressionValue) {
	c.set(key, cachedImpressionValue{Found: true, Value: value}, c.ttl)
}

func (c *valueCache) setMiss(key lookupKey) {
	c.set(key, cachedImpressionValue{Found: false}, c.negTTL)
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
