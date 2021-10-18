package events

import (
	"context"
	"encoding/json"

	"github.com/prebid/prebid-server/stored_requests"
)

// Save represents a bulk save
type Save struct {
	Requests map[string]json.RawMessage `json:"requests"`
	Imps     map[string]json.RawMessage `json:"imps"`
	Accounts map[string]json.RawMessage `json:"accounts"`
}

// Invalidation represents a bulk invalidation
type Invalidation struct {
	Requests []string `json:"requests"`
	Imps     []string `json:"imps"`
	Accounts []string `json:"accounts"`
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

// Listen is meant to be run as a goroutine that updates/invalidates the cache when events occur
func (e *EventListener) Listen(cache stored_requests.Cache, events EventProducer) {
	for {
		select {
		case save := <-events.Saves():
			cache.Requests.Save(context.Background(), save.Requests)
			cache.Imps.Save(context.Background(), save.Imps)
			cache.Accounts.Save(context.Background(), save.Accounts)
			if e.onSave != nil {
				e.onSave()
			}
		case invalidation := <-events.Invalidations():
			cache.Requests.Invalidate(context.Background(), invalidation.Requests)
			cache.Imps.Invalidate(context.Background(), invalidation.Imps)
			cache.Accounts.Invalidate(context.Background(), invalidation.Accounts)
			if e.onInvalidate != nil {
				e.onInvalidate()
			}
		case <-e.stop:
			break
		}
	}
}
