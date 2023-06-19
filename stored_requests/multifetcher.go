package stored_requests

import (
	"context"
	"encoding/json"
	"fmt"
)

// MultiFetcher is a Fetcher composed of multiple sub-Fetchers that are all polled for results.
type MultiFetcher []AllFetcher

// FetchRequests implements the Fetcher interface for MultiFetcher
func (mf MultiFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	requestData = make(map[string]json.RawMessage, len(requestIDs))
	impData = make(map[string]json.RawMessage, len(impIDs))

	// Loop over the fetchers
	for _, f := range mf {
		remainingRequestIDs := filter(requestIDs, requestData)
		requestIDs = remainingRequestIDs
		remainingImpIDs := filter(impIDs, impData)
		impIDs = remainingImpIDs

		theseRequestData, theseImpData, rerrs := f.FetchRequests(ctx, remainingRequestIDs, remainingImpIDs)
		// Drop NotFound errors, as other fetchers may have them. Also don't want multiple NotFound errors per ID.
		rerrs = dropMissingIDs(rerrs)
		if len(rerrs) > 0 {
			errs = append(errs, rerrs...)
		}
		addAll(requestData, theseRequestData)
		addAll(impData, theseImpData)
	}
	// Add missing ID errors back in for any IDs that are still missing
	errs = appendNotFoundErrors("Request", requestIDs, requestData, errs)
	errs = appendNotFoundErrors("Imp", impIDs, impData, errs)
	return
}

func (mf MultiFetcher) FetchResponses(ctx context.Context, ids []string) (data map[string]json.RawMessage, errs []error) {
	return nil, nil
}

func (mf MultiFetcher) FetchAccount(ctx context.Context, accountDefaultJSON json.RawMessage, accountID string) (account json.RawMessage, errs []error) {
	for _, f := range mf {
		if af, ok := f.(AccountFetcher); ok {
			if account, accErrs := af.FetchAccount(ctx, accountDefaultJSON, accountID); len(accErrs) == 0 {
				return account, nil
			} else {
				accErrs = dropMissingIDs(accErrs)
				errs = append(errs, accErrs...)
			}
		}
	}
	errs = append(errs, NotFoundError{accountID, "Account"})
	return nil, errs
}

func (mf MultiFetcher) FetchCategories(ctx context.Context, primaryAdServer, publisherId, iabCategory string) (string, error) {
	for _, f := range mf {
		if cf, ok := f.(CategoryFetcher); ok {
			iabCategory, _ := cf.FetchCategories(ctx, primaryAdServer, publisherId, iabCategory)
			if iabCategory != "" {
				return iabCategory, nil
			}
		}
	}

	// For now just return a NotFoundError if we didn't find it for some reason
	errtype := fmt.Sprintf("%s_%s.%s", primaryAdServer, publisherId, iabCategory)
	return "", NotFoundError{errtype, "Category"}
}

func addAll(base map[string]json.RawMessage, toAdd map[string]json.RawMessage) {
	for k, v := range toAdd {
		base[k] = v
	}
}

func filter(original []string, exclude map[string]json.RawMessage) (filtered []string) {
	if len(exclude) == 0 {
		filtered = original
		return
	}
	filtered = make([]string, 0, len(original))
	for _, id := range original {
		if _, ok := exclude[id]; !ok {
			filtered = append(filtered, id)
		}
	}
	return
}

func appendNotFoundErrors(dataType string, expected []string, contains map[string]json.RawMessage, errs []error) []error {
	for _, id := range expected {
		if _, ok := contains[id]; !ok {
			errs = append(errs, NotFoundError{id, dataType})
		}
	}
	return errs
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
