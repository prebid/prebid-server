package exchange

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	metricsConfig "github.com/prebid/prebid-server/v3/metrics/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"

	"github.com/prebid/openrtb/v20/openrtb2"
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
	bids := runTargetingAuction(t, mockBids, true, true, true, false, "", "")

	// Make sure that the cache keys exist on the bids where they're expected to
	assertKeyExists(t, bids["winning-bid"], DefaultKeyPrefix+string(openrtb_ext.CacheKey), true)
	assertKeyExists(t, bids["winning-bid"], openrtb_ext.CacheKey.BidderKey(DefaultKeyPrefix, openrtb_ext.BidderAppnexus, MaxKeyLength), true)

	assertKeyExists(t, bids["contending-bid"], DefaultKeyPrefix+string(openrtb_ext.CacheKey), false)
	assertKeyExists(t, bids["contending-bid"], openrtb_ext.CacheKey.BidderKey(DefaultKeyPrefix, openrtb_ext.BidderRubicon, MaxKeyLength), true)

	assertKeyExists(t, bids["losing-bid"], DefaultKeyPrefix+string(openrtb_ext.CacheKey), false)
	assertKeyExists(t, bids["losing-bid"], openrtb_ext.CacheKey.BidderKey(DefaultKeyPrefix, openrtb_ext.BidderAppnexus, MaxKeyLength), false)

	//assert hb_cache_host was included
	assert.Contains(t, string(bids["winning-bid"].Ext), DefaultKeyPrefix+string(openrtb_ext.CacheHostKey))
	assert.Contains(t, string(bids["winning-bid"].Ext), "www.pbcserver.com")

	//assert hb_cache_path was included
	assert.Contains(t, string(bids["winning-bid"].Ext), DefaultKeyPrefix+string(openrtb_ext.CachePathKey))
	assert.Contains(t, string(bids["winning-bid"].Ext), "/pbcache/endpoint")
}

func TestTargetingCacheRequestPrefix(t *testing.T) {
	reqPrefix := "req"
	bids := runTargetingAuction(t, mockBids, true, true, true, false, reqPrefix, "acc")

	// Make sure that the cache keys exist on the bids where they're expected to
	assertKeyExists(t, bids["winning-bid"], reqPrefix+string(openrtb_ext.CacheKey), true)
	assertKeyExists(t, bids["winning-bid"], openrtb_ext.CacheKey.BidderKey(reqPrefix, openrtb_ext.BidderAppnexus, MaxKeyLength), true)

	assertKeyExists(t, bids["contending-bid"], reqPrefix+string(openrtb_ext.CacheKey), false)
	assertKeyExists(t, bids["contending-bid"], openrtb_ext.CacheKey.BidderKey(reqPrefix, openrtb_ext.BidderRubicon, MaxKeyLength), true)

	assertKeyExists(t, bids["losing-bid"], reqPrefix+string(openrtb_ext.CacheKey), false)
	assertKeyExists(t, bids["losing-bid"], openrtb_ext.CacheKey.BidderKey(reqPrefix, openrtb_ext.BidderAppnexus, MaxKeyLength), false)

	//assert hb_cache_host was included
	assert.Contains(t, string(bids["winning-bid"].Ext), reqPrefix+string(openrtb_ext.CacheHostKey))
	assert.Contains(t, string(bids["winning-bid"].Ext), "www.pbcserver.com")

	//assert hb_cache_path was included
	assert.Contains(t, string(bids["winning-bid"].Ext), reqPrefix+string(openrtb_ext.CachePathKey))
	assert.Contains(t, string(bids["winning-bid"].Ext), "/pbcache/endpoint")
}

func TestTargetingCacheAccountPrefix(t *testing.T) {
	accPrefix := "acc"
	bids := runTargetingAuction(t, mockBids, true, true, true, false, "", accPrefix)

	// Make sure that the cache keys exist on the bids where they're expected to
	assertKeyExists(t, bids["winning-bid"], accPrefix+string(openrtb_ext.CacheKey), true)
	assertKeyExists(t, bids["winning-bid"], openrtb_ext.CacheKey.BidderKey(accPrefix, openrtb_ext.BidderAppnexus, MaxKeyLength), true)

	assertKeyExists(t, bids["contending-bid"], accPrefix+string(openrtb_ext.CacheKey), false)
	assertKeyExists(t, bids["contending-bid"], openrtb_ext.CacheKey.BidderKey(accPrefix, openrtb_ext.BidderRubicon, MaxKeyLength), true)

	assertKeyExists(t, bids["losing-bid"], accPrefix+string(openrtb_ext.CacheKey), false)
	assertKeyExists(t, bids["losing-bid"], openrtb_ext.CacheKey.BidderKey(accPrefix, openrtb_ext.BidderAppnexus, MaxKeyLength), false)

	//assert hb_cache_host was included
	assert.Contains(t, string(bids["winning-bid"].Ext), accPrefix+string(openrtb_ext.CacheHostKey))
	assert.Contains(t, string(bids["winning-bid"].Ext), "www.pbcserver.com")

	//assert hb_cache_path was included
	assert.Contains(t, string(bids["winning-bid"].Ext), accPrefix+string(openrtb_ext.CachePathKey))
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
func runTargetingAuction(t *testing.T, mockBids map[openrtb_ext.BidderName][]*openrtb2.Bid, includeCache bool, includeWinners bool, includeBidderKeys bool, isApp bool, requestPrefix string, accountPrefix string) map[string]*openrtb2.Bid {
	server := httptest.NewServer(http.HandlerFunc(mockServer))
	defer server.Close()

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	gdprPermsBuilder := fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder

	ex := &exchange{
		adapterMap:        buildAdapterMap(mockBids, server.URL, server.Client()),
		me:                &metricsConfig.NilMetricsEngine{},
		cache:             &wellBehavedCache{},
		cacheTime:         time.Duration(0),
		gdprPermsBuilder:  gdprPermsBuilder,
		currencyConverter: currency.NewRateConverter(&http.Client{}, "", time.Duration(0)),
		gdprDefaultValue:  gdpr.SignalYes,
		categoriesFetcher: categoriesFetcher,
		bidIDGenerator:    &fakeBidIDGenerator{GenerateBidID: false, ReturnError: false},
	}
	ex.requestSplitter = requestSplitter{
		me:               ex.me,
		gdprPermsBuilder: ex.gdprPermsBuilder,
	}

	imps := buildImps(t, mockBids)

	req := &openrtb2.BidRequest{
		Imp: imps,
		Ext: buildTargetingExt(includeCache, includeWinners, includeBidderKeys, requestPrefix),
	}
	if isApp {
		req.App = &openrtb2.App{}
	} else {
		req.Site = &openrtb2.Site{}
	}

	auctionRequest := &AuctionRequest{
		BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: req},
		Account: config.Account{
			TargetingPrefix: accountPrefix,
		},
		UserSyncs:    &emptyUsersync{},
		HookExecutor: &hookexecution.EmptyHookExecutor{},
		TCF2Config:   gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
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

func buildAdapterMap(bids map[openrtb_ext.BidderName][]*openrtb2.Bid, mockServerURL string, client *http.Client) map[openrtb_ext.BidderName]AdaptedBidder {
	adapterMap := make(map[openrtb_ext.BidderName]AdaptedBidder, len(bids))
	for bidder, bids := range bids {
		adapterMap[bidder] = AdaptBidder(&mockTargetingBidder{
			mockServerURL: mockServerURL,
			bids:          bids,
		}, client, &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, nil, "")
	}
	return adapterMap
}

func buildTargetingExt(includeCache bool, includeWinners bool, includeBidderKeys bool, prefix string) json.RawMessage {
	var targeting string
	if includeWinners && includeBidderKeys {
		targeting = `{"prefix":"` + prefix + `","pricegranularity":{"precision":2,"ranges": [{"min": 0,"max": 20,"increment": 0.1}]},"includewinners": true, "includebidderkeys": true}`
	} else if !includeWinners && includeBidderKeys {
		targeting = `{"prefix":"` + prefix + `","precision":2,"includewinners": false}`
	} else if includeWinners && !includeBidderKeys {
		targeting = `{"prefix":"` + prefix + `","precision":2,"includebidderkeys": false}`
	} else {
		targeting = `{"prefix":"` + prefix + `","precision":2,"includewinners": false, "includebidderkeys": false}`
	}

	if includeCache {
		return json.RawMessage(`{"prebid":{"targeting":` + targeting + `,"cache":{"bids":{}}}}`)
	}

	return json.RawMessage(`{"prebid":{"targeting":` + targeting + `}}`)
}

func buildParams(t *testing.T, mockBids map[openrtb_ext.BidderName][]*openrtb2.Bid) json.RawMessage {
	params := make(map[string]interface{})
	paramsPrebid := make(map[string]interface{})
	paramsPrebidBidders := make(map[string]json.RawMessage)

	for bidder := range mockBids {
		paramsPrebidBidders[string(bidder)] = json.RawMessage(`{"whatever":true}`)
	}

	paramsPrebid["bidder"] = paramsPrebidBidders
	params["prebid"] = paramsPrebid
	ext, err := jsonutil.Marshal(params)
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
	if err := jsonutil.UnmarshalValid(bid.Ext, &parsed); err != nil {
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
	Description        string
	TargetData         targetData
	Auction            auction
	IsApp              bool
	CategoryMapping    map[string]string
	ExpectedPbsBids    map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid
	TruncateTargetAttr *int
	MultiBidMap        map[string]openrtb_ext.ExtMultiBid
	DefaultBidLimit    int
}

type ExpectedPbsBid struct {
	BidTargets       map[string]string
	TargetBidderCode string
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

var bid1p001 *openrtb2.Bid = &openrtb2.Bid{
	Price: 0.01,
}

var bid1p077 *openrtb2.Bid = &openrtb2.Bid{
	Price: 0.77,
}

var bid1p120 *openrtb2.Bid = &openrtb2.Bid{
	Price: 1.20,
}

var bid2p123 *openrtb2.Bid = &openrtb2.Bid{
	Price: 1.23,
}

var bid2p144 *openrtb2.Bid = &openrtb2.Bid{
	Price: 1.44,
}

var bid2p155 *openrtb2.Bid = &openrtb2.Bid{
	Price: 1.55,
}

var bid2p166 *openrtb2.Bid = &openrtb2.Bid{
	Price: 1.66,
}

var bid175 *openrtb2.Bid = &openrtb2.Bid{
	Price:  1.75,
	DealID: "mydeal2",
}

var (
	truncateTargetAttrValue10       int = 10
	truncateTargetAttrValue5        int = 5
	truncateTargetAttrValue25       int = 25
	truncateTargetAttrValueNegative int = -1
)

func lookupPriceGranularity(v string) openrtb_ext.PriceGranularity {
	priceGranularity, _ := openrtb_ext.NewPriceGranularityFromLegacyID(v)
	return priceGranularity
}

var TargetingTests []TargetingTestData = []TargetingTestData{
	{
		Description: "Targeting winners only (most basic targeting example)",
		TargetData: targetData{
			priceGranularity: lookupPriceGranularity("med"),
			includeWinners:   true,
			prefix:           DefaultKeyPrefix,
		},
		Auction: auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {{
						Bid:     bid123,
						BidType: openrtb_ext.BidTypeBanner,
					}},
					openrtb_ext.BidderRubicon: {{
						Bid:     bid084,
						BidType: openrtb_ext.BidTypeBanner,
					}},
				},
			},
		},
		ExpectedPbsBids: map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder": "appnexus",
							"hb_pb":     "1.20",
						},
					},
				},
				openrtb_ext.BidderRubicon: []ExpectedPbsBid{},
			},
		},
		TruncateTargetAttr: nil,
	},
	{
		Description: "Targeting on bidders only",
		TargetData: targetData{
			priceGranularity:  lookupPriceGranularity("med"),
			includeBidderKeys: true,
			prefix:            DefaultKeyPrefix,
		},
		Auction: auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {{
						Bid:     bid123,
						BidType: openrtb_ext.BidTypeBanner,
					}},
					openrtb_ext.BidderRubicon: {{
						Bid:     bid084,
						BidType: openrtb_ext.BidTypeBanner,
					}},
				},
			},
		},
		ExpectedPbsBids: map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder_appnexus": "appnexus",
							"hb_pb_appnexus":     "1.20",
						},
					},
				},
				openrtb_ext.BidderRubicon: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder_rubicon": "rubicon",
							"hb_pb_rubicon":     "0.80",
						},
					},
				},
			},
		},
		TruncateTargetAttr: nil,
	},
	{
		Description: "Targeting with alwaysIncludeDeals",
		TargetData: targetData{
			priceGranularity:   lookupPriceGranularity("med"),
			alwaysIncludeDeals: true,
			prefix:             DefaultKeyPrefix,
		},
		Auction: auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {{
						Bid:     bid111,
						BidType: openrtb_ext.BidTypeBanner,
					}},
					openrtb_ext.BidderRubicon: {{
						Bid:     bid175,
						BidType: openrtb_ext.BidTypeBanner,
					}},
					openrtb_ext.BidderPubmatic: {{
						Bid:     bid123,
						BidType: openrtb_ext.BidTypeBanner,
					}},
				},
			},
		},
		ExpectedPbsBids: map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder_appnexus": "appnexus",
							"hb_pb_appnexus":     "1.10",
							"hb_deal_appnexus":   "mydeal",
						},
					},
				},
				openrtb_ext.BidderRubicon: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder_rubicon": "rubicon",
							"hb_pb_rubicon":     "1.70",
							"hb_deal_rubicon":   "mydeal2",
						},
					},
				},
			},
		},
		TruncateTargetAttr: nil,
	},
	{
		Description: "Full basic targeting with hd_format",
		TargetData: targetData{
			priceGranularity:  lookupPriceGranularity("med"),
			includeWinners:    true,
			includeBidderKeys: true,
			includeFormat:     true,
			prefix:            DefaultKeyPrefix,
		},
		Auction: auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {{
						Bid:     bid123,
						BidType: openrtb_ext.BidTypeBanner,
					}},
					openrtb_ext.BidderRubicon: {{
						Bid:     bid084,
						BidType: openrtb_ext.BidTypeBanner,
					}},
				},
			},
		},
		ExpectedPbsBids: map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder":          "appnexus",
							"hb_bidder_appnexus": "appnexus",
							"hb_pb":              "1.20",
							"hb_pb_appnexus":     "1.20",
							"hb_format":          "banner",
							"hb_format_appnexus": "banner",
						},
					},
				},
				openrtb_ext.BidderRubicon: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder_rubicon": "rubicon",
							"hb_pb_rubicon":     "0.80",
							"hb_format_rubicon": "banner",
						},
					},
				},
			},
		},
		TruncateTargetAttr: nil,
	},
	{
		Description: "Cache and deal targeting test",
		TargetData: targetData{
			priceGranularity:  lookupPriceGranularity("med"),
			includeBidderKeys: true,
			cacheHost:         "cache.prebid.com",
			cachePath:         "cache",
			prefix:            DefaultKeyPrefix,
		},
		Auction: auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {{
						Bid:     bid123,
						BidType: openrtb_ext.BidTypeBanner,
					}},
					openrtb_ext.BidderRubicon: {{
						Bid:     bid111,
						BidType: openrtb_ext.BidTypeBanner,
					}},
				},
			},
			cacheIds: map[*openrtb2.Bid]string{
				bid123: "55555",
				bid111: "cacheme",
			},
		},
		ExpectedPbsBids: map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder_appnexus":   "appnexus",
							"hb_pb_appnexus":       "1.20",
							"hb_cache_id_appnexus": "55555",
							"hb_cache_host_appnex": "cache.prebid.com",
							"hb_cache_path_appnex": "cache",
						},
					},
				},
				openrtb_ext.BidderRubicon: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
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
		},
		TruncateTargetAttr: nil,
	},
	{
		Description: "Cache and deal targeting test custom prefix",
		TargetData: targetData{
			priceGranularity:  lookupPriceGranularity("med"),
			includeBidderKeys: true,
			cacheHost:         "cache.prebid.com",
			cachePath:         "cache",
			prefix:            "prefix",
		},
		Auction: auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {{
						Bid:     bid123,
						BidType: openrtb_ext.BidTypeBanner,
					}},
					openrtb_ext.BidderRubicon: {{
						Bid:     bid111,
						BidType: openrtb_ext.BidTypeBanner,
					}},
				},
			},
			cacheIds: map[*openrtb2.Bid]string{
				bid123: "55555",
				bid111: "cacheme",
			},
		},
		ExpectedPbsBids: map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"prefix_bidder_appnex": "appnexus",
							"prefix_pb_appnexus":   "1.20",
							"prefix_cache_id_appn": "55555",
							"prefix_cache_host_ap": "cache.prebid.com",
							"prefix_cache_path_ap": "cache",
						},
					},
				},
				openrtb_ext.BidderRubicon: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"prefix_bidder_rubico": "rubicon",
							"prefix_pb_rubicon":    "1.10",
							"prefix_cache_id_rubi": "cacheme",
							"prefix_deal_rubicon":  "mydeal",
							"prefix_cache_host_ru": "cache.prebid.com",
							"prefix_cache_path_ru": "cache",
						},
					},
				},
			},
		},
		TruncateTargetAttr: nil,
	},
	{
		Description: "bidder with no dealID should not have deal targeting",
		TargetData: targetData{
			priceGranularity:  lookupPriceGranularity("med"),
			includeBidderKeys: true,
			prefix:            DefaultKeyPrefix,
		},
		Auction: auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {{
						Bid:     bid123,
						BidType: openrtb_ext.BidTypeBanner,
					}},
				},
			},
		},
		ExpectedPbsBids: map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder_appnexus": "appnexus",
							"hb_pb_appnexus":     "1.20",
						},
					},
				},
			},
		},
		TruncateTargetAttr: nil,
	},
	{
		Description: "Truncate Targeting Attribute value is given and is less than const MaxKeyLength",
		TargetData: targetData{
			priceGranularity:  lookupPriceGranularity("med"),
			includeBidderKeys: true,
			prefix:            DefaultKeyPrefix,
		},
		Auction: auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {{
						Bid:     bid123,
						BidType: openrtb_ext.BidTypeBanner,
					}},
					openrtb_ext.BidderRubicon: {{
						Bid:     bid084,
						BidType: openrtb_ext.BidTypeBanner,
					}},
				},
			},
		},
		ExpectedPbsBids: map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder_": "appnexus",
							"hb_pb_appn": "1.20",
						},
					},
				},
				openrtb_ext.BidderRubicon: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder_": "rubicon",
							"hb_pb_rubi": "0.80",
						},
					},
				},
			},
		},
		TruncateTargetAttr: &truncateTargetAttrValue10,
	},
	{
		Description: "Truncate Targeting Attribute value is given and is greater than const MaxKeyLength",
		TargetData: targetData{
			priceGranularity:  lookupPriceGranularity("med"),
			includeBidderKeys: true,
			prefix:            DefaultKeyPrefix,
		},
		Auction: auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {{
						Bid:     bid123,
						BidType: openrtb_ext.BidTypeBanner,
					}},
					openrtb_ext.BidderRubicon: {{
						Bid:     bid084,
						BidType: openrtb_ext.BidTypeBanner,
					}},
				},
			},
		},
		ExpectedPbsBids: map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder_appnexus": "appnexus",
							"hb_pb_appnexus":     "1.20",
						},
					},
				},
				openrtb_ext.BidderRubicon: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder_rubicon": "rubicon",
							"hb_pb_rubicon":     "0.80",
						},
					},
				},
			},
		},
		TruncateTargetAttr: &truncateTargetAttrValue25,
	},
	{
		Description: "Truncate Targeting Attribute value is given and is negative",
		TargetData: targetData{
			priceGranularity:  lookupPriceGranularity("med"),
			includeBidderKeys: true,
			prefix:            DefaultKeyPrefix,
		},
		Auction: auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {{
						Bid:     bid123,
						BidType: openrtb_ext.BidTypeBanner,
					}},
					openrtb_ext.BidderRubicon: {{
						Bid:     bid084,
						BidType: openrtb_ext.BidTypeBanner,
					}},
				},
			},
		},
		ExpectedPbsBids: map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder_appnexus": "appnexus",
							"hb_pb_appnexus":     "1.20",
						},
					},
				},
				openrtb_ext.BidderRubicon: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder_rubicon": "rubicon",
							"hb_pb_rubicon":     "0.80",
						},
					},
				},
			},
		},
		TruncateTargetAttr: &truncateTargetAttrValueNegative,
	},
	{
		Description: "Check that key gets truncated properly when value is smaller than key",
		TargetData: targetData{
			priceGranularity: lookupPriceGranularity("med"),
			includeWinners:   true,
			prefix:           DefaultKeyPrefix,
		},
		Auction: auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {{
						Bid:     bid123,
						BidType: openrtb_ext.BidTypeBanner,
					}},
					openrtb_ext.BidderRubicon: {{
						Bid:     bid084,
						BidType: openrtb_ext.BidTypeBanner,
					}},
				},
			},
		},
		ExpectedPbsBids: map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bi": "appnexus",
							"hb_pb": "1.20",
						},
					},
				},
				openrtb_ext.BidderRubicon: []ExpectedPbsBid{},
			},
		},
		TruncateTargetAttr: &truncateTargetAttrValue5,
	},
	{
		Description: "Check that key gets truncated properly when value is greater than key",
		TargetData: targetData{
			priceGranularity: lookupPriceGranularity("med"),
			includeWinners:   true,
			prefix:           DefaultKeyPrefix,
		},
		Auction: auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {{
						Bid:     bid123,
						BidType: openrtb_ext.BidTypeBanner,
					}},
					openrtb_ext.BidderRubicon: {{
						Bid:     bid084,
						BidType: openrtb_ext.BidTypeBanner,
					}},
				},
			},
		},
		ExpectedPbsBids: map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder": "appnexus",
							"hb_pb":     "1.20",
						},
					},
				},
				openrtb_ext.BidderRubicon: []ExpectedPbsBid{},
			},
		},
		TruncateTargetAttr: &truncateTargetAttrValue25,
	},
	{
		Description: "Check that key gets truncated properly when value is negative",
		TargetData: targetData{
			priceGranularity: lookupPriceGranularity("med"),
			includeWinners:   true,
			prefix:           DefaultKeyPrefix,
		},
		Auction: auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {{
						Bid:     bid123,
						BidType: openrtb_ext.BidTypeBanner,
					}},
					openrtb_ext.BidderRubicon: {{
						Bid:     bid084,
						BidType: openrtb_ext.BidTypeBanner,
					}},
				},
			},
		},
		ExpectedPbsBids: map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder": "appnexus",
							"hb_pb":     "1.20",
						},
					},
				},
				openrtb_ext.BidderRubicon: []ExpectedPbsBid{},
			},
		},
		TruncateTargetAttr: &truncateTargetAttrValueNegative,
	},
	{
		Description: "Full basic targeting with multibid",
		TargetData: targetData{
			priceGranularity:  lookupPriceGranularity("med"),
			includeWinners:    true,
			includeBidderKeys: true,
			includeFormat:     true,
			prefix:            DefaultKeyPrefix,
		},
		Auction: auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"ImpId-1": {
					openrtb_ext.BidderAppnexus: {
						{
							Bid:     bid1p120,
							BidType: openrtb_ext.BidTypeBanner,
						},
						{
							Bid:     bid1p077,
							BidType: openrtb_ext.BidTypeBanner,
						},
						{
							Bid:     bid1p001,
							BidType: openrtb_ext.BidTypeBanner,
						},
					},
					openrtb_ext.BidderRubicon: {
						{
							Bid:     bid123,
							BidType: openrtb_ext.BidTypeBanner,
						},
						{
							Bid:     bid111,
							BidType: openrtb_ext.BidTypeBanner,
						},
						{
							Bid:     bid084,
							BidType: openrtb_ext.BidTypeBanner,
						},
					},
				},
				"ImpId-2": {
					openrtb_ext.BidderPubmatic: {
						{
							Bid:     bid2p166,
							BidType: openrtb_ext.BidTypeBanner,
						},
						{
							Bid:     bid2p155,
							BidType: openrtb_ext.BidTypeBanner,
						},
						{
							Bid:     bid2p144,
							BidType: openrtb_ext.BidTypeBanner,
						},
						{
							Bid:     bid2p123,
							BidType: openrtb_ext.BidTypeBanner,
						},
					},
				},
			},
		},
		ExpectedPbsBids: map[string]map[openrtb_ext.BidderName][]ExpectedPbsBid{
			"ImpId-1": {
				openrtb_ext.BidderAppnexus: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder_appnexus": "appnexus",
							"hb_pb_appnexus":     "1.10",
							"hb_format_appnexus": "banner",
						},
						TargetBidderCode: "appnexus",
					},
					{},
					{},
				},
				openrtb_ext.BidderRubicon: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder":         "rubicon",
							"hb_bidder_rubicon": "rubicon",
							"hb_pb":             "1.20",
							"hb_pb_rubicon":     "1.20",
							"hb_format":         "banner",
							"hb_format_rubicon": "banner",
						},
					},
					{},
					{},
				},
			},
			"ImpId-2": {
				openrtb_ext.BidderPubmatic: []ExpectedPbsBid{
					{
						BidTargets: map[string]string{
							"hb_bidder":          "pubmatic",
							"hb_bidder_pubmatic": "pubmatic",
							"hb_pb":              "1.60",
							"hb_pb_pubmatic":     "1.60",
							"hb_format":          "banner",
							"hb_format_pubmatic": "banner",
						},
						TargetBidderCode: "pubmatic",
					},
					{
						BidTargets: map[string]string{
							"hb_bidder_pm2": "pm2",
							"hb_pb_pm2":     "1.50",
							"hb_format_pm2": "banner",
						},
						TargetBidderCode: "pm2",
					},
					{
						BidTargets: map[string]string{
							"hb_bidder_pm3": "pm3",
							"hb_pb_pm3":     "1.40",
							"hb_format_pm3": "banner",
						},
						TargetBidderCode: "pm3",
					},
					{},
				},
			},
		},
		TruncateTargetAttr: nil,
		MultiBidMap: map[string]openrtb_ext.ExtMultiBid{
			string(openrtb_ext.BidderPubmatic): {
				MaxBids:                ptrutil.ToPtr(3),
				TargetBidderCodePrefix: "pm",
			},
			string(openrtb_ext.BidderAppnexus): {
				MaxBids: ptrutil.ToPtr(2),
			},
		},
	},
}

func TestSetTargeting(t *testing.T) {
	for _, test := range TargetingTests {
		auc := &test.Auction
		// Set rounded prices from the auction data
		auc.setRoundedPrices(test.TargetData)
		winningBids := make(map[string]*entities.PbsOrtbBid)
		// Set winning bids from the auction data
		for imp, bidsByBidder := range auc.allBidsByBidder {
			for _, bids := range bidsByBidder {
				for _, bid := range bids {
					if winningBid, ok := winningBids[imp]; ok {
						if winningBid.Bid.Price < bid.Bid.Price {
							winningBids[imp] = bid
						}
					} else {
						winningBids[imp] = bid
					}
				}
			}
		}
		auc.winningBids = winningBids
		targData := test.TargetData
		targData.setTargeting(auc, test.IsApp, test.CategoryMapping, test.TruncateTargetAttr, test.MultiBidMap)
		for imp, targetsByBidder := range test.ExpectedPbsBids {
			for bidder, expectedTargets := range targetsByBidder {
				for i, expected := range expectedTargets {
					assert.Equal(t,
						expected.BidTargets,
						auc.allBidsByBidder[imp][bidder][i].BidTargets,
						"Test: %s\nTargeting failed for bidder %s on imp %s.",
						test.Description,
						string(bidder),
						imp)
					assert.Equal(t, expected.TargetBidderCode, auc.allBidsByBidder[imp][bidder][i].TargetBidderCode)
				}
			}
		}
	}
}
