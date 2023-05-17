package timeutil

import (
	"time"
)

type Time interface {
	Now() time.Time
}

// RealTime wraps the time package for testability
type RealTime struct{}

func (c *RealTime) Now() time.Time {
	return time.Now()
}
