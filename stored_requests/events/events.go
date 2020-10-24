package events

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/buger/jsonparser"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests"
)

// Save represents a bulk save
type Save struct {
	DataType config.DataType
	Data     map[string]json.RawMessage `json:"data"`
}

// Invalidation represents a bulk invalidation
type Invalidation struct {
	DataType config.DataType
	Data     []string `json:"data"`
}

// EventProducer will produce cache update and invalidation events on its channels
type EventProducer interface {
	Saves() <-chan Save
	Invalidations() <-chan Invalidation
}

// EventListener provides information about how many events a listener has processed
// and a mechanism to stop the listener goroutine
type EventListener struct {
	stop         chan struct{}
	onSave       func()
	onInvalidate func()
}

// SimpleEventListener creates a new EventListener that solely propagates cache updates and invalidations
func SimpleEventListener() *EventListener {
	return &EventListener{
		stop:         make(chan struct{}),
		onSave:       nil,
		onInvalidate: nil,
	}
}

// NewEventListener creates a new EventListener that may perform additional work after propagating cache saves and invalidations
func NewEventListener(onSave func(), onInvalidate func()) *EventListener {
	return &EventListener{
		stop:         make(chan struct{}),
		onSave:       onSave,
		onInvalidate: onInvalidate,
	}
}

// Stop the event listener
func (e *EventListener) Stop() {
	e.stop <- struct{}{}
}

// getCacheByType looks up the subcache by type - maybe this should be a Cache method
func getCacheByType(cache stored_requests.Cache, dataType config.DataType) stored_requests.CacheJSON {
	return map[config.DataType]stored_requests.CacheJSON{
		config.RequestDataType: cache.Requests,
		config.ImpDataType:     cache.Imps,
		config.AccountDataType: cache.Accounts,
	}[dataType]
}

// Listen is meant to be run as a goroutine that updates/invalidates the cache when events occur
func (e *EventListener) Listen(cache stored_requests.Cache, events EventProducer) {
	for {
		select {
		case save := <-events.Saves():
			getCacheByType(cache, save.DataType).Save(context.Background(), save.Data)
			if e.onSave != nil {
				e.onSave()
			}
		case invalidation := <-events.Invalidations():
			getCacheByType(cache, invalidation.DataType).Invalidate(context.Background(), invalidation.Data)
			if e.onInvalidate != nil {
				e.onInvalidate()
			}
		case <-e.stop:
			break
		}
	}
}

// SendInvalidations destructively extracts the ids with {deleted: true} from changes and sends a cache invalidation message
func SendInvalidations(invalidations chan<- Invalidation, dataType config.DataType, changes map[string]json.RawMessage) {
	deletedIDs := make([]string, 0, len(changes))
	for id, msg := range changes {
		if value, _, _, err := jsonparser.Get(msg, "deleted"); err == nil && bytes.Equal(value, []byte("true")) {
			delete(changes, id)
			deletedIDs = append(deletedIDs, id)
		}
	}
	if len(deletedIDs) > 0 {
		invalidations <- Invalidation{
			DataType: dataType,
			Data:     deletedIDs,
		}
	}
}

// SendSaves sends an update (save) message with all the changes
func SendSaves(saves chan<- Save, dataType config.DataType, changes map[string]json.RawMessage) {
	if len(changes) > 0 {
		saves <- Save{DataType: dataType, Data: changes}
	}
}
