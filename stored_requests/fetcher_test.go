package stored_requests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestPerfectCache(t *testing.T) {
	cache := &mockCache{
		mockGetReq: json.RawMessage(`{}`),
		mockGetImps: map[string]json.RawMessage{
			"known": json.RawMessage(`{}`),
		},
	}
	fetcher := &mockFetcher{}
	composed := WithCache(fetcher, cache)
	ids := []string{"known"}
	composed.FetchRequests(context.Background(), "req-id", ids)

	if cache.gotGetReq != "req-id" {
		t.Errorf("The cache called with the wrong request ID. Expected req-id, got %s.", cache.gotGetReq)
	}
	if len(cache.gotGetImps) != 1 {
		t.Errorf("The cache called with the wrong number of Imp IDs. Expected 1, got %d.", len(cache.gotGetImps))
		return
	}
	if cache.gotGetImps[0] != "known" {
		t.Errorf(`The cache called with the wrong Imp ID. Expected "known", got %s.`, cache.gotGetImps[0])
	}

	if len(fetcher.gotReqQuery) != 0 {
		t.Errorf("The delegate fetcher should not have been called with any Req ID. Got %#v", fetcher.gotReqQuery)
	}
	if len(fetcher.gotImpQuery) != 0 {
		t.Errorf("The delegate fetcher should not have been called with any Imp IDs. Got %#v", fetcher.gotImpQuery)
	}
}

func TestImperfectCache(t *testing.T) {
	cache := &mockCache{
		mockGetImps: map[string]json.RawMessage{
			"cached": json.RawMessage(`true`),
		},
	}
	fetcher := &mockFetcher{
		mockGetImps: map[string]json.RawMessage{
			"uncached": json.RawMessage(`false`),
		},
	}
	composed := WithCache(fetcher, cache)
	ids := []string{"cached", "uncached"}
	reqData, fetchedData, errs := composed.FetchRequests(context.Background(), "", ids)

	if len(reqData) != 0 {
		t.Errorf("Got unexpected Request data: %s", string(reqData))
	}
	if len(cache.gotGetImps) != 2 {
		t.Errorf("The cache called with the wrong number of Imp IDs. Expected 2, got %d.", len(cache.gotGetImps))
	}
	if cache.gotGetImps[0] != "cached" {
		t.Errorf(`Wrong cache.get on id[0]. Expected "cached", got %s.`, cache.gotGetImps[0])
	}
	if cache.gotGetImps[1] != "uncached" {
		t.Errorf(`Wrong cache.get on id[1]. Expected "uncached", got %s.`, cache.gotGetImps[1])
	}
	if !bytes.Equal(cache.gotSaveImps["uncached"], []byte("false")) {
		t.Errorf("Failed to save cache miss data. Expected false, got %s", cache.gotSaveImps["uncached"])
	}

	if len(fetcher.gotImpQuery) != 1 {
		t.Errorf("The delegate fetcher should have been called with 1 Imp ID. Got %d.", len(fetcher.gotImpQuery))
	}

	if fetcher.gotImpQuery[0] != "uncached" {
		t.Errorf("The delegate fetcher was called with the wrong imp id. Expected uncached, got %s", fetcher.gotImpQuery[0])
	}
	if len(errs) != 0 {
		t.Errorf("Got unexpected errors: %v", errs)
	}
	if len(fetchedData) != 2 {
		t.Errorf("Unexpected data fetched. Expected 2 entries, but got %d", len(fetchedData))
	}
	if cachedData, _ := fetchedData["cached"]; !bytes.Equal(cachedData, []byte("true")) {
		t.Errorf("Cached data was corrupted. Expected true, got %s", string(cachedData))
	}
	if cachedData, _ := fetchedData["uncached"]; !bytes.Equal(cachedData, []byte("false")) {
		t.Errorf("Uncached data was corrupted. Expected false, got %s", string(cachedData))
	}
}

func TestMissingData(t *testing.T) {
	cache := &mockCache{}
	fetcher := &mockFetcher{
		returnErrs: []error{errors.New("Data not found")},
	}
	composed := WithCache(fetcher, cache)
	_, fetchedData, errs := composed.FetchRequests(context.Background(), "", []string{"unknown"})
	if len(errs) != 1 {
		t.Errorf("Errors from the delegate fetcher should be returned. Got %d errors.", len(errs))
	}
	if errs[0].Error() != "Data not found" {
		t.Errorf(`Unexpected error message. Expected "Data not found", got "%s"`, errs[0].Error())
	}
	if len(fetchedData) != 0 {
		t.Errorf("WithCache inserted unexpected data: %v", fetchedData)
	}
}

// Prevents #311
func TestCacheSaves(t *testing.T) {
	cache := &mockCache{
		mockGetImps: map[string]json.RawMessage{
			"abc": json.RawMessage(`{}`),
		},
	}
	fetcher := &mockFetcher{}
	composed := WithCache(fetcher, cache)
	composed.FetchRequests(context.Background(), "", []string{"abc", "abc"})
	if len(fetcher.gotImpQuery) != 0 {
		t.Errorf("No IDs should be requested from the fetcher for requests with duplicate ID. Got %#v", fetcher.gotImpQuery)
	}
}

type mockFetcher struct {
	mockGetReq  json.RawMessage
	mockGetImps map[string]json.RawMessage
	returnErrs  []error

	gotReqQuery string
	gotImpQuery []string
}

func (f *mockFetcher) FetchRequests(ctx context.Context, requestID string, impIDs []string) (json.RawMessage, map[string]json.RawMessage, []error) {
	f.gotReqQuery = requestID
	f.gotImpQuery = impIDs
	return f.mockGetReq, f.mockGetImps, f.returnErrs
}

type mockCache struct {
	gotSaveReqs map[string]json.RawMessage
	gotSaveImps map[string]json.RawMessage

	gotGetReq   string
	gotGetImps  []string
	mockGetReq  json.RawMessage
	mockGetImps map[string]json.RawMessage
}

func (c *mockCache) GetRequests(ctx context.Context, requestID string, impIDs []string) (json.RawMessage, map[string]json.RawMessage) {
	c.gotGetReq = requestID
	c.gotGetImps = impIDs
	return c.mockGetReq, c.mockGetImps
}

func (c *mockCache) SaveRequests(ctx context.Context, storedRequests map[string]json.RawMessage, storedImps map[string]json.RawMessage) {
	c.gotSaveReqs = storedRequests
	c.gotSaveImps = storedImps
	return
}
