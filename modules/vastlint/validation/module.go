// Package validation is a Prebid Server module that validates the VAST XML
// inside video bids as they come back from bidders, reports findings through
// analytics tags and a bid extension, and can drop bids that fail host-chosen
// rules.
//
// Validation runs in-process. By default the engine is the vastlint rule set
// compiled to WebAssembly and executed by wazero, which needs nothing installed.
// Hosts that want native speed build Prebid Server with -tags vastlint_cgo and
// install libvastlint; the module code is the same either way.
package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"

	vastlint "github.com/aleksUIX/vastlint-go"
	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
)

// Module holds the engine and the host configuration. It is safe for
// concurrent use by the hook executor and implements modules.Shutdowner.
type Module struct {
	cfg       Config
	engine    vastlint.Engine
	catalog   *catalog // the engine's rule ids; validates and canonicalizes overrides
	sample    func() int
	parallel  int // bids of one response validated at once
	async     *asyncValidator
	cache     *verdictCache
	stop      chan struct{}
	closeOnce sync.Once
}

// Builder is the Prebid Server module entry point for vastlint.validation.
// Prebid Server calls it only when the module is enabled in the host config.
func Builder(rawCfg json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	cfg, err := parseConfig(rawCfg)
	if err != nil {
		return nil, err
	}
	engine, err := vastlint.New(context.Background(), vastlint.Config{
		MaxInputBytes: cfg.MaxDocumentBytes,
		PoolSize:      cfg.PoolSize,
	})
	if err != nil {
		return nil, fmt.Errorf("vastlint: engine: %w", err)
	}
	ids := make([]string, 0, len(engine.Rules()))
	for _, r := range engine.Rules() {
		ids = append(ids, r.ID)
	}
	cat := newCatalog(ids)
	rules, rejectOn, err := cat.resolve(cfg.Rules, cfg.RejectOn)
	if err != nil {
		engine.Close()
		return nil, fmt.Errorf("vastlint: %w", err)
	}
	cfg.Rules, cfg.RejectOn = rules, rejectOn
	workers := cfg.PoolSize
	if workers <= 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	// Off-path workers share the engine pool with on-path hooks, so they take
	// at most half of it (at least one); the rest stays free for hooks that
	// must answer inside the group timeout.
	asyncWorkers := max(1, workers/2)
	logger.Infof("vastlint module: engine=%s version=%s rules=%d mode=%s reject_on=%v max_document_bytes=%d sample_percent=%d annotate=%v annotate_rules=%v async_queue=%d async_queue_bytes=%d cache_size=%d cache_ttl_seconds=%d report_interval_seconds=%d",
		engine.Kind(), engine.Version(), len(ids), cfg.Mode, cfg.RejectOn, cfg.MaxDocumentBytes, cfg.SamplePercent, *cfg.AnnotateBids, *cfg.AnnotateRules, cfg.AsyncQueue, cfg.AsyncQueueBytes, cfg.CacheSize, cfg.CacheTTLSeconds, *cfg.ReportIntervalSeconds)
	cache := newVerdictCache(cfg.CacheSize, cfg.CacheTTLSeconds)
	m := &Module{
		cfg:      cfg,
		engine:   engine,
		catalog:  cat,
		sample:   func() int { return rand.Intn(100) },
		parallel: min(maxParallel, workers),
		cache:    cache,
		async:    newAsyncValidator(engine, cache, asyncWorkers, cfg.AsyncQueue, int64(cfg.AsyncQueueBytes)),
		stop:     make(chan struct{}),
	}
	if *cfg.ReportIntervalSeconds > 0 {
		go m.report(time.Duration(*cfg.ReportIntervalSeconds) * time.Second)
	}
	return m, nil
}

// Rules returns the effective host rule overrides with canonical ids.
func (m *Module) Rules() map[string]string {
	out := make(map[string]string, len(m.cfg.Rules))
	for id, level := range m.cfg.Rules {
		out[id] = level
	}
	return out
}

// Shutdown stops the off-path workers and the periodic report and closes the
// engine. Prebid Server calls it once at process shutdown through
// modules.Shutdowner; it is safe to call again and while hooks are running.
func (m *Module) Shutdown() error {
	var err error
	m.closeOnce.Do(func() {
		close(m.stop)
		if m.async != nil {
			m.async.close()
		}
		if m.engine != nil {
			err = m.engine.Close()
		}
	})
	return err
}

// report writes the module's own counters to the server log at a fixed
// interval, only when they changed: the off-path validator's totals and the
// verdict cache's hit rate. Off by default; the framework metrics cover the
// hook itself and the analytics tags carry the same facts per bid.
func (m *Module) report(every time.Duration) {
	tick := time.NewTicker(every)
	defer tick.Stop()
	var lastAsync, lastCache int64
	for {
		select {
		case <-m.stop:
			return
		case <-tick.C:
		}
		if m.async != nil {
			if s := m.async.activity(); s != lastAsync {
				lastAsync = s
				m.async.report()
			}
		}
		if m.cache != nil {
			hits, misses := m.cache.hits.Load(), m.cache.misses.Load()
			if n := hits + misses; n != lastCache {
				lastCache = n
				logger.Infof("vastlint cache: hits=%d misses=%d hit_rate_pct=%d entries=%d", hits, misses, 100*hits/n, m.cache.c.EntryCount())
			}
		}
	}
}

// newWithEngine is used by tests to inject an engine.
func newWithEngine(cfg Config, engine vastlint.Engine, sample func() int) *Module {
	if sample == nil {
		sample = func() int { return 0 }
	}
	cache := newVerdictCache(cfg.CacheSize, cfg.CacheTTLSeconds)
	return &Module{cfg: cfg, engine: engine, sample: sample, parallel: maxParallel, cache: cache, async: newAsyncValidator(engine, cache, 2, 64, 1<<20), stop: make(chan struct{})}
}
