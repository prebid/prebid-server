package cache

// NilCache is a pass-through cache: every Get is a miss and Save is a no-op. Paired
// with the engine's single-flight coalescing it yields "always fetch, still
// deduplicate" behaviour for direct-source / live tenants.
type NilCache[K comparable, V any] struct{}

func (NilCache[K, V]) Get(K) (value V, found bool, stale bool) {
	return value, false, false
}
func (NilCache[K, V]) Save(K, V)    {}
func (NilCache[K, V]) Invalidate(K) {}
