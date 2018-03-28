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
		// Drop NotFound errors, as other fetchers may have them. Also don't want multiple NotFound errors per ID.
		rerrs = dropMissingIDs(rerrs)
		if len(rerrs) > 0 {
			errs = append(errs, rerrs...)
		}
		// Loop over the results
		for k, v := range thisResult {
			result[k] = v
		}
	}
	// Add missing ID errors back in for any IDs that are still missing
	for _, id := range ids {
		if _, ok := result[id]; !ok {
			errs = append(errs, NotFoundError(id))
		}
	}
	return result, errs
}

// dropMissingIDs will scrub the NotFoundError's from a slice of errors.
// The order of the errors will not be preserved.
func dropMissingIDs(errs []error) []error {
	// filtered errors
	ferrs := errs[:0]
	for _, e := range errs {
		if _, ok := e.(NotFoundError); !ok {
			ferrs = append(ferrs, e)
		}
	}
	return ferrs
}
