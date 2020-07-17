package timeutil

import (
	"time"
)

type Time interface {
	// Now returns the current time.
	Now() time.Time
}
