package events

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/caches/memory"
)

func TestListen(t *testing.T) {
	ep := &dummyProducer{
		saves:         make(chan Save),
		invalidations: make(chan Invalidation),
	}
	cache := stored_requests.Cache{
		Requests: memory.NewCache(256*1024, -1, "Requests"),
		Imps:     memory.NewCache(256*1024, -1, "Imps"),
		Accounts: memory.NewCache(256*1024, -1, "Account"),
	}

	// create channels to synchronize
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
		Accounts: data,
	}
	cache.Requests.Save(context.Background(), save.Requests)
	cache.Imps.Save(context.Background(), save.Imps)
	cache.Accounts.Save(context.Background(), save.Accounts)

	config = fmt.Sprintf(`{"id": "%s", "updated": true}`, id)
	data = map[string]json.RawMessage{id: json.RawMessage(config)}
	save = Save{
		Requests: data,
		Imps:     data,
		Accounts: data,
	}

	ep.saves <- save
	<-saveOccurred

	requestData := cache.Requests.Get(context.Background(), idSlice)
	impData := cache.Imps.Get(context.Background(), idSlice)
	accountData := cache.Accounts.Get(context.Background(), idSlice)
	if !reflect.DeepEqual(requestData, data) || !reflect.DeepEqual(impData, data) || !reflect.DeepEqual(accountData, data) {
		t.Error("Update failed")
	}

	invalidation := Invalidation{
		Requests: idSlice,
		Imps:     idSlice,
		Accounts: idSlice,
	}

	ep.invalidations <- invalidation
	<-invalidateOccurred

	requestData = cache.Requests.Get(context.Background(), idSlice)
	impData = cache.Imps.Get(context.Background(), idSlice)
	accountData = cache.Accounts.Get(context.Background(), idSlice)
	if len(requestData) > 0 || len(impData) > 0 || len(accountData) > 0 {
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
