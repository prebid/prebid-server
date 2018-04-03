package file_fetcher

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/prebid/prebid-server/stored_requests"
)

// NewFileFetcher returns a Fetcher which uses the Client to pull data from the endpoint.
//
// This file expects the endpoint to satisfy the following API:
//
// GET {endpoint}?id=reqID&imp-ids=imp1,imp2,imp3
//
// This endpoint should return a payload like:
//
// {
//   "request": { ... stored request data ... },
//   "imps": {
//     "imp1": { ... stored data for imp1 ... },
//     "imp2": { ... stored data for imp2 ... },
//     "imp3": null // If imp3 is not found
//   }
// }
//
// If the request, or any of the imps are not found, then
func NewFetcher(client *http.Client, endpoint string) stored_requests.Fetcher {
	return &httpFetcher{
		client:   client,
		endpoint: endpoint,
	}
}

type httpFetcher struct {
	client   *http.Client
	endpoint string
}

func (fetcher *httpFetcher) FetchRequests(ctx context.Context, ids []string) (map[string]json.RawMessage, []error) {
	// TODO: Implement this
	// var errors []error = nil
	// for _, id := range ids {
	// 	if _, ok := fetcher.storedReqs[id]; !ok {
	// 		errors = append(errors, fmt.Errorf("No config found for id: %s", id))
	// 	}
	// }

	// // Even though there may be many other IDs here, the interface contract doesn't prohibit this.
	// // Returning the whole slice is much cheaper than making partial copies on each call.
	// return fetcher.storedReqs, errors
}
