package fetcher

import (
	"errors"
	"fmt"
)

// ErrNotFound is the sentinel for definitive per-key misses. Get returns a
// NotFoundError wrapping this sentinel so callers can inspect the key while
// existing errors.Is(err, ErrNotFound) checks keep working.
var ErrNotFound = errors.New("fetcher: not found")

// NotFoundError is returned by Get when the requested key does not exist.
type NotFoundError struct {
	Key any
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("fetcher: key %v not found", e.Key)
}

func (e NotFoundError) Unwrap() error {
	return ErrNotFound
}
