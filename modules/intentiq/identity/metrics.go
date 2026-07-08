package identity

import (
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// metricPrefix namespaces every module series so they group with the framework's
// modules_module_intentiq_identity_* tree (the Prometheus rendering of the Java
// modules.module.intentiq-identity.custom.* metrics). The Java per-partner "_<dpi>" name suffix
// becomes a "partner" label, and the layer/keytype the Java names embedded become "layer"/"keytype"
// labels — so the same per-partner dashboards apply while cardinality stays bounded.
const metricPrefix = "modules_module_intentiq_identity_custom_"

// l2LatencyBuckets is an ms-scaled bucket set for the Redis (L2) GET/PUT probes, which are typically
// sub-millisecond to a few milliseconds — finer than the default second-scaled buckets.
var l2LatencyBuckets = []float64{.0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1}

// newMetrics returns the module metrics implementation. When reg is nil (Prometheus disabled) or
// enabled is false, it returns a no-op so every call site is metric-agnostic.
func newMetrics(reg prometheus.Registerer, enabled bool) Metrics {
	if reg == nil || !enabled {
		return noopMetrics{}
	}
	return newPromMetrics(reg)
}

// promMetrics is the Prometheus-backed Metrics implementation. Collectors are registered into the
// supplied registry (the dedicated module registry threaded via ModuleDeps.MetricsRegisterer, which
// the server gathers at /metrics). The capacity gauges are registered lazily by RegisterL1Gauges /
// RegisterL2Gauges once the backing closures are known.
type promMetrics struct {
	reg prometheus.Registerer

	// Cache outcome counters (recorded from the cache Result in the enrich hook).
	cacheHit        *prometheus.CounterVec // {layer, keytype, partner}
	cacheMiss       *prometheus.CounterVec // {keytype, partner}
	cacheNegative   *prometheus.CounterVec // {layer, keytype, partner}
	cacheInProgress *prometheus.CounterVec // {layer, keytype, partner}

	// Per-partner business counters ({partner}).
	apiSuccess         *prometheus.CounterVec
	apiError           *prometheus.CounterVec
	enriched           *prometheus.CounterVec
	eidsNone           *prometheus.CounterVec
	skipNoEndpoint     *prometheus.CounterVec
	impressionReported *prometheus.CounterVec
	impressionError    *prometheus.CounterVec
	terminationCause   *prometheus.CounterVec // {tc, partner}

	// Per-partner latency histograms ({partner}).
	apiLatency  *prometheus.HistogramVec
	flowLatency *prometheus.HistogramVec

	// Global (unlabeled) L2 latency histograms.
	l2GetLatency prometheus.Histogram
	l2PutLatency prometheus.Histogram

	// Global (unlabeled) cache health counters.
	l1GetError prometheus.Counter
	l1PutError prometheus.Counter
	l2GetError prometheus.Counter
	l2PutError prometheus.Counter

	// Guards for the lazily-registered capacity gauges (RegisterL1Gauges/RegisterL2Gauges may be
	// called more than once; only the first wins).
	l1GaugesOnce sync.Once
	l2GaugesOnce sync.Once
}

func newPromMetrics(reg prometheus.Registerer) *promMetrics {
	f := promauto.With(reg)

	counterVec := func(name, help string, labels ...string) *prometheus.CounterVec {
		return f.NewCounterVec(prometheus.CounterOpts{Name: metricPrefix + name, Help: help}, labels)
	}
	counter := func(name, help string) prometheus.Counter {
		return f.NewCounter(prometheus.CounterOpts{Name: metricPrefix + name, Help: help})
	}
	histVec := func(name, help string, buckets []float64, labels ...string) *prometheus.HistogramVec {
		return f.NewHistogramVec(prometheus.HistogramOpts{Name: metricPrefix + name, Help: help, Buckets: buckets}, labels)
	}
	hist := func(name, help string, buckets []float64) prometheus.Histogram {
		return f.NewHistogram(prometheus.HistogramOpts{Name: metricPrefix + name, Help: help, Buckets: buckets})
	}

	return &promMetrics{
		reg: reg,

		cacheHit:        counterVec("cache_hit_total", "Positive cache entries served, by layer/keytype/partner.", "layer", "keytype", "partner"),
		cacheMiss:       counterVec("cache_miss_total", "Full cache misses (neither L1 nor L2) that triggered an API call, by keytype/partner.", "keytype", "partner"),
		cacheNegative:   counterVec("cache_negative_hit_total", "Negative-sentinel cache hits (counted as miss, no API call), by layer/keytype/partner.", "layer", "keytype", "partner"),
		cacheInProgress: counterVec("cache_in_progress_total", "In-progress-marker cache hits (duplicate API call skipped), by layer/keytype/partner.", "layer", "keytype", "partner"),

		apiSuccess:         counterVec("api_success_total", "Resolution API responded and parsed OK, by partner.", "partner"),
		apiError:           counterVec("api_error_total", "Resolution API failed/timed out/unparseable, by partner.", "partner"),
		enriched:           counterVec("enriched_total", "Requests whose user.eids were enriched (a match), by partner.", "partner"),
		eidsNone:           counterVec("eids_none_total", "Resolutions that produced no eids, by partner.", "partner"),
		skipNoEndpoint:     counterVec("skip_no_endpoint_total", "Resolutions skipped because no api-endpoint is configured, by partner.", "partner"),
		impressionReported: counterVec("impression_reported_total", "Winning bids reported to the reports-endpoint, by partner.", "partner"),
		impressionError:    counterVec("impression_error_total", "Impression-report calls that failed, by partner.", "partner"),
		terminationCause:   counterVec("termination_cause_total", "Enumerated termination-cause ids (0<=tc<200), by tc/partner.", "tc", "partner"),

		apiLatency:  histVec("api_latency_seconds", "Resolution API call duration in seconds, by partner.", prometheus.DefBuckets, "partner"),
		flowLatency: histVec("flow_latency_seconds", "Whole-flow latency (enrich hook to bid release) in seconds, by partner.", prometheus.DefBuckets, "partner"),

		l2GetLatency: hist("l2_get_latency_seconds", "L2 (Redis) GET duration in seconds.", l2LatencyBuckets),
		l2PutLatency: hist("l2_put_latency_seconds", "L2 (Redis) PUT duration in seconds.", l2LatencyBuckets),

		l1GetError: counter("l1_get_error_total", "L1 (Caffeine-equivalent) read errors (treated as a miss)."),
		l1PutError: counter("l1_put_error_total", "L1 write errors (entry did not land in L1)."),
		l2GetError: counter("l2_get_error_total", "L2 (Redis) GET failures that fell through to a live API call."),
		l2PutError: counter("l2_put_error_total", "L2 (Redis) PUT failures (entry still in L1, not in shared store)."),
	}
}

// --- Cache outcome counters -----------------------------------------------------------------------

func (m *promMetrics) CacheHit(layer, keyType, dpi string) {
	m.cacheHit.WithLabelValues(layer, keyType, dpi).Inc()
}

func (m *promMetrics) CacheMiss(keyType, dpi string) {
	m.cacheMiss.WithLabelValues(keyType, dpi).Inc()
}

func (m *promMetrics) CacheNegativeHit(layer, keyType, dpi string) {
	m.cacheNegative.WithLabelValues(layer, keyType, dpi).Inc()
}

func (m *promMetrics) CacheInProgress(layer, keyType, dpi string) {
	m.cacheInProgress.WithLabelValues(layer, keyType, dpi).Inc()
}

// --- Resolution / enrichment counters -------------------------------------------------------------

func (m *promMetrics) APISuccess(dpi string) { m.apiSuccess.WithLabelValues(dpi).Inc() }
func (m *promMetrics) APIError(dpi string)   { m.apiError.WithLabelValues(dpi).Inc() }
func (m *promMetrics) Enriched(dpi string)   { m.enriched.WithLabelValues(dpi).Inc() }
func (m *promMetrics) EidsNone(dpi string)   { m.eidsNone.WithLabelValues(dpi).Inc() }

func (m *promMetrics) SkipNoEndpoint(dpi string) { m.skipNoEndpoint.WithLabelValues(dpi).Inc() }

func (m *promMetrics) APILatency(d time.Duration, dpi string) {
	m.apiLatency.WithLabelValues(dpi).Observe(d.Seconds())
}

func (m *promMetrics) FlowLatency(d time.Duration, dpi string) {
	m.flowLatency.WithLabelValues(dpi).Observe(d.Seconds())
}

// TerminationCause records only 0<=tc<200 to bound label cardinality (matches the Java impl).
func (m *promMetrics) TerminationCause(tc int64, dpi string) {
	if tc >= 0 && tc < 200 {
		m.terminationCause.WithLabelValues(strconv.FormatInt(tc, 10), dpi).Inc()
	}
}

// --- Impression-report counters -------------------------------------------------------------------

func (m *promMetrics) ImpressionReported(dpi string) { m.impressionReported.WithLabelValues(dpi).Inc() }
func (m *promMetrics) ImpressionError(dpi string)    { m.impressionError.WithLabelValues(dpi).Inc() }

// --- cache.Metrics: shared L1/L2 health -----------------------------------------------------------

func (m *promMetrics) L1GetError() { m.l1GetError.Inc() }
func (m *promMetrics) L1PutError() { m.l1PutError.Inc() }
func (m *promMetrics) L2GetError() { m.l2GetError.Inc() }
func (m *promMetrics) L2PutError() { m.l2PutError.Inc() }

func (m *promMetrics) L2GetLatency(d time.Duration) { m.l2GetLatency.Observe(d.Seconds()) }
func (m *promMetrics) L2PutLatency(d time.Duration) { m.l2PutLatency.Observe(d.Seconds()) }

// RegisterL1Gauges wires the in-process (L1) capacity gauges. Idempotent: only the first call
// registers, so a repeat call cannot trigger an AlreadyRegisteredError.
func (m *promMetrics) RegisterL1Gauges(size, evictions func() int64) {
	m.l1GaugesOnce.Do(func() {
		m.registerGaugeFunc("l1_size", "Current L1 entry count (vs cache-max-size).", size)
		m.registerGaugeFunc("l1_eviction", "Cumulative L1 evictions.", evictions)
	})
}

// RegisterL2Gauges wires the Redis (L2) capacity gauges. Idempotent (see RegisterL1Gauges).
func (m *promMetrics) RegisterL2Gauges(size, evictions func() int64) {
	m.l2GaugesOnce.Do(func() {
		m.registerGaugeFunc("l2_size", "Redis DBSIZE (instance-wide; polled).", size)
		m.registerGaugeFunc("l2_eviction", "Redis cumulative evicted_keys (instance-wide; polled).", evictions)
	})
}

// registerGaugeFunc registers a global GaugeFunc reading the supplied closure. An
// AlreadyRegisteredError (e.g. two Metrics instances sharing a registry) is tolerated so the module
// keeps working rather than panicking.
func (m *promMetrics) registerGaugeFunc(name, help string, fn func() int64) {
	g := prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: metricPrefix + name, Help: help}, func() float64 {
		return float64(fn())
	})
	if err := m.reg.Register(g); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			panic(err)
		}
	}
}

// noopMetrics satisfies Metrics and does nothing. Used when metrics are disabled/unavailable.
type noopMetrics struct{}

func (noopMetrics) CacheHit(layer, keyType, dpi string)           {}
func (noopMetrics) CacheMiss(keyType, dpi string)                 {}
func (noopMetrics) CacheNegativeHit(layer, keyType, dpi string)   {}
func (noopMetrics) CacheInProgress(layer, keyType, dpi string)    {}
func (noopMetrics) APISuccess(dpi string)                         {}
func (noopMetrics) APIError(dpi string)                           {}
func (noopMetrics) APILatency(d time.Duration, dpi string)        {}
func (noopMetrics) Enriched(dpi string)                           {}
func (noopMetrics) EidsNone(dpi string)                           {}
func (noopMetrics) SkipNoEndpoint(dpi string)                     {}
func (noopMetrics) TerminationCause(tc int64, dpi string)         {}
func (noopMetrics) FlowLatency(d time.Duration, dpi string)       {}
func (noopMetrics) ImpressionReported(dpi string)                 {}
func (noopMetrics) ImpressionError(dpi string)                    {}
func (noopMetrics) L1GetError()                                   {}
func (noopMetrics) L1PutError()                                   {}
func (noopMetrics) L2GetLatency(d time.Duration)                  {}
func (noopMetrics) L2PutLatency(d time.Duration)                  {}
func (noopMetrics) L2GetError()                                   {}
func (noopMetrics) L2PutError()                                   {}
func (noopMetrics) RegisterL1Gauges(size, evictions func() int64) {}
func (noopMetrics) RegisterL2Gauges(size, evictions func() int64) {}
