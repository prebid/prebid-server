package currencies

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMockClock(t *testing.T) {
	// Since we can't know the exact time that something gets created
	// at least make sure a new mock clock picks up a time around now
	// instead of an exact time
	clock := NewMockClock()

	assert.WithinDuration(t, clock.fakeTime, time.Now(), time.Duration(100)*time.Millisecond)
}

func TestNewMockClockAt(t *testing.T) {
	clock := NewMockClockAt(time.Unix(100, 0))

	assert.Equal(t, clock.fakeTime, time.Unix(100, 0))
}

func TestSetCurrent(t *testing.T) {
	clock := NewMockClock()
	clock.SetCurrent(time.Unix(100, 0))

	assert.Equal(t, clock.Now(), time.Unix(100, 0))
}

func TestAdvance(t *testing.T) {
	clock := NewMockClock()

	clock.SetCurrent(time.Unix(100, 0))
	assert.Equal(t, clock.Now(), time.Unix(100, 0))

	clock.Advance(1 * time.Second)
	assert.Equal(t, clock.Now(), time.Unix(101, 0))

	clock.Advance(1 * time.Hour)
	assert.Equal(t, clock.Now(), time.Unix(3701, 0))
}

func TestNow(t *testing.T) {
	clock := NewMockClock()

	assert.Equal(t, clock.Now(), clock.fakeTime)
}

func TestSince(t *testing.T) {
	clock := NewMockClockAt(time.Unix(10, 0))

	assert.Equal(t, clock.Since(time.Unix(10, 0)), time.Duration(0))
	assert.Equal(t, clock.Since(time.Unix(5, 0)), time.Duration(5)*time.Second)
	assert.Equal(t, clock.Since(time.Unix(15, 0)), time.Duration(-5)*time.Second)
}

func TestUntil(t *testing.T) {
	clock := NewMockClockAt(time.Unix(10, 0))

	assert.Equal(t, clock.Until(time.Unix(10, 0)), time.Duration(0))
	assert.Equal(t, clock.Until(time.Unix(5, 0)), time.Duration(-5)*time.Second)
	assert.Equal(t, clock.Until(time.Unix(15, 0)), time.Duration(5)*time.Second)
}
