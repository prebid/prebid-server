package inmemorycache

import (
	"github.com/coocood/freecache"
	"github.com/prebid/prebid-server/stored_requests/cache/cacher"
)

type client struct {
	lru *freecache.Cache
}

func init() {
	var c = &client{}
	cacher.Register(c)
}

const (
	defaultCacheSize = int64(512 * 1024 * 1024)
	minimumCacheSize = int64(512 * 1024)
)

// Name of the client
func (c *client) Name() string {
	return cacher.InMemoryCache
}

// Ping will always return nil because it is in-memory
func (c *client) Ping() error {
	return nil
}

func (c *client) Close() {
	c.lru.Clear()
}

// Configure will set the size of the in-memory cache
// The cache size will be set to 512KB at minimum.
// If the size is set relatively large, you should call
// `debug.SetGCPercent()`, set it to a much smaller value
// to limit the memory consumption and GC pause time.
func (c *client) Configure(config interface{}) error {

	// use this size by default
	var cacheSize = defaultCacheSize

	// NOTE: ignore all of this for now. changing how this works
	var settings = &cacher.Settings{}

	// if we are provided with settings and the cache size is greater than 0, then use that value
	if settings != nil && settings.Size > 0 {
		cacheSize = settings.Size
	}

	// make sure that the cache size is at least 512kb
	// (freecache requires us to have at least 512kb)
	if cacheSize < minimumCacheSize {
		cacheSize = minimumCacheSize
	}

	c.lru = freecache.NewCache(int(cacheSize))
	return nil
}

// Get a string by its key
func (c *client) Get(key string) (string, error) {
	value, err := c.lru.Get([]byte(key))
	if err != nil {
		return "", cacher.ErrDoesNotExist
	}
	return string(value), nil
}

// Set a key and value with TTL in seconds
func (c *client) Set(key, value string, ttl uint) error {
	return c.lru.Set([]byte(key), []byte(value), int(ttl))
}
