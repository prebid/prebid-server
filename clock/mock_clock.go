// parts copied from: https://github.com/efritz/glock

package clock

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
var _ Clock = &MockClock{}

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

// SetCurrent sets the internal MockClock time to the supplied time.
func (mc *MockClock) SetCurrent(current time.Time) {
	mc.nowLock.Lock()
	defer mc.nowLock.Unlock()

	mc.fakeTime = current
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

// Since returns the time elapsed since t.
func (mc *MockClock) Since(t time.Time) time.Duration {
	return mc.Now().Sub(t)
}

// Until returns the duration until t.
func (mc *MockClock) Until(t time.Time) time.Duration {
	return t.Sub(mc.Now())
}
