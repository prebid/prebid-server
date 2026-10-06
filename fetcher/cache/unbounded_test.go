package cache

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnboundedCacheSaveGetAndStaleState(t *testing.T) {
	clk := newFakeTime()
	cache, err := NewUnboundedCache[string, string](time.Hour, clk)
	require.NoError(t, err)

	cache.Save("a", "v1")

	v, ok, stale := cache.Get("a")
	assert.True(t, ok)
	assert.False(t, stale)
	assert.Equal(t, "v1", v)

	clk.Add(2 * time.Hour)

	v, ok, stale = cache.Get("a")
	assert.True(t, ok, "stale entries are still returned; the fetcher decides how to refresh")
	assert.True(t, stale)
	assert.Equal(t, "v1", v)
}

func TestUnboundedCacheNonPositiveTTLNeverStales(t *testing.T) {
	clk := newFakeTime()
	cache, err := NewUnboundedCache[string, string](0, clk)
	require.NoError(t, err)

	cache.Save("a", "v1")
	clk.Add(24 * time.Hour)

	v, ok, stale := cache.Get("a")
	assert.True(t, ok)
	assert.False(t, stale)
	assert.Equal(t, "v1", v)
}

func TestUnboundedCacheDoesNotEvictByItemCount(t *testing.T) {
	cache, err := NewUnboundedCache[string, string](time.Hour, newFakeTime())
	require.NoError(t, err)

	for i := 0; i < 1000; i++ {
		key := strconv.Itoa(i)
		cache.Save(key, "v"+key)
	}

	for i := 0; i < 1000; i++ {
		key := strconv.Itoa(i)
		v, ok, stale := cache.Get(key)
		assert.True(t, ok)
		assert.False(t, stale)
		assert.Equal(t, "v"+key, v)
	}
}

func TestUnboundedCacheInvalidateRemovesEntry(t *testing.T) {
	cache, err := NewUnboundedCache[string, string](time.Hour, newFakeTime())
	require.NoError(t, err)

	cache.Save("a", "v1")
	cache.Invalidate("a")

	_, ok, stale := cache.Get("a")
	assert.False(t, ok)
	assert.False(t, stale)
}

func TestNewUnboundedCacheRequiresTime(t *testing.T) {
	cache, err := NewUnboundedCache[string, string](time.Hour, nil)

	assert.Nil(t, cache)
	assert.EqualError(t, err, "time is required")
}
