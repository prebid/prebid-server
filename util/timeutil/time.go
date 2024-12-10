package timeutil

import (
	"sync"
	"time"
	_ "time/tzdata"
)

type Time interface {
	Now() time.Time
}

// RealTime wraps the time package for testability
type RealTime struct{}

func (c *RealTime) Now() time.Time {
	return time.Now()
}

type LocationCache struct {
	cache map[string]*LocationCacheResult
	mu    sync.RWMutex
}

func NewLocationCache() *LocationCache {
	return &LocationCache{
		cache: make(map[string]*LocationCacheResult),
	}
}

type LocationCacheResult struct {
	loc *time.Location
	err error
}

// LoadLocation wraps standard package time.LoadLocation, cache the results
func (l *LocationCache) LoadLocation(name string) (*time.Location, error) {
	l.mu.RLock()
	result, ok := l.cache[name]
	l.mu.RUnlock()

	if ok {
		return result.loc, result.err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	result, ok = l.cache[name]
	if ok {
		return result.loc, result.err
	}

	loc, err := time.LoadLocation(name)
	// cache it whether it succeeds or fails. avoid cache penetration caused by invalid timezones.
	l.cache[name] = &LocationCacheResult{loc: loc, err: err}
	return loc, err
}

var defaultLocationCache = NewLocationCache()

func LoadLocation(name string) (*time.Location, error) {
	return defaultLocationCache.LoadLocation(name)
}
