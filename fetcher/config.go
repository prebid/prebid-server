package fetcher

import (
	"fmt"
	"time"

	fetchercache "github.com/prebid/prebid-server/v4/fetcher/cache"
	fetchersource "github.com/prebid/prebid-server/v4/fetcher/source"
	"github.com/prebid/prebid-server/v4/util/timeutil"
)

// Config contains generic fetcher behavior. It is intentionally data-type
// agnostic; callers provide only Source, Transform, Time and Metrics separately.
type Config struct {
	Cache            CacheConfig
	Refresh          RefreshConfig
	Negative         NegativeConfig
	CoalesceRequests bool
}

// CacheConfig selects the positive-cache policy.
type CacheConfig struct {
	Type       string
	MaxEntries int
	TTL        time.Duration
}

// RefreshConfig selects how cached values are refreshed.
type RefreshConfig struct {
	Mode                     string
	ServeStale               bool
	BackgroundRefreshTimeout time.Duration
	BackgroundRefreshBackoff time.Duration
}

// NegativeConfig selects the negative-cache policy.
type NegativeConfig struct {
	Enabled    bool
	Type       string
	MaxEntries int
	TTL        time.Duration
}

func buildCache[K comparable, V any](cfg CacheConfig, t timeutil.Time) (Cache[K, V], error) {
	switch cfg.Type {
	case "none":
		return fetchercache.NilCache[K, V]{}, nil
	case "unbounded":
		return fetchercache.NewUnboundedCache[K, V](cfg.TTL, t)
	case "", "lru":
		return fetchercache.NewLRUCache[K, V](cfg.MaxEntries, cfg.TTL, t)
	default:
		return nil, fmt.Errorf("cache.type %q is not supported (expected none, unbounded or lru)", cfg.Type)
	}
}

func effectiveCacheConfig(cfg Config) CacheConfig {
	cacheCfg := cfg.Cache
	if cfg.Refresh.Mode == "none" {
		cacheCfg.TTL = 0
	}
	return cacheCfg
}

func buildNegativeStore[K comparable](cfg NegativeConfig, t timeutil.Time) (*NegativeStore[K], error) {
	if !cfg.Enabled {
		return nil, nil
	}

	var negativeCache Cache[K, error]
	var err error
	switch cfg.Type {
	case "", "lru":
		negativeCache, err = fetchercache.NewLRUCache[K, error](cfg.MaxEntries, cfg.TTL, t)
	case "unbounded":
		negativeCache, err = fetchercache.NewUnboundedCache[K, error](cfg.TTL, t)
	default:
		return nil, fmt.Errorf("negative.type %q is not supported (expected lru or unbounded)", cfg.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("negative: %w", err)
	}
	store, err := NewNegativeStore[K](negativeCache)
	if err != nil {
		return nil, fmt.Errorf("negative: %w", err)
	}
	return store, nil
}

func applyRefreshConfig[K comparable](cfg RefreshConfig, src fetchersource.Source[K]) (refreshInBackground bool, preload fetchersource.BulkSource[K], err error) {
	refreshInBackground = cfg.ServeStale
	switch cfg.Mode {
	case "":
	case "ttl":
		refreshInBackground = true
	case "none":
	case "preload":
		refreshInBackground = true
		bulk, ok := src.(fetchersource.BulkSource[K])
		if !ok {
			return false, nil, fmt.Errorf("cache.refresh %q requires a source that supports bulk loading (FetchAll)", cfg.Mode)
		}
		preload = bulk
	// Delta/incremental refresh is planned for the next iteration. It needs a
	// source contract with cursor and delete semantics before it can be wired here.
	default:
		return false, nil, fmt.Errorf("cache.refresh %q is not supported (expected ttl, none or preload)", cfg.Mode)
	}
	return refreshInBackground, preload, nil
}
