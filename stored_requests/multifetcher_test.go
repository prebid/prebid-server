package stored_requests

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestMultiFetcher(t *testing.T) {
	mf0 := &mockFetcher{
		mockGetReqs: map[string]json.RawMessage{
			"abc": json.RawMessage(`{}`),
		},
		mockGetImps: map[string]json.RawMessage{
			"imp-0": json.RawMessage(`{}`),
		},
		returnErrs: []error{NotFoundError{"def", "Request"}, NotFoundError{"imp-1", "Imp"}},
	}
	mf1 := &mockFetcher{
		mockGetReqs: map[string]json.RawMessage{
			"def": json.RawMessage(`{}`),
		},
		mockGetImps: map[string]json.RawMessage{
			"imp-1": json.RawMessage(`{}`),
		},
		returnErrs: []error{NotFoundError{"abc", "Request"}, NotFoundError{"imp-0", "Imp"}},
	}
	mf := &MultiFetcher{mf0, mf1}

	// Verify we can use multifetcher as a fetcher
	var fetcher Fetcher = mf

	requestData, impData, errs := fetcher.FetchRequests(context.Background(), []string{"abc", "def"}, []string{"imp-0", "imp-1"})

	assertResults(t, "results", 2, len(requestData))
	assertResults(t, "results", 2, len(impData))
	assertResults(t, "errors", 0, len(errs))
}

func TestMissingID(t *testing.T) {
	mf0 := &mockFetcher{
		mockGetReqs: map[string]json.RawMessage{
			"abc": json.RawMessage(`{}`),
		},
		mockGetImps: map[string]json.RawMessage{
			"123": json.RawMessage(`{}`),
		},
		returnErrs: []error{NotFoundError{"def", "Request"}, NotFoundError{"ghi", "Request"}, NotFoundError{"456", "Imp"}, NotFoundError{"789", "Imp"}},
	}
	mf1 := &mockFetcher{
		mockGetReqs: map[string]json.RawMessage{
			"def": json.RawMessage(`{}`),
		},
		mockGetImps: map[string]json.RawMessage{
			"456": json.RawMessage(`{}`),
		},
		returnErrs: []error{NotFoundError{"abc", "Request"}, NotFoundError{"ghi", "Request"}, NotFoundError{"123", "Imp"}, NotFoundError{"789", "Imp"}},
	}
	mf := &MultiFetcher{mf0, mf1}
	reqIDs := []string{"abc", "def", "ghi"}
	impIDs := []string{"123", "456", "789"}

	requestData, impData, errs := mf.FetchRequests(context.Background(), reqIDs, impIDs)

	assertResults(t, "requests", 2, len(requestData))
	assertResults(t, "imps", 2, len(impData))
	assertResults(t, "errors", 2, len(errs))
}

func TestOtherError(t *testing.T) {
	mf0 := &mockFetcher{
		mockGetReqs: map[string]json.RawMessage{
			"abc": json.RawMessage(`{}`),
		},
		returnErrs: []error{NotFoundError{"def", "Request"}, NotFoundError{"123", "Imp"}, errors.New("Other error")},
	}
	mf1 := &mockFetcher{
		mockGetReqs: map[string]json.RawMessage{
			"def": json.RawMessage(`{}`),
		},
		mockGetImps: map[string]json.RawMessage{
			"123": json.RawMessage(`{}`),
		},
	}
	mf := &MultiFetcher{mf0, mf1}
	ids := []string{"abc", "def"}

	// Verify we can use multifetcher as a fetcher
	var fetcher Fetcher = mf

	requestData, impData, errs := fetcher.FetchRequests(context.Background(), ids, []string{"123"})

	assertResults(t, "results", 2, len(requestData))
	assertResults(t, "results", 1, len(impData))
	assertResults(t, "errors", 1, len(errs))
}

func assertResults(t *testing.T, obj string, expect int, found int) {
	if expect != found {
		t.Errorf("Expected %d %s, found %d", expect, obj, found)
	}
}
