# Fetcher package

The `fetcher` package provides a generic read-through fetching engine for Prebid Server packages that need to load, transform, and optionally cache typed values.

A caller wires together:

- `Source`: fetches raw data by key.
- `Transform`: converts raw data into the typed value stored by the cache.
- `Cache`: controls positive-cache retention. The package includes bounded LRU,
  unbounded and nil cache implementations.
- Optional `NegativeStore`: caches definitive not-found results.
- Optional `BulkSource`: preloads the cache at startup.

The cache stores the transformed typed value, not raw JSON. `Transform` runs when a value is inserted into the cache, so cache hits are simple lookups without repeated unmarshalling or derivation work.

When serve-stale mode is enabled, stale entries are returned immediately and refreshed in the background. Optional request coalescing collapses concurrent misses for the same key into one upstream fetch per process.

## Cache choices

`NilCache` disables positive caching: every lookup reads from the source.

`UnboundedCache` keeps every saved value until it is explicitly invalidated. This matches the legacy stored-requests memory-cache behavior when it was configured without a TTL or size limit.

`LRUCache` bounds cache size by **number of entries** using `max_entries`. This is different from the legacy memory cache, which bounded some caches by **bytes**. Use `LRUCache` only when an item-count limit is acceptable for the caller's data shape.

Files:

- `fetcher.go`: engine setup and lookup flow.
- `contracts.go`: interfaces implemented by callers.
- `cache/`: positive cache implementations.
- `negative.go`: negative cache store.
- `revalidate.go`: background stale revalidation.
