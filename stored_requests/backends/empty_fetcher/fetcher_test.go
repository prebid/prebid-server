package empty_fetcher

import (
	"context"
	"testing"
)

func TestErrorLength(t *testing.T) {
	fetcher := EmptyFetcher()

	storedReqs, errs := fetcher.FetchRequests(context.Background(), []string{"a", "b"})
	if len(storedReqs) != 0 {
		t.Errorf("The empty fetcher should never return stored requests. Got %d", len(storedReqs))
	}
	if len(errs) != 2 {
		t.Errorf("The empty fetcher should return 2 errors. Got %d", len(errs))
	}
}
