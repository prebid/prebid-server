// parts copied from: https://github.com/efritz/glock

package currencies

import (
	"runtime"
	"sort"
	"sync"
	"time"
)

type (
	// MockClock is an implementation of Clock that can be moved forward in time
	// in increments for testing code that relies on timeouts or other time-sensitive
	// constructs.
	MockClock struct {
		fakeTime   time.Time
		triggers   mockTriggers
		tickers    mockTickers
		afterArgs  []time.Duration
		tickerArgs []time.Duration
		nowLock    sync.RWMutex
		afterLock  sync.RWMutex
		tickerLock sync.Mutex
	}

	mockTrigger struct {
		trigger time.Time
		ch      chan time.Time
	}

	mockTriggers []*mockTrigger
	mockTickers  []*mockTicker
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
		fakeTime:   now,
		tickers:    make([]*mockTicker, 0),
		afterArgs:  make([]time.Duration, 0),
		tickerArgs: make([]time.Duration, 0),
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

	mc.processTriggers(now)
	mc.processTickers(now)
}

func (mc *MockClock) processTriggers(now time.Time) {
	mc.afterLock.Lock()
	defer mc.afterLock.Unlock()

	triggered := 0
	for _, trigger := range mc.triggers {
		if trigger.trigger.Before(now) || trigger.trigger.Equal(now) {
			trigger.ch <- trigger.trigger
			triggered++
		}
	}

	mc.triggers = mc.triggers[triggered:]
}

func (mc *MockClock) processTickers(now time.Time) {
	mc.tickerLock.Lock()
	defer mc.tickerLock.Unlock()

	for _, ticker := range mc.tickers {
		ticker.publish(now)
	}
}

// BlockingAdvance will call Advance but only after there is another routine
// which is blocking on the channel result of a call to After.
func (mc *MockClock) BlockingAdvance(duration time.Duration) {
	for mc.BlockedOnAfter() == 0 {
		runtime.Gosched()
	}

	mc.Advance(duration)
}

// Now returns the current time internal to the MockClock
func (mc *MockClock) Now() time.Time {
	mc.nowLock.RLock()
	defer mc.nowLock.RUnlock()

	return mc.fakeTime
}

// After returns a channel that will be sent the current internal MockClock
// time once the MockClock's internal time is at or past the provided duration
func (mc *MockClock) After(duration time.Duration) <-chan time.Time {
	mc.nowLock.RLock()
	triggerTime := mc.fakeTime.Add(duration)
	mc.nowLock.RUnlock()

	mc.afterLock.Lock()
	defer mc.afterLock.Unlock()

	trigger := &mockTrigger{
		trigger: triggerTime,
		ch:      make(chan time.Time, 1),
	}

	mc.triggers = append(mc.triggers, trigger)
	mc.afterArgs = append(mc.afterArgs, duration)

	sort.Slice(mc.triggers, func(i, j int) bool {
		return mc.triggers[i].trigger.Before(mc.triggers[j].trigger)
	})

	return trigger.ch
}

// BlockedOnAfter returns the number of calls to After that are blocked
// waiting for a call to Advance to trigger them.
func (mc *MockClock) BlockedOnAfter() int {
	mc.afterLock.RLock()
	defer mc.afterLock.RUnlock()

	return len(mc.triggers)
}

// Sleep will block until the internal MockClock time is at or past the
// provided duration
func (mc *MockClock) Sleep(duration time.Duration) {
	<-mc.After(duration)
}

// Since returns the time elapsed since t.
func (mc *MockClock) Since(t time.Time) time.Duration {
	return mc.Now().Sub(t)
}

// Until returns the duration until t.
func (mc *MockClock) Until(t time.Time) time.Duration {
	return t.Sub(mc.Now())
}

// GetAfterArgs returns the duration of each call to After in the
// same order as they were called. The list is cleared each time
// GetAfterArgs is called.
func (mc *MockClock) GetAfterArgs() []time.Duration {
	mc.afterLock.Lock()
	defer mc.afterLock.Unlock()

	args := mc.afterArgs
	mc.afterArgs = mc.afterArgs[:0]
	return args
}

// GetTickerArgs returns the duration of each call to create a new
// ticker in the same order as they were called. The list is cleared
// each time GetTickerArgs is called.
func (mc *MockClock) GetTickerArgs() []time.Duration {
	mc.tickerLock.Lock()
	defer mc.tickerLock.Unlock()

	args := mc.tickerArgs
	mc.tickerArgs = mc.tickerArgs[:0]
	return args
}

type mockTicker struct {
	clock        *MockClock
	duration     time.Duration
	started      time.Time
	nextTick     time.Time
	processQueue []time.Time
	ch           chan time.Time
	wakeup       chan struct{}
	stopped      bool
	processLock  sync.Mutex
	stoppedLock  sync.RWMutex
}

// NewTicker creates a new Ticker tied to the internal MockClock time that ticks
// at intervals similar to time.NewTicker().  It will also skip or drop ticks
// for slow readers similar to time.NewTicker() as well.

func (mc *MockClock) NewTicker(duration time.Duration) Ticker {
	if duration == 0 {
		panic("duration cannot be 0")
	}

	now := mc.Now()
	ticker := &mockTicker{
		clock:        mc,
		duration:     duration,
		started:      now,
		nextTick:     now.Add(duration),
		processQueue: make([]time.Time, 0),
		ch:           make(chan time.Time),
		wakeup:       make(chan struct{}, 1),
	}

	mc.tickerLock.Lock()
	defer mc.tickerLock.Unlock()

	mc.tickers = append(mc.tickers, ticker)
	mc.tickerArgs = append(mc.tickerArgs, duration)
	go ticker.process()
	return ticker
}

// Chan returns a channel which will receive the MockClock's internal time
// at the interval given when creating the ticker.
func (mt *mockTicker) Chan() <-chan time.Time {
	return mt.ch
}

// Stop will stop the ticker from ticking
func (mt *mockTicker) Stop() {
	mt.stoppedLock.Lock()
	defer mt.stoppedLock.Unlock()

	mt.stopped = true
	mt.wakeup <- struct{}{}
}

func (mt *mockTicker) publish(now time.Time) {
	if mt.isStopped() {
		return
	}

	mt.processLock.Lock()
	mt.processQueue = append(mt.processQueue, now)
	mt.processLock.Unlock()

	select {
	case mt.wakeup <- struct{}{}:
	default:
	}
}

func (mt *mockTicker) process() {
	defer close(mt.wakeup)

	for !mt.isStopped() {
		for {
			first, ok := mt.pop()
			if !ok {
				break
			}

			if mt.nextTick.After(first) {
				continue
			}

			mt.ch <- mt.nextTick

			durationMod := first.Sub(mt.started) % mt.duration

			if durationMod == 0 {
				mt.nextTick = first.Add(mt.duration)
			} else if first.Sub(mt.nextTick) > mt.duration {
				mt.nextTick = first.Add(mt.duration - durationMod)
			} else {
				mt.nextTick = mt.nextTick.Add(mt.duration)
			}
		}

		<-mt.wakeup
	}
}

func (mt *mockTicker) pop() (time.Time, bool) {
	mt.processLock.Lock()
	defer mt.processLock.Unlock()

	if len(mt.processQueue) == 0 {
		return time.Unix(0, 0), false
	}

	first := mt.processQueue[0]
	mt.processQueue = mt.processQueue[1:]
	return first, true
}

func (mt *mockTicker) isStopped() bool {
	mt.stoppedLock.RLock()
	defer mt.stoppedLock.RUnlock()

	return mt.stopped
}
