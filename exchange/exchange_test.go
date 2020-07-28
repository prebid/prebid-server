package exchange

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currencies"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	metricsConf "github.com/prebid/prebid-server/pbsmetrics/config"
	pbc "github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/backends/file_fetcher"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
)

func TestNewExchange(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	knownAdapters := openrtb_ext.BidderList()

	cfg := &config.Configuration{
		CacheURL: config.Cache{
			ExpectedTimeMillis: 20,
		},
		Adapters: blankAdapterConfig(openrtb_ext.BidderList()),
	}

	e := NewExchange(server.Client(), nil, cfg, pbsmetrics.NewMetrics(metrics.NewRegistry(), knownAdapters, config.DisabledMetrics{}), adapters.ParseBidderInfos(cfg.Adapters, "../static/bidder-info", openrtb_ext.BidderList()), gdpr.AlwaysAllow{}, currencies.NewRateConverterDefault()).(*exchange)
	for _, bidderName := range knownAdapters {
		if _, ok := e.adapterMap[bidderName]; !ok {
			t.Errorf("NewExchange produced an Exchange without bidder %s", bidderName)
		}
	}
	if e.cacheTime != time.Duration(cfg.CacheURL.ExpectedTimeMillis)*time.Millisecond {
		t.Errorf("Bad cacheTime. Expected 20 ms, got %s", e.cacheTime.String())
	}
}

// The objective is to get to execute e.buildBidResponse(ctx.Background(), liveA... ) (*openrtb.BidResponse, error)
// and check whether the returned request successfully prints any '&' characters as it should
// To do so, we:
// 	1) Write the endpoint adapter URL with an '&' character into a new config,Configuration struct
// 	   as specified in https://github.com/prebid/prebid-server/issues/465
// 	2) Initialize a new exchange with said configuration
// 	3) Build all the parameters e.buildBidResponse(ctx.Background(), liveA... ) needs including the
// 	   sample request as specified in https://github.com/prebid/prebid-server/issues/465
// 	4) Build a BidResponse struct using exchange.buildBidResponse(ctx.Background(), liveA... )
// 	5) Assert we have no '&' characters in the response that exchange.buildBidResponse returns
func TestCharacterEscape(t *testing.T) {
	/* 1) Adapter with a '& char in its endpoint property 		*/
	/*    https://github.com/prebid/prebid-server/issues/465	*/
	cfg := &config.Configuration{
		Adapters: make(map[string]config.Adapter, 1),
	}
	cfg.Adapters["appnexus"] = config.Adapter{
		Endpoint: "http://ib.adnxs.com/openrtb2?query1&query2", //Note the '&' character in there
	}

	/* 	2) Init new exchange with said configuration			*/
	//Other parameters also needed to create exchange
	handlerNoBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	server := httptest.NewServer(http.HandlerFunc(handlerNoBidServer))
	defer server.Close()

	e := NewExchange(server.Client(), nil, cfg, pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{}), adapters.ParseBidderInfos(cfg.Adapters, "../static/bidder-info", openrtb_ext.BidderList()), gdpr.AlwaysAllow{}, currencies.NewRateConverterDefault()).(*exchange)

	/* 	3) Build all the parameters e.buildBidResponse(ctx.Background(), liveA... ) needs */
	//liveAdapters []openrtb_ext.BidderName,
	liveAdapters := make([]openrtb_ext.BidderName, 1)
	liveAdapters[0] = "appnexus"

	//adapterBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid,
	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid, 1)
	adapterBids["appnexus"] = &pbsOrtbSeatBid{currency: "USD"}

	//An openrtb.BidRequest struct as specified in https://github.com/prebid/prebid-server/issues/465
	bidRequest := &openrtb.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb.Banner{Format: []openrtb.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext:    json.RawMessage(`{"appnexus": {"placementId": 1}}`),
		}},
		Site:   &openrtb.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
		Device: &openrtb.Device{UA: "curl/7.54.0", IP: "::1"},
		AT:     1,
		TMax:   500,
		Ext:    json.RawMessage(`{"id": "some-request-id","site": {"page": "prebid.org"},"imp": [{"id": "some-impression-id","banner": {"format": [{"w": 300,"h": 250},{"w": 300,"h": 600}]},"ext": {"appnexus": {"placementId": 1}}}],"tmax": 500}`),
	}

	//resolvedRequest json.RawMessage
	resolvedRequest := json.RawMessage(`{"id": "some-request-id","site": {"page": "prebid.org"},"imp": [{"id": "some-impression-id","banner": {"format": [{"w": 300,"h": 250},{"w": 300,"h": 600}]},"ext": {"appnexus": {"placementId": 1}}}],"tmax": 500}`)

	//adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra,
	adapterExtra := make(map[openrtb_ext.BidderName]*seatResponseExtra, 1)
	adapterExtra["appnexus"] = &seatResponseExtra{
		ResponseTimeMillis: 5,
		Errors:             []openrtb_ext.ExtBidderError{{Code: 999, Message: "Post ib.adnxs.com/openrtb2?query1&query2: unsupported protocol scheme \"\""}},
	}

	//errList []error
	var errList []error

	/* 	4) Build bid response 									*/
	bidResp, err := e.buildBidResponse(context.Background(), liveAdapters, adapterBids, bidRequest, resolvedRequest, adapterExtra, nil, nil, errList)

	/* 	5) Assert we have no errors and one '&' character as we are supposed to 	*/
	if err != nil {
		t.Errorf("exchange.buildBidResponse returned unexpected error: %v", err)
	}
	if len(errList) > 0 {
		t.Errorf("exchange.buildBidResponse returned %d errors", len(errList))
	}
	if bytes.Contains(bidResp.Ext, []byte("u0026")) {
		t.Errorf("exchange.buildBidResponse() did not correctly print the '&' characters %s", string(bidResp.Ext))
	}
}

func TestGetBidCacheInfo(t *testing.T) {
	testUUID := "CACHE_UUID_1234"
	testExternalCacheHost := "https://www.externalprebidcache.net"
	testExternalCachePath := "endpoints/cache"

	/* 1) An adapter 											*/
	bidderName := openrtb_ext.BidderName("appnexus")

	cfg := &config.Configuration{
		Adapters: map[string]config.Adapter{
			string(bidderName): {
				Endpoint: "http://ib.adnxs.com/endpoint",
			},
		},
		CacheURL: config.Cache{
			Host: "www.internalprebidcache.net",
		},
		ExtCacheURL: config.ExternalCache{
			Host: testExternalCacheHost,
			Path: testExternalCachePath,
		},
	}
	adapterList := make([]openrtb_ext.BidderName, 0, 2)
	testEngine := metricsConf.NewMetricsEngine(cfg, adapterList)

	/* 	2) Init new exchange with said configuration			*/
	handlerNoBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	server := httptest.NewServer(http.HandlerFunc(handlerNoBidServer))
	defer server.Close()

	e := NewExchange(server.Client(), pbc.NewClient(&http.Client{}, &cfg.CacheURL, &cfg.ExtCacheURL, testEngine), cfg, pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{}), adapters.ParseBidderInfos(cfg.Adapters, "../static/bidder-info", openrtb_ext.BidderList()), gdpr.AlwaysAllow{}, currencies.NewRateConverterDefault()).(*exchange)

	/* 	3) Build all the parameters e.buildBidResponse(ctx.Background(), liveA... ) needs */
	liveAdapters := []openrtb_ext.BidderName{bidderName}

	//adapterBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid,
	bids := []*openrtb.Bid{
		{
			ID:             "some-imp-id",
			ImpID:          "",
			Price:          9.517803,
			NURL:           "",
			BURL:           "",
			LURL:           "",
			AdM:            "",
			AdID:           "",
			ADomain:        nil,
			Bundle:         "",
			IURL:           "",
			CID:            "",
			CrID:           "",
			Tactic:         "",
			Cat:            nil,
			Attr:           nil,
			API:            0,
			Protocol:       0,
			QAGMediaRating: 0,
			Language:       "",
			DealID:         "",
			W:              300,
			H:              250,
			WRatio:         0,
			HRatio:         0,
			Exp:            0,
			Ext:            nil,
		},
	}
	auc := &auction{
		cacheIds: map[*openrtb.Bid]string{
			bids[0]: testUUID,
		},
	}
	aPbsOrtbBidArr := []*pbsOrtbBid{
		{
			bid:     bids[0],
			bidType: openrtb_ext.BidTypeBanner,
			bidTargets: map[string]string{
				"pricegranularity":  "med",
				"includewinners":    "true",
				"includebidderkeys": "false",
			},
		},
	}
	adapterBids := map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
		bidderName: {
			bids:     aPbsOrtbBidArr,
			currency: "USD",
		},
	}

	//resolvedRequest json.RawMessage
	resolvedRequest := json.RawMessage(`{"id": "some-request-id","site": {"page": "prebid.org"},"imp": [{"id": "some-impression-id","banner": {"format": [{"w": 300,"h": 250},{"w": 300,"h": 600}]},"ext": {"appnexus": {"placementId": 1}}}],"tmax": 500}`)

	//adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra,
	adapterExtra := map[openrtb_ext.BidderName]*seatResponseExtra{
		bidderName: {
			ResponseTimeMillis: 5,
			Errors: []openrtb_ext.ExtBidderError{
				{
					Code:    999,
					Message: "Post ib.adnxs.com/openrtb2?query1&query2: unsupported protocol scheme \"\"",
				},
			},
		},
	}
	bidRequest := &openrtb.BidRequest{
		ID:   "some-request-id",
		TMax: 1000,
		Imp: []openrtb.Imp{
			{
				ID:     "test-div",
				Secure: openrtb.Int8Ptr(0),
				Banner: &openrtb.Banner{Format: []openrtb.Format{{W: 300, H: 250}}},
				Ext: json.RawMessage(` {
    "rubicon": {
        "accountId": 1001,
        "siteId": 113932,
        "zoneId": 535510
    },
    "appnexus": { "placementId": 1 },
    "pubmatic": { "publisherId": "156209", "adSlot": "pubmatic_test2@300x250" },
    "pulsepoint": { "cf": "300X250", "cp": 512379, "ct": 486653 },
    "conversant": { "site_id": "108060" },
    "ix": { "siteId": "287415" }
}`),
			},
		},
		Site: &openrtb.Site{
			Page:      "http://rubitest.com/index.html",
			Publisher: &openrtb.Publisher{ID: "1001"},
		},
		Test: 1,
		Ext:  json.RawMessage(`{"prebid": { "cache": { "bids": {}, "vastxml": {} }, "targeting": { "pricegranularity": "med", "includewinners": true, "includebidderkeys": false } }}`),
	}

	var errList []error

	/* 	4) Build bid response 									*/
	bid_resp, err := e.buildBidResponse(context.Background(), liveAdapters, adapterBids, bidRequest, resolvedRequest, adapterExtra, auc, nil, errList)

	/* 	5) Assert we have no errors and the bid response we expected*/
	assert.NoError(t, err, "[TestGetBidCacheInfo] buildBidResponse() threw an error")

	expectedBidResponse := &openrtb.BidResponse{
		SeatBid: []openrtb.SeatBid{
			{
				Seat: string(bidderName),
				Bid: []openrtb.Bid{
					{
						Ext: json.RawMessage(`{ "prebid": { "cache": { "bids": { "cacheId": "` + testUUID + `", "url": "` + testExternalCacheHost + `/` + testExternalCachePath + `?uuid=` + testUUID + `" }, "key": "", "url": "" }`),
					},
				},
			},
		},
	}
	// compare cache UUID
	expCacheUUID, err := jsonparser.GetString(expectedBidResponse.SeatBid[0].Bid[0].Ext, "prebid", "cache", "bids", "cacheId")
	assert.NoErrorf(t, err, "[TestGetBidCacheInfo] Error found while trying to json parse the cacheId field from expected build response. Message: %v \n", err)

	cacheUUID, err := jsonparser.GetString(bid_resp.SeatBid[0].Bid[0].Ext, "prebid", "cache", "bids", "cacheId")
	assert.NoErrorf(t, err, "[TestGetBidCacheInfo] bid_resp.SeatBid[0].Bid[0].Ext = %s \n", bid_resp.SeatBid[0].Bid[0].Ext)

	assert.Equal(t, expCacheUUID, cacheUUID, "[TestGetBidCacheInfo] cacheId field in ext should equal \"%s\" \n", expCacheUUID)

	// compare cache UUID
	expCacheURL, err := jsonparser.GetString(expectedBidResponse.SeatBid[0].Bid[0].Ext, "prebid", "cache", "bids", "url")
	assert.NoErrorf(t, err, "[TestGetBidCacheInfo] Error found while trying to json parse the url field from expected build response. Message: %v \n", err)

	cacheURL, err := jsonparser.GetString(bid_resp.SeatBid[0].Bid[0].Ext, "prebid", "cache", "bids", "url")
	assert.NoErrorf(t, err, "[TestGetBidCacheInfo] Error found while trying to json parse the url field from actual build response. Message: %v \n", err)

	assert.Equal(t, expCacheURL, cacheURL, "[TestGetBidCacheInfo] cacheId field in ext should equal \"%s\" \n", expCacheURL)
}

func TestBidResponseCurrency(t *testing.T) {
	// Init objects
	cfg := &config.Configuration{Adapters: make(map[string]config.Adapter, 1)}
	cfg.Adapters["appnexus"] = config.Adapter{Endpoint: "http://ib.adnxs.com"}

	handlerNoBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	server := httptest.NewServer(http.HandlerFunc(handlerNoBidServer))
	defer server.Close()

	e := NewExchange(server.Client(), nil, cfg, pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{}), adapters.ParseBidderInfos(cfg.Adapters, "../static/bidder-info", openrtb_ext.BidderList()), gdpr.AlwaysAllow{}, currencies.NewRateConverterDefault()).(*exchange)

	liveAdapters := make([]openrtb_ext.BidderName, 1)
	liveAdapters[0] = "appnexus"

	bidRequest := &openrtb.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb.Banner{Format: []openrtb.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext:    json.RawMessage(`{"appnexus": {"placementId": 10433394}}`),
		}},
		Site:   &openrtb.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
		Device: &openrtb.Device{UA: "curl/7.54.0", IP: "::1"},
		AT:     1,
		TMax:   500,
		Ext:    json.RawMessage(`{"id": "some-request-id","site": {"page": "prebid.org"},"imp": [{"id": "some-impression-id","banner": {"format": [{"w": 300,"h": 250},{"w": 300,"h": 600}]},"ext": {"appnexus": {"placementId": 10433394}}}],"tmax": 500}`),
	}

	resolvedRequest := json.RawMessage(`{"id": "some-request-id","site": {"page": "prebid.org"},"imp": [{"id": "some-impression-id","banner": {"format": [{"w": 300,"h": 250},{"w": 300,"h": 600}]},"ext": {"appnexus": {"placementId": 1}}}],"tmax": 500}`)

	adapterExtra := map[openrtb_ext.BidderName]*seatResponseExtra{
		"appnexus": {ResponseTimeMillis: 5},
	}

	var errList []error

	sampleBid := &openrtb.Bid{
		ID:    "some-imp-id",
		Price: 9.517803,
		W:     300,
		H:     250,
		Ext:   nil,
	}
	aPbsOrtbBidArr := []*pbsOrtbBid{{bid: sampleBid, bidType: openrtb_ext.BidTypeBanner}}
	sampleSeatBid := []openrtb.SeatBid{
		{
			Seat: "appnexus",
			Bid: []openrtb.Bid{
				{
					ID:    "some-imp-id",
					Price: 9.517803,
					W:     300,
					H:     250,
					Ext:   json.RawMessage(`{"prebid":{"type":"banner"}}`),
				},
			},
		},
	}
	emptySeatBid := []openrtb.SeatBid{}

	// Test cases
	type aTest struct {
		description         string
		adapterBids         map[openrtb_ext.BidderName]*pbsOrtbSeatBid
		expectedBidResponse *openrtb.BidResponse
	}
	testCases := []aTest{
		{
			description: "1) Adapter to bids map comes with a non-empty currency field and non-empty bid array",
			adapterBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				openrtb_ext.BidderName("appnexus"): {
					bids:     aPbsOrtbBidArr,
					currency: "USD",
				},
			},
			expectedBidResponse: &openrtb.BidResponse{
				ID:      "some-request-id",
				SeatBid: sampleSeatBid,
				Cur:     "USD",
				Ext: json.RawMessage(`{"responsetimemillis":{"appnexus":5},"tmaxrequest":500}
`),
			},
		},
		{
			description: "2) Adapter to bids map comes with a non-empty currency field but an empty bid array",
			adapterBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				openrtb_ext.BidderName("appnexus"): {
					bids:     nil,
					currency: "USD",
				},
			},
			expectedBidResponse: &openrtb.BidResponse{
				ID:      "some-request-id",
				SeatBid: emptySeatBid,
				Cur:     "",
				Ext: json.RawMessage(`{"responsetimemillis":{"appnexus":5},"tmaxrequest":500}
`),
			},
		},
		{
			description: "3) Adapter to bids map comes with an empty currency string and a non-empty bid array",
			adapterBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				openrtb_ext.BidderName("appnexus"): {
					bids:     aPbsOrtbBidArr,
					currency: "",
				},
			},
			expectedBidResponse: &openrtb.BidResponse{
				ID:      "some-request-id",
				SeatBid: sampleSeatBid,
				Cur:     "",
				Ext: json.RawMessage(`{"responsetimemillis":{"appnexus":5},"tmaxrequest":500}
`),
			},
		},
		{
			description: "4) Adapter to bids map comes with an empty currency string and an empty bid array",
			adapterBids: map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
				openrtb_ext.BidderName("appnexus"): {
					bids:     nil,
					currency: "",
				},
			},
			expectedBidResponse: &openrtb.BidResponse{
				ID:      "some-request-id",
				SeatBid: emptySeatBid,
				Cur:     "",
				Ext: json.RawMessage(`{"responsetimemillis":{"appnexus":5},"tmaxrequest":500}
`),
			},
		},
	}

	// Run tests
	for i := range testCases {
		actualBidResp, err := e.buildBidResponse(context.Background(), liveAdapters, testCases[i].adapterBids, bidRequest, resolvedRequest, adapterExtra, nil, nil, errList)
		assert.NoError(t, err, fmt.Sprintf("[TEST_FAILED] e.buildBidResponse resturns error in test: %s Error message: %s \n", testCases[i].description, err))
		assert.Equalf(t, testCases[i].expectedBidResponse, actualBidResp, fmt.Sprintf("[TEST_FAILED] Objects must be equal for test: %s \n Expected: >>%s<< \n Actual: >>%s<< ", testCases[i].description, testCases[i].expectedBidResponse.Ext, actualBidResp.Ext))
	}
}

// TestRaceIntegration runs an integration test using all the sample params from
// adapters/{bidder}/{bidder}test/params/race/*.json files.
//
// Its primary goal is to catch race conditions, since parts of the BidRequest passed into MakeBids()
// are shared across many goroutines.
//
// The "known" file names right now are "banner.json" and "video.json". These files should hold params
// which the Bidder would expect on banner or video Imps, respectively.
func TestRaceIntegration(t *testing.T) {
	noBidServer := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}
	server := httptest.NewServer(http.HandlerFunc(noBidServer))
	defer server.Close()

	cfg := &config.Configuration{
		Adapters: make(map[string]config.Adapter, len(openrtb_ext.BidderMap)),
	}
	for _, bidder := range openrtb_ext.BidderList() {
		cfg.Adapters[strings.ToLower(string(bidder))] = config.Adapter{
			Endpoint: server.URL,
		}
	}
	cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderFacebook))] = config.Adapter{
		Endpoint:   server.URL,
		PlatformID: "abc",
	}
	cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderBeachfront))] = config.Adapter{
		Endpoint:         server.URL,
		ExtraAdapterInfo: "{\"video_endpoint\":\"" + server.URL + "\"}",
	}

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}
	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{})
	ex := NewExchange(server.Client(), &wellBehavedCache{}, cfg, theMetrics, adapters.ParseBidderInfos(cfg.Adapters, "../static/bidder-info", openrtb_ext.BidderList()), gdpr.AlwaysAllow{}, currencies.NewRateConverterDefault())
	_, err := ex.HoldAuction(context.Background(), newRaceCheckingRequest(t), &emptyUsersync{}, pbsmetrics.Labels{}, &categoriesFetcher, nil)
	if err != nil {
		t.Errorf("HoldAuction returned unexpected error: %v", err)
	}
}

func newCategoryFetcher(directory string) (stored_requests.CategoryFetcher, error) {
	fetcher, err := file_fetcher.NewFileFetcher(directory)
	if err != nil {
		return nil, err
	}
	catfetcher, ok := fetcher.(stored_requests.CategoryFetcher)
	if !ok {
		return nil, fmt.Errorf("Failed to type cast fetcher to CategoryFetcher")
	}
	return catfetcher, nil
}

// newRaceCheckingRequest builds a BidRequest from all the params in the
// adapters/{bidder}/{bidder}test/params/race/*.json files
func newRaceCheckingRequest(t *testing.T) *openrtb.BidRequest {
	dnt := int8(1)
	return &openrtb.BidRequest{
		Site: &openrtb.Site{
			Page:   "www.some.domain.com",
			Domain: "domain.com",
			Publisher: &openrtb.Publisher{
				ID: "some-publisher-id",
			},
		},
		Device: &openrtb.Device{
			UA:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.87 Safari/537.36",
			IFA:      "ifa",
			IP:       "132.173.230.74",
			DNT:      &dnt,
			Language: "EN",
		},
		Source: &openrtb.Source{
			TID: "61018dc9-fa61-4c41-b7dc-f90b9ae80e87",
		},
		User: &openrtb.User{
			ID:       "our-id",
			BuyerUID: "their-id",
			Ext:      json.RawMessage(`{"consent":"BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw","digitrust":{"id":"digi-id","keyv":1,"pref":1}}`),
		},
		Regs: &openrtb.Regs{
			COPPA: 1,
			Ext:   json.RawMessage(`{"gdpr":1}`),
		},
		Imp: []openrtb.Imp{{
			ID: "some-imp-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}, {
					W: 300,
					H: 600,
				}},
			},
			Ext: buildImpExt(t, "banner"),
		}, {
			Video: &openrtb.Video{
				MIMEs:       []string{"video/mp4"},
				MinDuration: 1,
				MaxDuration: 300,
				W:           300,
				H:           600,
			},
			Ext: buildImpExt(t, "video"),
		}},
	}
}

func TestPanicRecovery(t *testing.T) {
	cfg := &config.Configuration{
		CacheURL: config.Cache{
			ExpectedTimeMillis: 20,
		},
		Adapters: blankAdapterConfig(openrtb_ext.BidderList()),
	}

	theMetrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{})
	e := NewExchange(&http.Client{}, nil, cfg, theMetrics, adapters.ParseBidderInfos(cfg.Adapters, "../static/bidder-info", openrtb_ext.BidderList()), gdpr.AlwaysAllow{}, currencies.NewRateConverterDefault()).(*exchange)
	chBids := make(chan *bidResponseWrapper, 1)
	panicker := func(aName openrtb_ext.BidderName, coreBidder openrtb_ext.BidderName, request *openrtb.BidRequest, bidlabels *pbsmetrics.AdapterLabels, conversions currencies.Conversions) {
		panic("panic!")
	}
	cleanReqs := map[openrtb_ext.BidderName]*openrtb.BidRequest{
		"bidder1": {
			ID: "b-1",
		},
		"bidder2": {
			ID: "b-2",
		},
	}
	recovered := e.recoverSafely(cleanReqs, panicker, chBids)
	apnLabels := pbsmetrics.AdapterLabels{
		Source:      pbsmetrics.DemandWeb,
		RType:       pbsmetrics.ReqTypeORTB2Web,
		Adapter:     openrtb_ext.BidderAppnexus,
		PubID:       "test1",
		Browser:     pbsmetrics.BrowserSafari,
		CookieFlag:  pbsmetrics.CookieFlagYes,
		AdapterBids: pbsmetrics.AdapterBidNone,
	}
	recovered(openrtb_ext.BidderAppnexus, openrtb_ext.BidderAppnexus, nil, &apnLabels, nil)
}

func buildImpExt(t *testing.T, jsonFilename string) json.RawMessage {
	adapterFolders, err := ioutil.ReadDir("../adapters")
	if err != nil {
		t.Fatalf("Failed to open adapters directory: %v", err)
	}
	bidderExts := make(map[string]json.RawMessage, len(openrtb_ext.BidderMap))
	for _, adapterFolder := range adapterFolders {
		if adapterFolder.IsDir() && adapterFolder.Name() != "adapterstest" {
			bidderName := adapterFolder.Name()
			sampleParams := "../adapters/" + bidderName + "/" + bidderName + "test/params/race/" + jsonFilename + ".json"
			// If the file doesn't exist, don't worry about it. I don't think the Go APIs offer a reliable way to check for this.
			fileContents, err := ioutil.ReadFile(sampleParams)
			if err == nil {
				bidderExts[bidderName] = json.RawMessage(fileContents)
			}
		}
	}
	toReturn, err := json.Marshal(bidderExts)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	return json.RawMessage(toReturn)
}

func TestPanicRecoveryHighLevel(t *testing.T) {
	noBidServer := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}
	server := httptest.NewServer(http.HandlerFunc(noBidServer))
	defer server.Close()

	cfg := &config.Configuration{
		Adapters: make(map[string]config.Adapter, len(openrtb_ext.BidderMap)),
	}
	for _, bidder := range openrtb_ext.BidderList() {
		cfg.Adapters[strings.ToLower(string(bidder))] = config.Adapter{
			Endpoint: server.URL,
		}
	}
	e := NewExchange(server.Client(), &mockCache{}, cfg, pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{}), adapters.ParseBidderInfos(cfg.Adapters, "../static/bidder-info", openrtb_ext.BidderList()), gdpr.AlwaysAllow{}, currencies.NewRateConverterDefault()).(*exchange)

	e.adapterMap[openrtb_ext.BidderBeachfront] = panicingAdapter{}
	e.adapterMap[openrtb_ext.BidderAppnexus] = panicingAdapter{}

	request := &openrtb.BidRequest{
		Site: &openrtb.Site{
			Page:   "www.some.domain.com",
			Domain: "domain.com",
			Publisher: &openrtb.Publisher{
				ID: "some-publisher-id",
			},
		},
		User: &openrtb.User{
			ID:       "our-id",
			BuyerUID: "their-id",
			Ext:      json.RawMessage(`{"consent":"BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw","digitrust":{"id":"digi-id","keyv":1,"pref":1}}`),
		},
		Imp: []openrtb.Imp{{
			ID: "some-imp-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}, {
					W: 300,
					H: 600,
				}},
			},
			Ext: buildImpExt(t, "banner"),
		}},
	}

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}
	_, err := e.HoldAuction(context.Background(), request, &emptyUsersync{}, pbsmetrics.Labels{}, &categoriesFetcher, nil)
	if err != nil {
		t.Errorf("HoldAuction returned unexpected error: %v", err)
	}

}

func TestTimeoutComputation(t *testing.T) {
	cacheTimeMillis := 10
	ex := exchange{
		cacheTime: time.Duration(cacheTimeMillis) * time.Millisecond,
	}
	deadline := time.Now()
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	auctionCtx, cancel := ex.makeAuctionContext(ctx, true)
	defer cancel()

	if finalDeadline, ok := auctionCtx.Deadline(); !ok || deadline.Add(-time.Duration(cacheTimeMillis)*time.Millisecond) != finalDeadline {
		t.Errorf("The auction should allocate cacheTime amount of time from the whole request timeout.")
	}
}

// TestExchangeJSON executes tests for all the *.json files in exchangetest.
func TestExchangeJSON(t *testing.T) {
	if specFiles, err := ioutil.ReadDir("./exchangetest"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./exchangetest/" + specFile.Name()
			fileDisplayName := "exchange/exchangetest/" + specFile.Name()
			specData, err := loadFile(fileName)
			if err != nil {
				t.Fatalf("Failed to load contents of file %s: %v", fileDisplayName, err)
			}

			runSpec(t, fileDisplayName, specData)
		}
	}
}

// LoadFile reads and parses a file as a test case. If something goes wrong, it returns an error.
func loadFile(filename string) (*exchangeSpec, error) {
	specData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file %s: %v", filename, err)
	}

	var spec exchangeSpec
	if err := json.Unmarshal(specData, &spec); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal JSON from file: %v", err)
	}

	return &spec, nil
}

func runSpec(t *testing.T, filename string, spec *exchangeSpec) {
	aliases, errs := parseAliases(&spec.IncomingRequest.OrtbRequest)
	if len(errs) != 0 {
		t.Fatalf("%s: Failed to parse aliases", filename)
	}

	privacyConfig := config.Privacy{
		CCPA: config.CCPA{
			Enforce: spec.EnforceCCPA,
		},
		LMT: config.LMT{
			Enforce: spec.EnforceLMT,
		},
	}

	ex := newExchangeForTests(t, filename, spec.OutgoingRequests, aliases, privacyConfig)
	biddersInAuction := findBiddersInAuction(t, filename, &spec.IncomingRequest.OrtbRequest)
	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}
	debugLog := &DebugLog{}
	if spec.DebugLog != nil {
		*debugLog = *spec.DebugLog
		debugLog.Regexp = regexp.MustCompile(`[<>]`)
	}
	bid, err := ex.HoldAuction(context.Background(), &spec.IncomingRequest.OrtbRequest, mockIdFetcher(spec.IncomingRequest.Usersyncs), pbsmetrics.Labels{}, &categoriesFetcher, debugLog)
	responseTimes := extractResponseTimes(t, filename, bid)
	for _, bidderName := range biddersInAuction {
		if _, ok := responseTimes[bidderName]; !ok {
			t.Errorf("%s: Response JSON missing expected ext.responsetimemillis.%s", filename, bidderName)
		}
	}
	if spec.Response.Bids != nil {
		diffOrtbResponses(t, filename, spec.Response.Bids, bid)
		if err == nil {
			if spec.Response.Error != "" {
				t.Errorf("%s: Exchange did not return expected error: %s", filename, spec.Response.Error)
			}
		} else {
			if err.Error() != spec.Response.Error {
				t.Errorf("%s: Exchange returned different errors. Expected %s, got %s", filename, spec.Response.Error, err.Error())
			}
		}
	}
	if spec.DebugLog != nil {
		if spec.DebugLog.Enabled {
			if len(debugLog.Data.Response) == 0 {
				t.Errorf("%s: DebugLog response was not modified when it should have been", filename)
			}
		} else {
			if len(debugLog.Data.Response) != 0 {
				t.Errorf("%s: DebugLog response was modified when it shouldn't have been", filename)
			}
		}
	}
	if spec.IncomingRequest.OrtbRequest.Test == 1 {
		//compare debug info
		diffJson(t, "Debug info modified", bid.Ext, spec.Response.Ext)

	}
}

func findBiddersInAuction(t *testing.T, context string, req *openrtb.BidRequest) []string {
	if splitImps, err := splitImps(req.Imp); err != nil {
		t.Errorf("%s: Failed to parse Bidders from request: %v", context, err)
		return nil
	} else {
		bidders := make([]string, 0, len(splitImps))
		for bidderName := range splitImps {
			bidders = append(bidders, bidderName)
		}
		return bidders
	}
}

// extractResponseTimes validates the format of bid.ext.responsetimemillis, and then removes it.
// This is done because the response time will change from run to run, so it's impossible to hardcode a value
// into the JSON. The best we can do is make sure that the property exists.
func extractResponseTimes(t *testing.T, context string, bid *openrtb.BidResponse) map[string]int {
	if data, dataType, _, err := jsonparser.Get(bid.Ext, "responsetimemillis"); err != nil || dataType != jsonparser.Object {
		t.Errorf("%s: Exchange did not return ext.responsetimemillis object: %v", context, err)
		return nil
	} else {
		responseTimes := make(map[string]int)
		if err := json.Unmarshal(data, &responseTimes); err != nil {
			t.Errorf("%s: Failed to unmarshal ext.responsetimemillis into map[string]int: %v", context, err)
			return nil
		}

		// Delete the response times so that they don't appear in the JSON, because they can't be tested reliably anyway.
		// If there's no other ext, just delete it altogether.
		bid.Ext = jsonparser.Delete(bid.Ext, "responsetimemillis")
		if diff, err := gojsondiff.New().Compare(bid.Ext, []byte("{}")); err == nil && !diff.Modified() {
			bid.Ext = nil
		}
		return responseTimes
	}
}

func newExchangeForTests(t *testing.T, filename string, expectations map[string]*bidderSpec, aliases map[string]string, privacyConfig config.Privacy) Exchange {
	adapters := make(map[openrtb_ext.BidderName]adaptedBidder)
	for _, bidderName := range openrtb_ext.BidderMap {
		if spec, ok := expectations[string(bidderName)]; ok {
			adapters[bidderName] = &validatingBidder{
				t:             t,
				fileName:      filename,
				bidderName:    string(bidderName),
				expectations:  map[string]*bidderRequest{string(bidderName): spec.ExpectedRequest},
				mockResponses: map[string]bidderResponse{string(bidderName): spec.MockResponse},
			}
		}
	}
	for alias, coreBidder := range aliases {
		if spec, ok := expectations[alias]; ok {
			if bidder, ok := adapters[openrtb_ext.BidderName(coreBidder)]; ok {
				bidder.(*validatingBidder).expectations[alias] = spec.ExpectedRequest
				bidder.(*validatingBidder).mockResponses[alias] = spec.MockResponse
			} else {
				adapters[openrtb_ext.BidderName(coreBidder)] = &validatingBidder{
					t:             t,
					fileName:      filename,
					bidderName:    coreBidder,
					expectations:  map[string]*bidderRequest{coreBidder: spec.ExpectedRequest},
					mockResponses: map[string]bidderResponse{coreBidder: spec.MockResponse},
				}
			}
		}
	}

	return &exchange{
		adapterMap:          adapters,
		me:                  metricsConf.NewMetricsEngine(&config.Configuration{}, openrtb_ext.BidderList()),
		cache:               &wellBehavedCache{},
		cacheTime:           0,
		gDPR:                gdpr.AlwaysAllow{},
		currencyConverter:   currencies.NewRateConverterDefault(),
		UsersyncIfAmbiguous: false,
		privacyConfig:       privacyConfig,
	}
}

func newExtRequest() openrtb_ext.ExtRequest {
	priceGran := openrtb_ext.PriceGranularity{
		Precision: 2,
		Ranges: []openrtb_ext.GranularityRange{
			{
				Min:       0.0,
				Max:       20.0,
				Increment: 2.0,
			},
		},
	}

	translateCategories := true
	brandCat := openrtb_ext.ExtIncludeBrandCategory{PrimaryAdServer: 1, WithCategory: true, TranslateCategories: &translateCategories}

	reqExt := openrtb_ext.ExtRequestTargeting{
		PriceGranularity:     priceGran,
		IncludeWinners:       true,
		IncludeBrandCategory: &brandCat,
	}

	return openrtb_ext.ExtRequest{
		Prebid: openrtb_ext.ExtRequestPrebid{
			Targeting: &reqExt,
		},
	}
}

func newExtRequestNoBrandCat() openrtb_ext.ExtRequest {
	priceGran := openrtb_ext.PriceGranularity{
		Precision: 2,
		Ranges: []openrtb_ext.GranularityRange{
			{
				Min:       0.0,
				Max:       20.0,
				Increment: 2.0,
			},
		},
	}

	brandCat := openrtb_ext.ExtIncludeBrandCategory{WithCategory: false}

	reqExt := openrtb_ext.ExtRequestTargeting{
		PriceGranularity:     priceGran,
		IncludeWinners:       true,
		IncludeBrandCategory: &brandCat,
	}

	return openrtb_ext.ExtRequest{
		Prebid: openrtb_ext.ExtRequestPrebid{
			Targeting: &reqExt,
		},
	}
}

func TestCategoryMapping(t *testing.T) {

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequest()

	targData := &targetData{
		priceGranularity: requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	requestExt.Prebid.Targeting.DurationRangeSec = []int{15, 30, 50}

	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}
	cats3 := []string{"IAB1-1000"}
	cats4 := []string{"IAB1-2000"}
	bid1 := openrtb.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 20.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 30.0000, Cat: cats3, W: 1, H: 1}
	bid4 := openrtb.Bid{ID: "bid_id4", ImpID: "imp_id4", Price: 40.0000, Cat: cats4, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, 0}
	bid1_2 := pbsOrtbBid{&bid2, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 40}, 0}
	bid1_3 := pbsOrtbBid{&bid3, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30, PrimaryCategory: "AdapterOverride"}, 0}
	bid1_4 := pbsOrtbBid{&bid4, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, 0}

	innerBids := []*pbsOrtbBid{
		&bid1_1,
		&bid1_2,
		&bid1_3,
		&bid1_4,
	}

	seatBid := pbsOrtbSeatBid{innerBids, "USD", nil, nil}
	bidderName1 := openrtb_ext.BidderName("appnexus")

	adapterBids[bidderName1] = &seatBid

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, requestExt, adapterBids, categoriesFetcher, targData)

	assert.Equal(t, nil, err, "Category mapping error should be empty")
	assert.Equal(t, 1, len(rejections), "There should be 1 bid rejection message")
	assert.Equal(t, "bid rejected [bid ID: bid_id4] reason: Category mapping file for primary ad server: 'freewheel', publisher: '' not found", rejections[0], "Rejection message did not match expected")
	assert.Equal(t, "10.00_Electronics_30s", bidCategory["bid_id1"], "Category mapping doesn't match")
	assert.Equal(t, "20.00_Sports_50s", bidCategory["bid_id2"], "Category mapping doesn't match")
	assert.Equal(t, "20.00_AdapterOverride_30s", bidCategory["bid_id3"], "Category mapping override from adapter didn't take")
	assert.Equal(t, 3, len(adapterBids[bidderName1].bids), "Bidders number doesn't match")
	assert.Equal(t, 3, len(bidCategory), "Bidders category mapping doesn't match")
}

func TestCategoryMappingNoIncludeBrandCategory(t *testing.T) {

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequestNoBrandCat()

	targData := &targetData{
		priceGranularity: requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}
	requestExt.Prebid.Targeting.DurationRangeSec = []int{15, 30, 40, 50}

	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}
	cats3 := []string{"IAB1-1000"}
	cats4 := []string{"IAB1-2000"}
	bid1 := openrtb.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 20.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 30.0000, Cat: cats3, W: 1, H: 1}
	bid4 := openrtb.Bid{ID: "bid_id4", ImpID: "imp_id4", Price: 40.0000, Cat: cats4, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, 0}
	bid1_2 := pbsOrtbBid{&bid2, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 40}, 0}
	bid1_3 := pbsOrtbBid{&bid3, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30, PrimaryCategory: "AdapterOverride"}, 0}
	bid1_4 := pbsOrtbBid{&bid4, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 50}, 0}

	innerBids := []*pbsOrtbBid{
		&bid1_1,
		&bid1_2,
		&bid1_3,
		&bid1_4,
	}

	seatBid := pbsOrtbSeatBid{innerBids, "USD", nil, nil}
	bidderName1 := openrtb_ext.BidderName("appnexus")

	adapterBids[bidderName1] = &seatBid

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, requestExt, adapterBids, categoriesFetcher, targData)

	assert.Equal(t, nil, err, "Category mapping error should be empty")
	assert.Empty(t, rejections, "There should be no bid rejection messages")
	assert.Equal(t, "10.00_30s", bidCategory["bid_id1"], "Category mapping doesn't match")
	assert.Equal(t, "20.00_40s", bidCategory["bid_id2"], "Category mapping doesn't match")
	assert.Equal(t, "20.00_30s", bidCategory["bid_id3"], "Category mapping doesn't match")
	assert.Equal(t, "20.00_50s", bidCategory["bid_id4"], "Category mapping doesn't match")
	assert.Equal(t, 4, len(adapterBids[bidderName1].bids), "Bidders number doesn't match")
	assert.Equal(t, 4, len(bidCategory), "Bidders category mapping doesn't match")
}

func TestCategoryMappingTranslateCategoriesNil(t *testing.T) {

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequestTranslateCategories(nil)

	targData := &targetData{
		priceGranularity: requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	requestExt.Prebid.Targeting.DurationRangeSec = []int{15, 30, 50}

	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}
	cats3 := []string{"IAB1-1000"}
	bid1 := openrtb.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 20.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 30.0000, Cat: cats3, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, 0}
	bid1_2 := pbsOrtbBid{&bid2, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 40}, 0}
	bid1_3 := pbsOrtbBid{&bid3, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, 0}

	innerBids := []*pbsOrtbBid{
		&bid1_1,
		&bid1_2,
		&bid1_3,
	}

	seatBid := pbsOrtbSeatBid{innerBids, "USD", nil, nil}
	bidderName1 := openrtb_ext.BidderName("appnexus")

	adapterBids[bidderName1] = &seatBid

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, requestExt, adapterBids, categoriesFetcher, targData)

	assert.Equal(t, nil, err, "Category mapping error should be empty")
	assert.Equal(t, 1, len(rejections), "There should be 1 bid rejection message")
	assert.Equal(t, "bid rejected [bid ID: bid_id3] reason: Category mapping file for primary ad server: 'freewheel', publisher: '' not found", rejections[0], "Rejection message did not match expected")
	assert.Equal(t, "10.00_Electronics_30s", bidCategory["bid_id1"], "Category mapping doesn't match")
	assert.Equal(t, "20.00_Sports_50s", bidCategory["bid_id2"], "Category mapping doesn't match")
	assert.Equal(t, 2, len(adapterBids[bidderName1].bids), "Bidders number doesn't match")
	assert.Equal(t, 2, len(bidCategory), "Bidders category mapping doesn't match")
}

func newExtRequestTranslateCategories(translateCategories *bool) openrtb_ext.ExtRequest {
	priceGran := openrtb_ext.PriceGranularity{
		Precision: 2,
		Ranges: []openrtb_ext.GranularityRange{
			{
				Min:       0.0,
				Max:       20.0,
				Increment: 2.0,
			},
		},
	}

	brandCat := openrtb_ext.ExtIncludeBrandCategory{WithCategory: true, PrimaryAdServer: 1}
	if translateCategories != nil {
		brandCat.TranslateCategories = translateCategories
	}

	reqExt := openrtb_ext.ExtRequestTargeting{
		PriceGranularity:     priceGran,
		IncludeWinners:       true,
		IncludeBrandCategory: &brandCat,
	}

	return openrtb_ext.ExtRequest{
		Prebid: openrtb_ext.ExtRequestPrebid{
			Targeting: &reqExt,
		},
	}
}

func TestCategoryMappingTranslateCategoriesFalse(t *testing.T) {

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	translateCategories := false
	requestExt := newExtRequestTranslateCategories(&translateCategories)

	targData := &targetData{
		priceGranularity: requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	requestExt.Prebid.Targeting.DurationRangeSec = []int{15, 30, 50}

	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}
	cats3 := []string{"IAB1-1000"}
	bid1 := openrtb.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 20.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 30.0000, Cat: cats3, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, 0}
	bid1_2 := pbsOrtbBid{&bid2, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 40}, 0}
	bid1_3 := pbsOrtbBid{&bid3, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, 0}

	innerBids := []*pbsOrtbBid{
		&bid1_1,
		&bid1_2,
		&bid1_3,
	}

	seatBid := pbsOrtbSeatBid{innerBids, "USD", nil, nil}
	bidderName1 := openrtb_ext.BidderName("appnexus")

	adapterBids[bidderName1] = &seatBid

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, requestExt, adapterBids, categoriesFetcher, targData)

	assert.Equal(t, nil, err, "Category mapping error should be empty")
	assert.Empty(t, rejections, "There should be no bid rejection messages")
	assert.Equal(t, "10.00_IAB1-3_30s", bidCategory["bid_id1"], "Category should not be translated")
	assert.Equal(t, "20.00_IAB1-4_50s", bidCategory["bid_id2"], "Category should not be translated")
	assert.Equal(t, "20.00_IAB1-1000_30s", bidCategory["bid_id3"], "Bid should not be rejected")
	assert.Equal(t, 3, len(adapterBids[bidderName1].bids), "Bidders number doesn't match")
	assert.Equal(t, 3, len(bidCategory), "Bidders category mapping doesn't match")
}

func TestCategoryDedupe(t *testing.T) {

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequest()

	targData := &targetData{
		priceGranularity: requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}
	// bid3 will be same price, category, and duration as bid1 so one of them should get removed
	cats4 := []string{"IAB1-2000"}
	bid1 := openrtb.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 15.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid4 := openrtb.Bid{ID: "bid_id4", ImpID: "imp_id4", Price: 20.0000, Cat: cats4, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, 0}
	bid1_2 := pbsOrtbBid{&bid2, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 50}, 0}
	bid1_3 := pbsOrtbBid{&bid3, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, 0}
	bid1_4 := pbsOrtbBid{&bid4, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, 0}

	selectedBids := make(map[string]int)
	expectedCategories := map[string]string{
		"bid_id1": "10.00_Electronics_30s",
		"bid_id2": "14.00_Sports_50s",
		"bid_id3": "10.00_Electronics_30s",
	}

	numIterations := 10

	// Run the function many times, this should be enough for the 50% chance of which bid to remove to remove bid1 sometimes
	// and bid3 others. It's conceivably possible (but highly unlikely) that the same bid get chosen every single time, but
	// if you notice false fails from this test increase numIterations to make it even less likely to happen.
	for i := 0; i < numIterations; i++ {
		innerBids := []*pbsOrtbBid{
			&bid1_1,
			&bid1_2,
			&bid1_3,
			&bid1_4,
		}

		seatBid := pbsOrtbSeatBid{innerBids, "USD", nil, nil}
		bidderName1 := openrtb_ext.BidderName("appnexus")

		adapterBids[bidderName1] = &seatBid

		bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, requestExt, adapterBids, categoriesFetcher, targData)

		assert.Equal(t, nil, err, "Category mapping error should be empty")
		assert.Equal(t, 2, len(rejections), "There should be 2 bid rejection messages")
		assert.Regexpf(t, regexp.MustCompile(`bid rejected \[bid ID: bid_id(1|3)\] reason: Bid was deduplicated`), rejections[0], "Rejection message did not match expected")
		assert.Equal(t, "bid rejected [bid ID: bid_id4] reason: Category mapping file for primary ad server: 'freewheel', publisher: '' not found", rejections[1], "Rejection message did not match expected")
		assert.Equal(t, 2, len(adapterBids[bidderName1].bids), "Bidders number doesn't match")
		assert.Equal(t, 2, len(bidCategory), "Bidders category mapping doesn't match")

		for bidId, bidCat := range bidCategory {
			assert.Equal(t, expectedCategories[bidId], bidCat, "Category mapping doesn't match")
			selectedBids[bidId]++
		}
	}

	assert.Equal(t, numIterations, selectedBids["bid_id2"], "Bid 2 did not make it through every time")
	assert.NotEqual(t, numIterations, selectedBids["bid_id1"], "Bid 1 made it through every time")
	assert.NotEqual(t, numIterations, selectedBids["bid_id3"], "Bid 3 made it through every time")
}

func TestBidRejectionErrors(t *testing.T) {
	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequest()
	requestExt.Prebid.Targeting.DurationRangeSec = []int{15, 30, 50}

	targData := &targetData{
		priceGranularity: requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	invalidReqExt := newExtRequest()
	invalidReqExt.Prebid.Targeting.DurationRangeSec = []int{15, 30, 50}
	invalidReqExt.Prebid.Targeting.IncludeBrandCategory.PrimaryAdServer = 2
	invalidReqExt.Prebid.Targeting.IncludeBrandCategory.Publisher = "some_publisher"

	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)
	bidderName := openrtb_ext.BidderName("appnexus")

	testCases := []struct {
		description        string
		reqExt             openrtb_ext.ExtRequest
		bids               []*openrtb.Bid
		duration           int
		expectedRejections []string
		expectedCatDur     string
	}{
		{
			description: "Bid should be rejected due to not containing a category",
			reqExt:      requestExt,
			bids: []*openrtb.Bid{
				{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: []string{}, W: 1, H: 1},
			},
			duration: 30,
			expectedRejections: []string{
				"bid rejected [bid ID: bid_id1] reason: Bid did not contain a category",
			},
		},
		{
			description: "Bid should be rejected due to missing category mapping file",
			reqExt:      invalidReqExt,
			bids: []*openrtb.Bid{
				{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: []string{"IAB1-1"}, W: 1, H: 1},
			},
			duration: 30,
			expectedRejections: []string{
				"bid rejected [bid ID: bid_id1] reason: Category mapping file for primary ad server: 'dfp', publisher: 'some_publisher' not found",
			},
		},
		{
			description: "Bid should be rejected due to duration exceeding maximum",
			reqExt:      requestExt,
			bids: []*openrtb.Bid{
				{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: []string{"IAB1-1"}, W: 1, H: 1},
			},
			duration: 70,
			expectedRejections: []string{
				"bid rejected [bid ID: bid_id1] reason: Bid duration exceeds maximum allowed",
			},
		},
		{
			description: "Bid should be rejected due to duplicate bid",
			reqExt:      requestExt,
			bids: []*openrtb.Bid{
				{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: []string{"IAB1-1"}, W: 1, H: 1},
				{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: []string{"IAB1-1"}, W: 1, H: 1},
			},
			duration: 30,
			expectedRejections: []string{
				"bid rejected [bid ID: bid_id1] reason: Bid was deduplicated",
			},
			expectedCatDur: "10.00_VideoGames_30s",
		},
	}

	for _, test := range testCases {
		innerBids := []*pbsOrtbBid{}
		for _, bid := range test.bids {
			currentBid := pbsOrtbBid{
				bid, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: test.duration}, 0,
			}
			innerBids = append(innerBids, &currentBid)
		}

		seatBid := pbsOrtbSeatBid{innerBids, "USD", nil, nil}

		adapterBids[bidderName] = &seatBid

		bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, test.reqExt, adapterBids, categoriesFetcher, targData)

		if len(test.expectedCatDur) > 0 {
			// Bid deduplication case
			assert.Equal(t, 1, len(adapterBids[bidderName].bids), "Bidders number doesn't match")
			assert.Equal(t, 1, len(bidCategory), "Bidders category mapping doesn't match")
			assert.Equal(t, test.expectedCatDur, bidCategory["bid_id1"], "Bid category did not contain expected hb_pb_cat_dur")
		} else {
			assert.Empty(t, adapterBids[bidderName].bids, "Bidders number doesn't match")
			assert.Empty(t, bidCategory, "Bidders category mapping doesn't match")
		}

		assert.Empty(t, err, "Category mapping error should be empty")
		assert.Equal(t, test.expectedRejections, rejections, test.description)
	}
}

func TestUpdateRejections(t *testing.T) {
	rejections := []string{}

	rejections = updateRejections(rejections, "bid_id1", "some reason 1")
	rejections = updateRejections(rejections, "bid_id2", "some reason 2")

	assert.Equal(t, 2, len(rejections), "Rejections should contain 2 rejection messages")
	assert.Containsf(t, rejections, "bid rejected [bid ID: bid_id1] reason: some reason 1", "Rejection message did not match expected")
	assert.Containsf(t, rejections, "bid rejected [bid ID: bid_id2] reason: some reason 2", "Rejection message did not match expected")
}

func TestApplyDealSupport(t *testing.T) {
	testCases := []struct {
		description        string
		dealPriority       int
		impExt             json.RawMessage
		targ               map[string]string
		expectedHbPbCatDur string
		expectedDealErr    string
	}{
		{
			description:  "hb_pb_cat_dur should be modified",
			dealPriority: 5,
			impExt:       json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}`),
			targ: map[string]string{
				"hb_pb_cat_dur": "12.00_movies_30s",
			},
			expectedHbPbCatDur: "tier5_movies_30s",
			expectedDealErr:    "",
		},
		{
			description:  "hb_pb_cat_dur should not be modified due to priority not exceeding min",
			dealPriority: 9,
			impExt:       json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 10, "prefix": "tier"}, "placementId": 10433394}}`),
			targ: map[string]string{
				"hb_pb_cat_dur": "12.00_medicine_30s",
			},
			expectedHbPbCatDur: "12.00_medicine_30s",
			expectedDealErr:    "",
		},
		{
			description:  "hb_pb_cat_dur should not be modified due to invalid config",
			dealPriority: 5,
			impExt:       json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": ""}, "placementId": 10433394}}`),
			targ: map[string]string{
				"hb_pb_cat_dur": "12.00_games_30s",
			},
			expectedHbPbCatDur: "12.00_games_30s",
			expectedDealErr:    "dealTier configuration invalid for bidder 'appnexus', imp ID 'imp_id1'",
		},
		{
			description:  "hb_pb_cat_dur should not be modified due to deal priority of 0",
			dealPriority: 0,
			impExt:       json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}`),
			targ: map[string]string{
				"hb_pb_cat_dur": "12.00_auto_30s",
			},
			expectedHbPbCatDur: "12.00_auto_30s",
			expectedDealErr:    "",
		},
	}

	bidderName := openrtb_ext.BidderName("appnexus")
	for _, test := range testCases {
		bidRequest := &openrtb.BidRequest{
			ID: "some-request-id",
			Imp: []openrtb.Imp{
				{
					ID:  "imp_id1",
					Ext: test.impExt,
				},
			},
		}

		bid := pbsOrtbBid{&openrtb.Bid{ID: "123456"}, "video", map[string]string{}, &openrtb_ext.ExtBidPrebidVideo{}, test.dealPriority}
		bidCategory := map[string]string{
			bid.bid.ID: test.targ["hb_pb_cat_dur"],
		}

		auc := &auction{
			winningBidsByBidder: map[string]map[openrtb_ext.BidderName]*pbsOrtbBid{
				"imp_id1": {
					bidderName: &bid,
				},
			},
		}

		dealErrs := applyDealSupport(bidRequest, auc, bidCategory)

		assert.Equal(t, test.expectedHbPbCatDur, bidCategory[auc.winningBidsByBidder["imp_id1"][bidderName].bid.ID], test.description)
		if len(test.expectedDealErr) > 0 {
			assert.Containsf(t, dealErrs, errors.New(test.expectedDealErr), "Expected error message not found in deal errors")
		}
	}
}

func TestGetDealTiers(t *testing.T) {
	testCases := []struct {
		impExt       json.RawMessage
		bidderResult map[string]bool // true indicates bidder had valid config, false indicates invalid
	}{
		{
			impExt: json.RawMessage(`{"validbase": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}`),
			bidderResult: map[string]bool{
				"validbase": true,
			},
		},
		{
			impExt: json.RawMessage(`{"validmultiple1": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}, "validmultiple2": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}`),
			bidderResult: map[string]bool{
				"validmultiple1": true,
				"validmultiple2": true,
			},
		},
		{
			impExt: json.RawMessage(`{"nodealtier": {"placementId": 10433394}}`),
			bidderResult: map[string]bool{
				"nodealtier": false,
			},
		},
		{
			impExt: json.RawMessage(`{"validbase": {"placementId": 10433394}, "onedealTier2": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}`),
			bidderResult: map[string]bool{
				"onedealTier2": true,
				"validbase":    false,
			},
		},
	}

	filledDealTier := DealTier{
		Info: &DealTierInfo{
			Prefix:      "tier",
			MinDealTier: 5,
		},
	}
	emptyDealTier := DealTier{}

	for _, test := range testCases {
		bidRequest := &openrtb.BidRequest{
			ID: "some-request-id",
			Imp: []openrtb.Imp{
				{
					ID:  "imp_id1",
					Ext: test.impExt,
				},
			},
		}

		impDealMap := getDealTiers(bidRequest)

		for bidder, valid := range test.bidderResult {
			if valid {
				assert.Equal(t, &filledDealTier, impDealMap["imp_id1"].DealInfo[bidder], "DealTier should be filled with config data")
			} else {
				assert.Equal(t, &emptyDealTier, impDealMap["imp_id1"].DealInfo[bidder], "DealTier should be empty")
			}
		}
	}
}

func TestValidateAndNormalizeDealTier(t *testing.T) {
	testCases := []struct {
		description    string
		params         json.RawMessage
		expectedResult bool
	}{
		{
			description:    "BidderDealTier should be valid",
			params:         json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}`),
			expectedResult: true,
		},
		{
			description:    "BidderDealTier should be invalid due to empty prefix",
			params:         json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": ""}, "placementId": 10433394}}`),
			expectedResult: false,
		},
		{
			description:    "BidderDealTier should be invalid due to empty dealTier",
			params:         json.RawMessage(`{"appnexus": {"dealTier": {}, "placementId": 10433394}}`),
			expectedResult: false,
		},
		{
			description:    "BidderDealTier should be invalid due to missing minDealTier",
			params:         json.RawMessage(`{"appnexus": {"dealTier": {"prefix": "tier"}, "placementId": 10433394}}`),
			expectedResult: false,
		},
		{
			description:    "BidderDealTier should be invalid due to missing dealTier",
			params:         json.RawMessage(`{"appnexus": {"placementId": 10433394}}`),
			expectedResult: false,
		},
		{
			description:    "BidderDealTier should be invalid due to prefix containing all whitespace",
			params:         json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "    "}, "placementId": 10433394}}`),
			expectedResult: false,
		},
		{
			description:    "BidderDealTier should be valid after removing whitespace",
			params:         json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "    prefixwith  sp aces "}, "placementId": 10433394}}`),
			expectedResult: true,
		},
	}

	for _, test := range testCases {
		var bidderDealTier BidderDealTier
		err := json.Unmarshal(test.params, &bidderDealTier.DealInfo)
		if err != nil {
			assert.Fail(t, "Unable to unmarshal JSON data for testing BidderDealTier")
		}

		assert.Equal(t, test.expectedResult, validateAndNormalizeDealTier(bidderDealTier.DealInfo["appnexus"]), test.description)
	}
}

func TestUpdateHbPbCatDur(t *testing.T) {
	testCases := []struct {
		description        string
		targ               map[string]string
		dealTier           *DealTierInfo
		dealPriority       int
		expectedHbPbCatDur string
	}{
		{
			description: "hb_pb_cat_dur should be updated with prefix and tier",
			targ: map[string]string{
				"hb_pb":         "12.00",
				"hb_pb_cat_dur": "12.00_movies_30s",
			},
			dealTier: &DealTierInfo{
				Prefix:      "tier",
				MinDealTier: 5,
			},
			dealPriority:       5,
			expectedHbPbCatDur: "tier5_movies_30s",
		},
		{
			description: "hb_pb_cat_dur should not be updated due to bid priority",
			targ: map[string]string{
				"hb_pb":         "12.00",
				"hb_pb_cat_dur": "12.00_auto_30s",
			},
			dealTier: &DealTierInfo{
				Prefix:      "tier",
				MinDealTier: 10,
			},
			dealPriority:       6,
			expectedHbPbCatDur: "12.00_auto_30s",
		},
		{
			description: "hb_pb_cat_dur should be updated with prefix and tier",
			targ: map[string]string{
				"hb_pb":         "12.00",
				"hb_pb_cat_dur": "12.00_medicine_30s",
			},
			dealTier: &DealTierInfo{
				Prefix:      "tier",
				MinDealTier: 1,
			},
			dealPriority:       7,
			expectedHbPbCatDur: "tier7_medicine_30s",
		},
	}

	for _, test := range testCases {
		bid := pbsOrtbBid{&openrtb.Bid{ID: "123456"}, "video", map[string]string{}, &openrtb_ext.ExtBidPrebidVideo{}, test.dealPriority}
		bidCategory := map[string]string{
			bid.bid.ID: test.targ["hb_pb_cat_dur"],
		}

		updateHbPbCatDur(&bid, test.dealTier, bidCategory)

		assert.Equal(t, test.expectedHbPbCatDur, bidCategory[bid.bid.ID], test.description)
	}
}

type exchangeSpec struct {
	IncomingRequest  exchangeRequest        `json:"incomingRequest"`
	OutgoingRequests map[string]*bidderSpec `json:"outgoingRequests"`
	Response         exchangeResponse       `json:"response,omitempty"`
	EnforceCCPA      bool                   `json:"enforceCcpa"`
	EnforceLMT       bool                   `json:"enforceLmt"`
	DebugLog         *DebugLog              `json:"debuglog,omitempty"`
}

type exchangeRequest struct {
	OrtbRequest openrtb.BidRequest `json:"ortbRequest"`
	Usersyncs   map[string]string  `json:"usersyncs"`
}

type exchangeResponse struct {
	Bids  *openrtb.BidResponse `json:"bids"`
	Error string               `json:"error,omitempty"`
	Ext   json.RawMessage      `json:"ext,omitempty"`
}

type bidderSpec struct {
	ExpectedRequest *bidderRequest `json:"expectRequest"`
	MockResponse    bidderResponse `json:"mockResponse"`
}

type bidderRequest struct {
	OrtbRequest   openrtb.BidRequest `json:"ortbRequest"`
	BidAdjustment float64            `json:"bidAdjustment"`
}

type bidderResponse struct {
	SeatBid   *bidderSeatBid             `json:"pbsSeatBid,omitempty"`
	Errors    []string                   `json:"errors,omitempty"`
	HttpCalls []*openrtb_ext.ExtHttpCall `json:"httpCalls,omitempty"`
}

// bidderSeatBid is basically a subset of pbsOrtbSeatBid from exchange/bidder.go.
// The only real reason I'm not reusing that type is because I don't want people to think that the
// JSON property tags on those types are contracts in prod.
type bidderSeatBid struct {
	Bids []bidderBid `json:"pbsBids,omitempty"`
}

// bidderBid is basically a subset of pbsOrtbBid from exchange/bidder.go.
// See the comment on bidderSeatBid for more info.
type bidderBid struct {
	Bid  *openrtb.Bid `json:"ortbBid,omitempty"`
	Type string       `json:"bidType,omitempty"`
}

type mockIdFetcher map[string]string

func (f mockIdFetcher) GetId(bidder openrtb_ext.BidderName) (id string, ok bool) {
	id, ok = f[string(bidder)]
	return
}

type validatingBidder struct {
	t          *testing.T
	fileName   string
	bidderName string

	// These are maps because they may contain aliases. They should _at least_ contain an entry for bidderName.
	expectations  map[string]*bidderRequest
	mockResponses map[string]bidderResponse
}

func (b *validatingBidder) requestBid(ctx context.Context, request *openrtb.BidRequest, name openrtb_ext.BidderName, bidAdjustment float64, conversions currencies.Conversions, reqInfo *adapters.ExtraRequestInfo) (seatBid *pbsOrtbSeatBid, errs []error) {
	if expectedRequest, ok := b.expectations[string(name)]; ok {
		if expectedRequest != nil {
			if expectedRequest.BidAdjustment != bidAdjustment {
				b.t.Errorf("%s: Bidder %s got wrong bid adjustment. Expected %f, got %f", b.fileName, name, expectedRequest.BidAdjustment, bidAdjustment)
			}
			diffOrtbRequests(b.t, fmt.Sprintf("Request to %s in %s", string(name), b.fileName), &expectedRequest.OrtbRequest, request)
		}
	} else {
		b.t.Errorf("%s: Bidder %s got unexpected request for alias %s. No input assertions.", b.fileName, b.bidderName, name)
	}

	if mockResponse, ok := b.mockResponses[string(name)]; ok {
		if mockResponse.SeatBid != nil {
			bids := make([]*pbsOrtbBid, len(mockResponse.SeatBid.Bids))
			for i := 0; i < len(bids); i++ {
				bids[i] = &pbsOrtbBid{
					bid:     mockResponse.SeatBid.Bids[i].Bid,
					bidType: openrtb_ext.BidType(mockResponse.SeatBid.Bids[i].Type),
				}
			}

			seatBid = &pbsOrtbSeatBid{
				bids:      bids,
				httpCalls: mockResponse.HttpCalls,
			}
		} else {
			seatBid = &pbsOrtbSeatBid{
				bids:      nil,
				httpCalls: mockResponse.HttpCalls,
			}
		}

		for _, err := range mockResponse.Errors {
			errs = append(errs, errors.New(err))
		}
	} else {
		b.t.Errorf("%s: Bidder %s got unexpected request for alias %s. No mock responses.", b.fileName, b.bidderName, name)
	}

	return
}

func diffOrtbRequests(t *testing.T, description string, expected *openrtb.BidRequest, actual *openrtb.BidRequest) {
	t.Helper()
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("%s failed to marshal actual BidRequest into JSON. %v", description, err)
	}

	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("%s failed to marshal expected BidRequest into JSON. %v", description, err)
	}

	diffJson(t, description, actualJSON, expectedJSON)
}

func diffOrtbResponses(t *testing.T, description string, expected *openrtb.BidResponse, actual *openrtb.BidResponse) {
	t.Helper()
	// The OpenRTB spec is wonky here. Since "bidresponse.seatbid" is an array, order technically matters to any JSON diff or
	// deep equals method. However, for all intents and purposes it really *doesn't* matter. ...so this nasty logic makes compares
	// the seatbids in an order-independent way.
	//
	// Note that the same thing is technically true of the "seatbid[i].bid" array... but since none of our exchange code relies on
	// this implementation detail, I'm cutting a corner and ignoring it here.
	actualSeats := mapifySeatBids(t, description, actual.SeatBid)
	expectedSeats := mapifySeatBids(t, description, expected.SeatBid)
	actualJSON, err := json.Marshal(actualSeats)
	if err != nil {
		t.Fatalf("%s failed to marshal actual BidResponse into JSON. %v", description, err)
	}

	expectedJSON, err := json.Marshal(expectedSeats)
	if err != nil {
		t.Fatalf("%s failed to marshal expected BidResponse into JSON. %v", description, err)
	}

	diffJson(t, description, actualJSON, expectedJSON)
}

func mapifySeatBids(t *testing.T, context string, seatBids []openrtb.SeatBid) map[string]*openrtb.SeatBid {
	seatMap := make(map[string]*openrtb.SeatBid, len(seatBids))
	for i := 0; i < len(seatBids); i++ {
		seatName := seatBids[i].Seat
		if _, ok := seatMap[seatName]; ok {
			t.Fatalf("%s: Contains duplicate Seat: %s", context, seatName)
		} else {
			seatMap[seatName] = &seatBids[i]
		}
	}
	return seatMap
}

// diffJson compares two JSON byte arrays for structural equality. It will produce an error if either
// byte array is not actually JSON.
func diffJson(t *testing.T, description string, actual []byte, expected []byte) {
	t.Helper()
	diff, err := gojsondiff.New().Compare(actual, expected)
	if err != nil {
		t.Fatalf("%s json diff failed. %v", description, err)
	}

	if diff.Modified() {
		var left interface{}
		if err := json.Unmarshal(actual, &left); err != nil {
			t.Fatalf("%s json did not match, but unmarshalling failed. %v", description, err)
		}
		printer := formatter.NewAsciiFormatter(left, formatter.AsciiFormatterConfig{
			ShowArrayIndex: true,
		})
		output, err := printer.Format(diff)
		if err != nil {
			t.Errorf("%s did not match, but diff formatting failed. %v", description, err)
		} else {
			t.Errorf("%s json did not match expected.\n\n%s", description, output)
		}
	}
}

func mockHandler(statusCode int, getBody string, postBody string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		if r.Method == "GET" {
			w.Write([]byte(getBody))
		} else {
			w.Write([]byte(postBody))
		}
	})
}

func mockSlowHandler(delay time.Duration, statusCode int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)

		w.WriteHeader(statusCode)
		w.Write([]byte(body))
	})
}

type wellBehavedCache struct{}

func (c *wellBehavedCache) GetExtCacheData() (string, string) {
	return "www.pbcserver.com", "/pbcache/endpoint"
}

func (c *wellBehavedCache) PutJson(ctx context.Context, values []pbc.Cacheable) ([]string, []error) {
	ids := make([]string, len(values))
	for i := 0; i < len(values); i++ {
		ids[i] = strconv.Itoa(i)
	}
	return ids, nil
}

type emptyUsersync struct{}

func (e *emptyUsersync) GetId(bidder openrtb_ext.BidderName) (string, bool) {
	return "", false
}

type mockUsersync struct {
	syncs map[string]string
}

func (e *mockUsersync) GetId(bidder openrtb_ext.BidderName) (id string, exists bool) {
	id, exists = e.syncs[string(bidder)]
	return
}

type panicingAdapter struct{}

func (panicingAdapter) requestBid(ctx context.Context, request *openrtb.BidRequest, name openrtb_ext.BidderName, bidAdjustment float64, conversions currencies.Conversions, reqInfo *adapters.ExtraRequestInfo) (posb *pbsOrtbSeatBid, errs []error) {
	panic("Panic! Panic! The world is ending!")
}
