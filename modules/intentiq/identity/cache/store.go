package cache

import (
	"context"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
)

// Store is a generic, backend-agnostic key/value store used as the shared (L2) layer of
// IdentityCache. The default implementation is Redis (RedisStore); a partner can provide a different
// backend by supplying another implementation. Values are opaque strings (the cache handles
// serialization), so the store stays decoupled from the eid model.
type Store interface {
	Get(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key, value string, ttl time.Duration) error
}

// Entry is the serialized value held in both cache layers. The same JSON shape is stored in L1 and
// L2 so the two layers stay uniform.
//
//   - a positive entry carries Eids, with Negative=false, InProgress=false
//   - a negative sentinel has Negative=true (id known-unresolvable)
//   - an in-progress marker has InProgress=true (a resolution call is in flight)
//
// Exp is the absolute expiry as Unix milliseconds; a read must treat an entry at/after Exp as absent.
type Entry struct {
	Eids       []openrtb2.EID `json:"eids,omitempty"`
	Negative   bool           `json:"negative,omitempty"`
	InProgress bool           `json:"inProgress,omitempty"`
	Exp        int64          `json:"exp"`
}

// Metrics is the cache-layer health contract (implemented by the module's Prometheus metrics). It
// is defined here, in the cache package, so the cache subsystem does not import the parent package
// (which would create an import cycle). Business counters (hit/miss/negative/in_progress by
// layer+keytype+partner) are recorded by the parent package from the Result, not here.
type Metrics interface {
	L1GetError()
	L1PutError()
	L2GetLatency(d time.Duration)
	L2PutLatency(d time.Duration)
	L2GetError()
	L2PutError()
	// RegisterL1Gauges wires the in-process capacity gauges (current size, cumulative evictions).
	RegisterL1Gauges(size, evictions func() int64)
	// RegisterL2Gauges wires the Redis capacity gauges (DBSIZE, cumulative evicted_keys).
	RegisterL2Gauges(size, evictions func() int64)
}
