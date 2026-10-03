// Package identity implements the IntentIQ Identity module for prebid-server.
//
// At the processed-auction-request stage it calls the IntentIQ Bid Enhancement S2S API and merges
// the resolved eids into user.eids before the request is sent to bidders. Optionally, at the
// auction-response stage it reports each winning bid to the IntentIQ impression API. A two-layer
// (in-process + a shared store) alias cache with negative caching and in-progress dedup fronts the
// resolution call. This is a Go port of the prebid-server-java extra/modules/intentiq-identity
// module.
package identity

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/enrichment"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/metrics"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// Module implements the processed-auction-request (enrich) and auction-response (impression) hooks.
type Module struct {
	cfg          Config
	httpClient   *http.Client
	api          *enrichment.Client
	keyExtractor *FirstPartyKeyExtractor
	metrics      metrics.Metrics

	// The following are non-nil only when caching is enabled.
	cache *cache.IdentityCache
	store cache.Store
}

var (
	_ hookstage.ProcessedAuctionRequest = (*Module)(nil)
	_ hookstage.AuctionResponse         = (*Module)(nil)
)

func Builder(rawCfg json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	cfg := defaultConfig()
	if len(rawCfg) > 0 {
		if err := jsonutil.Unmarshal(rawCfg, &cfg); err != nil {
			return nil, fmt.Errorf("intentiq-identity: failed to parse config: %w", err)
		}
	}

	// Recording needs both an opt-in and somewhere to publish to: the host leaves
	// deps.MetricsRegisterer nil when it exposes no Prometheus registry.
	metricsOn := cfg.MetricsEnabled && deps.MetricsRegisterer != nil

	m := &Module{
		cfg:          cfg,
		httpClient:   deps.HTTPClient,
		api:          enrichment.NewClient(deps.HTTPClient),
		keyExtractor: NewFirstPartyKeyExtractor(cfg.Cache.MaxKeys),
		metrics:      metrics.New(deps.MetricsRegisterer, cfg.MetricsEnabled),
	}

	if cfg.Cache.Enabled {
		store, err := provider.New(cfg.GetCacheType(), cfg.GetCacheConfigs())
		if err != nil {
			return nil, fmt.Errorf("intentiq-identity: cache backend: %w", err)
		}
		m.store = store
		m.cache = cache.NewIdentityCache(cfg.Cache.MaxSize, cfg.Cache.TTLPolicy(), store, m.metrics)
	}

	cacheDesc := "off"
	if cfg.Cache.Enabled {
		cacheDesc = cfg.Cache.Provider
	}
	logger.Infof("intentiq-identity: started dpi=%s cache=%s metrics=%t trace=%t timeout=%s",
		cfg.PartnerID, cacheDesc, metricsOn, cfg.TraceEnabled, cfg.GetTimeout())

	return m, nil
}

func (m *Module) Shutdown() error {
	if m.store != nil {
		return m.store.Close()
	}
	return nil
}
