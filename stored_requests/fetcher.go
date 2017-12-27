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
