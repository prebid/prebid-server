package fetcher

import "encoding/json"

// TransformFunc converts raw bytes into the typed value V. It runs exactly once
// per key, at cache insert time. The key is provided so transforms that need it
// (e.g. filling an ID) can do so without mutating the shared cached value later.
type TransformFunc[K comparable, V any] func(key K, raw json.RawMessage) (V, error)
