package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// statsPollTimeout bounds a single DBSIZE / INFO poll so a slow or stuck Redis instance cannot wedge
// the poller goroutine.
const statsPollTimeout = 2 * time.Second

// RedisStatsReporter periodically polls Redis (L2) DBSIZE and INFO stats evicted_keys and exposes
// them as the global l2.size / l2.eviction gauges. Redis stats are asynchronous and instance-wide,
// so (unlike freecache's in-process L1 counters) they can't be read inside a synchronous gauge —
// this caches the latest poll into atomics the gauges read. Both values are Redis-instance-wide,
// not module-scoped.
type RedisStatsReporter struct {
	store        *RedisStore
	metrics      Metrics
	pollInterval time.Duration

	size      atomic.Int64
	evictions atomic.Int64

	stop     chan struct{}
	stopOnce sync.Once
	done     chan struct{}
}

// NewRedisStatsReporter builds the poller and registers the L2 gauges.
func NewRedisStatsReporter(store *RedisStore, metrics Metrics, pollInterval time.Duration) *RedisStatsReporter {
	r := &RedisStatsReporter{
		store:        store,
		metrics:      metrics,
		pollInterval: pollInterval,
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
	}
	r.metrics.RegisterL2Gauges(
		func() int64 { return r.size.Load() },
		func() int64 { return r.evictions.Load() },
	)
	return r
}

// Start polls once, then launches the periodic poller goroutine. Returns the receiver for fluent
// wiring.
func (r *RedisStatsReporter) Start() *RedisStatsReporter {
	r.poll()
	go func() {
		defer close(r.done)
		ticker := time.NewTicker(r.pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-r.stop:
				return
			case <-ticker.C:
				r.poll()
			}
		}
	}()
	return r
}

// Stop halts the poller goroutine. Safe to call multiple times.
func (r *RedisStatsReporter) Stop() {
	r.stopOnce.Do(func() {
		close(r.stop)
	})
}

// poll reads DBSIZE and evicted_keys and caches them into the gauge atomics. Failures are swallowed
// (the previous value is retained) so a transient Redis blip does not zero the gauges.
func (r *RedisStatsReporter) poll() {
	ctx, cancel := context.WithTimeout(context.Background(), statsPollTimeout)
	defer cancel()

	if size, err := r.store.DBSize(ctx); err == nil {
		r.size.Store(size)
	}
	if evictions, err := r.store.EvictedKeys(ctx); err == nil {
		r.evictions.Store(evictions)
	}
}
