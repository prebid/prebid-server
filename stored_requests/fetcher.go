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

type CategoryFetcher interface {
	// FetchCategories fetches the ad-server/publisher specific category for the given IAB category
	FetchCategories(primaryAdServer, publisherId, iabCategory string) (string, error)
}

// AllFetcher is an iterface that encapsulates both the original Fetcher and the CategoryFetcher
type AllFetcher interface {
	FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error)
	FetchCategories(primaryAdServer, publisherId, iabCategory string) (string, error)
}

// NotFoundError is an error type to flag that an ID was not found by the Fetcher.
// This was added to support Multifetcher and any other case where we might expect
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
// To add a Cache layer in front of a Fetcher, see WithCache()
type Cache interface {
	// Get works much like Fetcher.FetchRequests, with a few exceptions:
	//
	// 1. Any (actionable) errors should be logged by the implementation, rather than returned.
	// 2. The returned maps _may_ be written to.
	// 3. The returned maps must _not_ contain keys unless they were present in the argument ID list.
	// 4. Callers _should not_ assume that the returned maps contain key for every argument id.
	//    The returned map will miss entries for keys which don't exist in the cache.
	//
	// Nil slices and empty strings are treated as "no ops". That is, a nil requestID will always produce a nil
	// "stored request data" in the response.
	Get(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage)

	// Invalidate will ensure that all values associated with the given IDs
	// are no longer returned by the cache until new values are saved via Update
	Invalidate(ctx context.Context, requestIDs []string, impIDs []string)

	// Save will add or overwrite the data in the cache at the given keys
	Save(ctx context.Context, requestData map[string]json.RawMessage, impData map[string]json.RawMessage)
}

// ComposedCache creates an interface to treat a slice of caches as a single cache
type ComposedCache []Cache

// Get will attempt to Get from the caches in the order in which they are in the slice,
// stopping as soon as a value is found (or when all caches have been exhausted)
func (c ComposedCache) Get(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage) {
	requestData = make(map[string]json.RawMessage, len(requestIDs))
	impData = make(map[string]json.RawMessage, len(impIDs))

	remainingReqIDs := requestIDs
	remainingImpIDs := impIDs

	for _, cache := range c {
		cachedReqData, cachedImpData := cache.Get(ctx, remainingReqIDs, remainingImpIDs)

		requestData, remainingReqIDs = updateFromCache(requestData, remainingReqIDs, cachedReqData)
		impData, remainingImpIDs = updateFromCache(impData, remainingImpIDs, cachedImpData)

		// return if all ids filled
		if len(remainingReqIDs) == 0 && len(remainingImpIDs) == 0 {
			return
		}
	}

	return
}

func updateFromCache(data map[string]json.RawMessage, ids []string, newData map[string]json.RawMessage) (map[string]json.RawMessage, []string) {
	remainingIDs := ids

	if len(newData) > 0 {
		remainingIDs = make([]string, 0, len(ids))

		for _, id := range ids {
			if config, ok := newData[id]; ok {
				data[id] = config
			} else {
				remainingIDs = append(remainingIDs, id)
			}
		}
	}

	return data, remainingIDs
}

// Invalidate will propagate invalidations to all underlying caches
func (c ComposedCache) Invalidate(ctx context.Context, requestIDs []string, impIDs []string) {
	for _, cache := range c {
		cache.Invalidate(ctx, requestIDs, impIDs)
	}
}

// Save will propagate saves to all underlying caches
func (c ComposedCache) Save(ctx context.Context, requestData map[string]json.RawMessage, impData map[string]json.RawMessage) {
	for _, cache := range c {
		cache.Save(ctx, requestData, impData)
	}
}

type fetcherWithCache struct {
	fetcher AllFetcher
	cache   Cache
}

// WithCache returns a Fetcher which uses the given Cache before delegating to the original.
// This can be called multiple times to compose Cache layers onto the backing Fetcher, though
// it is usually more desirable to first compose caches with Compose, ensuring propagation of updates
// and invalidations through all cache layers.
func WithCache(fetcher AllFetcher, cache Cache) AllFetcher {
	return &fetcherWithCache{
		cache:   cache,
		fetcher: fetcher,
	}
}

func (f *fetcherWithCache) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	requestData, impData = f.cache.Get(ctx, requestIDs, impIDs)

	// Fixes #311
	leftoverImps := findLeftovers(impIDs, impData)
	leftoverReqs := findLeftovers(requestIDs, requestData)

	if len(leftoverReqs) > 0 || len(leftoverImps) > 0 {
		fetcherReqData, fetcherImpData, fetcherErrs := f.fetcher.FetchRequests(ctx, leftoverReqs, leftoverImps)
		errs = fetcherErrs

		f.cache.Save(ctx, fetcherReqData, fetcherImpData)

		requestData = mergeData(requestData, fetcherReqData)
		impData = mergeData(impData, fetcherImpData)
	}

	return
}

func (f *fetcherWithCache) FetchCategories(primaryAdServer, publisherId, iabCategory string) (string, error) {
	return "", nil
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
