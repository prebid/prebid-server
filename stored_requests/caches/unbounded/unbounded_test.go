package unbounded

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/caches/cachestest"
)

func TestUnboundedRobustness(t *testing.T) {
	cachestest.AssertCacheRobustness(t, func() stored_requests.Cache {
		return NewUnboundedCache()
	})
}

func TestRaceConcurrency(t *testing.T) {
	cache := NewUnboundedCache()

	go writeLots(cache, 100)
	go readLots(cache, 100)
	go invalidateLots(cache, 100)
}

func readLots(cache stored_requests.Cache, numWrites int) {
	for i := 0; i < numWrites; i++ {
		cache.Get(context.Background(), sliceForVal(i), sliceForVal(-i))
	}
}

func writeLots(cache stored_requests.Cache, numWrites int) {
	for i := 0; i < numWrites; i++ {
		cache.Save(context.Background(), mapForVal(i), mapForVal(-i))
	}
}

func invalidateLots(cache stored_requests.Cache, numWrites int) {
	for i := 0; i < numWrites; i++ {
		cache.Invalidate(context.Background(), sliceForVal(i), sliceForVal(i))
	}
}

func mapForVal(val int) map[string]json.RawMessage {
	return map[string]json.RawMessage{
		strconv.Itoa(val): json.RawMessage(strconv.Itoa(val)),
	}
}

func sliceForVal(val int) []string {
	return []string{strconv.Itoa(val)}
}
