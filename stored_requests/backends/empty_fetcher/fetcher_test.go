package empty_fetcher

import (
	"context"
	"testing"
)

func TestErrorLength(t *testing.T) {
	fetcher := EmptyFetcher{}

	storedReqs, storedImps, errs := fetcher.FetchRequests(context.Background(), []string{"a", "b"}, []string{"c"})
	if len(storedReqs) != 0 {
		t.Errorf("The empty fetcher should never return stored requests. Got %d", len(storedReqs))
	}
	if len(storedImps) != 0 {
		t.Errorf("The empty fetcher should never return stored imps. Got %d", len(storedImps))
	}
	if len(errs) != 3 {
		t.Errorf("The empty fetcher should return 3 errors. Got %d", len(errs))
	}
}
