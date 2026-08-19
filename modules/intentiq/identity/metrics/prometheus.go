package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var l2LatencyBuckets = []float64{.0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1}

const (
	Namespace = "iiq"
	Subsystem = "identity"
	Prefix    = Namespace + "_" + Subsystem + "_"
)

type promMetrics struct {
	reg prometheus.Registerer

	cacheLookup *prometheus.CounterVec // {result, layer, partner_id}

	requests           *prometheus.CounterVec // {partner_id}
	apiSuccess         *prometheus.CounterVec // {partner_id}
	apiError           *prometheus.CounterVec // {reason, status_code, partner_id}
	enriched           *prometheus.CounterVec // {partner_id}
	notEnriched        *prometheus.CounterVec // {reason, partner_id}
	impressionReported *prometheus.CounterVec // {partner_id}
	impressionError    *prometheus.CounterVec // {partner_id}

	apiLatency *prometheus.HistogramVec // {partner_id}

	// Unlabeled: L2 latency and L1 health are process-wide, not per-partner.
	l2GetLatency prometheus.Histogram
	l2PutLatency prometheus.Histogram
	l2Requests   *prometheus.CounterVec // {op, result}
	l1PutError   prometheus.Counter

	l1GaugesOnce sync.Once
}

func newPromMetrics(reg prometheus.Registerer) *promMetrics {
	f := promauto.With(reg)

	opts := func(name, help string) prometheus.CounterOpts {
		return prometheus.CounterOpts{Namespace: Namespace, Subsystem: Subsystem, Name: name, Help: help}
	}
	counterVec := func(name, help string, labels ...string) *prometheus.CounterVec {
		return f.NewCounterVec(opts(name, help), labels)
	}
	counter := func(name, help string) prometheus.Counter {
		return f.NewCounter(opts(name, help))
	}
	histVec := func(name, help string, buckets []float64, labels ...string) *prometheus.HistogramVec {
		return f.NewHistogramVec(prometheus.HistogramOpts{Namespace: Namespace, Subsystem: Subsystem, Name: name, Help: help, Buckets: buckets}, labels)
	}
	hist := func(name, help string, buckets []float64) prometheus.Histogram {
		return f.NewHistogram(prometheus.HistogramOpts{Namespace: Namespace, Subsystem: Subsystem, Name: name, Help: help, Buckets: buckets})
	}

	return &promMetrics{
		reg: reg,

		cacheLookup: counterVec("cache_lookup_total", "Cache lookups by result: hit=positive entry served (no API call), miss=no positive entry (true miss, negative sentinel or in-flight marker). layer is l1/l2 on a hit, none on a miss. Sum = total lookups.", "result", "layer", "partner_id"),

		requests:           counterVec("requests_total", "Enrich-hook invocations (identity ingress QPS), by partner_id.", "partner_id"),
		apiSuccess:         counterVec("api_success_total", "Resolution API responded 2xx and parsed OK, by partner_id.", "partner_id"),
		apiError:           counterVec("api_error_total", "Resolution S2S failures, by reason (timeout|transport|status|body_read|parse|request), status_code (HTTP code when a response was received, else empty), and partner_id.", "reason", "status_code", "partner_id"),
		enriched:           counterVec("enriched_total", "Resolutions that added >=1 eid to user.eids (a match), by partner_id.", "partner_id"),
		notEnriched:        counterVec("not_enriched_total", "Resolutions that added no eids, by reason (no_ids|no_ids_cached|in_progress|no_endpoint) and partner_id. S2S failures are counted in api_error_total.", "reason", "partner_id"),
		impressionReported: counterVec("impression_reported_total", "Winning bids reported to the reports_endpoint, by partner_id.", "partner_id"),
		impressionError:    counterVec("impression_error_total", "Impression-report calls that failed, by partner_id.", "partner_id"),

		apiLatency: histVec("api_latency_seconds", "Resolution API call duration in seconds, by partner_id.", prometheus.DefBuckets, "partner_id"),

		l2GetLatency: hist("l2_get_latency_seconds", "L2 (shared store) GET duration in seconds.", l2LatencyBuckets),
		l2PutLatency: hist("l2_put_latency_seconds", "L2 (shared store) PUT duration in seconds.", l2LatencyBuckets),
		l2Requests:   counterVec("l2_requests_total", "L2 (shared store) operations, by op (get|put) and result (hit|miss|stored|error).", "op", "result"),

		l1PutError: counter("l1_put_error_total", "L1 writes that did not land, chiefly freecache ErrLargeEntry (entry over cache.max_size/1024)."),
	}
}

func (m *promMetrics) CacheLookup(result, layer, dpi string) {
	m.cacheLookup.WithLabelValues(result, layer, dpi).Inc()
}

func (m *promMetrics) Requests(dpi string)   { m.requests.WithLabelValues(dpi).Inc() }
func (m *promMetrics) APISuccess(dpi string) { m.apiSuccess.WithLabelValues(dpi).Inc() }
func (m *promMetrics) Enriched(dpi string)   { m.enriched.WithLabelValues(dpi).Inc() }

func (m *promMetrics) APIError(dpi, reason, statusCode string) {
	m.apiError.WithLabelValues(reason, statusCode, dpi).Inc()
}

func (m *promMetrics) NotEnriched(reason, dpi string) {
	m.notEnriched.WithLabelValues(reason, dpi).Inc()
}

func (m *promMetrics) ImpressionReported(dpi string) { m.impressionReported.WithLabelValues(dpi).Inc() }
func (m *promMetrics) ImpressionError(dpi string)    { m.impressionError.WithLabelValues(dpi).Inc() }

func (m *promMetrics) APILatency(d time.Duration, dpi string) {
	m.apiLatency.WithLabelValues(dpi).Observe(d.Seconds())
}

func (m *promMetrics) L1PutError() { m.l1PutError.Inc() }

func (m *promMetrics) L2GetLatency(d time.Duration) { m.l2GetLatency.Observe(d.Seconds()) }
func (m *promMetrics) L2PutLatency(d time.Duration) { m.l2PutLatency.Observe(d.Seconds()) }

func (m *promMetrics) L2Request(op, result string) { m.l2Requests.WithLabelValues(op, result).Inc() }

func (m *promMetrics) RegisterL1Gauges(size, evictions func() int64) {
	m.l1GaugesOnce.Do(func() {
		m.registerGaugeFunc("l1_size", "Current L1 entry count (vs cache.max_size).", size)
		m.registerGaugeFunc("l1_eviction", "Cumulative L1 evictions.", evictions)
	})
}

func (m *promMetrics) registerGaugeFunc(name, help string, fn func() int64) {
	g := prometheus.NewGaugeFunc(prometheus.GaugeOpts{Name: Prefix + name, Help: help}, func() float64 {
		return float64(fn())
	})
	if err := m.reg.Register(g); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			panic(err)
		}
	}
}
