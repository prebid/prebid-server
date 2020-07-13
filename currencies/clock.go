// parts copied from: https://github.com/efritz/glock

package currencies

import (
	"time"
)

type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// Since returns the time elapsed since t.
	Since(t time.Time) time.Duration

	// Until returns the duration until t.
	Until(t time.Time) time.Duration
}
