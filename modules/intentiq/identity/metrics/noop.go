package metrics

import "time"

// Noop does nothing, so call sites stay metric-agnostic when metrics are disabled. Exported so tests
// can embed it and override only the methods they assert on.
type Noop struct{}

func (Noop) Requests(dpi string)                           {}
func (Noop) CacheLookup(result, layer, dpi string)         {}
func (Noop) APISuccess(dpi string)                         {}
func (Noop) APIError(dpi, reason, statusCode string)       {}
func (Noop) APILatency(d time.Duration, dpi string)        {}
func (Noop) Enriched(dpi string)                           {}
func (Noop) NotEnriched(reason, dpi string)                {}
func (Noop) ImpressionReported(dpi string)                 {}
func (Noop) ImpressionError(dpi string)                    {}
func (Noop) L1PutError()                                   {}
func (Noop) L2GetLatency(d time.Duration)                  {}
func (Noop) L2PutLatency(d time.Duration)                  {}
func (Noop) L2Request(op, result string)                   {}
func (Noop) RegisterL1Gauges(size, evictions func() int64) {}
