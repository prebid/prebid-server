package stored_requests

import (
	"context"
	"encoding/json"
	"fmt"
)

// Fetcher knows how to fetch Stored Request data by id.
//
// Implementations must be safe for concurrent access by multiple goroutines.
// Callers are expected to share a single instance as much as possible.
type Fetcher interface {
	// FetchRequests fetches the stored requests for the given IDs.
	//
	// The first return value will be the Stored Request data, or nil if it doesn't exist.
	// If requestID is an empty string, then this value will always be nil.
	//
	// The second return value will be a map from Stored Imp data. It will have a key for every ID
	// in the impIDs list, unless errors exist.
	//
	// The returned objects can only be read from. They may not be written to.
	FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error)
}

// NotFoundError is an error type to flag that an ID was not found, but there was otherwise no issue
// with the query. This was added to support Multifetcher and any other case where we might expect
// that all IDs would not be found, and want to disentangle those errors from the others.
type NotFoundError struct {
	ID       string
	DataType string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf(`Stored %s with ID="%s" not found.`, e.DataType, e.ID)
}

// Cache is an intermediate layer which can be used to create more complex Fetchers by composition.
// Implementations must be safe for concurrent access by multiple goroutines.
// To add a Cache layer in front of the Fetcher, see WithCache()
type Cache interface {
	// GetRequests works much like Fetcher.FetchRequests, with a few exceptions:
	//
	// 1. Any (actionable) errors should be logged by the implementation, rather than returned.
	// 2. The returned maps _may_ be written to.
	// 3. The returned maps must _not_ contain keys unless they were present in the argument ID list.
	// 4. Callers _should not_ assume that the returned maps contain key for every argument id.
	//    The returned map will miss entries for keys which don't exist in the cache.
	//
	// Nil slices and empty strings are treated as "no ops". That is, a nil requestID will always produce a nil
	// "stored request data" in the response.
	GetRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage)

	// SaveRequests stores some data in the cache. The maps are from ID to the cached value.
	//
	// This is a best-effort method. If the cache call fails, implementations should log the error.
	SaveRequests(ctx context.Context, storedRequests map[string]json.RawMessage, storedImps map[string]json.RawMessage)
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

func (f *fetcherWithCache) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	requestData, impData = f.cache.GetRequests(ctx, requestIDs, impIDs)

	// Fixes #311
	leftoverImps := findLeftovers(impIDs, impData)
	leftoverReqs := findLeftovers(requestIDs, requestData)

	fetcherReqData, fetcherImpData, errs := f.fetcher.FetchRequests(ctx, leftoverReqs, leftoverImps)

	f.cache.SaveRequests(ctx, fetcherReqData, fetcherImpData)

	requestData = mergeData(requestData, fetcherReqData)
	impData = mergeData(impData, fetcherImpData)
	return
}

func findLeftovers(ids []string, data map[string]json.RawMessage) (leftovers []string) {
	leftovers = make([]string, 0, len(ids)-len(data))
	for _, id := range ids {
		if _, ok := data[id]; !ok {
			leftovers = append(leftovers, id)
		}
	}
	return
}

func mergeData(cachedData map[string]json.RawMessage, fetchedData map[string]json.RawMessage) (mergedData map[string]json.RawMessage) {
	mergedData = cachedData
	if mergedData == nil {
		mergedData = fetchedData
	} else {
		for key, value := range fetchedData {
			mergedData[key] = value
		}
	}
	return
}
