package fetchutil

// IdFetcher can find the user's ID for a specific Bidder.
type IdFetcher interface {
	GetUID(key string) (uid string, exists bool, notExpired bool)
	HasAnyLiveSyncs() bool
}
