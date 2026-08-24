package fetcher

// Cache is a keyed store of composed typed values. Implementations must be safe
// for concurrent use.
type Cache[K comparable, V any] interface {
	// Get returns the value if present, and whether it is stale (past its refresh
	// time). Stale values are still returned so the read path never blocks on the
	// backend; the engine revalidates them in the background.
	Get(key K) (v V, ok bool, stale bool)
	Save(key K, v V)
	// Invalidate drops a key so the next Get is a miss. Used when a background
	// revalidation finds the key was deleted upstream.
	Invalidate(key K)
}
