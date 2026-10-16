package provider

import (
	"fmt"

	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider/aerospike"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider/redis"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider/valkey"
)

type CacheType string

const (
	CacheTypeRedis     CacheType = "redis"
	CacheTypeValkey    CacheType = "valkey"
	CacheTypeAerospike CacheType = "aerospike"
)

// Configs holds every backend block the host config may carry; only the one named by cacheType is
// validated and connected.
type Configs struct {
	Redis     *redis.Config
	Valkey    *valkey.Config
	Aerospike *aerospike.Config
}

func New(cacheType CacheType, cfgs Configs) (cache.Store, error) {
	switch cacheType {
	case CacheTypeAerospike:
		if cfgs.Aerospike == nil {
			return nil, fmt.Errorf("aerospike: no aerospike config")
		}
		if err := cfgs.Aerospike.Validate(); err != nil {
			return nil, fmt.Errorf("aerospike: %w", err)
		}
		store, err := aerospike.New(*cfgs.Aerospike)
		if err != nil {
			return nil, err
		}
		return store, nil

	case CacheTypeRedis:
		if cfgs.Redis == nil {
			return nil, fmt.Errorf("redis: no redis config")
		}
		if err := cfgs.Redis.Validate(); err != nil {
			return nil, fmt.Errorf("redis: %w", err)
		}
		store, err := redis.New(*cfgs.Redis)
		if err != nil {
			return nil, err
		}
		return store, nil

	case CacheTypeValkey:
		if cfgs.Valkey == nil {
			return nil, fmt.Errorf("valkey: no valkey config")
		}
		if err := cfgs.Valkey.Validate(); err != nil {
			return nil, fmt.Errorf("valkey: %w", err)
		}
		store, err := valkey.New(*cfgs.Valkey)
		if err != nil {
			return nil, err
		}
		return store, nil

	default:
		return nil, fmt.Errorf("unknown cache type %q (want %q, %q or %q)",
			cacheType, CacheTypeRedis, CacheTypeValkey, CacheTypeAerospike)
	}
}
