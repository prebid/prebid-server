// parts copied from: https://github.com/efritz/glock

package timeutil

import (
	"sync"
	"time"
)

type (
	// MockClock is an implementation of Clock that can be moved forward in time
	// in increments for testing code that relies on timeouts or other time-sensitive
	// constructs.
	MockClock struct {
		fakeTime time.Time
		nowLock  sync.RWMutex
	}
)

// Make sure MockClock conforms to the interfaces
var _ Time = &MockClock{}

// NewMockClock creates a new MockClock with the internal time set
// to time.Now()
func NewMockClock() *MockClock {
	return NewMockClockAt(time.Now())
}

// NewMockClockAt creates a new MockClick with the internal time set
// to the provided time.
func NewMockClockAt(now time.Time) *MockClock {
	return &MockClock{
		fakeTime: now,
	}
}

// Advance will advance the internal MockClock time by the supplied time.
func (mc *MockClock) Advance(duration time.Duration) {
	mc.nowLock.Lock()
	now := mc.fakeTime.Add(duration)
	mc.fakeTime = now
	mc.nowLock.Unlock()
}

// Now returns the current time internal to the MockClock
func (mc *MockClock) Now() time.Time {
	mc.nowLock.RLock()
	defer mc.nowLock.RUnlock()

	return mc.fakeTime
}
