package exchange

import (
	"context"
	"encoding/json"
	"github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

func TestBidSerialization(t *testing.T) {
	a := newAuction(1)
	dealBid := &openrtb.Bid{
		ID:     "foo",
		ImpID:  "a",
		DealID: "deal1",
		Price:  0.5,
	}
	winningBid := &openrtb.Bid{
		ID:    "bar",
		ImpID: "a",
		Price: 1.5,
	}
	expectedJson := []json.RawMessage{
		json.RawMessage(`{"id":"bar","impid":"a","price":1.5}`),
		json.RawMessage(`{"id":"foo","impid":"a","dealid":"deal1","price":0.5}`),
	}
	a.addBid(openrtb_ext.BidderAppnexus, dealBid)
	a.addBid(openrtb_ext.BidderAppnexus, winningBid)

	mockClient := &mockCacheClient{
		mockReturn: []string{"0", "1"},
	}

	cacheBids(context.Background(), mockClient, a)
	assertJSONMatch(t, expectedJson, mockClient.capturedRequest)
	assertStringValue(t, `bid "bar"`, "0", a.cachedBids[winningBid])
	assertStringValue(t, `bid "foo"`, "1", a.cachedBids[dealBid])
}

func TestCacheFailures(t *testing.T) {
	a := newAuction(1)
	dealBid := &openrtb.Bid{
		ID:     "foo",
		ImpID:  "a",
		DealID: "deal1",
		Price:  0.5,
	}
	winningBid := &openrtb.Bid{
		ID:    "bar",
		ImpID: "a",
		Price: 1.5,
	}
	a.addBid(openrtb_ext.BidderAppnexus, winningBid)
	a.addBid(openrtb_ext.BidderAppnexus, dealBid)

	mockClient := &mockCacheClient{
		mockReturn: []string{"", "1"},
	}
	cacheBids(context.Background(), mockClient, a)
	assertStringValue(t, `bid "foo"`, "1", a.cachedBids[dealBid])
	if _, ok := a.cachedBids[winningBid]; ok {
		t.Error("If the cache call fails, no ID should be saved.")
	}
}

func assertJSONMatch(t *testing.T, expected []json.RawMessage, actual []json.RawMessage) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Errorf("Mismatched lengths. Expected %d, actual %d", len(expected), len(actual))
	}
	for i := 0; i < len(expected); i++ {
		if !jsonpatch.Equal(actual[i], expected[i]) {
			t.Errorf("Wrong JSON at index %d. Expected %s, got %s", i, string(expected[i]), string(actual[i]))
		}
	}
}

func assertStringsMatch(t *testing.T, expected string, actual string) {
	t.Helper()
	if actual != expected {
		t.Errorf("Strings do not match. Expected %s, got %s", expected, actual)
	}
}

type mockCacheClient struct {
	capturedRequest []json.RawMessage
	mockReturn      []string
}

func (c *mockCacheClient) PutJson(ctx context.Context, values []json.RawMessage) []string {
	c.capturedRequest = values
	return c.mockReturn
}
