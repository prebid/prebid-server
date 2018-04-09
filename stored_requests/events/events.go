package events

import (
	"context"
	"encoding/json"
	"time"

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
	stop              chan struct{}
	updateCount       int
	invalidationCount int
}

// Stop the event listener
func (e *EventListener) Stop() {
	e.stop <- struct{}{}
}

// Counts returns the number of updates and invalidations that were propagated
func (e *EventListener) Counts() (updates int, invalidations int) {
	return e.updateCount, e.invalidationCount
}

// InvalidationCount is the number of propagated Invalidations
func (e *EventListener) InvalidationCount() int {
	return e.invalidationCount
}

// WaitFor the specified number of events to be propagated
func (e *EventListener) WaitFor(ctx context.Context, updates int, invalidations int) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if e.updateCount >= updates && e.invalidationCount >= invalidations {
				return
			}
			time.Sleep(1 * time.Millisecond)
		}
	}
}

// Listen will run a goroutine that updates/invalidates the cache when events occur
func Listen(cache stored_requests.Cache, events EventProducer) *EventListener {
	listener := &EventListener{
		stop:              make(chan struct{}),
		updateCount:       0,
		invalidationCount: 0,
	}

	go func() {
		defer close(listener.stop)
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
