package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatsReporterRegistersL2Gauges(t *testing.T) {
	store, _, _ := newTestStore(t)
	m := &countingMetrics{}
	NewRedisStatsReporter(store, m, time.Minute)
	require.NotNil(t, m.l2Size)
	require.NotNil(t, m.l2Evictions)
	// Before any poll the gauges read zero.
	assert.Equal(t, int64(0), m.l2Size())
	assert.Equal(t, int64(0), m.l2Evictions())
}

func TestStatsReporterPollsImmediatelyAndPeriodically(t *testing.T) {
	store, _, client := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, client.Set(ctx, "a", "1", time.Minute).Err())
	require.NoError(t, client.Set(ctx, "b", "2", time.Minute).Err())

	m := &countingMetrics{}
	r := NewRedisStatsReporter(store, m, 20*time.Millisecond)
	r.Start()
	t.Cleanup(r.Stop)

	// The immediate poll in Start() should have populated the size gauge.
	assert.Equal(t, int64(2), m.l2Size())

	// A third key should be picked up by a later periodic poll.
	require.NoError(t, client.Set(ctx, "c", "3", time.Minute).Err())
	assert.Eventually(t, func() bool {
		return m.l2Size() == 3
	}, time.Second, 10*time.Millisecond)
}

func TestStatsReporterStopIsIdempotent(t *testing.T) {
	store, _, _ := newTestStore(t)
	m := &countingMetrics{}
	r := NewRedisStatsReporter(store, m, 10*time.Millisecond)
	r.Start()

	r.Stop()
	// Second and third Stop must not panic (sync.Once).
	assert.NotPanics(t, func() {
		r.Stop()
		r.Stop()
	})

	// The poller goroutine must have terminated.
	select {
	case <-r.done:
	case <-time.After(time.Second):
		t.Fatal("poller goroutine did not terminate after Stop")
	}
}

func TestStatsReporterStopWithoutStart(t *testing.T) {
	store, _, _ := newTestStore(t)
	m := &countingMetrics{}
	r := NewRedisStatsReporter(store, m, time.Minute)
	// Stop before Start must be safe.
	assert.NotPanics(t, r.Stop)
}

func TestStatsReporterPollSurvivesRedisFailure(t *testing.T) {
	store, mr, client := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, client.Set(ctx, "a", "1", time.Minute).Err())

	m := &countingMetrics{}
	r := NewRedisStatsReporter(store, m, 20*time.Millisecond)
	r.Start()
	t.Cleanup(r.Stop)

	assert.Equal(t, int64(1), m.l2Size())

	// Kill Redis: the previously polled value must be retained (poll failures are swallowed).
	mr.Close()
	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, int64(1), m.l2Size())
}
