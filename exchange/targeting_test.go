package exchange

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/prebid-server/pbsmetrics"
	metrics "github.com/rcrowley/go-metrics"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestTargeting(t *testing.T) {
	impId := "some-imp"
	mockBids := map[openrtb_ext.BidderName][]*openrtb.Bid{
		openrtb_ext.BidderAppnexus: []*openrtb.Bid{&openrtb.Bid{
			ID:    "losing-bid",
			ImpID: impId,
			Price: 0.5,
		}, &openrtb.Bid{
			ID:    "winning-bid",
			ImpID: impId,
			Price: 0.7,
		}},
		openrtb_ext.BidderRubicon: []*openrtb.Bid{&openrtb.Bid{
			ID:    "contending-bid",
			ImpID: impId,
			Price: 0.6,
		}},
	}
	bids := runAuction(t, mockBids, false)

	var winner *openrtb.Bid
	var loser *openrtb.Bid
	var contender *openrtb.Bid
	for i := 0; i < len(bids[impId]); i++ {
		switch bids[impId][i].ID {
		case "winning-bid":
			winner = bids[impId][i]
		case "losing-bid":
			loser = bids[impId][i]
		case "contending-bid":
			contender = bids[impId][i]
		default:
			t.Fatalf("Unexpected bid: %s", bids[impId][i].ImpID)
		}
	}

	// Make sure that the normal keys exist on the bids where they're expected to exist
	assertKeyExists(t, winner, string(openrtb_ext.HbpbConstantKey), true)
	assertKeyExists(t, winner, openrtb_ext.HbpbConstantKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), true)

	assertKeyExists(t, contender, string(openrtb_ext.HbpbConstantKey), false)
	assertKeyExists(t, contender, openrtb_ext.HbpbConstantKey.BidderKey(openrtb_ext.BidderRubicon, maxKeyLength), true)

	assertKeyExists(t, loser, string(openrtb_ext.HbpbConstantKey), false)
	assertKeyExists(t, loser, openrtb_ext.HbpbConstantKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), false)

	// Make sure that the cache keys don't exist on any bids, because they weren't requested
	assertKeyExists(t, winner, string(openrtb_ext.HbCacheKey), false)
	assertKeyExists(t, winner, openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), false)

	assertKeyExists(t, contender, string(openrtb_ext.HbCacheKey), false)
	assertKeyExists(t, contender, openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderRubicon, maxKeyLength), false)

	assertKeyExists(t, loser, string(openrtb_ext.HbCacheKey), false)
	assertKeyExists(t, loser, openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), false)
}
func TestTargetingCache(t *testing.T) {
	impId := "some-imp"

	mockBids := map[openrtb_ext.BidderName][]*openrtb.Bid{
		openrtb_ext.BidderAppnexus: []*openrtb.Bid{&openrtb.Bid{
			ID:    "losing-bid",
			ImpID: impId,
			Price: 0.5,
		}, &openrtb.Bid{
			ID:    "winning-bid",
			ImpID: impId,
			Price: 0.7,
		}},
		openrtb_ext.BidderRubicon: []*openrtb.Bid{&openrtb.Bid{
			ID:    "contending-bid",
			ImpID: impId,
			Price: 0.6,
		}},
	}
	bids := runAuction(t, mockBids, true)

	var winner *openrtb.Bid
	var loser *openrtb.Bid
	var contender *openrtb.Bid
	for i := 0; i < len(bids[impId]); i++ {
		switch bids[impId][i].ID {
		case "winning-bid":
			winner = bids[impId][i]
		case "losing-bid":
			loser = bids[impId][i]
		case "contending-bid":
			contender = bids[impId][i]
		default:
			t.Fatalf("Unexpected bid: %s", bids[impId][i].ImpID)
		}
	}

	// Make sure that the cache keys exist on the bids where they're expected to exist
	assertKeyExists(t, winner, string(openrtb_ext.HbCacheKey), true)
	assertKeyExists(t, winner, openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), true)

	assertKeyExists(t, contender, string(openrtb_ext.HbCacheKey), false)
	assertKeyExists(t, contender, openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderRubicon, maxKeyLength), true)

	assertKeyExists(t, loser, string(openrtb_ext.HbCacheKey), false)
	assertKeyExists(t, loser, openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), false)
}

func assertKeyExists(t *testing.T, bid *openrtb.Bid, key string, expected bool) {
	targets := parseTargets(t, bid)
	if _, ok := targets[string(key)]; ok != expected {
		t.Errorf("Bid %s has wrong key: %s. Expected? %t, Exists? %t", bid.ID, key, expected, ok)
	}
}

// runAuction takes a bunch of mock bids by Bidder and runs an auction. It returns a map of Bids indexed by their ImpID.
// If includeCache is true, the auction will be run with cacheing as well, so the cache targeting keys should exist.
func runAuction(t *testing.T, mockBids map[openrtb_ext.BidderName][]*openrtb.Bid, includeCache bool) map[string][]*openrtb.Bid {
	server := httptest.NewServer(http.HandlerFunc(mockServer))
	defer server.Close()

	ex := &exchange{
		adapterMap: buildAdapterMap(mockBids, server.URL, server.Client()),
		m:          pbsmetrics.NewMetrics(metrics.NewRegistry(), buildBidderList(mockBids)),
		cache:      &wellBehavedCache{},
		cacheTime:  time.Duration(0),
	}

	imps := buildImps(t, mockBids)
	bidResp, err := ex.HoldAuction(context.Background(), &openrtb.BidRequest{
		Imp: imps,
		Ext: buildTargetingExt(includeCache),
	}, &mockFetcher{})

	if err != nil {
		t.Fatalf("Unexpected errors running auction: %v", err)
	}
	if len(bidResp.SeatBid) != len(mockBids) {
		t.Fatalf("Unexpected number of SeatBids. Expected %d, got %d", len(mockBids), len(bidResp.SeatBid))
	}
	return splitByImp(bidResp.SeatBid, len(imps))
}

func buildBidderList(bids map[openrtb_ext.BidderName][]*openrtb.Bid) []openrtb_ext.BidderName {
	bidders := make([]openrtb_ext.BidderName, 0, len(bids))
	for name, _ := range bids {
		bidders = append(bidders, name)
	}
	return bidders
}

func buildAdapterMap(bids map[openrtb_ext.BidderName][]*openrtb.Bid, mockServerURL string, client *http.Client) map[openrtb_ext.BidderName]adaptedBidder {
	adapterMap := make(map[openrtb_ext.BidderName]adaptedBidder, len(bids))
	for bidder, bids := range bids {
		adapterMap[bidder] = adaptBidder(&mockTargetingBidder{
			mockServerURL: mockServerURL,
			bids:          bids,
		}, client)
	}
	return adapterMap
}

func buildTargetingExt(includeCache bool) openrtb.RawJSON {
	if includeCache {
		return openrtb.RawJSON(`{"prebid":{"targeting":{},"cache":{"bids":{}}}}`)
	}

	return openrtb.RawJSON(`{"prebid":{"targeting":{}}}`)
}

func buildParams(t *testing.T, mockBids map[openrtb_ext.BidderName][]*openrtb.Bid) openrtb.RawJSON {
	params := make(map[string]openrtb.RawJSON)
	for bidder, _ := range mockBids {
		params[string(bidder)] = openrtb.RawJSON(`{"whatever":true}`)
	}
	ext, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to make imp exts: %v", err)
	}
	return ext
}

func buildImps(t *testing.T, mockBids map[openrtb_ext.BidderName][]*openrtb.Bid) []openrtb.Imp {
	impExt := buildParams(t, mockBids)

	var s struct{}
	impIds := make(map[string]struct{}, 2*len(mockBids))
	for _, bidList := range mockBids {
		for _, bid := range bidList {
			impIds[bid.ImpID] = s
		}
	}

	imps := make([]openrtb.Imp, 0, len(impIds))
	for impId, _ := range impIds {
		imps = append(imps, openrtb.Imp{
			ID:  impId,
			Ext: impExt,
		})
	}
	return imps
}

func splitByImp(seatBids []openrtb.SeatBid, numImps int) map[string][]*openrtb.Bid {
	bids := make(map[string][]*openrtb.Bid, numImps)
	for _, seatBid := range seatBids {
		for i := 0; i < len(seatBid.Bid); i++ {
			bid := seatBid.Bid[i]
			bids[bid.ImpID] = append(bids[bid.ImpID], &bid)
		}
	}
	return bids
}

func parseTargets(t *testing.T, bid *openrtb.Bid) map[string]string {
	t.Helper()
	var parsed openrtb_ext.ExtBid
	if err := json.Unmarshal(bid.Ext, &parsed); err != nil {
		t.Fatalf("Unexpected error parsing targeting params: %v", err)
	}
	return parsed.Prebid.Targeting
}

type mockTargetingBidder struct {
	mockServerURL string
	bids          []*openrtb.Bid
}

func (m *mockTargetingBidder) MakeRequests(request *openrtb.BidRequest) ([]*adapters.RequestData, []error) {
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     m.mockServerURL,
		Body:    []byte(""),
		Headers: http.Header{},
	}}, nil
}

func (m *mockTargetingBidder) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) ([]*adapters.TypedBid, []error) {
	typedBids := make([]*adapters.TypedBid, len(m.bids))
	for i := 0; i < len(m.bids); i++ {
		typedBids[i] = &adapters.TypedBid{
			Bid:     m.bids[i],
			BidType: openrtb_ext.BidTypeBanner,
		}
	}
	return typedBids, nil
}

type mockFetcher struct{}

func (f *mockFetcher) GetId(bidder openrtb_ext.BidderName) (string, bool) {
	return "", false
}

func mockServer(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("{}"))
}
