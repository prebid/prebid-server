// parts copied from: https://github.com/efritz/glock

package currencies

import (
	"time"
)

type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// After returns a channel which receives the current time after
	// the given duration elapses.
	After(duration time.Duration) <-chan time.Time

	// Sleep blocks until the given duration elapses.
	Sleep(duration time.Duration)

	// Since returns the time elapsed since t.
	Since(t time.Time) time.Duration

	// Until returns the duration until t.
	Until(t time.Time) time.Duration
}

// Ticker is a wrapper around a time.Ticker, which allows interface
// access  to the underlying channel (instead of bare access like the
// time.Ticker struct allows).
type Ticker interface {
	// Chan returns the underlying ticker channel.
	Chan() <-chan time.Time

	// Stop stops the ticker.
	Stop()
}
