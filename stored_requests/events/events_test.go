package events

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

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

	id := "1"
	config := fmt.Sprintf(`{"id": "%s"}`, id)
	update := Update{
		Requests: map[string]json.RawMessage{id: json.RawMessage(config)},
		Imps:     map[string]json.RawMessage{id: json.RawMessage(config)},
	}
	cache.Save(context.Background(), update.Requests, update.Imps)

	config = fmt.Sprintf(`{"id": "%s", "updated": true}`, id)
	update = Update{
		Requests: map[string]json.RawMessage{id: json.RawMessage(config)},
		Imps:     map[string]json.RawMessage{id: json.RawMessage(config)},
	}

	updates, invalidations := listener.Counts()
	ep.updates <- update
	waitFor(t, listener, updates+1, invalidations)

	invalidation := Invalidation{
		Requests: []string{id},
		Imps:     []string{id},
	}

	updates, invalidations = listener.Counts()
	ep.invalidations <- invalidation
	waitFor(t, listener, updates, invalidations+1)
}

func waitFor(t *testing.T, listener *EventListener, updates int, invalidations int) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	listener.WaitFor(ctx, updates, invalidations)
	if err := ctx.Err(); err != nil {
		t.Error(err.Error())
	}
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
