package validation

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	vastlint "github.com/aleksUIX/vastlint-go"
	"github.com/prebid/prebid-server/v4/logger"
)

// asyncValidator validates creatives after the hook has returned, so the
// auction never waits for the engine. Results cannot reach the bid or the
// analytics tags of the auction that produced them; they are counted and
// reported in the server log at a fixed interval, and with the cache on they
// serve the next sighting of the same creative. The queue is bounded by count
// and by bytes of markup held: when it is full a creative is counted as not
// validated rather than held.
type asyncValidator struct {
	engine    vastlint.Engine
	cache     *verdictCache
	jobs      chan asyncJob
	done      chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
	maxBytes  int64
	bytes     atomic.Int64
	submitted atomic.Int64
	validated atomic.Int64
	invalid   atomic.Int64
	errors    atomic.Int64 // engine errors, including panics contained in run
	timeouts  atomic.Int64 // waited longer than asyncValidateTimeout for an engine slot
	full      atomic.Int64
	durNs     atomic.Int64
}

type asyncJob struct {
	adm   string
	rules ruleLevels
	key   []byte // cache key, nil when the cache is off
}

// asyncValidateTimeout bounds how long an off-path job waits for an engine
// slot; a job that waits longer is counted under timeouts and dropped, so a
// backlog never pins a worker while the pool is busy with on-path hooks. The
// engine bounds its own work per document once it has a slot.
const asyncValidateTimeout = 100 * time.Millisecond

func newAsyncValidator(engine vastlint.Engine, cache *verdictCache, workers, queue int, queueBytes int64) *asyncValidator {
	if workers < 1 {
		workers = 1
	}
	if queue < 1 {
		queue = 1
	}
	if queueBytes < 1 {
		queueBytes = 64 << 20
	}
	a := &asyncValidator{engine: engine, cache: cache, jobs: make(chan asyncJob, queue), done: make(chan struct{}), maxBytes: queueBytes}
	a.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go a.worker()
	}
	return a
}

// submit queues one creative. It never blocks and never panics: false means
// the queue was full, by count or by bytes of markup held, or the validator
// has been closed. The jobs channel is never closed, so a hook that races a
// shutdown cannot send on a closed channel.
func (a *asyncValidator) submit(adm string, rules ruleLevels, key []byte) bool {
	select {
	case <-a.done:
		a.full.Add(1)
		return false
	default:
	}
	n := int64(len(adm))
	if a.bytes.Add(n) > a.maxBytes {
		a.bytes.Add(-n)
		a.full.Add(1)
		return false
	}
	select {
	case a.jobs <- asyncJob{adm: adm, rules: rules, key: key}:
		a.submitted.Add(1)
		return true
	default:
		a.bytes.Add(-n)
		a.full.Add(1)
		return false
	}
}

func (a *asyncValidator) worker() {
	defer a.wg.Done()
	for {
		select {
		case <-a.done:
			return
		case j := <-a.jobs:
			a.run(j)
		}
	}
}

func (a *asyncValidator) run(j asyncJob) {
	defer a.bytes.Add(-int64(len(j.adm)))
	defer func() {
		// A panic inside the engine binding must not take the process down.
		if r := recover(); r != nil {
			a.errors.Add(1)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), asyncValidateTimeout)
	t := time.Now()
	v, err := a.engine.Validate(ctx, j.adm, vastlint.Options{RuleOverrides: map[string]string(j.rules)})
	cancel()
	a.durNs.Add(int64(time.Since(t)))
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			a.timeouts.Add(1)
		} else {
			a.errors.Add(1)
		}
		return
	}
	a.validated.Add(1)
	if !v.Valid {
		a.invalid.Add(1)
	}
	if a.cache != nil && j.key != nil {
		a.cache.put(j.key, summarize(v, a.engine.Version()))
	}
}

// activity is a sum that changes whenever any counter does; the periodic
// report logs only when it moved.
func (a *asyncValidator) activity() int64 {
	return a.submitted.Load() + a.validated.Load() + a.errors.Load() + a.timeouts.Load() + a.full.Load()
}

func (a *asyncValidator) report() {
	done := a.validated.Load() + a.errors.Load() + a.timeouts.Load()
	var meanUs int64
	if done > 0 {
		meanUs = a.durNs.Load() / done / 1000
	}
	logger.Infof("vastlint async: submitted=%d validated=%d invalid=%d errors=%d timeouts=%d queue_full=%d queue_depth=%d queue_bytes=%d mean_us=%d",
		a.submitted.Load(), a.validated.Load(), a.invalid.Load(), a.errors.Load(), a.timeouts.Load(), a.full.Load(), len(a.jobs), a.bytes.Load(), meanUs)
}

// close stops the workers and waits for the one validation each may have in
// flight. Creatives still queued are dropped; the process is shutting down.
// Safe to call more than once and concurrently with submit.
func (a *asyncValidator) close() {
	a.closeOnce.Do(func() { close(a.done) })
	a.wg.Wait()
}
