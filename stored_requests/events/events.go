package events

import (
	"context"
	"encoding/json"

	"github.com/prebid/prebid-server/stored_requests"
)

// Update represents a bulk update
type Update struct {
	Requests map[string]json.RawMessage `json:"requests"`
	Imps     map[string]json.RawMessage `json:"imps"`
}

// Invalidation represents a bulk invalidation
type Invalidation struct {
	Requests []string `json:"requests"`
	Imps     []string `json:"imps"`
}

// EventProducer will produce cache update and invalidation events on its channels
type EventProducer interface {
	Updates() <-chan Update
	Invalidations() <-chan Invalidation
}

// EventListener provides information about how many events a listener has processed
// and a mechanism to stop the listener goroutine
type EventListener struct {
	stop         chan struct{}
	onUpdate     func()
	onInvalidate func()
}

// SimpleEventListener creates a new EventListener that solely propagates cache updates and invalidations
func SimpleEventListener() *EventListener {
	return &EventListener{
		stop:         make(chan struct{}),
		onUpdate:     nil,
		onInvalidate: nil,
	}
}

// NewEventListener creates a new EventListener that may perform additional work after propagating cache updates and invalidations
func NewEventListener(onUpdate func(), onInvalidate func()) *EventListener {
	return &EventListener{
		stop:         make(chan struct{}),
		onUpdate:     onUpdate,
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
		case update := <-events.Updates():
			cache.Update(context.Background(), update.Requests, update.Imps)
			if e.onUpdate != nil {
				e.onUpdate()
			}
		case invalidation := <-events.Invalidations():
			cache.Invalidate(context.Background(), invalidation.Requests, invalidation.Imps)
			if e.onInvalidate != nil {
				e.onInvalidate()
			}
		case <-e.stop:
			break
		}
	}
}
