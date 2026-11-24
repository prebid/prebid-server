package fetcher

import (
	"context"
	"encoding/json"
)

// Source pulls raw, undecoded bytes for one key. A false found value is treated
// as a definitive not-found; a non-nil error is a systemic failure.
type Source[K comparable] interface {
	Fetch(ctx context.Context, key K) (raw json.RawMessage, found bool, err error)
}

// BulkSource optionally extends Source with full-corpus loading for preload.
type BulkSource[K comparable] interface {
	Source[K]
	FetchAll(ctx context.Context) (map[K]json.RawMessage, error)
}
