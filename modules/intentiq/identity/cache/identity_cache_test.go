package cache

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider/memory"
)

var errL2Down = errors.New("l2 unreachable")

// countingMetrics counts calls and captures the registered gauges.
type countingMetrics struct {
	mu           sync.Mutex
	l1PutError   atomic.Int64
	l2GetLatency atomic.Int64
	l2PutLatency atomic.Int64
	// l2Requests counts by "<op>:<result>".
	l2Requests map[string]int64

	l1Size      func() int64
	l1Evictions func() int64
}

// CacheLookup is recorded by the enrich hook, not by this layer; stubbed to satisfy CacheMetrics.
func (m *countingMetrics) CacheLookup(result, layer, dpi string) {}

func (m *countingMetrics) L1PutError()                  { m.l1PutError.Add(1) }
func (m *countingMetrics) L2GetLatency(d time.Duration) { m.l2GetLatency.Add(1) }
func (m *countingMetrics) L2PutLatency(d time.Duration) { m.l2PutLatency.Add(1) }

func (m *countingMetrics) L2Request(op, result string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.l2Requests == nil {
		m.l2Requests = map[string]int64{}
	}
	m.l2Requests[op+":"+result]++
}

// l2Count reads one "<op>:<result>" tally.
func (m *countingMetrics) l2Count(op, result string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.l2Requests[op+":"+result]
}

func (m *countingMetrics) RegisterL1Gauges(size, evictions func() int64) {
	m.l1Size = size
	m.l1Evictions = evictions
}

// testTTL uses distinct ceilings so each one is identifiable in an assertion.
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

// newTestCache wires an IdentityCache over an in-process L2. Backend behavior is covered by the
// conformance suite each provider package runs, so these tests stay about the cache logic.
func newTestCache(t *testing.T) (*IdentityCache, *memory.Store, *countingMetrics) {
	t.Helper()
	store := memory.New()
	t.Cleanup(func() { _ = store.Close() })

	metrics := &countingMetrics{}
	c := NewIdentityCache(1024*1024, testTTL(), store, metrics)
	return c, store, metrics
}

func eids(source string) []openrtb2.EID {
	return []openrtb2.EID{{Source: source, UIDs: []openrtb2.UID{{ID: "abc"}}}}
}

func keysFor(pairs ...Key) []Key { return pairs }

func TestNewIdentityCacheRegistersL1Gauges(t *testing.T) {
	c, _, m := newTestCache(t)
	require.NotNil(t, m.l1Size)
	require.NotNil(t, m.l1Evictions)

	assert.Equal(t, int64(0), m.l1Size())
	c.Put(context.Background(), keysFor(Key{Key: "k1", Type: FirstParty}), eids("a.com"), "", nil, 0)
	assert.Equal(t, int64(1), m.l1Size())
	assert.Equal(t, int64(0), m.l1Evictions())
}

func TestGetEmptyKeysIsMiss(t *testing.T) {
	c, _, _ := newTestCache(t)
	assert.Equal(t, Miss, c.Get(context.Background(), nil).State)
	assert.Equal(t, Miss, c.Get(context.Background(), []Key{}).State)
}

func TestL1Hit(t *testing.T) {
	c, _, m := newTestCache(t)
	ctx := context.Background()
	keys := keysFor(Key{Key: "fp", Type: FirstParty})
	c.Put(ctx, keys, eids("a.com"), "", nil, 0)

	// An L1 hit must not move the L2 counter the Put left behind.
	before := m.l2GetLatency.Load()
	res := c.Get(ctx, keys)
	assert.Equal(t, Hit, res.State)
	assert.Equal(t, LayerL1, res.Layer)
	assert.Equal(t, FirstParty, res.KeyType)
	require.Len(t, res.Eids, 1)
	assert.Equal(t, "a.com", res.Eids[0].Source)
	assert.Equal(t, before, m.l2GetLatency.Load())
}

func TestL2HitPromotesToL1(t *testing.T) {
	c, store, m := newTestCache(t)
	ctx := context.Background()
	keys := keysFor(Key{Key: "fp", Type: FirstParty})

	// Seed L2 without touching c's L1, so only c's L2 probe can find the entry.
	seed := NewIdentityCache(1024*1024, testTTL(), store, &countingMetrics{})
	seed.Put(ctx, keys, eids("b.com"), "", nil, 0)

	res := c.Get(ctx, keys)
	assert.Equal(t, Hit, res.State)
	assert.Equal(t, LayerL2, res.Layer)
	require.Len(t, res.Eids, 1)
	assert.Equal(t, "b.com", res.Eids[0].Source)
	assert.GreaterOrEqual(t, m.l2GetLatency.Load(), int64(1))

	// Promoted into L1: the second Get must not issue another L2 GET.
	before := m.l2GetLatency.Load()
	res2 := c.Get(ctx, keys)
	assert.Equal(t, Hit, res2.State)
	assert.Equal(t, LayerL1, res2.Layer)
	assert.Equal(t, before, m.l2GetLatency.Load())
}

func TestAliasBackfill(t *testing.T) {
	c, _, _ := newTestCache(t)
	ctx := context.Background()
	k0 := Key{Key: "primary", Type: FirstParty}
	k1 := Key{Key: "secondary", Type: ThirdParty}

	c.Put(ctx, keysFor(k0), eids("a.com"), "", nil, 0)

	// A lookup carrying both keys hits under k0 and back-fills k1.
	res := c.Get(ctx, keysFor(k0, k1))
	assert.Equal(t, Hit, res.State)
	assert.Equal(t, FirstParty, res.KeyType)

	// A later lookup carrying only k1 now hits, from the backfill.
	res2 := c.Get(ctx, keysFor(k1))
	assert.Equal(t, Hit, res2.State)
	assert.Equal(t, ThirdParty, res2.KeyType)
	require.Len(t, res2.Eids, 1)
	assert.Equal(t, "a.com", res2.Eids[0].Source)
}

func TestNegativeSentinel(t *testing.T) {
	c, _, _ := newTestCache(t)
	ctx := context.Background()
	keys := keysFor(Key{Key: "neg", Type: ThirdParty})
	c.PutNegative(ctx, keys, "", nil, 0)

	res := c.Get(ctx, keys)
	assert.Equal(t, Negative, res.State)
	assert.Equal(t, LayerL1, res.Layer)
	assert.Equal(t, ThirdParty, res.KeyType)
	assert.Nil(t, res.Eids)
}

func TestInProgressMarkerFallback(t *testing.T) {
	c, _, _ := newTestCache(t)
	ctx := context.Background()
	k0 := Key{Key: "ip", Type: Device}
	c.PutInProgress(ctx, keysFor(k0))

	res := c.Get(ctx, keysFor(k0))
	assert.Equal(t, InProgress, res.State)
	assert.Equal(t, LayerL1, res.Layer)
	assert.Equal(t, Device, res.KeyType)
}

func TestResolvedWinsOverInProgress(t *testing.T) {
	c, _, _ := newTestCache(t)
	ctx := context.Background()
	k0 := Key{Key: "ipkey", Type: FirstParty}
	k1 := Key{Key: "reskey", Type: ThirdParty}
	c.PutInProgress(ctx, keysFor(k0))
	c.Put(ctx, keysFor(k1), eids("a.com"), "", nil, 0)

	// k0 (in-progress) is higher priority but a resolved entry under k1 wins.
	res := c.Get(ctx, keysFor(k0, k1))
	assert.Equal(t, Hit, res.State)
	assert.Equal(t, ThirdParty, res.KeyType)
}

func TestFullMiss(t *testing.T) {
	c, _, m := newTestCache(t)
	ctx := context.Background()
	res := c.Get(ctx, keysFor(Key{Key: "nope", Type: FirstParty}, Key{Key: "nada", Type: Device}))
	assert.Equal(t, Miss, res.State)
	assert.Equal(t, LayerNone, res.Layer)
	// Every key was probed in L2.
	assert.GreaterOrEqual(t, m.l2GetLatency.Load(), int64(2))
}

func TestPositiveTTLCappedByCeiling(t *testing.T) {
	c, store, _ := newTestCache(t)
	ctx := context.Background()
	// cttl far above the device ceiling (5m) -> capped at 5m.
	k := Key{Key: "devkey", Type: Device}
	c.Put(ctx, keysFor(k), eids("a.com"), "", nil, 24*time.Hour)

	ttl := store.TTL("devkey")
	assert.LessOrEqual(t, ttl, 5*time.Minute)
	assert.Greater(t, ttl, 4*time.Minute)
}

func TestPositiveTTLUsesCttlWhenBelowCeiling(t *testing.T) {
	c, store, _ := newTestCache(t)
	ctx := context.Background()
	// cttl below the first-party ceiling (1h) -> honored.
	k := Key{Key: "fpkey", Type: FirstParty}
	c.Put(ctx, keysFor(k), eids("a.com"), "", nil, 3*time.Minute)

	ttl := store.TTL("fpkey")
	assert.LessOrEqual(t, ttl, 3*time.Minute)
	assert.Greater(t, ttl, 2*time.Minute)
}

func TestNegativeTTL(t *testing.T) {
	c, store, _ := newTestCache(t)
	ctx := context.Background()

	// Without cttl -> configured NegativeTTL (2m).
	c.PutNegative(ctx, keysFor(Key{Key: "n1", Type: ThirdParty}), "", nil, 0)
	ttl := store.TTL("n1")
	assert.LessOrEqual(t, ttl, 2*time.Minute)
	assert.Greater(t, ttl, time.Minute)

	// With cttl (below first-party ceiling) -> honored.
	c.PutNegative(ctx, keysFor(Key{Key: "n2", Type: ThirdParty}), "", nil, 30*time.Second)
	ttl2 := store.TTL("n2")
	assert.LessOrEqual(t, ttl2, 30*time.Second)
	assert.Greater(t, ttl2, 15*time.Second)

	// With absurd cttl -> capped by first-party ceiling (1h).
	c.PutNegative(ctx, keysFor(Key{Key: "n3", Type: ThirdParty}), "", nil, 48*time.Hour)
	ttl3 := store.TTL("n3")
	assert.LessOrEqual(t, ttl3, time.Hour)
	assert.Greater(t, ttl3, 59*time.Minute)
}

func TestL2GetFailOpen(t *testing.T) {
	c, store, m := newTestCache(t)
	ctx := context.Background()
	keys := keysFor(Key{Key: "k", Type: FirstParty})

	// Close L2: a Get for a key absent from L1 must fall through to a miss, and count an L2 error.
	store.SetErr(errL2Down)

	res := c.Get(ctx, keys)
	assert.Equal(t, Miss, res.State)
	assert.GreaterOrEqual(t, m.l2Count("get", "error"), int64(1))
}

func TestL2PutFailOpenStillLivesInL1(t *testing.T) {
	c, store, m := newTestCache(t)
	ctx := context.Background()
	keys := keysFor(Key{Key: "k", Type: FirstParty})

	store.SetErr(errL2Down)
	c.Put(ctx, keys, eids("a.com"), "", nil, 0)
	assert.GreaterOrEqual(t, m.l2Count("put", "error"), int64(1))

	// Entry still lives in L1 despite the L2 write failure.
	res := c.Get(ctx, keys)
	assert.Equal(t, Hit, res.State)
	assert.Equal(t, LayerL1, res.Layer)
}

func TestExpiredEntryTreatedAsAbsent(t *testing.T) {
	c, store, _ := newTestCache(t)
	ctx := context.Background()

	// Backend TTL is long but the absolute expiry is past: Exp must win.
	past := Entry{Eids: eids("a.com"), Exp: time.Now().UnixMilli() - 1000}
	value, err := json.Marshal(past)
	require.NoError(t, err)
	require.NoError(t, store.Put(ctx, "stale", string(value), time.Hour))

	res := c.Get(ctx, keysFor(Key{Key: "stale", Type: FirstParty}))
	assert.Equal(t, Miss, res.State)
}

func TestPutKeepsAbTestUUIDAndTc(t *testing.T) {
	c, _, _ := newTestCache(t)
	ctx := context.Background()
	keys := keysFor(Key{Key: "fp", Type: FirstParty})

	c.Put(ctx, keys, eids("a.com"), "ab-1", tcPtr(120088), 0)

	res := c.Get(ctx, keys)
	require.Equal(t, Hit, res.State)
	assert.Equal(t, "ab-1", res.AbTestUUID)
	require.NotNil(t, res.Tc)
	assert.Equal(t, int64(120088), *res.Tc)
}

func TestNegativeKeepsAbTestUUIDAndTc(t *testing.T) {
	c, _, _ := newTestCache(t)
	ctx := context.Background()
	keys := keysFor(Key{Key: "neg", Type: ThirdParty})

	c.PutNegative(ctx, keys, "ab-2", tcPtr(120088), 0)

	res := c.Get(ctx, keys)
	require.Equal(t, Negative, res.State)
	assert.Equal(t, "ab-2", res.AbTestUUID)
	require.NotNil(t, res.Tc)
	assert.Equal(t, int64(120088), *res.Tc)
}

func TestBackfillKeepsAbTestUUIDAndTc(t *testing.T) {
	c, _, _ := newTestCache(t)
	ctx := context.Background()
	k0 := Key{Key: "primary", Type: FirstParty}
	k1 := Key{Key: "secondary", Type: ThirdParty}

	c.Put(ctx, keysFor(k0), eids("a.com"), "ab-3", tcPtr(7), 0)
	c.Get(ctx, keysFor(k0, k1)) // hits k0, back-fills k1

	res := c.Get(ctx, keysFor(k1))
	require.Equal(t, Hit, res.State)
	assert.Equal(t, "ab-3", res.AbTestUUID)
	require.NotNil(t, res.Tc)
	assert.Equal(t, int64(7), *res.Tc)
}

func tcPtr(v int64) *int64 { return &v }
