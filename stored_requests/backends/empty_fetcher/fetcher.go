package empty_fetcher

import (
	"context"
	"encoding/json"

	"github.com/prebid/prebid-server/stored_requests"
)

// EmptyFetcher is a nil-object which has no Stored Requests.
// If PBS is configured to use this, then all the OpenRTB request data must be sent in the HTTP request.
type EmptyFetcher struct{}

func (fetcher EmptyFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string, respIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, respData map[string]json.RawMessage, errs []error) {

	//!!! resp
	errs = make([]error, 0, len(requestIDs)+len(impIDs)+len(respIDs))
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
	for _, id := range respIDs {
		errs = append(errs, stored_requests.NotFoundError{
			ID:       id,
			DataType: "Response",
		})
	}
	return
}

func (fetcher EmptyFetcher) FetchAccount(ctx context.Context, accountID string) (json.RawMessage, []error) {
	return nil, []error{stored_requests.NotFoundError{accountID, "Account"}}
}

func (fetcher EmptyFetcher) FetchCategories(ctx context.Context, primaryAdServer, publisherId, iabCategory string) (string, error) {
	return "", nil
}
