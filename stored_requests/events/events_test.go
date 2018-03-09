package events

import (
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests/caches/in_memory"
)

func TestListen(t *testing.T) {
	ep := &dummyProducer{
		updates:       make(chan Update),
		invalidations: make(chan Invalidation),
	}

	cache := in_memory.NewLRUCache(&config.InMemoryCache{
		RequestCacheSize: 256 * 1024,
		ImpCacheSize:     256 * 1024,
		TTL:              -1,
	})

	listener := Listen(cache, ep)
	defer listener.Stop()

	// id := "1"
	// config := fmt.Sprintf(`{"id": "%s"}`, id)
	// cache.Save(context.Background(), map[string]json.RawMessage{id: json.RawMessage(config)})

	// config = fmt.Sprintf(`{"id": "%s", "updated": true}`, id)
	// ep.updates <- map[string]json.RawMessage{id: json.RawMessage(config)}

	// for listener.UpdateCount() < 1 {
	// 	// wait for listener goroutine to process the event
	// }
	// data := cache.Get(context.Background(), []string{id})
	// if value, ok := data[id]; !ok || string(value) != config {
	// 	t.Errorf("Updated key/value not present in cache after update.")
	// }

	// ep.invalidations <- []string{id}
	// for listener.InvalidationCount() < 1 {
	// 	// wait for listener goroutine to process the event
	// }
	// data = cache.Get(context.Background(), []string{id})
	// if _, ok := data[id]; ok {
	// 	t.Errorf("Key/Value still present in cache after invalidation.")
	// }
}

type dummyProducer struct {
	updates       chan Update
	invalidations chan Invalidation
}

func (p *dummyProducer) Updates() <-chan Update {
	return p.updates
}

func (p *dummyProducer) Invalidations() <-chan Invalidation {
	return p.invalidations
}
