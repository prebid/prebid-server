package cachekit

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLRUCacheSaveGetAndStaleState(t *testing.T) {
	clk := clock.NewMock()
	cache, err := NewLRUCache[string, string](2, clk)
	require.NoError(t, err)

	cache.Save("a", "v1", time.Hour)

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

func TestLRUCacheNonPositiveTTLNeverStales(t *testing.T) {
	clk := clock.NewMock()
	cache, err := NewLRUCache[string, string](2, clk)
	require.NoError(t, err)

	cache.Save("a", "v1", 0)
	clk.Add(24 * time.Hour)

	v, ok, stale := cache.Get("a")
	assert.True(t, ok)
	assert.False(t, stale)
	assert.Equal(t, "v1", v)
}

func TestLRUCacheEvictsLeastRecentlyUsedEntry(t *testing.T) {
	cache, err := NewLRUCache[string, string](2, clock.NewMock())
	require.NoError(t, err)

	cache.Save("a", "v1", time.Hour)
	cache.Save("b", "v2", time.Hour)
	_, ok, _ := cache.Get("a")
	require.True(t, ok, "touch a so b becomes least recently used")
	cache.Save("c", "v3", time.Hour)

	_, ok, _ = cache.Get("b")
	assert.False(t, ok, "least recently used entry should be evicted")

	v, ok, stale := cache.Get("a")
	assert.True(t, ok)
	assert.False(t, stale)
	assert.Equal(t, "v1", v)

	v, ok, stale = cache.Get("c")
	assert.True(t, ok)
	assert.False(t, stale)
	assert.Equal(t, "v3", v)
}

func TestLRUCacheInvalidateRemovesEntry(t *testing.T) {
	cache, err := NewLRUCache[string, string](2, clock.NewMock())
	require.NoError(t, err)

	cache.Save("a", "v1", time.Hour)
	cache.Invalidate("a")

	_, ok, stale := cache.Get("a")
	assert.False(t, ok)
	assert.False(t, stale)
}

func TestNewLRUCacheRejectsInvalidSize(t *testing.T) {
	cache, err := NewLRUCache[string, string](0, clock.NewMock())

	assert.Nil(t, cache)
	assert.Error(t, err)
}

func TestNoCacheAlwaysMisses(t *testing.T) {
	cache := NoCache[string, string]{}

	cache.Save("a", "v1", time.Hour)
	v, ok, stale := cache.Get("a")
	assert.Empty(t, v)
	assert.False(t, ok)
	assert.False(t, stale)

	cache.Invalidate("a")
	v, ok, stale = cache.Get("a")
	assert.Empty(t, v)
	assert.False(t, ok)
	assert.False(t, stale)
}
