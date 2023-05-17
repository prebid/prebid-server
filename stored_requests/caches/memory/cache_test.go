package memory

import (
	"context"
	"encoding/json"
	"math/rand"
	"strconv"
	"testing"

	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/caches/cachestest"
)

func TestLRURobustness(t *testing.T) {
	cachestest.AssertCacheRobustness(t, func() stored_requests.CacheJSON {
		return NewCache(256*1024, -1, "TestData")
	})
}

func TestUnboundedRobustness(t *testing.T) {
	cachestest.AssertCacheRobustness(t, func() stored_requests.CacheJSON {
		return NewCache(0, -1, "TestData")
	})
}

func TestRaceLRUConcurrency(t *testing.T) {
	cache := NewCache(256*1024, -1, "TestData")
	doRaceTest(t, cache)
}

func TestRaceUnboundedConcurrency(t *testing.T) {
	cache := NewCache(0, -1, "TestData")

	doRaceTest(t, cache)
}

func doRaceTest(t *testing.T, cache stored_requests.CacheJSON) {
	done := make(chan struct{})
	sets := [][]int{rand.Perm(100), rand.Perm(100), rand.Perm(100)}

	readOrder := rand.Perm(3)
	writeOrder := rand.Perm(3)
	invalidateOrder := rand.Perm(3)
	for i := 0; i < 3; i++ {
		go writeLots(cache, done, sets[writeOrder[i]])
		go readLots(cache, done, sets[readOrder[i]])
		go invalidateLots(cache, done, sets[invalidateOrder[i]])
	}

	for i := 0; i < 9; i++ {
		<-done
	}
}

func readLots(cache stored_requests.CacheJSON, done chan<- struct{}, reads []int) {
	var s struct{}
	for _, i := range reads {
		cache.Get(context.Background(), sliceForVal(i))
	}
	done <- s
}

func writeLots(cache stored_requests.CacheJSON, done chan<- struct{}, writes []int) {
	var s struct{}
	for _, i := range writes {
		cache.Save(context.Background(), mapForVal(i))
	}
	done <- s
}

func invalidateLots(cache stored_requests.CacheJSON, done chan<- struct{}, invalidates []int) {
	var s struct{}
	for _, i := range invalidates {
		cache.Invalidate(context.Background(), sliceForVal(i))
	}
	done <- s
}

func mapForVal(val int) map[string]json.RawMessage {
	return map[string]json.RawMessage{
		strconv.Itoa(val): json.RawMessage(strconv.Itoa(val)),
	}
}

func sliceForVal(val int) []string {
	return []string{strconv.Itoa(val)}
}
