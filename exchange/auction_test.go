package exchange

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currencies"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/prebid_cache_client"

	"github.com/rcrowley/go-metrics"

	"github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
)

func TestCacheVersusTargets(t *testing.T) {
	//Load JSON file
	testInfo, readErr := ioutil.ReadFile("./cachetest/targetedVersusCachedTest.json")
	if readErr != nil {
		fmt.Errorf("Failed to read JSON file ./cachetest/targetedVersusCachedTest.json: %v", readErr)
		//failed test
	}

	//Unmarshal JSON into TargetSpec struct
	var specData TargetSpec
	if unmarshalErr := json.Unmarshal(testInfo, &specData); unmarshalErr != nil {
		fmt.Errorf("Failed to unmarshal JSON from file: %v", unmarshalErr)
		//failed test
	}

	//Put said specData into bid objects
	var bid *pbsOrtbBid
	winningBids := make(map[string]*pbsOrtbBid)
	winningBidsByBidder := make(map[string]map[openrtb_ext.BidderName]*pbsOrtbBid)
	roundedPrices := make(map[*pbsOrtbBid]string)
	bidCategory := make(map[string]string)
	for _, pbsBid := range specData.PbsBids {
		if _, ok := winningBidsByBidder[pbsBid.Bid.ImpID]; !ok {
			winningBidsByBidder[pbsBid.Bid.ImpID] = make(map[openrtb_ext.BidderName]*pbsOrtbBid)
		}
		bid = &pbsOrtbBid{
			bid:     pbsBid.Bid,
			bidType: pbsBid.BidType,
		}
		if _, ok := winningBids[pbsBid.Bid.ImpID]; !ok {
			winningBids[pbsBid.Bid.ImpID] = bid
		}
		winningBidsByBidder[pbsBid.Bid.ImpID][pbsBid.Bidder] = bid
		if len(pbsBid.Bid.Cat) == 1 {
			bidCategory[pbsBid.Bid.ImpID] = pbsBid.Bid.Cat[0]
		}
		roundedPrices[bid] = strconv.FormatFloat(bid.bid.Price, 'f', 2, 64)
	}

	//Get a  mock cache
	cache := &mockCache{}

	server := httptest.NewServer(http.HandlerFunc(dummyServer))
	defer server.Close()

	cfg := &config.Configuration{
		CacheURL: config.Cache{
			ExpectedTimeMillis: 20,
		},
		Adapters: blankAdapterConfig(openrtb_ext.BidderList()),
	}
	knownAdapters := openrtb_ext.BidderList()
	e := NewExchange(server.Client(), nil, cfg, pbsmetrics.NewMetrics(metrics.NewRegistry(), knownAdapters), adapters.ParseBidderInfos("../static/bidder-info", openrtb_ext.BidderList()), gdpr.AlwaysAllow{}, currencies.NewRateConverterDefault()).(*exchange)

	//Get a Context object
	ctx := context.Background()
	auctionCtx, cancel := e.makeAuctionContext(ctx, true)
	defer cancel()

	//Get a clean request    --   //map[openrtb_ext.BidderName]*openrtb.BidRequest
	cleanRequests := make(map[openrtb_ext.BidderName]*openrtb.BidRequest)
	cleanRequests["appnexus"] = &openrtb.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb.Imp{
			{
				ID: "some-request-id",
				Banner: &openrtb.Banner{
					Format: []openrtb.Format{
						openrtb.Format{
							W: 300,
							H: 250,
						},
					},
				},
				Ext: []byte("{\"bidder\":{\"placementId\":10433394}}"),
			},
		},
		Site: &openrtb.Site{
			Page: "prebid.org",
			Ext:  []byte("{\"amp\":0}"),
		},
		Device: &openrtb.Device{
			UA: "curl/7.54.0",
			IP: "127.0.0.1",
		},
		AT:   1,
		TMax: 500,
	}
	liveAdapters := make([]openrtb_ext.BidderName, len(cleanRequests))
	i := 0
	for a := range cleanRequests {
		liveAdapters[i] = a
		i++
	}

	//Get adapterBids and adapterExtra using the getAllBids function (requires the clean requests)
	aliases := map[string]string{
		"test1": "appnexus",
		"test2": "rubicon",
		"test3": "openx",
		"test4": "pubmatic",
	}
	bidAdjustmentFactors := map[string]float64{
		"appnexus": 0.00,
		"rubicon":  0.00,
		"openx":    0.00,
		"pubmatic": 0.00,
		"test1":    0.00,
		"test2":    0.00,
		"test3":    0.00,
		"test4":    0.00,
	}
	//bidAdjustmentFactors := requestExt.Prebid.BidAdjustmentFactors
	labels := pbsmetrics.Labels{
		Source:        pbsmetrics.DemandUnknown,
		RType:         pbsmetrics.ReqTypeLegacy,
		PubID:         "",
		Browser:       pbsmetrics.BrowserOther,
		CookieFlag:    pbsmetrics.CookieFlagUnknown,
		RequestStatus: pbsmetrics.RequestStatusOK,
	}
	bidder := &pbs.PBSBidder{
		BidderCode: "appnexus", //string                 `json:"bidder"`
	}
	blabels := make(map[openrtb_ext.BidderName]*pbsmetrics.AdapterLabels)
	blabels["appnexus"] = &pbsmetrics.AdapterLabels{
		Source:      labels.Source,
		RType:       labels.RType,
		Adapter:     openrtb_ext.BidderMap[bidder.BidderCode],
		PubID:       labels.PubID,
		Browser:     labels.Browser,
		CookieFlag:  labels.CookieFlag,
		AdapterBids: pbsmetrics.AdapterBidPresent,
	}

	conversions := e.currencyConverter.Rates()

	adapterBids, _ := e.getAllBids(auctionCtx, cleanRequests, aliases, bidAdjustmentFactors, blabels, conversions)

	testAuction := &auction{
		winningBids:         winningBids,
		winningBidsByBidder: winningBidsByBidder,
		roundedPrices:       roundedPrices,
	}

	//Instantiate target data and call setTargeting() on it
	targData := &targetData{
		priceGranularity: openrtb_ext.PriceGranularity{
			Precision: 2,
			Ranges: []openrtb_ext.GranularityRange{
				{
					Min:       0,
					Max:       5,
					Increment: 0.05,
				},
				{
					Min:       5,
					Max:       10,
					Increment: 0.1,
				},
				{
					Min:       10,
					Max:       20,
					Increment: 0.5,
				},
			},
		},
		includeWinners:    true,
		includeBidderKeys: true,
		includeCacheBids:  true,
		includeCacheVast:  false,
	}
	//Call doCache()
	if adapterBids != nil {
		errs := testAuction.doCache(ctx, cache, targData, &specData.BidRequest, 60, &specData.DefaultTTLs, bidCategory)
		if len(errs) > 0 {
			var all_errors string
			for _, an_error := range errs {
				all_errors += an_error.Error() + "|"
			}
		}

		targData.setTargeting(testAuction, true, bidCategory)
	}

	//Traverse it like this:
	for _, topBidsPerImp := range testAuction.winningBidsByBidder {
		for _, topBidPerBidder := range topBidsPerImp {
			var i int = 0
			for targetKey, target := range topBidPerBidder.bidTargets {
				//if targetKey != specData.ExpectedTargets[i].targetKey || target != specData.ExpectedTargets[i].target {
				t.Errorf("targetKey != specData.ExpectedTargets[i].targetKey || target != specData.ExpectedTargets[i].target \n [  %s  ] != [  %s  ] || [ %s ] != [ %s ] \n", targetKey, specData.ExpectedTargets[i].targetKey, target, specData.ExpectedTargets[i].target)
				//t.Run("Fail because target is not what expected", func(b *testing.T) {
				//	assert.Equal(b, "+", "-")
				//})
				//}
				fmt.Printf("topBidPerBidder.bidTargets[%s] = %s \n", targetKey, target)
				fmt.Printf("specData.ExpectedTargets[i].targetKey = %s -> specData.ExpectedTargets[i].target = %s \n", specData.ExpectedTargets[i].targetKey, specData.ExpectedTargets[i].target)
				i += 1
			}
		}
	}
	assert.Equal(t, "+", "+")
}

func dummyServer(w http.ResponseWriter, r *http.Request) {
	w.Write(
		[]byte(`{
            "id":"some-request-id",
            "seatbid":
            [
                {
                  "bid":
                  [
                     {
                        "id":"4625436751433509010",
                        "impid":"my-imp-id",
                        "price":0.5,
                        "adm":"\u003cscript type=\"application/javascript\" src=\"http://nym1-ib.adnxs.com/ab?e=wqT_3QKABqAAAwAAAwDWAAUBCM-OiNAFELuV09Pqi86EVRj6t-7QyLin_REqLQkAAAECCOA_EQEHNAAA4D8ZAAAAgOtR4D8hERIAKREJoDDy5vwEOL4HQL4HSAJQ1suTDljhgEhgAGiRQHixhQSAAQGKAQNVU0SSBQbwUpgBrAKgAfoBqAEBsAEAuAECwAEDyAEC0AEA2AEA4AEB8AEAigI6dWYoJ2EnLCA0OTQ0NzIsIDE1MTAwODIzODMpO3VmKCdyJywgMjk2ODExMTAsMh4A8JySAvkBIVR6WGNkQWk2MEljRUVOYkxrdzRZQUNEaGdFZ3dBRGdBUUFSSXZnZFE4dWI4QkZnQVlQX19fXzhQYUFCd0FYZ0JnQUVCaUFFQmtBRUJtQUVCb0FFQnFBRURzQUVBdVFFcGk0aURBQURnUDhFQktZdUlnd0FBNERfSkFTZlJKRUdtbi00XzJRRUFBQUFBQUFEd1AtQUJBUFVCBQ8oSmdDQUtBQ0FMVUMFEARMMAkI8ExNQUNBY2dDQWRBQ0FkZ0NBZUFDQU9nQ0FQZ0NBSUFEQVpBREFKZ0RBYWdEdXRDSEJMb0RDVTVaVFRJNk16STNOdy4umgItITh3aENuZzb8ALg0WUJJSUFRb0FEb0pUbGxOTWpvek1qYzPYAugH4ALH0wHyAhAKBkFEVl9JRBIGNCV1HPICEQoGQ1BHARMcBzE5Nzc5MzMBJwgFQ1AFE_B-ODUxMzU5NIADAYgDAZADAJgDFKADAaoDAMADrALIAwDYAwDgAwDoAwD4AwCABACSBAkvb3BlbnJ0YjKYBACoBACyBAwIABAAGAAgADAAOAC4BADABADIBADSBAlOWU0yOjMyNzfaBAIIAeAEAPAE1suTDogFAZgFAKAF_____wUDXAGqBQ9zb21lLXJlcXVlc3QtaWTABQDJBUmbTPA_0gUJCQAAAAAAAAAA2AUB4AUB\u0026s=61dc0e8770543def5a3a77b4589830d1274b26f1\u0026test=1\u0026pp=${AUCTION_PRICE}\u0026\"\u003e\u003c/script\u003e",
                        "adid":"29681110",
                        "adomain":["appnexus.com"],
                        "iurl":"http://nym1-ib.adnxs.com/cr?id=29681110",
                        "cid":"958",
                        "crid":"29681110",
                        "w":300,
                        "h":250,
                        "ext":
                        {
                           "bidder":
                           {
                                "appnexus":
                                {
                                    "brand_id":1,
                                    "auction_id":6127490747252132539,
                                    "bidder_id":2
                                }
                            }
                        }
                     }
                  ],
                  "seat":"appnexus"
                }
            ],
            "ext":
            {
                "debug":
                {
                    "httpcalls":
                    {
                        "appnexus":
                        [
                             {
                                "uri":"http://ib.adnxs.com/openrtb2",
                                "requestbody":"{\"id\":\"some-request-id\",\"imp\":[{\"id\":\"my-imp-id\",\"banner\":{\"format\":[{\"w\":300,\"h\":250},{\"w\":300,\"h\":600}]},\"ext\":{\"appnexus\":{\"placement_id\":10433394}}}],\"test\":1,\"tmax\":500}",
                                "responsebody":"{\"id\":\"some-request-id\",\"seatbid\":[{\"bid\":[{\"id\":\"4625436751433509010\",\"impid\":\"my-imp-id\",\"price\": 0.500000,\"adid\":\"29681110\",\"adm\":\"\u003cscript type=\\\"application/javascript\\\" src=\\\"http://nym1-ib.adnxs.com/ab?e=wqT_3QKABqAAAwAAAwDWAAUBCM-OiNAFELuV09Pqi86EVRj6t-7QyLin_REqLQkAAAECCOA_EQEHNAAA4D8ZAAAAgOtR4D8hERIAKREJoDDy5vwEOL4HQL4HSAJQ1suTDljhgEhgAGiRQHixhQSAAQGKAQNVU0SSBQbwUpgBrAKgAfoBqAEBsAEAuAECwAEDyAEC0AEA2AEA4AEB8AEAigI6dWYoJ2EnLCA0OTQ0NzIsIDE1MTAwODIzODMpO3VmKCdyJywgMjk2ODExMTAsMh4A8JySAvkBIVR6WGNkQWk2MEljRUVOYkxrdzRZQUNEaGdFZ3dBRGdBUUFSSXZnZFE4dWI4QkZnQVlQX19fXzhQYUFCd0FYZ0JnQUVCaUFFQmtBRUJtQUVCb0FFQnFBRURzQUVBdVFFcGk0aURBQURnUDhFQktZdUlnd0FBNERfSkFTZlJKRUdtbi00XzJRRUFBQUFBQUFEd1AtQUJBUFVCBQ8oSmdDQUtBQ0FMVUMFEARMMAkI8ExNQUNBY2dDQWRBQ0FkZ0NBZUFDQU9nQ0FQZ0NBSUFEQVpBREFKZ0RBYWdEdXRDSEJMb0RDVTVaVFRJNk16STNOdy4umgItITh3aENuZzb8ALg0WUJJSUFRb0FEb0pUbGxOTWpvek1qYzPYAugH4ALH0wHyAhAKBkFEVl9JRBIGNCV1HPICEQoGQ1BHARMcBzE5Nzc5MzMBJwgFQ1AFE_B-ODUxMzU5NIADAYgDAZADAJgDFKADAaoDAMADrALIAwDYAwDgAwDoAwD4AwCABACSBAkvb3BlbnJ0YjKYBACoBACyBAwIABAAGAAgADAAOAC4BADABADIBADSBAlOWU0yOjMyNzfaBAIIAeAEAPAE1suTDogFAZgFAKAF_____wUDXAGqBQ9zb21lLXJlcXVlc3QtaWTABQDJBUmbTPA_0gUJCQAAAAAAAAAA2AUB4AUB\u0026s=61dc0e8770543def5a3a77b4589830d1274b26f1\u0026test=1\u0026pp=${AUCTION_PRICE}\u0026\\\"\u003e\u003c/script\u003e\",\"adomain\":[\"appnexus.com\"],\"iurl\":\"http://nym1-ib.adnxs.com/cr?id=29681110\",\"cid\":\"958\",\"crid\":\"29681110\",\"h\": 250,\"w\": 300,\"ext\":{\"appnexus\":{\"brand_id\": 1,\"auction_id\": 6127490747252132539,\"bidder_id\": 2}}}],\"seat\":\"958\"}],\"bidid\":\"8271358638249766712\",\"cur\":\"USD\"}",
                                "status":200
                             }
                        ]
                    }
                },
                "responsetimemillis":{"appnexus":42}
            }
		}`),
	)
}

func TestMakeVASTGiven(t *testing.T) {
	const expect = `<VAST version="3.0"></VAST>`
	bid := &openrtb.Bid{
		AdM: expect,
	}
	vast := makeVAST(bid)
	assert.Equal(t, expect, vast)
}

func TestMakeVASTNurl(t *testing.T) {
	const url = "http://domain.com/win-notify/1"
	const expect = `<VAST version="3.0"><Ad><Wrapper>` +
		`<AdSystem>prebid.org wrapper</AdSystem>` +
		`<VASTAdTagURI><![CDATA[` + url + `]]></VASTAdTagURI>` +
		`<Impression></Impression><Creatives></Creatives>` +
		`</Wrapper></Ad></VAST>`
	bid := &openrtb.Bid{
		NURL: url,
	}
	vast := makeVAST(bid)
	assert.Equal(t, expect, vast)
}

// TestCacheJSON executes tests for all the *.json files in cachetest.
// customcachekey.json test here verifies custom cache key not used for non-vast video
func TestCacheJSON(t *testing.T) {
	if specFiles, err := ioutil.ReadDir("./cachetest"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./cachetest/" + specFile.Name()
			fileDisplayName := "exchange/cachetest/" + specFile.Name()
			specData, err := loadCacheSpec(fileName)
			if err != nil {
				t.Fatalf("Failed to load contents of file %s: %v", fileDisplayName, err)
			}

			runCacheSpec(t, fileDisplayName, specData, true, false)
		}
	} else {
		t.Fatalf("Failed to read contents of directory exchange/cachetest/: %v", err)
	}
}

// TestCacheJSON executes tests for all the *.json files in customcachekeytest.
// customcachekey.json test here verifies custom cache key is used for vast video
func TestCustomCacheKeyJSON(t *testing.T) {
	if specFiles, err := ioutil.ReadDir("./customcachekeytest"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./customcachekeytest/" + specFile.Name()
			fileDisplayName := "exchange/customcachekeytest/" + specFile.Name()
			specData, err := loadCacheSpec(fileName)
			if err != nil {
				t.Fatalf("Failed to load contents of file %s: %v", fileDisplayName, err)
			}

			runCacheSpec(t, fileDisplayName, specData, false, true)
		}
	} else {
		t.Fatalf("Failed to read contents of directory exchange/customcachekeytest/: %v", err)
	}
}

// LoadCacheSpec reads and parses a file as a test case. If something goes wrong, it returns an error.
func loadCacheSpec(filename string) (*cacheSpec, error) {
	specData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file %s: %v", filename, err)
	}

	var spec cacheSpec
	if err := json.Unmarshal(specData, &spec); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal JSON from file: %v", err)
	}

	return &spec, nil
}

func runCacheSpec(t *testing.T, fileDisplayName string, specData *cacheSpec, bids bool, vast bool) {
	// bid := make([]pbsOrtbBid, 5)
	var bid *pbsOrtbBid
	winningBids := make(map[string]*pbsOrtbBid)
	winningBidsByBidder := make(map[string]map[openrtb_ext.BidderName]*pbsOrtbBid)
	roundedPrices := make(map[*pbsOrtbBid]string)
	bidCategory := make(map[string]string)
	for i, pbsBid := range specData.PbsBids {
		if _, ok := winningBidsByBidder[pbsBid.Bid.ImpID]; !ok {
			winningBidsByBidder[pbsBid.Bid.ImpID] = make(map[openrtb_ext.BidderName]*pbsOrtbBid)
		}
		bid = &pbsOrtbBid{
			bid:     pbsBid.Bid,
			bidType: pbsBid.BidType,
		}
		if _, ok := winningBids[pbsBid.Bid.ImpID]; !ok {
			winningBids[pbsBid.Bid.ImpID] = bid
		}
		winningBidsByBidder[pbsBid.Bid.ImpID][pbsBid.Bidder] = bid
		if len(pbsBid.Bid.Cat) == 1 {
			bidCategory[pbsBid.Bid.ImpID] = pbsBid.Bid.Cat[0]
		}
		roundedPrices[bid] = strconv.FormatFloat(bid.bid.Price, 'f', 2, 64)
		// Marshal the bid for the expected cacheables
		cjson, _ := json.Marshal(bid.bid)
		specData.ExpectedCacheables[i].Data = cjson
	}
	ctx := context.Background()
	cache := &mockCache{}

	testAuction := &auction{
		winningBids:         winningBids,
		winningBidsByBidder: winningBidsByBidder,
		roundedPrices:       roundedPrices,
	}
	//define targetData here
	targData := &targetData{
		priceGranularity: openrtb_ext.PriceGranularity{
			Precision: 2,
			Ranges: []openrtb_ext.GranularityRange{
				{
					Min:       0,
					Max:       5,
					Increment: 0.05,
				},
				{
					Min:       5,
					Max:       10,
					Increment: 0.1,
				},
				{
					Min:       10,
					Max:       20,
					Increment: 0.5,
				},
			},
		},
		includeWinners:    true,
		includeBidderKeys: true,
		includeCacheBids:  bids,
		includeCacheVast:  vast,
	}
	_ = testAuction.doCache(ctx, cache, targData, &specData.BidRequest, 60, &specData.DefaultTTLs, bidCategory)
	found := 0

	for _, cExpected := range specData.ExpectedCacheables {
		for _, cFound := range cache.items {
			// make sure Data section matches exactly
			if !vast {
				// not testing VAST XML for custom cache key test
				eq := jsonpatch.Equal(cExpected.Data, cFound.Data)
				if !eq {
					continue
				}
			}

			// make sure Key value is as expected
			keymatch := false
			if len(cExpected.Key) > 0 {
				// can only verify prefix; remainder is random
				if strings.HasPrefix(cFound.Key, cExpected.Key) {
					keymatch = true
				}
			} else {
				if len(cFound.Key) == 0 {
					// Key is expected to be empty
					keymatch = true
				}
			}
			if !keymatch {
				continue
			}

			// make sure TTLSeconds section matches exactly
			if cExpected.TTLSeconds == cFound.TTLSeconds {
				found++
			}
		}
	}

	if found != len(specData.ExpectedCacheables) {
		fmt.Printf("Expected:\n%v\n\n", specData.ExpectedCacheables)
		fmt.Printf("Found:\n%v\n\n", cache.items)
		t.Errorf("%s:  All expected cacheables not found. Expected %d, found %d.", fileDisplayName, len(specData.ExpectedCacheables), found)
	}

	// bid := make([]pbsOrtbBid, 5)
}

type cacheSpec struct {
	BidRequest         openrtb.BidRequest              `json:"bidRequest"`
	PbsBids            []pbsBid                        `json:"pbsBids"`
	ExpectedCacheables []prebid_cache_client.Cacheable `json:"expectedCacheables"`
	DefaultTTLs        config.DefaultTTLs              `json:"defaultTTLs"`
}

type TargetSpec struct {
	BidRequest openrtb.BidRequest `json:"bidRequest"`
	PbsBids    []pbsBid           `json:"pbsBids"`
	//ExpectedTargets []prebid_cache_client.Cacheable `json:"expectedCacheables"`
	//ExpectedTargets map[string]string               `json:"expectedTargets"`
	ExpectedTargets []mappedBidTargets `json:"expectedTargets"`
	DefaultTTLs     config.DefaultTTLs `json:"defaultTTLs"`
}
type mappedBidTargets struct {
	targetKey string `json:"targetKey"`
	target    string `json:"target"`
}
type pbsBid struct {
	Bid     *openrtb.Bid           `json:"bid"`
	BidType openrtb_ext.BidType    `json:"bidType"`
	Bidder  openrtb_ext.BidderName `json:"bidder"`
}

type mockCache struct {
	items []prebid_cache_client.Cacheable
}

func (c *mockCache) PutJson(ctx context.Context, values []prebid_cache_client.Cacheable) ([]string, []error) {
	c.items = values
	return []string{"", "", "", "", ""}, nil
}
