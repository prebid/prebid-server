package exchange

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"

	"github.com/prebid/prebid-server/gdpr"

	metricsConf "github.com/prebid/prebid-server/metrics/config"
	metricsConfig "github.com/prebid/prebid-server/metrics/config"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

// Using this set of bids in more than one test
var mockBids = map[openrtb_ext.BidderName][]*openrtb2.Bid{
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
	assertKeyExists(t, bids["winning-bid"], openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderAppnexus, MaxKeyLength), true)

	assertKeyExists(t, bids["contending-bid"], string(openrtb_ext.HbCacheKey), false)
	assertKeyExists(t, bids["contending-bid"], openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderRubicon, MaxKeyLength), true)

	assertKeyExists(t, bids["losing-bid"], string(openrtb_ext.HbCacheKey), false)
	assertKeyExists(t, bids["losing-bid"], openrtb_ext.HbCacheKey.BidderKey(openrtb_ext.BidderAppnexus, MaxKeyLength), false)

	//assert hb_cache_host was included
	assert.Contains(t, string(bids["winning-bid"].Ext), string(openrtb_ext.HbConstantCacheHostKey))
	assert.Contains(t, string(bids["winning-bid"].Ext), "www.pbcserver.com")

	//assert hb_cache_path was included
	assert.Contains(t, string(bids["winning-bid"].Ext), string(openrtb_ext.HbConstantCachePathKey))
	assert.Contains(t, string(bids["winning-bid"].Ext), "/pbcache/endpoint")

}

func assertKeyExists(t *testing.T, bid *openrtb2.Bid, key string, expected bool) {
	t.Helper()
	targets := parseTargets(t, bid)
	if _, ok := targets[key]; ok != expected {
		t.Errorf("Bid %s has wrong key: %s. Expected? %t, Exists? %t", bid.ID, key, expected, ok)
	}
}

// runAuction takes a bunch of mock bids by Bidder and runs an auction. It returns a map of Bids indexed by their ImpID.
// If includeCache is true, the auction will be run with cacheing as well, so the cache targeting keys should exist.
func runTargetingAuction(t *testing.T, mockBids map[openrtb_ext.BidderName][]*openrtb2.Bid, includeCache bool, includeWinners bool, includeBidderKeys bool, isApp bool) map[string]*openrtb2.Bid {
	server := httptest.NewServer(http.HandlerFunc(mockServer))
	defer server.Close()

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	ex := &exchange{
		adapterMap:        buildAdapterMap(mockBids, server.URL, server.Client()),
		me:                &metricsConf.DummyMetricsEngine{},
		cache:             &wellBehavedCache{},
		cacheTime:         time.Duration(0),
		gDPR:              gdpr.AlwaysAllow{},
		currencyConverter: currency.NewRateConverter(&http.Client{}, "", time.Duration(0)),
		gdprDefaultValue:  gdpr.SignalYes,
		categoriesFetcher: categoriesFetcher,
		bidIDGenerator:    &mockBidIDGenerator{false, false},
	}

	imps := buildImps(t, mockBids)

	req := &openrtb2.BidRequest{
		Imp: imps,
		Ext: buildTargetingExt(includeCache, includeWinners, includeBidderKeys),
	}
	if isApp {
		req.App = &openrtb2.App{}
	} else {
		req.Site = &openrtb2.Site{}
	}

	auctionRequest := AuctionRequest{
		BidRequest: req,
		Account:    config.Account{},
		UserSyncs:  &emptyUsersync{},
	}

	debugLog := DebugLog{}
	bidResp, err := ex.HoldAuction(context.Background(), auctionRequest, &debugLog)

	if err != nil {
		t.Fatalf("Unexpected errors running auction: %v", err)
	}
	if len(bidResp.SeatBid) != len(mockBids) {
		t.Fatalf("Unexpected number of SeatBids. Expected %d, got %d", len(mockBids), len(bidResp.SeatBid))
	}

	return buildBidMap(bidResp.SeatBid, len(mockBids))
}

func buildBidderList(bids map[openrtb_ext.BidderName][]*openrtb2.Bid) []openrtb_ext.BidderName {
	bidders := make([]openrtb_ext.BidderName, 0, len(bids))
	for name := range bids {
		bidders = append(bidders, name)
	}
	return bidders
}

func buildAdapterMap(bids map[openrtb_ext.BidderName][]*openrtb2.Bid, mockServerURL string, client *http.Client) map[openrtb_ext.BidderName]adaptedBidder {
	adapterMap := make(map[openrtb_ext.BidderName]adaptedBidder, len(bids))
	for bidder, bids := range bids {
		adapterMap[bidder] = adaptBidder(&mockTargetingBidder{
			mockServerURL: mockServerURL,
			bids:          bids,
		}, client, &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, nil)
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

func buildParams(t *testing.T, mockBids map[openrtb_ext.BidderName][]*openrtb2.Bid) json.RawMessage {
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

func buildImps(t *testing.T, mockBids map[openrtb_ext.BidderName][]*openrtb2.Bid) []openrtb2.Imp {
	impExt := buildParams(t, mockBids)

	var s struct{}
	impIds := make(map[string]struct{}, 2*len(mockBids))
	for _, bidList := range mockBids {
		for _, bid := range bidList {
			impIds[bid.ImpID] = s
		}
	}

	imps := make([]openrtb2.Imp, 0, len(impIds))
	for impId := range impIds {
		imps = append(imps, openrtb2.Imp{
			ID:  impId,
			Ext: impExt,
		})
	}
	return imps
}

func buildBidMap(seatBids []openrtb2.SeatBid, numBids int) map[string]*openrtb2.Bid {
	bids := make(map[string]*openrtb2.Bid, numBids)
	for _, seatBid := range seatBids {
		for i := 0; i < len(seatBid.Bid); i++ {
			bid := seatBid.Bid[i]
			bids[bid.ID] = &bid
		}
	}
	return bids
}

func parseTargets(t *testing.T, bid *openrtb2.Bid) map[string]string {
	t.Helper()
	var parsed openrtb_ext.ExtBid
	if err := json.Unmarshal(bid.Ext, &parsed); err != nil {
		t.Fatalf("Unexpected error parsing targeting params: %v", err)
	}
	return parsed.Prebid.Targeting
}

type mockTargetingBidder struct {
	mockServerURL string
	bids          []*openrtb2.Bid
}

func (m *mockTargetingBidder) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     m.mockServerURL,
		Body:    []byte(""),
		Headers: http.Header{},
	}}, nil
}

func (m *mockTargetingBidder) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

func mockServer(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("{}"))
}

type TargetingTestData struct {
	Description                string
	TargetData                 targetData
	Auction                    auction
	IsApp                      bool
	CategoryMapping            map[string]string
	ExpectedBidTargetsByBidder map[string]map[openrtb_ext.BidderName]map[string]string
}

var bid123 *openrtb2.Bid = &openrtb2.Bid{
	Price: 1.23,
}

var bid111 *openrtb2.Bid = &openrtb2.Bid{
	Price:  1.11,
	DealID: "mydeal",
}
var bid084 *openrtb2.Bid = &openrtb2.Bid{
	Price: 0.84,
}

var TargetingTests []TargetingTestData = []TargetingTestData{
	{
		Description: "Targeting winners only (most basic targeting example)",
		TargetData: targetData{
			priceGranularity: openrtb_ext.PriceGranularityFromString("med"),
			includeWinners:   true,
		},
		Auction: auction{
			winningBidsByBidder: map[string]map[openrtb_ext.BidderName]*pbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {
						bid:     bid123,
						bidType: openrtb_ext.BidTypeBanner,
					},
					openrtb_ext.BidderRubicon: {
						bid:     bid084,
						bidType: openrtb_ext.BidTypeBanner,
					},
				},
			},
		},
		ExpectedBidTargetsByBidder: map[string]map[openrtb_ext.BidderName]map[string]string{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: {
					"hb_bidder": "appnexus",
					"hb_pb":     "1.20",
				},
				openrtb_ext.BidderRubicon: {},
			},
		},
	},
	{
		Description: "Targeting on bidders only",
		TargetData: targetData{
			priceGranularity:  openrtb_ext.PriceGranularityFromString("med"),
			includeBidderKeys: true,
		},
		Auction: auction{
			winningBidsByBidder: map[string]map[openrtb_ext.BidderName]*pbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {
						bid:     bid123,
						bidType: openrtb_ext.BidTypeBanner,
					},
					openrtb_ext.BidderRubicon: {
						bid:     bid084,
						bidType: openrtb_ext.BidTypeBanner,
					},
				},
			},
		},
		ExpectedBidTargetsByBidder: map[string]map[openrtb_ext.BidderName]map[string]string{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: {
					"hb_bidder_appnexus": "appnexus",
					"hb_pb_appnexus":     "1.20",
				},
				openrtb_ext.BidderRubicon: {
					"hb_bidder_rubicon": "rubicon",
					"hb_pb_rubicon":     "0.80",
				},
			},
		},
	},
	{
		Description: "Full basic targeting with hd_format",
		TargetData: targetData{
			priceGranularity:  openrtb_ext.PriceGranularityFromString("med"),
			includeWinners:    true,
			includeBidderKeys: true,
			includeFormat:     true,
		},
		Auction: auction{
			winningBidsByBidder: map[string]map[openrtb_ext.BidderName]*pbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {
						bid:     bid123,
						bidType: openrtb_ext.BidTypeBanner,
					},
					openrtb_ext.BidderRubicon: {
						bid:     bid084,
						bidType: openrtb_ext.BidTypeBanner,
					},
				},
			},
		},
		ExpectedBidTargetsByBidder: map[string]map[openrtb_ext.BidderName]map[string]string{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: {
					"hb_bidder":          "appnexus",
					"hb_bidder_appnexus": "appnexus",
					"hb_pb":              "1.20",
					"hb_pb_appnexus":     "1.20",
					"hb_format":          "banner",
					"hb_format_appnexus": "banner",
				},
				openrtb_ext.BidderRubicon: {
					"hb_bidder_rubicon": "rubicon",
					"hb_pb_rubicon":     "0.80",
					"hb_format_rubicon": "banner",
				},
			},
		},
	},
	{
		Description: "Cache and deal targeting test",
		TargetData: targetData{
			priceGranularity:  openrtb_ext.PriceGranularityFromString("med"),
			includeBidderKeys: true,
			cacheHost:         "cache.prebid.com",
			cachePath:         "cache",
		},
		Auction: auction{
			winningBidsByBidder: map[string]map[openrtb_ext.BidderName]*pbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {
						bid:     bid123,
						bidType: openrtb_ext.BidTypeBanner,
					},
					openrtb_ext.BidderRubicon: {
						bid:     bid111,
						bidType: openrtb_ext.BidTypeBanner,
					},
				},
			},
			cacheIds: map[*openrtb2.Bid]string{
				bid123: "55555",
				bid111: "cacheme",
			},
		},
		ExpectedBidTargetsByBidder: map[string]map[openrtb_ext.BidderName]map[string]string{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: {
					"hb_bidder_appnexus":   "appnexus",
					"hb_pb_appnexus":       "1.20",
					"hb_cache_id_appnexus": "55555",
					"hb_cache_host_appnex": "cache.prebid.com",
					"hb_cache_path_appnex": "cache",
				},
				openrtb_ext.BidderRubicon: {
					"hb_bidder_rubicon":    "rubicon",
					"hb_pb_rubicon":        "1.10",
					"hb_cache_id_rubicon":  "cacheme",
					"hb_deal_rubicon":      "mydeal",
					"hb_cache_host_rubico": "cache.prebid.com",
					"hb_cache_path_rubico": "cache",
				},
			},
		},
	},
}

func TestSetTargeting(t *testing.T) {
	for _, test := range TargetingTests {
		auc := &test.Auction
		// Set rounded prices from the auction data
		auc.setRoundedPrices(test.TargetData.priceGranularity)
		winningBids := make(map[string]*pbsOrtbBid)
		// Set winning bids from the auction data
		for imp, bidsByBidder := range auc.winningBidsByBidder {
			for _, bid := range bidsByBidder {
				if winningBid, ok := winningBids[imp]; ok {
					if winningBid.bid.Price < bid.bid.Price {
						winningBids[imp] = bid
					}
				} else {
					winningBids[imp] = bid
				}
			}
		}
		auc.winningBids = winningBids
		targData := test.TargetData
		targData.setTargeting(auc, test.IsApp, test.CategoryMapping)
		for imp, targetsByBidder := range test.ExpectedBidTargetsByBidder {
			for bidder, expected := range targetsByBidder {
				assert.Equal(t,
					expected,
					auc.winningBidsByBidder[imp][bidder].bidTargets,
					"Test: %s\nTargeting failed for bidder %s on imp %s.",
					test.Description,
					string(bidder),
					imp)
			}
		}
	}

}
