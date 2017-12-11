package empty_fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/stored_requests"
)

// EmptyFetcher is a nil-object which has no Stored Requests.
// If PBS is configured to use this, then all the OpenRTB request data must be sent in the HTTP request.
func EmptyFetcher() stored_requests.Fetcher {
	return &instance
}

type emptyFetcher struct{}

func (fetcher *emptyFetcher) FetchRequests(ctx context.Context, ids []string) (map[string]json.RawMessage, []error) {
	errs := make([]error, 0, len(ids))
	for _, id := range ids {
		errs = append(errs, fmt.Errorf(`Stored request with id="%s" not found.`, id))
	}
	return nil, errs
}

var instance = emptyFetcher{}
