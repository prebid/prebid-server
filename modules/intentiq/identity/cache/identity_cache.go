package cache

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/coocood/freecache"
	"github.com/prebid/openrtb/v20/openrtb2"
)

// minFreeCacheSize is freecache's minimum backing-buffer size (512 KiB). A smaller byte budget is
// bumped up to this floor, matching freecache's own internal clamp.
const minFreeCacheSize = 512 * 1024

// IdentityCache is a dual-layer, multi-key (alias) cache for resolved eids: an in-process L1
// (freecache) backed by a pluggable Store (L2, shared; Redis by default).
//
// A request yields an ordered list of Keys (one per first-party id present). On read, the
// highest-priority key with a live entry wins and that entry is back-filled under every other key
// that missed, so the alias graph grows over time and a later request carrying any of those ids
// hits. On a full miss the caller fetches once and writes the entry under all keys.
//
// Entries carry the IntentIQ cttl capped by a per-KeyType ceiling (see TTLPolicy). Unresolvable ids
// are cached as a short-lived negative sentinel so they do not re-hit the upstream API. L2 failures
// are swallowed (fail-open) so the auction can fall through to a live call. Differing resolutions
// are never merged — only the single winning entry propagates.
type IdentityCache struct {
	local   *freecache.Cache
	store   Store
	ttl     TTLPolicy
	metrics Metrics
	maxSize int
}

// NewIdentityCache builds the two-layer cache. maxSizeBytes bounds the in-process layer (freecache
// is byte-budget bounded; the Java cache-max-size count becomes a byte budget in Go — document in
// README). store is the L2 backend, ttl the TTL policy, metrics the health contract.
func NewIdentityCache(maxSizeBytes int, ttl TTLPolicy, store Store, metrics Metrics) *IdentityCache {
	size := maxSizeBytes
	if size < minFreeCacheSize {
		size = minFreeCacheSize
	}
	local := freecache.NewCache(size)
	c := &IdentityCache{
		local:   local,
		store:   store,
		ttl:     ttl,
		metrics: metrics,
		maxSize: size,
	}
	// L1 capacity gauges: current size (vs cache-max-size) and cumulative evictions.
	c.metrics.RegisterL1Gauges(
		func() int64 { return local.EntryCount() },
		func() int64 { return local.EvacuateCount() },
	)
	return c
}

// Get sweeps keys in priority order and returns the first live outcome (see Result). A full miss
// returns MissResult(). ctx bounds the L2 probes.
func (c *IdentityCache) Get(ctx context.Context, keys []Key) Result {
	if len(keys) == 0 {
		return MissResult()
	}

	// L1 sweep in priority order. A resolved entry always wins; an in-progress marker is a fallback
	// that short-circuits the L2 probe (this instance already knows a call is in flight) without
	// firing a duplicate.
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

	// Full L1 miss: probe keys in L2 in priority order. Prefer the highest-priority resolved entry;
	// fall back to an in-progress marker only if no resolved entry is found under any key. Every L2
	// entry read is promoted into L1.
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

// Put writes a positive entry (the resolved eids) under every key, each capped by its ceiling.
func (c *IdentityCache) Put(ctx context.Context, keys []Key, eids []openrtb2.EID, cttl time.Duration) {
	for _, k := range keys {
		ttl := c.ttl.EffectiveTTL(k.Type, cttl)
		exp := time.Now().UnixMilli() + ttl.Milliseconds()
		c.writeBoth(ctx, k.Key, Entry{Eids: eids, Exp: exp}, ttl)
	}
}

// PutNegative writes a negative sentinel under every key with the negative TTL.
func (c *IdentityCache) PutNegative(ctx context.Context, keys []Key, cttl time.Duration) {
	ttl := c.ttl.NegativeTTLFor(cttl)
	for _, k := range keys {
		exp := time.Now().UnixMilli() + ttl.Milliseconds()
		c.writeBoth(ctx, k.Key, Entry{Negative: true, Exp: exp}, ttl)
	}
}

// PutInProgress marks a resolution as in flight: an IN_PROGRESS sentinel under every key with the
// short in-progress TTL, so a concurrent request for the same id reads it and skips a duplicate
// upstream call. Overwritten by Put/PutNegative when the call completes; expires otherwise.
func (c *IdentityCache) PutInProgress(ctx context.Context, keys []Key) {
	ttl := c.ttl.InProgressTTL
	for _, k := range keys {
		exp := time.Now().UnixMilli() + ttl.Milliseconds()
		c.writeBoth(ctx, k.Key, Entry{InProgress: true, Exp: exp}, ttl)
	}
}

// backfill propagates the winning entry under every other key that currently misses in L1, capped by
// min(remaining, ceilingFor(type)). Faithful port of Java backfill.
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
		entry := Entry{Eids: hit.Eids, Negative: hit.Negative, InProgress: hit.InProgress, Exp: exp}
		c.writeBoth(ctx, k.Key, entry, time.Duration(ttlMs)*time.Millisecond)
	}
}

// writeBoth writes the entry into L1 and L2. The entry lives in L1 even if the L2 write fails.
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
		c.metrics.L2PutError()
	}
}

// l1Set stores a pre-encoded value into freecache with a per-entry expiry (expireSeconds = ceil of
// the TTL, min 1). L1 failures are counted and swallowed (fail open).
func (c *IdentityCache) l1Set(key string, value []byte, ttl time.Duration) {
	expireSeconds := int(math.Ceil(ttl.Seconds()))
	if expireSeconds < 1 {
		expireSeconds = 1
	}
	if err := c.local.Set([]byte(key), value, expireSeconds); err != nil {
		c.metrics.L1PutError()
	}
}

// l1Promote writes an entry read from L2 into L1, deriving the L1 TTL from the entry's absolute
// expiry. A no-op if the entry has already expired.
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

// l1Get reads and decodes a live entry from L1, or nil when absent/expired. A pathological freecache
// failure is counted and swallowed (fail open).
func (c *IdentityCache) l1Get(key string) *Entry {
	value, err := c.local.Get([]byte(key))
	if err != nil {
		if errors.Is(err, freecache.ErrNotFound) {
			return nil
		}
		c.metrics.L1GetError()
		return nil
	}
	return decodeValid(value)
}

// l2Get reads and decodes a live entry from L2, timing the GET. An L2 error is counted and treated
// as a miss (fail open).
func (c *IdentityCache) l2Get(ctx context.Context, key string) *Entry {
	start := time.Now()
	value, err := c.store.Get(ctx, key)
	c.metrics.L2GetLatency(time.Since(start))
	if err != nil {
		c.metrics.L2GetError()
		return nil
	}
	if value == "" {
		return nil
	}
	return decodeValid([]byte(value))
}

// decodeValid decodes a serialized Entry and treats an entry at/after its absolute expiry as absent.
func decodeValid(value []byte) *Entry {
	if len(value) == 0 {
		return nil
	}
	var entry Entry
	if err := json.Unmarshal(value, &entry); err != nil {
		return nil
	}
	if entry.Exp <= time.Now().UnixMilli() {
		return nil
	}
	return &entry
}

// toResult maps a live entry to a Result for the given key type and serving layer.
func toResult(entry Entry, keyType KeyType, layer Layer) Result {
	if entry.InProgress {
		return InProgressResult(keyType, layer)
	}
	if entry.Negative {
		return NegativeResult(keyType, layer)
	}
	return HitResult(entry.Eids, keyType, layer)
}
