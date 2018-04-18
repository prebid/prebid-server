package events

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests/caches/memory"
)

func TestListen(t *testing.T) {
	ep := &dummyProducer{
		saves:         make(chan Save),
		invalidations: make(chan Invalidation),
	}

	cache := memory.NewCache(&config.InMemoryCache{
		RequestCacheSize: 256 * 1024,
		ImpCacheSize:     256 * 1024,
		TTL:              -1,
	})

	// create channels to syncronize
	saveOccurred := make(chan struct{})
	invalidateOccurred := make(chan struct{})
	listener := NewEventListener(
		func() { saveOccurred <- struct{}{} },
		func() { invalidateOccurred <- struct{}{} },
	)

	go listener.Listen(cache, ep)
	defer listener.Stop()

	id := "1"
	idSlice := []string{id}
	config := fmt.Sprintf(`{"id": "%s"}`, id)
	data := map[string]json.RawMessage{id: json.RawMessage(config)}
	save := Save{
		Requests: data,
		Imps:     data,
	}
	cache.Save(context.Background(), save.Requests, save.Imps)

	config = fmt.Sprintf(`{"id": "%s", "updated": true}`, id)
	data = map[string]json.RawMessage{id: json.RawMessage(config)}
	save = Save{
		Requests: data,
		Imps:     data,
	}

	ep.saves <- save
	<-saveOccurred

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
	saves         chan Save
	invalidations chan Invalidation
}

func (p *dummyProducer) Saves() <-chan Save {
	return p.saves
}

func (p *dummyProducer) Invalidations() <-chan Invalidation {
	return p.invalidations
}
