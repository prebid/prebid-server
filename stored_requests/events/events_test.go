package events

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
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

	// create channels to syncronize
	updateOccurred := make(chan struct{})
	invalidateOccurred := make(chan struct{})
	listener := NewEventListener(
		func() { updateOccurred <- struct{}{} },
		func() { invalidateOccurred <- struct{}{} },
	)

	go listener.Listen(cache, ep)
	defer listener.Stop()

	id := "1"
	idSlice := []string{id}
	config := fmt.Sprintf(`{"id": "%s"}`, id)
	data := map[string]json.RawMessage{id: json.RawMessage(config)}
	update := Update{
		Requests: data,
		Imps:     data,
	}
	cache.Save(context.Background(), update.Requests, update.Imps)

	config = fmt.Sprintf(`{"id": "%s", "updated": true}`, id)
	data = map[string]json.RawMessage{id: json.RawMessage(config)}
	update = Update{
		Requests: data,
		Imps:     data,
	}

	ep.updates <- update
	<-updateOccurred

	requestData, impData := cache.Get(context.Background(), idSlice, idSlice)
	if !reflect.DeepEqual(requestData, data) || !reflect.DeepEqual(impData, data) {
		t.Error("Update failed")
	}

	invalidation := Invalidation{
		Requests: idSlice,
		Imps:     idSlice,
	}

	ep.invalidations <- invalidation
	<-invalidateOccurred

	requestData, impData = cache.Get(context.Background(), idSlice, idSlice)
	if len(requestData) > 0 || len(impData) > 0 {
		t.Error("Invalidate failed")
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
