package cache

import (
	"errors"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests/cache/cacher"
)

var caches = make([]cacher.Cacher, 0)

// Configure accepts a slice of cache configurations
// This should only be called once on startup
func Configure(configs []interface{}) error {

	for _, c := range configs {
		if _, ok := c.(config.RedisCache); ok {

			cacher.Get(cacher.RedisCache).Configure(c)

			// if redis cache
		} else if _, ok := c.(config.InMemoryCache); ok {
			// if in-memory cache
			cacher.Get(cacher.InMemoryCache).Configure(c)

		}
	}

	return nil
}

var ErrDoesNotExist = errors.New("cache: this key does not exist.")

var DefaultTTL = uint(60 * 60 * 24)

// Get will iterate through the slice of caches until there is a hit
func Get(key string) (resp string, err error) {

	var misses = make([]int, 0)

	for idx, cache := range caches {
		resp, err = cache.Get(key)
		if err == nil {
			break
		}
		misses = append(misses, idx)
	}

	// if we have no misses then we can return back our response and error
	if len(misses) == 0 {
		return resp, err
	}

	// if error is "does not exist" then we missed all of our caches
	if err == cacher.ErrDoesNotExist {
		return resp, ErrDoesNotExist // do not return the cacher error.
	}

	// iterate through all of the missed caches and set the key/value
	for _, idx := range misses {
		caches[idx].Set(key, resp, DefaultTTL)
	}

	return resp, err
}

func Set(key, value string, ttl uint) error {
	return nil

}
