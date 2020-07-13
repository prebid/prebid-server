package clock

import (
	"time"
)

type RealClock struct{}

// NewRealClock returns a Clock whose implementation falls back to the
// methods available in the time package.
func NewRealClock() Clock {
	return &RealClock{}
}

func (c *RealClock) Now() time.Time {
	return time.Now()
}

func (c *RealClock) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

func (c *RealClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

func (c *RealClock) Until(t time.Time) time.Duration {
	return time.Until(t)
}
