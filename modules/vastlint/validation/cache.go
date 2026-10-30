package validation

import (
	"crypto/sha256"
	"sort"
	"sync/atomic"

	"github.com/coocood/freecache"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// verdictCache remembers the summary for a creative, keyed by a hash of the
// markup together with the effective rule overrides and the engine version,
// so a different rule set or engine build never sees another's verdict. It
// stores verdicts only, never markup. Same shape as the scope3 module's
// segment cache: freecache, a size in bytes and a TTL in seconds.
type verdictCache struct {
	c      *freecache.Cache
	ttl    int
	hits   atomic.Int64
	misses atomic.Int64
}

func newVerdictCache(size, ttlSeconds int) *verdictCache {
	if size <= 0 {
		return nil
	}
	return &verdictCache{c: freecache.NewCache(size), ttl: ttlSeconds}
}

// rulesKey digests the rule overrides and the engine version once per hook call.
func rulesKey(rules ruleLevels, engineVersion string) []byte {
	h := sha256.New()
	h.Write([]byte(engineVersion))
	ids := make([]string, 0, len(rules))
	for id := range rules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		h.Write([]byte(id))
		h.Write([]byte{'='})
		h.Write([]byte(rules[id]))
		h.Write([]byte{'\n'})
	}
	return h.Sum(nil)
}

// cacheKey is 128 bits of SHA-256 over the rules digest and the creative. A
// collision would hand a creative another one's verdict, which is why a
// 64-bit hash is not used here.
func cacheKey(adm string, rk []byte) []byte {
	h := sha256.New()
	h.Write(rk)
	h.Write([]byte(adm))
	sum := h.Sum(nil)
	return sum[:16]
}

func (vc *verdictCache) get(key []byte) (bidSummary, bool) {
	b, err := vc.c.Get(key)
	if err != nil {
		vc.misses.Add(1)
		return bidSummary{}, false
	}
	var s bidSummary
	if jsonutil.Unmarshal(b, &s) != nil {
		vc.misses.Add(1)
		return bidSummary{}, false
	}
	vc.hits.Add(1)
	return s, true
}

func (vc *verdictCache) put(key []byte, s bidSummary) {
	if b, err := jsonutil.Marshal(s); err == nil {
		_ = vc.c.Set(key, b, vc.ttl) // a value too large for the cache is simply not kept
	}
}

func (vc *verdictCache) tag(cached bool) string {
	if cached {
		return "hit"
	}
	return "miss"
}
