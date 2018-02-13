package stored_requests

import (
	"context"
	"encoding/json"
)

// Fetcher knows how to fetch Stored Request data by id.
//
// Implementations must be safe for concurrent access by multiple goroutines.
// Callers are expected to share a single instance as much as possible.
type Fetcher interface {
	// FetchRequests fetches the stored requests for the given IDs.
	// The returned map will have keys for every ID in the argument list, unless errors exist.
	//
	// The returned objects can only be read from. They may not be written to.
	FetchRequests(ctx context.Context, ids []string) (map[string]json.RawMessage, []error)
}

// Cache is an intermediate layer which can be used to create more complex Fetchers by composition.
// Implementations must be safe for concurrent access by multiple goroutines.
// To add a Cache layer in front of the Fetcher, see WithCache()
type Cache interface {
	// GetRequests works much like Fetcher.FetchRequests, with a few exceptions:
	//
	// 1. Any errors should be logged by the implementation, rather than returned.
	// 2. The returned map _may_ be written to.
	// 3. The returned map must _not_ contain keys unless they were present in the argument ID list.
	// 4. Callers _should not_ assume that the returned map contains a key for every id.
	//    The returned map will miss entries for keys which don't exist in the cache.
	GetRequests(ctx context.Context, ids []string) map[string]json.RawMessage

	// SaveRequests stores some data in the cache. The map is from ID to the cached value.
	//
	// This is a best-effort method. If the cache call fails, implementations should log the error.
	SaveRequests(ctx context.Context, values map[string]json.RawMessage)
}

// WithCache returns a Fetcher which uses the given Cache before delegating to the original.
// This can be called multiple times to compose Cache layers onto the backing Fetcher.
func WithCache(fetcher Fetcher, cache Cache) Fetcher {
	return &fetcherWithCache{
		cache:   cache,
		fetcher: fetcher,
	}
}

type fetcherWithCache struct {
	cache   Cache
	fetcher Fetcher
}

func (f *fetcherWithCache) FetchRequests(ctx context.Context, ids []string) (data map[string]json.RawMessage, errs []error) {
	data = f.cache.GetRequests(ctx, ids)

	// Fixes #311
	leftoverIds := make([]string, 0, len(ids)-len(data))
	for _, id := range ids {
		if _, gotFromCache := data[id]; !gotFromCache {
			leftoverIds = append(leftoverIds, id)
		}
	}

	newData, errs := f.fetcher.FetchRequests(ctx, leftoverIds)
	f.cache.SaveRequests(ctx, newData)
	for key, value := range newData {
		data[key] = value
	}
	return
}
