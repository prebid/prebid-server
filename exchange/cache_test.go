package exchange

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestBidSerialization(t *testing.T) {
	a := newAuction(1)
	winningBid := &openrtb.Bid{
		ID:    "bar",
		ImpID: "a",
		Price: 1.5,
	}
	otherBid := &openrtb.Bid{
		ID:    "foo",
		ImpID: "a",
		Price: 0.5,
	}
	a.addBid(openrtb_ext.BidderAppnexus, winningBid)
	a.addBid(openrtb_ext.BidderIndex, otherBid)

	mockClient := &mockCacheClient{
		mockReturns: map[*openrtb.Bid]string{
			winningBid: "0",
			otherBid:   "1",
		},
	}

	cacheBids(context.Background(), mockClient, a, openrtb_ext.PriceGranularityMedium)
	assertStringValue(t, `bid "bar"`, "0", a.cachedBids[winningBid])
	assertStringValue(t, `bid "foo"`, "1", a.cachedBids[otherBid])
}

func TestCacheFailures(t *testing.T) {
	a := newAuction(1)
	winningBid := &openrtb.Bid{
		ID:    "bar",
		ImpID: "a",
		Price: 1.5,
	}
	otherBid := &openrtb.Bid{
		ID:    "foo",
		ImpID: "a",
		Price: 0.5,
	}
	a.addBid(openrtb_ext.BidderAppnexus, winningBid)
	a.addBid(openrtb_ext.BidderIndex, otherBid)

	mockClient := &mockCacheClient{
		mockReturns: map[*openrtb.Bid]string{
			winningBid: "",
			otherBid:   "1",
		},
	}
	cacheBids(context.Background(), mockClient, a, openrtb_ext.PriceGranularityMedium)
	assertStringValue(t, `bid "foo"`, "1", a.cachedBids[otherBid])
	if _, ok := a.cachedBids[winningBid]; ok {
		t.Error("If the cache call fails, no ID should exist for that bid.")
	}
}

func TestMarshalFailure(t *testing.T) {
	auc := newAuction(2)

	badBid := &openrtb.Bid{
		ImpID: "foo",
		Price: 1,
		Ext:   openrtb.RawJSON("{"),
	}
	goodBid := &openrtb.Bid{
		ImpID: "bar",
		Price: 2,
	}
	auc.addBid(openrtb_ext.BidderAppnexus, badBid)
	auc.addBid(openrtb_ext.BidderAppnexus, goodBid)

	mockClient := &mockCacheClient{
		mockReturns: map[*openrtb.Bid]string{
			badBid:  "0",
			goodBid: "1",
		},
	}

	cacheBids(context.Background(), mockClient, auc, openrtb_ext.PriceGranularityMedium)
	if _, ok := auc.cacheId(badBid); ok {
		t.Errorf("bids with malformed JSON should not be cached.")
	}
	if id, ok := auc.cacheId(goodBid); ok {
		if id != "1" {
			t.Errorf("Wrong id for good bid. Expected 1, got %s", id)
		}
	} else {
		t.Errorf("bids with malformed JSON should not prevent other bids from being cached.")
	}
}

type mockCacheClient struct {
	mockReturns map[*openrtb.Bid]string
}

func (c *mockCacheClient) PutJson(ctx context.Context, values []json.RawMessage) []string {
	returns := make([]string, len(values))
	for i, value := range values {
		for bid, id := range c.mockReturns {
			bidBytes, _ := json.Marshal(bid)
			if jsonpatch.Equal(bidBytes, value) {
				returns[i] = id
				break
			}
		}
	}
	return returns
}
