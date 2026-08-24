package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTime struct {
	now time.Time
}

func newFakeTime() *fakeTime {
	return &fakeTime{now: time.Unix(1000, 0)}
}

func (f *fakeTime) Now() time.Time {
	return f.now
}

func (f *fakeTime) Add(d time.Duration) {
	f.now = f.now.Add(d)
}

func TestLRUCacheSaveGetAndStaleState(t *testing.T) {
	clk := newFakeTime()
	cache, err := NewLRUCache[string, string](2, time.Hour, clk)
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

func TestLRUCacheNonPositiveTTLNeverStales(t *testing.T) {
	clk := newFakeTime()
	cache, err := NewLRUCache[string, string](2, 0, clk)
	require.NoError(t, err)

	cache.Save("a", "v1")
	clk.Add(24 * time.Hour)

	v, ok, stale := cache.Get("a")
	assert.True(t, ok)
	assert.False(t, stale)
	assert.Equal(t, "v1", v)
}

func TestLRUCacheEvictsLeastRecentlyUsedEntry(t *testing.T) {
	cache, err := NewLRUCache[string, string](2, time.Hour, newFakeTime())
	require.NoError(t, err)

	cache.Save("a", "v1")
	cache.Save("b", "v2")
	_, ok, _ := cache.Get("a")
	require.True(t, ok, "touch a so b becomes least recently used")
	cache.Save("c", "v3")

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
	cache, err := NewLRUCache[string, string](2, time.Hour, newFakeTime())
	require.NoError(t, err)

	cache.Save("a", "v1")
	cache.Invalidate("a")

	_, ok, stale := cache.Get("a")
	assert.False(t, ok)
	assert.False(t, stale)
}

func TestNewLRUCacheRejectsInvalidSize(t *testing.T) {
	cache, err := NewLRUCache[string, string](0, time.Hour, newFakeTime())

	assert.Nil(t, cache)
	assert.Error(t, err)
}

func TestNewLRUCacheRequiresTime(t *testing.T) {
	cache, err := NewLRUCache[string, string](1, time.Hour, nil)

	assert.Nil(t, cache)
	assert.EqualError(t, err, "time is required")
}
