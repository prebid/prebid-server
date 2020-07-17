package timeutil

import (
	"time"
)

type RealClock struct{}

// NewRealClock returns a Time whose implementation falls back to the
// methods available in the time package.
func NewRealClock() Time {
	return &RealClock{}
}

func (c *RealClock) Now() time.Time {
	return time.Now()
}

func (c *RealClock) Sleep(duration time.Duration) {
	time.Sleep(duration)
}
