package events

import (
	"context"
	"encoding/json"

	"github.com/prebid/prebid-server/stored_requests"
)

type Update struct {
	Requests map[string]json.RawMessage
	Imps     map[string]json.RawMessage
}

type Invalidation struct {
	Requests []string
	Imps     []string
}

// EventProducer will produce cache update and invalidation events on its channels
type EventProducer interface {
	Updates() <-chan Update
	Invalidations() <-chan Invalidation
}

// EventListener provides information about how many events a listener has processed
// and a mechanism to stop the listener goroutine
type EventListener interface {
	InvalidationCount() int
	UpdateCount() int
	Stop()
}

type eventListener struct {
	invalidationCount int
	updateCount       int
	stop              chan struct{}
}

func (e eventListener) InvalidationCount() int {
	return e.invalidationCount
}

func (e eventListener) UpdateCount() int {
	return e.updateCount
}

func (e *eventListener) Stop() {
	e.stop <- struct{}{}
}

// Listen will run a goroutine that updates/invalidates the cache when events occur
func Listen(cache stored_requests.Cache, events EventProducer) EventListener {
	listener := &eventListener{
		invalidationCount: 0,
		updateCount:       0,
		stop:              make(chan struct{}),
	}

	go func() {
		for {
			select {
			case update := <-events.Updates():
				cache.Update(context.Background(), update.Requests, update.Requests)
				listener.updateCount++
			case invalidation := <-events.Invalidations():
				cache.Invalidate(context.Background(), invalidation.Requests, invalidation.Imps)
				listener.invalidationCount++
			case <-listener.stop:
				break
			}
		}
	}()

	return listener
}
