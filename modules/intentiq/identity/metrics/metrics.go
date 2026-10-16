package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type CacheMetrics interface {
	// CacheLookup records the result of a cache lookup.
	// layer is l1 or l2 for hits, and none for misses.
	CacheLookup(result, layer, dpi string)

	// L1PutError counts failed L1 writes, such as entries rejected for being too large.
	L1PutError()

	RegisterL1Gauges(size, evictions func() int64)

	// L2Request records an L2 operation and its result.
	L2Request(op, result string)

	L2GetLatency(d time.Duration)
	L2PutLatency(d time.Duration)
}

type APIMetrics interface {
	APILatency(d time.Duration, dpi string)

	APISuccess(dpi string)

	// APIError records a failed API call.
	// statusCode is empty if no response was received.
	APIError(dpi, reason, statusCode string)
}

type Metrics interface {
	CacheMetrics
	APIMetrics

	// Requests counts enrich-hook invocations.
	Requests(dpi string)

	Enriched(dpi string)

	// NotEnriched records requests that added no EIDs, by reason.
	// API failures are recorded by APIError.
	NotEnriched(reason, dpi string)

	ImpressionReported(dpi string)
	ImpressionError(dpi string)
}

// New returns the Metrics implementation the hooks record through: the Prometheus adapter,
// registering its collectors into reg, or Noop.
//
// reg is the registerer the host handed the module in moduledeps.ModuleDeps. It is nil when the
// host exposes no Prometheus registry, in which case there is nowhere to publish to and recording
// degrades to a no-op rather than failing startup. Called once, from Builder: the collectors are
// registered on construction and re-registering the same series into one registry panics.
func New(reg prometheus.Registerer, enabled bool) Metrics {
	if !enabled || reg == nil {
		return Noop{}
	}
	return newPromMetrics(reg)
}
