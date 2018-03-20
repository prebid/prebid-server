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
		returnErrs: []error{errors.New("Id 'def' not found")},
	}
	mf1 := &mockFetcher{
		returnData: map[string]json.RawMessage{
			"def": json.RawMessage(`{}`),
		},
		returnErrs: []error{errors.New("Id 'abc' not found")},
	}
	mf := &MultiFetcher{mf0, mf1}
	ids := []string{"abc", "def"}

	// Verify we can use multifetcher as a fetcher
	var fetcher Fetcher = mf

	result, errs := fetcher.FetchRequests(context.Background(), ids)

	if len(result) != 2 {
		t.Errorf("Expected 2 results, found %d", len(result))
	}

	if len(errs) != 2 {
		t.Errorf("Expected 2 errors, found %d", len(errs))
	}
}
