package fetcher

import "time"

// Recorder receives low-cardinality telemetry. The subsystem label is applied by
// the implementation, not passed per call, to keep cardinality bounded.
type Recorder interface {
	CacheHit()
	CacheMiss()
	CacheNegative()
	// BackendFetch reports one upstream call. operation is "get", "start" or
	// "background_refresh"; result is "ok", "notfound" or "error".
	BackendFetch(operation string, result string, d time.Duration)
}

// NoopRecorder is an explicit Recorder for callers that do not want metrics.
type NoopRecorder struct{}

func (NoopRecorder) CacheHit()                                  {}
func (NoopRecorder) CacheMiss()                                 {}
func (NoopRecorder) CacheNegative()                             {}
func (NoopRecorder) BackendFetch(string, string, time.Duration) {}
