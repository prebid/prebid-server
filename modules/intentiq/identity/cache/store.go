package cache

import (
	"context"
	"time"
)

// Store is the shared (L2) layer.
//
// An absent key is not an error: Get returns ("", nil), so a cold key falls through to a live API
// call instead of being counted as a backend failure.
type Store interface {
	Get(ctx context.Context, key string) (string, error)
	Put(ctx context.Context, key, value string, ttl time.Duration) error
	Name() string
	Close() error
}
