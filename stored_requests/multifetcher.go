package stored_requests

import (
	"context"
	"encoding/json"
)

// MultiFetcher is a Fetcher composed of multiple sub-Fetchers that are all polled for results.
type MultiFetcher []Fetcher

// FetchRequests implements the Fetcher interface for MultiFetcher
func (mf *MultiFetcher) FetchRequests(ctx context.Context, ids []string) (map[string]json.RawMessage, []error) {
	var errs []error
	result := make(map[string]json.RawMessage, len(ids))
	// Loop over the fetchers
	for _, f := range *mf {
		remainingIDs := make([]string, 0, len(ids))
		for _, id := range ids {
			if _, ok := result[id]; !ok {
				remainingIDs = append(remainingIDs, id)
			}
		}
		ids = remainingIDs
		thisResult, rerrs := f.FetchRequests(ctx, ids)
		if len(rerrs) > 0 {
			errs = append(errs, rerrs...)
		}
		// Loop over the results
		for k, v := range thisResult {
			result[k] = v
		}
	}
	return result, errs
}
