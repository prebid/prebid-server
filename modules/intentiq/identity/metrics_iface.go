package identity

import (
	"time"

	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache"
)

// Metrics is the full module metrics contract. It embeds the cache-layer health methods
// (cache.Metrics) and adds the per-partner business counters recorded by the enrich and impression
// hooks. The concrete Prometheus implementation lives in metrics.go (agent D); a no-op
// implementation is used when metrics are disabled or no registry is available.
//
// dpi is the partner id (partner-facing data-provider id), attached as a Prometheus label rather
// than the Java "_<dpi>" name suffix. layer is "l1"/"l2" (cache.Layer.Token()); keyType is
// "first_party"/"third_party"/"device" (cache.KeyType.Token()).
type Metrics interface {
	cache.Metrics

	// Cache outcome counters (recorded from the cache Result in the enrich hook).
	CacheHit(layer, keyType, dpi string)
	CacheMiss(keyType, dpi string)
	CacheNegativeHit(layer, keyType, dpi string)
	CacheInProgress(layer, keyType, dpi string)

	// Resolution / enrichment counters.
	APISuccess(dpi string)
	APIError(dpi string)
	APILatency(d time.Duration, dpi string)
	Enriched(dpi string)
	EidsNone(dpi string)
	SkipNoEndpoint(dpi string)
	TerminationCause(tc int64, dpi string)

	// Whole-flow latency (enrich entry -> bid release), recorded once per auction.
	FlowLatency(d time.Duration, dpi string)

	// Impression-report counters.
	ImpressionReported(dpi string)
	ImpressionError(dpi string)
}
