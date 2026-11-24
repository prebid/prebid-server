package fetcher

import (
	"time"

	"github.com/prebid/prebid-server/v4/metrics"
)

// Recorder receives low-cardinality telemetry. The subsystem label is applied by
// the implementation, not passed per call, to keep cardinality bounded.
type Recorder interface {
	CacheHit()
	CacheMiss()
	CacheNegative()
	BackendFetch(operation metrics.FetcherOperation, result metrics.FetcherBackendResult, d time.Duration)
}

// NilRecorder is an explicit Recorder for callers that do not want metrics.
type NilRecorder struct{}

func (NilRecorder) CacheHit()      {}
func (NilRecorder) CacheMiss()     {}
func (NilRecorder) CacheNegative() {}
func (NilRecorder) BackendFetch(metrics.FetcherOperation, metrics.FetcherBackendResult, time.Duration) {
}
