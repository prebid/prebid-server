package config

import (
	"context"
	"testing"

	"github.com/prebid/prebid-server/config"
)

func TestNewEmptyFetcher(t *testing.T) {
	fetcher, ampFetcher, db := newFetchers(&config.StoredRequests{})
	if fetcher == nil || ampFetcher == nil {
		t.Errorf("The fetchers should be non-nil, even with an empty config.")
	}
	if db != nil {
		t.Errorf("The database should be nil, since none was used.")
	}
	if _, _, errs := fetcher.FetchRequests(context.Background(), []string{"some-id"}, []string{"other-id"}); len(errs) != 2 {
		t.Errorf("The returned fetcher should fail on any IDs.")
	}
	if _, _, errs := ampFetcher.FetchRequests(context.Background(), []string{"some-id"}, []string{"other-id"}); len(errs) != 2 {
		t.Errorf("The returned ampFetcher should fail on any IDs.")
	}
}
