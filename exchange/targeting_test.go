package exchange

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currencies"

	"github.com/prebid/prebid-server/gdpr"

	"github.com/prebid/prebid-server/pbsmetrics"
	metricsConf "github.com/prebid/prebid-server/pbsmetrics/config"
	metricsConfig "github.com/prebid/prebid-server/pbsmetrics/config"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

// Using this set of bids in more than one test
var mockBids = map[openrtb_ext.BidderName][]*openrtb.Bid{
	openrtb_ext.BidderAppnexus: {{
		ID:    "losing-bid",
		ImpID: "some-imp",
		Price: 0.5,
		CrID:  "1",
	}, {
		ID:    "winning-bid",
		ImpID: "some-imp",
		Price: 0.7,
		CrID:  "2",
	}},
	openrtb_ext.BidderRubicon: {{
		ID:    "contending-bid",
		ImpID: "some-imp",
		Price: 0.6,
		CrID:  "3",
	}},
}

// Prevents #378. This is not a JSON test because the cache ID values aren't reproducible, which makes them a pain to test in that format.
func TestTargetingCache(t *testing.T) {
	bids := runTargetingAuction(t, mockBids, true, true, true, false)

	// Make sure that the cache keys exist on the bids where they're expected to
	assertKeyExists(t, bids["winning-bid"], string(openrtb_ext.HbCacheKey), true)
	assertKeyExists(t, bids["winning-bid"], openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), true)

	assertKeyExists(t, bids["contending-bid"], string(openrtb_ext.HbCacheKey), false)
	assertKeyExists(t, bids["contending-bid"], openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderRubicon, maxKeyLength), true)

	assertKeyExists(t, bids["losing-bid"], string(openrtb_ext.HbCacheKey), false)
	assertKeyExists(t, bids["losing-bid"], openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderAppnexus, maxKeyLength), false)

	//assert hb_cache_host was included
	assert.Contains(t, string(bids["winning-bid"].Ext), string(openrtb_ext.HbConstantCacheHostKey))
	assert.Contains(t, string(bids["winning-bid"].Ext), "www.pbcserver.com")

	//assert hb_cache_path was included
	assert.Contains(t, string(bids["winning-bid"].Ext), string(openrtb_ext.HbConstantCachePathKey))
	assert.Contains(t, string(bids["winning-bid"].Ext), "/pbcache/endpoint")

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
func runTargetingAuction(t *testing.T, mockBids map[openrtb_ext.BidderName][]*openrtb.Bid, includeCache bool, includeWinners bool, includeBidderKeys bool, isApp bool) map[string]*openrtb.Bid {
	server := httptest.NewServer(http.HandlerFunc(mockServer))
	defer server.Close()

	ex := &exchange{
		adapterMap:          buildAdapterMap(mockBids, server.URL, server.Client()),
		me:                  &metricsConf.DummyMetricsEngine{},
		cache:               &wellBehavedCache{},
		cacheTime:           time.Duration(0),
		gDPR:                gdpr.AlwaysAllow{},
		currencyConverter:   currencies.NewRateConverterDefault(),
		UsersyncIfAmbiguous: false,
	}

	imps := buildImps(t, mockBids)

	req := &openrtb.BidRequest{
		Imp: imps,
		Ext: buildTargetingExt(includeCache, includeWinners, includeBidderKeys),
	}
	if isApp {
		req.App = &openrtb.App{}
	} else {
		req.Site = &openrtb.Site{}
	}

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}
	bidResp, err := ex.HoldAuction(context.Background(), req, &mockFetcher{}, pbsmetrics.Labels{}, &categoriesFetcher, nil)

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
	for name := range bids {
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
		}, client, &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus)
	}
	return adapterMap
}

func buildTargetingExt(includeCache bool, includeWinners bool, includeBidderKeys bool) json.RawMessage {
	var targeting string
	if includeWinners && includeBidderKeys {
		targeting = "{}"
	} else if !includeWinners && includeBidderKeys {
		targeting = `{"includewinners": false}`
	} else if includeWinners && !includeBidderKeys {
		targeting = `{"includebidderkeys": false}`
	} else {
		targeting = `{"includewinners": false, "includebidderkeys": false}`
	}

	if includeCache {
		return json.RawMessage(`{"prebid":{"targeting":` + targeting + `,"cache":{"bids":{}}}}`)
	}

	return json.RawMessage(`{"prebid":{"targeting":` + targeting + `}}`)
}

func buildParams(t *testing.T, mockBids map[openrtb_ext.BidderName][]*openrtb.Bid) json.RawMessage {
	params := make(map[string]json.RawMessage)
	for bidder := range mockBids {
		params[string(bidder)] = json.RawMessage(`{"whatever":true}`)
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
	for impId := range impIds {
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

func (m *mockTargetingBidder) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     m.mockServerURL,
		Body:    []byte(""),
		Headers: http.Header{},
	}}, nil
}

func (m *mockTargetingBidder) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	bidResponse := &adapters.BidderResponse{
		Bids: make([]*adapters.TypedBid, len(m.bids)),
	}
	for i := 0; i < len(m.bids); i++ {
		bidResponse.Bids[i] = &adapters.TypedBid{
			Bid:     m.bids[i],
			BidType: openrtb_ext.BidTypeBanner,
		}
	}
	return bidResponse, nil
}

type mockFetcher struct{}

func (f *mockFetcher) GetId(bidder openrtb_ext.BidderName) (string, bool) {
	return "", false
}

func mockServer(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("{}"))
}
