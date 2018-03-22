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

// Prevents #378
func TestTargetingWinners(t *testing.T) {
	doTargetingWinnersTest(t, true)
}

func TestTargetingWithoutWinners(t *testing.T) {
	doTargetingWinnersTest(t, false)
}

func doTargetingWinnersTest(t *testing.T, includeWinners bool) {
	mockBids := map[openrtb_ext.BidderName][]*openrtb.Bid{
		openrtb_ext.BidderAppnexus: []*openrtb.Bid{&openrtb.Bid{
			ID:    "losing-bid",
			ImpID: "some-imp",
			Price: 0.5,
		}, &openrtb.Bid{
			ID:    "winning-bid",
			ImpID: "some-imp",
			Price: 0.7,
		}},
		openrtb_ext.BidderRubicon: []*openrtb.Bid{&openrtb.Bid{
			ID:    "contending-bid",
			ImpID: "some-imp",
			Price: 0.6,
		}},
	}
	bids := runTargetingAuction(t, mockBids, false, includeWinners, false)

	// Make sure that the normal keys exist on the bids where they're expected to exist
	assertKeyExists(t, bids["winning-bid"], string(openrtb_ext.HbpbConstantKey), includeWinners)
	assertKeyExists(t, bids["winning-bid"], openrtb_ext.HbpbConstantKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), true)

	assertKeyExists(t, bids["contending-bid"], string(openrtb_ext.HbpbConstantKey), false)
	assertKeyExists(t, bids["contending-bid"], openrtb_ext.HbpbConstantKey.BidderKey(openrtb_ext.BidderRubicon, maxKeyLength), true)

	assertKeyExists(t, bids["losing-bid"], string(openrtb_ext.HbpbConstantKey), false)
	assertKeyExists(t, bids["losing-bid"], openrtb_ext.HbpbConstantKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), false)

	// Make sure that the unexpected keys *don't* exist
	assertKeyExists(t, bids["winning-bid"], string(openrtb_ext.HbCacheKey), false)
	assertKeyExists(t, bids["winning-bid"], openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), false)
	assertKeyExists(t, bids["contending-bid"], string(openrtb_ext.HbCacheKey), false)
	assertKeyExists(t, bids["contending-bid"], openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderRubicon, maxKeyLength), false)
	assertKeyExists(t, bids["losing-bid"], string(openrtb_ext.HbCacheKey), false)
	assertKeyExists(t, bids["losing-bid"], openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), false)

	assertKeyExists(t, bids["winning-bid"], string(openrtb_ext.HbEnvKey), false)
	assertKeyExists(t, bids["contending-bid"], string(openrtb_ext.HbEnvKey), false)
	assertKeyExists(t, bids["losing-bid"], string(openrtb_ext.HbEnvKey), false)
}

func TestEnvKey(t *testing.T) {
	mockBids := map[openrtb_ext.BidderName][]*openrtb.Bid{
		openrtb_ext.BidderAppnexus: []*openrtb.Bid{&openrtb.Bid{
			ID:    "losing-bid",
			ImpID: "some-imp",
			Price: 0.5,
		}, &openrtb.Bid{
			ID:    "winning-bid",
			ImpID: "some-imp",
			Price: 0.7,
		}},
		openrtb_ext.BidderRubicon: []*openrtb.Bid{&openrtb.Bid{
			ID:    "contending-bid",
			ImpID: "some-imp",
			Price: 0.6,
		}},
	}
	bids := runTargetingAuction(t, mockBids, false, true, true)

	assertKeyExists(t, bids["winning-bid"], string(openrtb_ext.HbEnvKey), true)
	assertKeyExists(t, bids["winning-bid"], openrtb_ext.HbEnvKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), true)
	assertKeyExists(t, bids["losing-bid"], string(openrtb_ext.HbEnvKey), false)
	assertKeyExists(t, bids["losing-bid"], openrtb_ext.HbEnvKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), false)
	assertKeyExists(t, bids["contending-bid"], string(openrtb_ext.HbEnvKey), false)
	assertKeyExists(t, bids["contending-bid"], openrtb_ext.HbEnvKey.BidderKey(openrtb_ext.BidderRubicon, maxKeyLength), true)
}

// Prevents #378
func TestTargetingCache(t *testing.T) {
	mockBids := map[openrtb_ext.BidderName][]*openrtb.Bid{
		openrtb_ext.BidderAppnexus: []*openrtb.Bid{&openrtb.Bid{
			ID:    "losing-bid",
			ImpID: "some-imp",
			Price: 0.5,
		}, &openrtb.Bid{
			ID:    "winning-bid",
			ImpID: "some-imp",
			Price: 0.7,
		}},
		openrtb_ext.BidderRubicon: []*openrtb.Bid{&openrtb.Bid{
			ID:    "contending-bid",
			ImpID: "some-imp",
			Price: 0.6,
		}},
	}
	bids := runTargetingAuction(t, mockBids, true, true, false)

	// Make sure that the cache keys exist on the bids where they're expected to
	assertKeyExists(t, bids["winning-bid"], string(openrtb_ext.HbCacheKey), true)
	assertKeyExists(t, bids["winning-bid"], openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), true)

	assertKeyExists(t, bids["contending-bid"], string(openrtb_ext.HbCacheKey), false)
	assertKeyExists(t, bids["contending-bid"], openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderRubicon, maxKeyLength), true)

	assertKeyExists(t, bids["losing-bid"], string(openrtb_ext.HbCacheKey), false)
	assertKeyExists(t, bids["losing-bid"], openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), false)
}

func TestTargetingKeys(t *testing.T) {
	mockBids := map[openrtb_ext.BidderName][]*openrtb.Bid{
		openrtb_ext.BidderAppnexus: []*openrtb.Bid{&openrtb.Bid{
			ID:    "some-bid",
			ImpID: "some-imp",
			Price: 0.5,
			W:     500,
			H:     200,
		}},
	}
	bids := runTargetingAuction(t, mockBids, true, true, false)

	assertKeyValue(t, bids["some-bid"], string(openrtb_ext.HbpbConstantKey), "0.50")
	assertKeyValue(t, bids["some-bid"], openrtb_ext.HbpbConstantKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), "0.50")

	assertKeyValue(t, bids["some-bid"], string(openrtb_ext.HbBidderConstantKey), "appnexus")
	assertKeyValue(t, bids["some-bid"], openrtb_ext.HbBidderConstantKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), "appnexus")

	assertKeyValue(t, bids["some-bid"], string(openrtb_ext.HbSizeConstantKey), "500x200")
	assertKeyValue(t, bids["some-bid"], openrtb_ext.HbSizeConstantKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), "500x200")
}

func assertKeyValue(t *testing.T, bid *openrtb.Bid, key string, expectedValue string) {
	t.Helper()
	targets := parseTargets(t, bid)
	if value, ok := targets[key]; ok {
		if value != expectedValue {
			t.Errorf("Bid %s has bad value for key %s. Expected %s, actual %s", bid.ID, key, expectedValue, value)
		}
	} else {
		t.Errorf("Bid %s missing expected key: %s.", bid.ID, key)
	}
}

func assertKeyExists(t *testing.T, bid *openrtb.Bid, key string, expected bool) {
	t.Helper()
	targets := parseTargets(t, bid)
	if _, ok := targets[key]; ok != expected {
		t.Errorf("Bid %s has wrong key: %s. Expected? %t, Exists? %t", bid.ID, key, expected, ok)
	}
}

// runAuction takes a bunch of mock bids by Bidder and runs an auction. It returns a map of Bids indexed by their ImpID.
// If includeCache is true, the auction will be run with cacheing as well, so the cache targeting keys should exist.
func runTargetingAuction(t *testing.T, mockBids map[openrtb_ext.BidderName][]*openrtb.Bid, includeCache bool, includeWinners bool, isApp bool) map[string]*openrtb.Bid {
	server := httptest.NewServer(http.HandlerFunc(mockServer))
	defer server.Close()

	ex := &exchange{
		adapterMap: buildAdapterMap(mockBids, server.URL, server.Client()),
		m:          pbsmetrics.NewMetrics(metrics.NewRegistry(), buildBidderList(mockBids)),
		cache:      &wellBehavedCache{},
		cacheTime:  time.Duration(0),
	}

	imps := buildImps(t, mockBids)

	req := &openrtb.BidRequest{
		Imp: imps,
		Ext: buildTargetingExt(includeCache, includeWinners),
	}
	if isApp {
		req.App = &openrtb.App{}
	} else {
		req.Site = &openrtb.Site{}
	}

	bidResp, err := ex.HoldAuction(context.Background(), req, &mockFetcher{})

	if err != nil {
		t.Fatalf("Unexpected errors running auction: %v", err)
	}
	if len(bidResp.SeatBid) != len(mockBids) {
		t.Fatalf("Unexpected number of SeatBids. Expected %d, got %d", len(mockBids), len(bidResp.SeatBid))
	}

	return buildBidMap(bidResp.SeatBid, len(mockBids))
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

func buildTargetingExt(includeCache bool, includeWinners bool) openrtb.RawJSON {
	var targeting string
	if includeWinners {
		targeting = "{}"
	} else {
		targeting = `{"includeWinners": false}`
	}

	if includeCache {
		return openrtb.RawJSON(`{"prebid":{"targeting":` + targeting + `,"cache":{"bids":{}}}}`)
	}

	return openrtb.RawJSON(`{"prebid":{"targeting":` + targeting + `}}`)
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

func buildBidMap(seatBids []openrtb.SeatBid, numBids int) map[string]*openrtb.Bid {
	bids := make(map[string]*openrtb.Bid, numBids)
	for _, seatBid := range seatBids {
		for i := 0; i < len(seatBid.Bid); i++ {
			bid := seatBid.Bid[i]
			bids[bid.ID] = &bid
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
