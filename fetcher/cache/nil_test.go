package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNilCacheAlwaysMisses(t *testing.T) {
	cache := NilCache[string, string]{}

	cache.Save("a", "v1")
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
