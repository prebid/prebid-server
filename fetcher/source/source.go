package source

import (
	"context"
	"encoding/json"
)

// Source pulls raw, undecoded bytes for one key. A false found value is treated
// as a definitive "not found" for that key; a non-nil error is a systemic
// failure (never cached, never negative-cached).
type Source[K comparable] interface {
	Fetch(ctx context.Context, key K) (raw json.RawMessage, found bool, err error)
}

// BulkSource is an optional capability a Source may implement to return the
// entire corpus in a single call. It is what the engine's preload warm-up uses
// to fill the cache at startup; sources that cannot enumerate everything simply
// do not implement it.
//
// Future extension: sources with a durable change can add an incremental
// refresh capability after preload. This package intentionally keeps preload as
// full warm-up only until a source exposes tested change and delete semantics.
type BulkSource[K comparable] interface {
	Source[K]
	FetchAll(ctx context.Context) (map[K]json.RawMessage, error)
}
