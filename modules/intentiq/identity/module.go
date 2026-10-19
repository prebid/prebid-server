// Package identity implements IntentIQ identity enrichment and impression reporting.
package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	iiqcache "github.com/Intent-IQ/identity-go/cache"
	iiqenrichment "github.com/Intent-IQ/identity-go/enrichment"
	iiqreportingapi "github.com/Intent-IQ/identity-go/iiqapi/reporting"
	iiqs2s "github.com/Intent-IQ/identity-go/iiqapi/s2s"
	iiqaerospike "github.com/Intent-IQ/identity-go/integrations/aerospike"
	iiqprometheus "github.com/Intent-IQ/identity-go/integrations/prometheus"
	iiqredis "github.com/Intent-IQ/identity-go/integrations/redis"
	iiqvalkey "github.com/Intent-IQ/identity-go/integrations/valkey"
	iiqreporting "github.com/Intent-IQ/identity-go/reporting"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
)

type Module struct {
	config   Config
	enricher iiqenrichment.Enricher
	reporter iiqreporting.Reporter
	store    iiqcache.Store
}

var (
	_ hookstage.ProcessedAuctionRequest = (*Module)(nil)
	_ hookstage.AuctionResponse         = (*Module)(nil)
)

func Builder(rawConfig json.RawMessage, dependencies moduledeps.ModuleDeps) (interface{}, error) {
	config := defaultConfig()
	if len(rawConfig) > 0 {
		if err := json.Unmarshal(rawConfig, &config); err != nil {
			return nil, fmt.Errorf("intentiq identity: parse config: %w", err)
		}
	}
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("intentiq identity: invalid config: %w", err)
	}

	httpClient := dependencies.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	log := prebidLogger{}

	var enrichmentMetrics iiqenrichment.Metrics = iiqenrichment.NoopMetrics{}
	var cacheMetrics iiqcache.Metrics = iiqcache.NoopMetrics{}
	var reportingMetrics iiqreporting.Metrics = iiqreporting.NoopMetrics{}
	if config.MetricsEnabled && dependencies.MetricsRegisterer != nil {
		metrics, err := iiqprometheus.New(dependencies.MetricsRegisterer)
		if err != nil {
			return nil, fmt.Errorf("intentiq identity: create metrics: %w", err)
		}
		enrichmentMetrics, cacheMetrics, reportingMetrics = metrics, metrics, metrics
	}

	store, err := newStore(config)
	if err != nil {
		return nil, fmt.Errorf("intentiq identity: create cache store: %w", err)
	}
	closeStore := func() {
		if store != nil {
			_ = store.Close()
		}
	}

	identityCache, err := iiqcache.New(iiqcache.Dependencies{Store: store, Metrics: cacheMetrics, Logger: log}, config.Cache)
	if err != nil {
		closeStore()
		return nil, fmt.Errorf("intentiq identity: create cache: %w", err)
	}

	enricher, err := iiqenrichment.New(iiqenrichment.Dependencies{
		S2S: iiqs2s.NewClient(httpClient), Cache: identityCache, Metrics: enrichmentMetrics,
		Logger: log, MaxConcurrentCalls: config.MaxConcurrentCalls,
	}, config.Cache.MaxKeys)
	if err != nil {
		closeStore()
		return nil, fmt.Errorf("intentiq identity: create enricher: %w", err)
	}

	reporter, err := iiqreporting.New(iiqreporting.Dependencies{
		API: iiqreportingapi.NewClient(httpClient), Metrics: reportingMetrics, Logger: log,
	}, iiqreporting.Config{
		MaxConcurrentCalls: config.Reporting.MaxConcurrentCalls,
	})
	if err != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = enricher.Shutdown(ctx)
		closeStore()
		return nil, fmt.Errorf("intentiq identity: create reporter: %w", err)
	}

	return &Module{config: config, enricher: enricher, reporter: reporter, store: store}, nil
}

func newStore(config Config) (iiqcache.Store, error) {
	if !config.Cache.Enabled {
		return nil, nil
	}
	switch config.Cache.Provider {
	case "aerospike":
		if config.Aerospike == nil {
			return nil, fmt.Errorf("aerospike configuration is required")
		}
		if err := config.Aerospike.Validate(); err != nil {
			return nil, err
		}
		return iiqaerospike.New(*config.Aerospike)
	case "redis":
		if config.Redis == nil {
			return nil, fmt.Errorf("redis configuration is required")
		}
		if err := config.Redis.Validate(); err != nil {
			return nil, err
		}
		return iiqredis.New(*config.Redis)
	case "valkey":
		if config.Valkey == nil {
			return nil, fmt.Errorf("valkey configuration is required")
		}
		if err := config.Valkey.Validate(); err != nil {
			return nil, err
		}
		return iiqvalkey.New(*config.Valkey)
	default:
		return nil, fmt.Errorf("unknown cache provider %q", config.Cache.Provider)
	}
}

func (m *Module) Shutdown() error {
	if m == nil {
		return nil
	}
	var errs []error
	if m.reporter != nil {
		timeout := time.Duration(m.config.Reporting.ShutdownTimeoutMS) * time.Millisecond
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		if err := m.reporter.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown identity reporting: %w", err))
		}
		cancel()
	}
	if m.enricher != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*m.config.timeout()+time.Second)
		if err := m.enricher.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown identity enricher: %w", err))
		}
		cancel()
	}
	if m.store != nil {
		if err := m.store.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close identity cache store: %w", err))
		}
	}
	return errors.Join(errs...)
}
