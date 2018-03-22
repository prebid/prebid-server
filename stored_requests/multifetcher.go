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
	missingIDs := 0 // The number of missing IDs that each fetcher reported.
	// If the number of errors == number of missing IDs, then it is likely that the errors are simply due
	// to the IDs missing from the return result.
	numIDs := len(ids)
	// suspect fetchers ... fetchers that returned errors that don't match the number of missing IDs.
	suspect := 0

	// Loop over the fetchers
	for _, f := range *mf {
		remainingIDs := make([]string, 0, len(ids))
		for _, id := range ids {
			if _, ok := result[id]; !ok {
				remainingIDs = append(remainingIDs, id)
			}
		}
		missingIDs = missingIDs + len(remainingIDs)
		if len(errs) > 0 && len(errs) != len(remainingIDs) {
			// This doesn't look like a simple error per missing ID.
			suspect++
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
	// If we have all the results and a number of errors == the number of missing IDs, then assume all is good.
	if len(result) == numIDs && len(errs) <= missingIDs && suspect == 0 {
		errs = []error{}
	}
	return result, errs
}
