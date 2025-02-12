package stored_requests

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/prebid/prebid-server/v3/metrics"
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
	FetchResponses(ctx context.Context, ids []string) (data map[string]json.RawMessage, errs []error)
}

type AccountFetcher interface {
	// FetchAccount fetches the host account configuration for a publisher
	FetchAccount(ctx context.Context, accountDefaultJSON json.RawMessage, accountID string) (json.RawMessage, []error)
}

type CategoryFetcher interface {
	// FetchCategories fetches the ad-server/publisher specific category for the given IAB category
	FetchCategories(ctx context.Context, primaryAdServer, publisherId, iabCategory string) (string, error)
}

// AllFetcher is an interface that encapsulates both the original Fetcher and the CategoryFetcher
type AllFetcher interface {
	Fetcher
	AccountFetcher
	CategoryFetcher
}

// NotFoundError is an error type to flag that an ID was not found by the Fetcher.
// This was added to support Multifetcher and any other case where we might expect
// that all IDs would not be found, and want to disentangle those errors from the others.
type NotFoundError struct {
	ID       string
	DataType string
}

type Category struct {
	Id   string
	Name string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf(`Stored %s with ID="%s" not found.`, e.DataType, e.ID)
}

// Cache is an intermediate layer which can be used to create more complex Fetchers by composition.
// Implementations must be safe for concurrent access by multiple goroutines.
// To add a Cache layer in front of a Fetcher, see WithCache()
type Cache struct {
	Requests  CacheJSON
	Imps      CacheJSON
	Responses CacheJSON
	Accounts  CacheJSON
}
type CacheJSON interface {
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
	Get(ctx context.Context, ids []string) (data map[string]json.RawMessage)

	// Invalidate will ensure that all values associated with the given IDs
	// are no longer returned by the cache until new values are saved via Update
	Invalidate(ctx context.Context, ids []string)

	// Save will add or overwrite the data in the cache at the given keys
	Save(ctx context.Context, data map[string]json.RawMessage)
}

// ComposedCache creates an interface to treat a slice of caches as a single cache
type ComposedCache []CacheJSON

// Get will attempt to Get from the caches in the order in which they are in the slice,
// stopping as soon as a value is found (or when all caches have been exhausted)
func (c ComposedCache) Get(ctx context.Context, ids []string) (data map[string]json.RawMessage) {
	data = make(map[string]json.RawMessage, len(ids))

	remainingIDs := ids

	for _, cache := range c {
		cachedData := cache.Get(ctx, remainingIDs)
		data, remainingIDs = updateFromCache(data, remainingIDs, cachedData)

		// finish early if all ids filled
		if len(remainingIDs) == 0 {
			break
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
func (c ComposedCache) Invalidate(ctx context.Context, ids []string) {
	for _, cache := range c {
		cache.Invalidate(ctx, ids)
	}
}

// Save will propagate saves to all underlying caches
func (c ComposedCache) Save(ctx context.Context, data map[string]json.RawMessage) {
	for _, cache := range c {
		cache.Save(ctx, data)
	}
}

type fetcherWithCache struct {
	fetcher       AllFetcher
	cache         Cache
	metricsEngine metrics.MetricsEngine
}

// WithCache returns a Fetcher which uses the given Caches before delegating to the original.
// This can be called multiple times to compose Cache layers onto the backing Fetcher, though
// it is usually more desirable to first compose caches with Compose, ensuring propagation of updates
// and invalidations through all cache layers.
func WithCache(fetcher AllFetcher, cache Cache, metricsEngine metrics.MetricsEngine) AllFetcher {
	return &fetcherWithCache{
		cache:         cache,
		fetcher:       fetcher,
		metricsEngine: metricsEngine,
	}
}

func (f *fetcherWithCache) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {

	requestData = f.cache.Requests.Get(ctx, requestIDs)
	impData = f.cache.Imps.Get(ctx, impIDs)

	// Fixes #311
	leftoverImps := findLeftovers(impIDs, impData)
	leftoverReqs := findLeftovers(requestIDs, requestData)

	// Record cache hits for stored requests and stored imps
	f.metricsEngine.RecordStoredReqCacheResult(metrics.CacheHit, len(requestIDs)-len(leftoverReqs))
	f.metricsEngine.RecordStoredImpCacheResult(metrics.CacheHit, len(impIDs)-len(leftoverImps))
	// Record cache misses for stored requests and stored imps
	f.metricsEngine.RecordStoredReqCacheResult(metrics.CacheMiss, len(leftoverReqs))
	f.metricsEngine.RecordStoredImpCacheResult(metrics.CacheMiss, len(leftoverImps))

	if len(leftoverReqs) > 0 || len(leftoverImps) > 0 {
		fetcherReqData, fetcherImpData, fetcherErrs := f.fetcher.FetchRequests(ctx, leftoverReqs, leftoverImps)
		errs = fetcherErrs

		f.cache.Requests.Save(ctx, fetcherReqData)
		f.cache.Imps.Save(ctx, fetcherImpData)

		requestData = mergeData(requestData, fetcherReqData)
		impData = mergeData(impData, fetcherImpData)
	}

	return
}

func (f *fetcherWithCache) FetchResponses(ctx context.Context, ids []string) (data map[string]json.RawMessage, errs []error) {
	data = f.cache.Responses.Get(ctx, ids)

	leftoverResp := findLeftovers(ids, data)

	if len(leftoverResp) > 0 {
		fetcherRespData, fetcherErrs := f.fetcher.FetchResponses(ctx, leftoverResp)
		errs = fetcherErrs

		f.cache.Responses.Save(ctx, fetcherRespData)

		data = mergeData(data, fetcherRespData)
	}

	return
}

func (f *fetcherWithCache) FetchAccount(ctx context.Context, acccountDefaultJSON json.RawMessage, accountID string) (account json.RawMessage, errs []error) {
	accountData := f.cache.Accounts.Get(ctx, []string{accountID})
	// TODO: add metrics
	if account, ok := accountData[accountID]; ok {
		f.metricsEngine.RecordAccountCacheResult(metrics.CacheHit, 1)
		return account, errs
	} else {
		f.metricsEngine.RecordAccountCacheResult(metrics.CacheMiss, 1)
	}
	account, errs = f.fetcher.FetchAccount(ctx, acccountDefaultJSON, accountID)
	if len(errs) == 0 {
		f.cache.Accounts.Save(ctx, map[string]json.RawMessage{accountID: account})
	}
	return account, errs
}

func (f *fetcherWithCache) FetchCategories(ctx context.Context, primaryAdServer, publisherId, iabCategory string) (string, error) {
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
