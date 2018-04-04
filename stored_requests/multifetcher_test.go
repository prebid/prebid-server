package stored_requests

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestMultiFetcher(t *testing.T) {
	mf0 := &mockFetcher{
		returnData: map[string]json.RawMessage{
			"abc": json.RawMessage(`{}`),
		},
		returnErrs: []error{NotFoundError("def")},
	}
	mf1 := &mockFetcher{
		returnData: map[string]json.RawMessage{
			"def": json.RawMessage(`{}`),
		},
		returnErrs: []error{NotFoundError("abc")},
	}
	mf := &MultiFetcher{mf0, mf1}
	ids := []string{"abc", "def"}

	// Verify we can use multifetcher as a fetcher
	var fetcher Fetcher = mf

	result, errs := fetcher.FetchRequests(context.Background(), ids)

	assertResults(t, "results", 2, len(result))
	assertResults(t, "errors", 0, len(errs))
}

func TestMissingID(t *testing.T) {
	mf0 := &mockFetcher{
		returnData: map[string]json.RawMessage{
			"abc": json.RawMessage(`{}`),
		},
		returnErrs: []error{NotFoundError("def"), NotFoundError("ghi")},
	}
	mf1 := &mockFetcher{
		returnData: map[string]json.RawMessage{
			"def": json.RawMessage(`{}`),
		},
		returnErrs: []error{NotFoundError("abc"), NotFoundError("ghi")},
	}
	mf := &MultiFetcher{mf0, mf1}
	ids := []string{"abc", "def", "ghi"}

	result, errs := mf.FetchRequests(context.Background(), ids)

	assertResults(t, "results", 2, len(result))
	assertResults(t, "errors", 1, len(errs))
}

func TestOtherError(t *testing.T) {
	mf0 := &mockFetcher{
		returnData: map[string]json.RawMessage{
			"abc": json.RawMessage(`{}`),
		},
		returnErrs: []error{NotFoundError("def"), errors.New("Other error")},
	}
	mf1 := &mockFetcher{
		returnData: map[string]json.RawMessage{
			"def": json.RawMessage(`{}`),
		},
		returnErrs: []error{},
	}
	mf := &MultiFetcher{mf0, mf1}
	ids := []string{"abc", "def"}

	// Verify we can use multifetcher as a fetcher
	var fetcher Fetcher = mf

	result, errs := fetcher.FetchRequests(context.Background(), ids)

	assertResults(t, "results", 2, len(result))
	assertResults(t, "errors", 1, len(errs))

}

func assertResults(t *testing.T, obj string, expect int, found int) {
	if expect != found {
		t.Errorf("Expected %d %s, found %d", expect, obj, found)
	}
}
