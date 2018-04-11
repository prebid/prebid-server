package empty_fetcher

import (
	"context"
	"encoding/json"

	"github.com/prebid/prebid-server/stored_requests"
)

// EmptyFetcher is a nil-object which has no Stored Requests.
// If PBS is configured to use this, then all the OpenRTB request data must be sent in the HTTP request.
func EmptyFetcher() stored_requests.Fetcher {
	return &instance
}

type emptyFetcher struct{}

func (fetcher *emptyFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	errs = make([]error, 0, len(requestIDs)+len(impIDs))
	for _, id := range requestIDs {
		errs = append(errs, stored_requests.NotFoundError{
			ID:       id,
			DataType: "Request",
		})
	}
	for _, id := range impIDs {
		errs = append(errs, stored_requests.NotFoundError{
			ID:       id,
			DataType: "Imp",
		})
	}
	return
}

var instance = emptyFetcher{}
