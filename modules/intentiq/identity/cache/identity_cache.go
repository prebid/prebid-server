package cache

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/coocood/freecache"
	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/metrics"
)

const minFreeCacheSize = 512 * 1024

// IdentityCache is a dual-layer, multi-key (alias) cache for resolved eids:
// an in-process L1 over a shared Store (L2).
//
// A request yields an ordered list of Keys, one per identifier present. On read the highest-priority
// key with a live entry wins, and that entry is back-filled under every other key that missed, so
// the alias graph grows over time. Differing resolutions are never merged: only the winning entry
// propagates.
//
// Both layers fail open. An L1 or L2 error is counted and treated as a miss, so the auction falls
// through to a live call rather than failing.
type IdentityCache struct {
	local   *freecache.Cache
	store   Store
	ttl     TTLPolicy
	metrics metrics.CacheMetrics
	maxSize int
}

// NewIdentityCache builds the two-layer cache. maxSizeBytes is a byte budget, not an entry count:
// freecache is byte-bounded, unlike the Java module's Caffeine cache.
func NewIdentityCache(maxSizeBytes int, ttl TTLPolicy, store Store, m metrics.CacheMetrics) *IdentityCache {
	size := maxSizeBytes
	if size < minFreeCacheSize {
		size = minFreeCacheSize
	}
	local := freecache.NewCache(size)
	c := &IdentityCache{
		local:   local,
		store:   store,
		ttl:     ttl,
		metrics: m,
		maxSize: size,
	}
	c.metrics.RegisterL1Gauges(
		func() int64 { return local.EntryCount() },
		func() int64 { return local.EvacuateCount() },
	)
	return c
}

// Get sweeps keys in priority order and returns the first live outcome. ctx bounds the L2 probes.
func (c *IdentityCache) Get(ctx context.Context, keys []Key) Result {
	if len(keys) == 0 {
		return MissResult()
	}

	// A resolved entry always wins over an in-progress marker, which is only a fallback. Finding one
	// still short-circuits the L2 probe: this instance already knows a call is in flight.
	var inProgressType KeyType
	inProgressFound := false
	for i, k := range keys {
		entry := c.l1Get(k.Key)
		if entry == nil {
			continue
		}
		if entry.InProgress {
			if !inProgressFound {
				inProgressType = k.Type
				inProgressFound = true
			}
			continue
		}
		c.backfill(ctx, keys, i, *entry)
		return toResult(*entry, k.Type, LayerL1)
	}
	if inProgressFound {
		return InProgressResult(inProgressType, LayerL1)
	}

	// Full L1 miss: same precedence in L2, promoting everything read into L1.
	var l2InProgressType KeyType
	l2InProgressFound := false
	for i, k := range keys {
		entry := c.l2Get(ctx, k.Key)
		if entry == nil {
			continue
		}
		if entry.InProgress {
			if !l2InProgressFound {
				c.l1Promote(k.Key, *entry)
				l2InProgressType = k.Type
				l2InProgressFound = true
			}
			continue
		}
		c.l1Promote(k.Key, *entry)
		c.backfill(ctx, keys, i, *entry)
		return toResult(*entry, k.Type, LayerL2)
	}
	if l2InProgressFound {
		return InProgressResult(l2InProgressType, LayerL2)
	}
	return MissResult()
}

// Put writes the resolved eids under every key, each capped by its ceiling.
func (c *IdentityCache) Put(ctx context.Context, keys []Key, eids []openrtb2.EID, abTestUUID string, tc *int64, cttl time.Duration) {
	for _, k := range keys {
		ttl := c.ttl.EffectiveTTL(k.Type, cttl)
		exp := time.Now().UnixMilli() + ttl.Milliseconds()
		c.writeBoth(ctx, k.Key, Entry{Eids: eids, AbTestUUID: abTestUUID, Tc: tc, Exp: exp}, ttl)
	}
}

// PutNegative suppresses an unresolvable id so it stops re-hitting the upstream API.
func (c *IdentityCache) PutNegative(ctx context.Context, keys []Key, abTestUUID string, tc *int64, cttl time.Duration) {
	ttl := c.ttl.NegativeTTLFor(cttl)
	for _, k := range keys {
		exp := time.Now().UnixMilli() + ttl.Milliseconds()
		c.writeBoth(ctx, k.Key, Entry{Negative: true, AbTestUUID: abTestUUID, Tc: tc, Exp: exp}, ttl)
	}
}

// PutInProgress marks a resolution as in flight so a concurrent request for the same id skips the
// duplicate upstream call. Overwritten by Put/PutNegative on completion; expires otherwise.
func (c *IdentityCache) PutInProgress(ctx context.Context, keys []Key) {
	ttl := c.ttl.InProgressTTL
	for _, k := range keys {
		exp := time.Now().UnixMilli() + ttl.Milliseconds()
		c.writeBoth(ctx, k.Key, Entry{InProgress: true, Exp: exp}, ttl)
	}
}

// backfill propagates the winning entry to every other key missing from L1, capped by
// min(remaining, ceiling).
func (c *IdentityCache) backfill(ctx context.Context, keys []Key, hitIndex int, hit Entry) {
	remaining := hit.Exp - time.Now().UnixMilli()
	if remaining <= 0 {
		return
	}
	for i, k := range keys {
		if i == hitIndex {
			continue
		}
		if c.l1Get(k.Key) != nil {
			continue
		}
		ttlMs := remaining
		if ceil := c.ttl.CeilingFor(k.Type).Milliseconds(); ceil < ttlMs {
			ttlMs = ceil
		}
		exp := time.Now().UnixMilli() + ttlMs
		entry := Entry{
			Eids:       hit.Eids,
			AbTestUUID: hit.AbTestUUID,
			Tc:         hit.Tc,
			Negative:   hit.Negative,
			InProgress: hit.InProgress,
			Exp:        exp,
		}
		c.writeBoth(ctx, k.Key, entry, time.Duration(ttlMs)*time.Millisecond)
	}
}

// writeBoth writes to both layers. The entry stays in L1 even if the L2 write fails.
func (c *IdentityCache) writeBoth(ctx context.Context, key string, entry Entry, ttl time.Duration) {
	value, err := json.Marshal(entry)
	if err != nil {
		c.metrics.L1PutError()
		return
	}
	c.l1Set(key, value, ttl)

	start := time.Now()
	putErr := c.store.Put(ctx, key, string(value), ttl)
	c.metrics.L2PutLatency(time.Since(start))
	if putErr != nil {
		c.metrics.L2Request(metrics.OpPut, metrics.ResultError)
		logger.Warnf("intentiq-identity: cache L2 put failed: %v", putErr)
	} else {
		c.metrics.L2Request(metrics.OpPut, metrics.ResultStored)
	}
}

// l1Set rounds the TTL up to whole seconds, freecache's unit, with a floor of 1 so a sub-second TTL
// never becomes an immediate expiry.
func (c *IdentityCache) l1Set(key string, value []byte, ttl time.Duration) {
	expireSeconds := int(math.Ceil(ttl.Seconds()))
	if expireSeconds < 1 {
		expireSeconds = 1
	}
	if err := c.local.Set([]byte(key), value, expireSeconds); err != nil {
		c.metrics.L1PutError()
	}
}

// l1Promote copies an entry read from L2 into L1, deriving the TTL from its absolute expiry.
func (c *IdentityCache) l1Promote(key string, entry Entry) {
	remaining := entry.Exp - time.Now().UnixMilli()
	if remaining <= 0 {
		return
	}
	value, err := json.Marshal(entry)
	if err != nil {
		c.metrics.L1PutError()
		return
	}
	c.l1Set(key, value, time.Duration(remaining)*time.Millisecond)
}

// l1Get returns nil when the key is absent, expired, or unreadable.
func (c *IdentityCache) l1Get(key string) *Entry {
	value, err := c.local.Get([]byte(key))
	if err != nil {
		// Absent and unreadable are both treated as a miss; the caller falls through to L2.
		return nil
	}
	return decodeValid(value)
}

// l2Get returns nil when the key is absent, expired, or the backend errored.
func (c *IdentityCache) l2Get(ctx context.Context, key string) *Entry {
	start := time.Now()
	value, err := c.store.Get(ctx, key)
	c.metrics.L2GetLatency(time.Since(start))
	if err != nil {
		c.metrics.L2Request(metrics.OpGet, metrics.ResultError)
		logger.Warnf("intentiq-identity: cache L2 get failed: %v", err)
		return nil
	}
	if value == "" {
		c.metrics.L2Request(metrics.OpGet, metrics.ResultMiss)
		return nil
	}
	entry := decodeValid([]byte(value))
	if entry == nil {
		c.metrics.L2Request(metrics.OpGet, metrics.ResultMiss)
		return nil
	}
	c.metrics.L2Request(metrics.OpGet, metrics.ResultHit)
	return entry
}
