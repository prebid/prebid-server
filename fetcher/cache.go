package fetcher

// Cache is a keyed store of composed typed values. Implementations must be safe
// for concurrent use.
type Cache[K comparable, V any] interface {
	// Get returns the cached value, whether it was found, and whether it is stale.
	// A stale value is still returned so the caller can choose how to refresh it.
	Get(key K) (value V, found bool, stale bool)
	Save(key K, v V)
	// Invalidate drops a key so the next Get is a miss. Used when a background
	// revalidation finds the key was deleted upstream.
	Invalidate(key K)
}
