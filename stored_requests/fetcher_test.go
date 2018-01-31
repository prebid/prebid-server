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
		mockGetData: map[string]json.RawMessage{
			"known": json.RawMessage(`{}`),
		},
	}
	fetcher := &mockFetcher{}
	composed := WithCache(fetcher, cache)
	ids := []string{"known"}
	composed.FetchRequests(context.Background(), ids)

	if len(cache.gotGetIds) != 1 {
		t.Errorf("The cache called with the wrong number of IDs. Expected 1, got %d.", len(cache.gotGetIds))
	}
	if cache.gotGetIds[0] != "known" {
		t.Errorf(`The cache called with the wrong ID. Expected "known", got %s.`, cache.gotGetIds[0])
	}
	if len(fetcher.gotRequest) != 0 {
		t.Errorf("The delegate fetcher should not have been called with any IDs. Got %#v", fetcher.gotRequest)
	}
}

func TestImperfectCache(t *testing.T) {
	cache := &mockCache{
		mockGetData: map[string]json.RawMessage{
			"cached": json.RawMessage(`true`),
		},
	}
	fetcher := &mockFetcher{
		returnData: map[string]json.RawMessage{
			"uncached": json.RawMessage(`false`),
		},
	}
	composed := WithCache(fetcher, cache)
	ids := []string{"cached", "uncached"}
	fetchedData, errs := composed.FetchRequests(context.Background(), ids)

	if len(cache.gotGetIds) != 2 {
		t.Errorf("The cache called with the wrong number of IDs. Expected 2, got %d.", len(cache.gotGetIds))
	}
	if cache.gotGetIds[0] != "cached" {
		t.Errorf(`Wrong cache.get on id[0]. Expected "cached", got %s.`, cache.gotGetIds[0])
	}
	if cache.gotGetIds[1] != "uncached" {
		t.Errorf(`Wrong cache.get on id[1]. Expected "uncached", got %s.`, cache.gotGetIds[1])
	}
	if !bytes.Equal(cache.gotSaveValues["uncached"], []byte("false")) {
		t.Errorf("Failed to save cache miss data. Expected false, got %s", cache.gotSaveValues["uncached"])
	}

	if len(fetcher.gotRequest) != 1 {
		t.Errorf("The delegate fetcher should have been called with 1 ID. Got %d.", len(fetcher.gotRequest))
	}

	if fetcher.gotRequest[0] != "uncached" {
		t.Errorf("The delegate fetcher was called with the wrong id. Expected uncached, got %s", fetcher.gotRequest[0])
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
	fetchedData, errs := composed.FetchRequests(context.Background(), []string{"unknown"})
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
		mockGetData: map[string]json.RawMessage{
			"abc": json.RawMessage(`{}`),
		},
	}
	fetcher := &mockFetcher{}
	composed := WithCache(fetcher, cache)
	composed.FetchRequests(context.Background(), []string{"abc", "abc"})
	if len(fetcher.gotRequest) != 0 {
		t.Errorf("No IDs should be requested from the fetcher for requests with duplicate ID. Got %#v", fetcher.gotRequest)
	}
}

type mockFetcher struct {
	returnData map[string]json.RawMessage
	returnErrs []error
	gotRequest []string
}

func (f *mockFetcher) FetchRequests(ctx context.Context, ids []string) (map[string]json.RawMessage, []error) {
	f.gotRequest = ids
	return f.returnData, f.returnErrs
}

type mockCache struct {
	gotSaveValues map[string]json.RawMessage
	gotGetIds     []string
	mockGetData   map[string]json.RawMessage
}

func (c *mockCache) GetRequests(ctx context.Context, ids []string) map[string]json.RawMessage {
	c.gotGetIds = ids
	return c.mockGetData
}

func (c *mockCache) SaveRequests(ctx context.Context, values map[string]json.RawMessage) {
	c.gotSaveValues = values
}
