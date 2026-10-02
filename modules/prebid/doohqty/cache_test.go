package doohqty

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValueCacheStoresHitsAndMisses(t *testing.T) {
	cache := newValueCache(1024 * 1024)
	lookup := lookupKey{AccountID: "acct", Path: lookupPathDOOHID, Key: "screen-1"}
	value := testLookupValue(lookupPathDOOHID, "screen-1", 7.5)

	cache.setValueWithTTL(lookup, value, 60)

	cachedValue, found, cached := cache.get(lookup)
	assert.True(t, cached)
	assert.True(t, found)
	assert.Equal(t, value, cachedValue)

	miss := lookupKey{AccountID: "acct", Path: lookupPathDOOHID, Key: "missing"}
	cache.setMissWithTTL(miss, 60)

	cachedValue, found, cached = cache.get(miss)
	assert.True(t, cached)
	assert.False(t, found)
	assert.Equal(t, impressionValue{}, cachedValue)
}

func TestValueCacheMisses(t *testing.T) {
	cache := newValueCache(1024 * 1024)
	lookup := lookupKey{AccountID: "acct", Path: lookupPathDOOHID, Key: "screen-1"}

	_, _, cached := cache.get(lookup)
	assert.False(t, cached)

	cache.setValueWithTTL(lookup, testLookupValue(lookupPathDOOHID, "screen-1", 1), 0)

	_, _, cached = cache.get(lookup)
	assert.False(t, cached)
}

func TestValueCacheTTLExpiry(t *testing.T) {
	cache := newValueCache(1024 * 1024)
	lookup := lookupKey{AccountID: "acct", Path: lookupPathDOOHID, Key: "screen-1"}

	cache.setValueWithTTL(lookup, testLookupValue(lookupPathDOOHID, "screen-1", 1), 1)
	time.Sleep(1100 * time.Millisecond)

	_, _, cached := cache.get(lookup)
	assert.False(t, cached)
}
