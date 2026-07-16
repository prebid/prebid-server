// Package identity implements the IntentIQ Identity module for prebid-server.
//
// At the processed-auction-request stage it calls the IntentIQ Bid Enhancement S2S API and merges
// the resolved eids into user.eids before the request is sent to bidders. Optionally, at the
// auction-response stage it reports each winning bid to the IntentIQ impression API. A two-layer
// (in-process + Redis) alias cache with negative caching and in-progress dedup fronts the resolution
// call. This is a Go port of the prebid-server-java extra/modules/intentiq-identity module.
package identity

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// redisStatsPollInterval is how often the L2 size/eviction gauges are refreshed.
const redisStatsPollInterval = 30 * time.Second

// Module implements the processed-auction-request (enrich) and auction-response (impression) hooks.
type Module struct {
	cfg          Config
	httpClient   *http.Client
	keyExtractor *FirstPartyKeyExtractor
	metrics      Metrics

	// The following are non-nil only when caching is enabled and Redis is configured.
	cache       *cache.IdentityCache
	reporter    *cache.RedisStatsReporter
	redisClient *redis.Client
}

var (
	_ hookstage.ProcessedAuctionRequest = (*Module)(nil)
	_ hookstage.AuctionResponse         = (*Module)(nil)
)

// Builder is the module entry point invoked by prebid-server to construct the module. Host-level
// config defaults are applied here; account-level config is merged per request (see Config.resolve).
func Builder(rawCfg json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	cfg := defaultConfig()
	if len(rawCfg) > 0 {
		if err := jsonutil.Unmarshal(rawCfg, &cfg); err != nil {
			return nil, fmt.Errorf("intentiq-identity: failed to parse config: %w", err)
		}
	}

	metrics := newMetrics(deps.MetricsRegisterer, cfg.MetricsEnabled)

	m := &Module{
		cfg:          cfg,
		httpClient:   deps.HTTPClient,
		keyExtractor: NewFirstPartyKeyExtractor(cfg.Cache.MaxKeys),
		metrics:      metrics,
	}

	if cfg.Cache.Enabled && cfg.Redis != nil && cfg.Redis.Host != "" {
		client := redis.NewClient(&redis.Options{
			Addr:     net.JoinHostPort(cfg.Redis.Host, strconv.Itoa(cfg.Redis.Port)),
			Password: cfg.Redis.Password,
		})
		store := cache.NewRedisStore(client)
		m.redisClient = client
		m.cache = cache.NewIdentityCache(cfg.CacheMaxSize, cfg.ttlPolicy(), store, metrics)
		m.reporter = cache.NewRedisStatsReporter(store, metrics, redisStatsPollInterval).Start()
	}

	return m, nil
}

// Shutdown stops the Redis stats poller and closes the Redis client. Detected structurally by the
// framework's shutdown handling (modules.Shutdowner).
func (m *Module) Shutdown() error {
	if m.reporter != nil {
		m.reporter.Stop()
	}
	if m.redisClient != nil {
		return m.redisClient.Close()
	}
	return nil
}
