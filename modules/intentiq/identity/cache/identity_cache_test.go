package cache

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingMetrics is a test-only cache.Metrics that counts calls and captures the registered gauges.
type countingMetrics struct {
	l1GetError   atomic.Int64
	l1PutError   atomic.Int64
	l2GetLatency atomic.Int64
	l2PutLatency atomic.Int64
	l2GetError   atomic.Int64
	l2PutError   atomic.Int64

	l1Size      func() int64
	l1Evictions func() int64
	l2Size      func() int64
	l2Evictions func() int64
}

func (m *countingMetrics) L1GetError()                  { m.l1GetError.Add(1) }
func (m *countingMetrics) L1PutError()                  { m.l1PutError.Add(1) }
func (m *countingMetrics) L2GetLatency(d time.Duration) { m.l2GetLatency.Add(1) }
func (m *countingMetrics) L2PutLatency(d time.Duration) { m.l2PutLatency.Add(1) }
func (m *countingMetrics) L2GetError()                  { m.l2GetError.Add(1) }
func (m *countingMetrics) L2PutError()                  { m.l2PutError.Add(1) }

func (m *countingMetrics) RegisterL1Gauges(size, evictions func() int64) {
	m.l1Size = size
	m.l1Evictions = evictions
}

func (m *countingMetrics) RegisterL2Gauges(size, evictions func() int64) {
	m.l2Size = size
	m.l2Evictions = evictions
}

// testTTL is a policy with distinct, easily-asserted ceilings.
func testTTL() TTLPolicy {
	return TTLPolicy{
		Default:           30 * time.Minute,
		FirstPartyCeiling: time.Hour,
		ThirdPartyCeiling: 10 * time.Minute,
		DeviceCeiling:     5 * time.Minute,
		NegativeTTL:       2 * time.Minute,
		InProgressTTL:     15 * time.Second,
	}
}

// newTestCache wires an IdentityCache over a real miniredis-backed L2.
func newTestCache(t *testing.T) (*IdentityCache, *miniredis.Miniredis, *redis.Client, *countingMetrics) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	metrics := &countingMetrics{}
	c := NewIdentityCache(1024*1024, testTTL(), NewRedisStore(client), metrics)
	return c, mr, client, metrics
}

func eids(source string) []openrtb2.EID {
	return []openrtb2.EID{{Source: source, UIDs: []openrtb2.UID{{ID: "abc"}}}}
}

func keysFor(pairs ...Key) []Key { return pairs }

func TestNewIdentityCacheRegistersL1Gauges(t *testing.T) {
	c, _, _, m := newTestCache(t)
	require.NotNil(t, m.l1Size)
	require.NotNil(t, m.l1Evictions)

	assert.Equal(t, int64(0), m.l1Size())
	c.Put(context.Background(), keysFor(Key{Key: "k1", Type: FirstParty}), eids("a.com"), 0)
	assert.Equal(t, int64(1), m.l1Size())
	assert.Equal(t, int64(0), m.l1Evictions())
}

func TestGetEmptyKeysIsMiss(t *testing.T) {
	c, _, _, _ := newTestCache(t)
	assert.Equal(t, Miss, c.Get(context.Background(), nil).State)
	assert.Equal(t, Miss, c.Get(context.Background(), []Key{}).State)
}

func TestL1Hit(t *testing.T) {
	c, _, _, m := newTestCache(t)
	ctx := context.Background()
	keys := keysFor(Key{Key: "fp", Type: FirstParty})
	c.Put(ctx, keys, eids("a.com"), 0)

	// Reset L2 latency counter from the Put so we can assert the hit served from L1.
	before := m.l2GetLatency.Load()
	res := c.Get(ctx, keys)
	assert.Equal(t, Hit, res.State)
	assert.Equal(t, LayerL1, res.Layer)
	assert.Equal(t, FirstParty, res.KeyType)
	require.Len(t, res.Eids, 1)
	assert.Equal(t, "a.com", res.Eids[0].Source)
	// L1 hit must not probe L2.
	assert.Equal(t, before, m.l2GetLatency.Load())
}

func TestL2HitPromotesToL1(t *testing.T) {
	c, _, client, m := newTestCache(t)
	ctx := context.Background()
	keys := keysFor(Key{Key: "fp", Type: FirstParty})

	// Seed L2 directly (bypass L1) via the store, using a fresh cache's Put and then wiping L1 by
	// building a second cache sharing the same redis.
	seed := NewIdentityCache(1024*1024, testTTL(), NewRedisStore(client), &countingMetrics{})
	seed.Put(ctx, keys, eids("b.com"), 0)

	res := c.Get(ctx, keys)
	assert.Equal(t, Hit, res.State)
	assert.Equal(t, LayerL2, res.Layer)
	require.Len(t, res.Eids, 1)
	assert.Equal(t, "b.com", res.Eids[0].Source)
	assert.GreaterOrEqual(t, m.l2GetLatency.Load(), int64(1))

	// Now it must be promoted into L1: a second Get serves from L1 (no new L2 GET).
	before := m.l2GetLatency.Load()
	res2 := c.Get(ctx, keys)
	assert.Equal(t, Hit, res2.State)
	assert.Equal(t, LayerL1, res2.Layer)
	assert.Equal(t, before, m.l2GetLatency.Load())
}

func TestAliasBackfill(t *testing.T) {
	c, _, _, _ := newTestCache(t)
	ctx := context.Background()
	k0 := Key{Key: "primary", Type: FirstParty}
	k1 := Key{Key: "secondary", Type: ThirdParty}

	// Only k0 is populated.
	c.Put(ctx, keysFor(k0), eids("a.com"), 0)

	// A lookup carrying both keys hits under k0 and back-fills k1.
	res := c.Get(ctx, keysFor(k0, k1))
	assert.Equal(t, Hit, res.State)
	assert.Equal(t, FirstParty, res.KeyType)

	// A later lookup carrying only k1 now hits (from L1 backfill).
	res2 := c.Get(ctx, keysFor(k1))
	assert.Equal(t, Hit, res2.State)
	assert.Equal(t, ThirdParty, res2.KeyType)
	require.Len(t, res2.Eids, 1)
	assert.Equal(t, "a.com", res2.Eids[0].Source)
}

func TestNegativeSentinel(t *testing.T) {
	c, _, _, _ := newTestCache(t)
	ctx := context.Background()
	keys := keysFor(Key{Key: "neg", Type: ThirdParty})
	c.PutNegative(ctx, keys, 0)

	res := c.Get(ctx, keys)
	assert.Equal(t, Negative, res.State)
	assert.Equal(t, LayerL1, res.Layer)
	assert.Equal(t, ThirdParty, res.KeyType)
	assert.Nil(t, res.Eids)
}

func TestInProgressMarkerFallback(t *testing.T) {
	c, _, _, _ := newTestCache(t)
	ctx := context.Background()
	k0 := Key{Key: "ip", Type: Device}
	c.PutInProgress(ctx, keysFor(k0))

	res := c.Get(ctx, keysFor(k0))
	assert.Equal(t, InProgress, res.State)
	assert.Equal(t, LayerL1, res.Layer)
	assert.Equal(t, Device, res.KeyType)
}

func TestResolvedWinsOverInProgress(t *testing.T) {
	c, _, _, _ := newTestCache(t)
	ctx := context.Background()
	k0 := Key{Key: "ipkey", Type: FirstParty}
	k1 := Key{Key: "reskey", Type: ThirdParty}
	c.PutInProgress(ctx, keysFor(k0))
	c.Put(ctx, keysFor(k1), eids("a.com"), 0)

	// k0 (in-progress) is higher priority but a resolved entry under k1 wins.
	res := c.Get(ctx, keysFor(k0, k1))
	assert.Equal(t, Hit, res.State)
	assert.Equal(t, ThirdParty, res.KeyType)
}

func TestFullMiss(t *testing.T) {
	c, _, _, m := newTestCache(t)
	ctx := context.Background()
	res := c.Get(ctx, keysFor(Key{Key: "nope", Type: FirstParty}, Key{Key: "nada", Type: Device}))
	assert.Equal(t, Miss, res.State)
	assert.Equal(t, LayerNone, res.Layer)
	// Every key was probed in L2.
	assert.GreaterOrEqual(t, m.l2GetLatency.Load(), int64(2))
}

func TestPositiveTTLCappedByCeiling(t *testing.T) {
	c, _, client, _ := newTestCache(t)
	ctx := context.Background()
	// cttl far above the device ceiling (5m) -> capped at 5m.
	k := Key{Key: "devkey", Type: Device}
	c.Put(ctx, keysFor(k), eids("a.com"), 24*time.Hour)

	ttl := client.PTTL(ctx, "devkey").Val()
	assert.LessOrEqual(t, ttl, 5*time.Minute)
	assert.Greater(t, ttl, 4*time.Minute)
}

func TestPositiveTTLUsesCttlWhenBelowCeiling(t *testing.T) {
	c, _, client, _ := newTestCache(t)
	ctx := context.Background()
	// cttl below the first-party ceiling (1h) -> honored.
	k := Key{Key: "fpkey", Type: FirstParty}
	c.Put(ctx, keysFor(k), eids("a.com"), 3*time.Minute)

	ttl := client.PTTL(ctx, "fpkey").Val()
	assert.LessOrEqual(t, ttl, 3*time.Minute)
	assert.Greater(t, ttl, 2*time.Minute)
}

func TestNegativeTTL(t *testing.T) {
	c, _, client, _ := newTestCache(t)
	ctx := context.Background()

	// Without cttl -> configured NegativeTTL (2m).
	c.PutNegative(ctx, keysFor(Key{Key: "n1", Type: ThirdParty}), 0)
	ttl := client.PTTL(ctx, "n1").Val()
	assert.LessOrEqual(t, ttl, 2*time.Minute)
	assert.Greater(t, ttl, time.Minute)

	// With cttl (below first-party ceiling) -> honored.
	c.PutNegative(ctx, keysFor(Key{Key: "n2", Type: ThirdParty}), 30*time.Second)
	ttl2 := client.PTTL(ctx, "n2").Val()
	assert.LessOrEqual(t, ttl2, 30*time.Second)
	assert.Greater(t, ttl2, 15*time.Second)

	// With absurd cttl -> capped by first-party ceiling (1h).
	c.PutNegative(ctx, keysFor(Key{Key: "n3", Type: ThirdParty}), 48*time.Hour)
	ttl3 := client.PTTL(ctx, "n3").Val()
	assert.LessOrEqual(t, ttl3, time.Hour)
	assert.Greater(t, ttl3, 59*time.Minute)
}

func TestL2GetFailOpen(t *testing.T) {
	c, mr, _, m := newTestCache(t)
	ctx := context.Background()
	keys := keysFor(Key{Key: "k", Type: FirstParty})

	// Close L2: a Get for a key absent from L1 must fall through to a miss, and count an L2 error.
	mr.Close()

	res := c.Get(ctx, keys)
	assert.Equal(t, Miss, res.State)
	assert.GreaterOrEqual(t, m.l2GetError.Load(), int64(1))
}

func TestL2PutFailOpenStillLivesInL1(t *testing.T) {
	c, mr, _, m := newTestCache(t)
	ctx := context.Background()
	keys := keysFor(Key{Key: "k", Type: FirstParty})

	mr.Close()
	c.Put(ctx, keys, eids("a.com"), 0)
	assert.GreaterOrEqual(t, m.l2PutError.Load(), int64(1))

	// Entry still lives in L1 despite the L2 write failure.
	res := c.Get(ctx, keys)
	assert.Equal(t, Hit, res.State)
	assert.Equal(t, LayerL1, res.Layer)
}

func TestExpiredEntryTreatedAsAbsent(t *testing.T) {
	c, _, client, _ := newTestCache(t)
	ctx := context.Background()

	// Write an entry directly to L2 whose absolute expiry is already in the past.
	past := Entry{Eids: eids("a.com"), Exp: time.Now().UnixMilli() - 1000}
	store := NewRedisStore(client)
	value, err := json.Marshal(past)
	require.NoError(t, err)
	require.NoError(t, store.Put(ctx, "stale", string(value), time.Hour))

	res := c.Get(ctx, keysFor(Key{Key: "stale", Type: FirstParty}))
	assert.Equal(t, Miss, res.State)
}
