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

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	metricsConf "github.com/prebid/prebid-server/metrics/config"
	metricsConfig "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	pbc "github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/backends/file_fetcher"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
)

func TestNewExchange(t *testing.T) {
	respStatus := 200
	respBody := "{\"bid\":false}"
	server := httptest.NewServer(mockHandler(respStatus, "getBody", respBody))
	defer server.Close()

	knownAdapters := openrtb_ext.CoreBidderNames()

	cfg := &config.Configuration{
		CacheURL: config.Cache{
			ExpectedTimeMillis: 20,
		},
		Adapters: blankAdapterConfig(knownAdapters),
		GDPR: config.GDPR{
			EEACountries: []string{"FIN", "FRA", "GUF"},
		},
	}

	biddersInfo, err := config.LoadBidderInfoFromDisk("../static/bidder-info", cfg.Adapters, openrtb_ext.BuildBidderStringSlice())
	if err != nil {
		t.Fatal(err)
	}

	adapters, adaptersErr := BuildAdapters(server.Client(), cfg, biddersInfo, &metricsConf.DummyMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}

	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	e := NewExchange(adapters, nil, cfg, &metricsConf.DummyMetricsEngine{}, biddersInfo, gdpr.AlwaysAllow{}, currencyConverter, nilCategoryFetcher{}).(*exchange)
	for _, bidderName := range knownAdapters {
		if _, ok := e.adapterMap[bidderName]; !ok {
			t.Errorf("NewExchange produced an Exchange without bidder %s", bidderName)
		}
	}
	if e.cacheTime != time.Duration(cfg.CacheURL.ExpectedTimeMillis)*time.Millisecond {
		t.Errorf("Bad cacheTime. Expected 20 ms, got %s", e.cacheTime.String())
	}
}

// The objective is to get to execute e.buildBidResponse(ctx.Background(), liveA... ) (*openrtb2.BidResponse, error)
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
	// 1) Adapter with a '& char in its endpoint property
	//    https://github.com/prebid/prebid-server/issues/465
	cfg := &config.Configuration{
		Adapters: make(map[string]config.Adapter, 1),
	}
	cfg.Adapters["appnexus"] = config.Adapter{
		Endpoint: "http://ib.adnxs.com/openrtb2?query1&query2", //Note the '&' character in there
	}

	// 	2) Init new exchange with said configuration
	//Other parameters also needed to create exchange
	handlerNoBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	server := httptest.NewServer(http.HandlerFunc(handlerNoBidServer))
	defer server.Close()

	biddersInfo, err := config.LoadBidderInfoFromDisk("../static/bidder-info", cfg.Adapters, openrtb_ext.BuildBidderStringSlice())
	if err != nil {
		t.Fatal(err)
	}

	adapters, adaptersErr := BuildAdapters(server.Client(), cfg, biddersInfo, &metricsConf.DummyMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}

	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	e := NewExchange(adapters, nil, cfg, &metricsConf.DummyMetricsEngine{}, biddersInfo, gdpr.AlwaysAllow{}, currencyConverter, nilCategoryFetcher{}).(*exchange)

	// 	3) Build all the parameters e.buildBidResponse(ctx.Background(), liveA... ) needs
	//liveAdapters []openrtb_ext.BidderName,
	liveAdapters := make([]openrtb_ext.BidderName, 1)
	liveAdapters[0] = "appnexus"

	//adapterBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid,
	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid, 1)
	adapterBids["appnexus"] = &pbsOrtbSeatBid{currency: "USD"}

	//An openrtb2.BidRequest struct as specified in https://github.com/prebid/prebid-server/issues/465
	bidRequest := &openrtb2.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb2.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext:    json.RawMessage(`{"appnexus": {"placementId": 1}}`),
		}},
		Site:   &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
		Device: &openrtb2.Device{UA: "curl/7.54.0", IP: "::1"},
		AT:     1,
		TMax:   500,
		Ext:    json.RawMessage(`{"id": "some-request-id","site": {"page": "prebid.org"},"imp": [{"id": "some-impression-id","banner": {"format": [{"w": 300,"h": 250},{"w": 300,"h": 600}]},"ext": {"appnexus": {"placementId": 1}}}],"tmax": 500}`),
	}

	//adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra,
	adapterExtra := make(map[openrtb_ext.BidderName]*seatResponseExtra, 1)
	adapterExtra["appnexus"] = &seatResponseExtra{
		ResponseTimeMillis: 5,
		Errors:             []openrtb_ext.ExtBidderMessage{{Code: 999, Message: "Post ib.adnxs.com/openrtb2?query1&query2: unsupported protocol scheme \"\""}},
	}

	var errList []error

	// 	4) Build bid response
	bidResp, err := e.buildBidResponse(context.Background(), liveAdapters, adapterBids, bidRequest, adapterExtra, nil, nil, true, nil, errList)

	// 	5) Assert we have no errors and one '&' character as we are supposed to
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

// TestDebugBehaviour asserts the HttpCalls object is included inside the json "debug" field of the bidResponse extension when the
// openrtb2.BidRequest "Test" value is set to 1 or the openrtb2.BidRequest.Ext.Debug boolean field is set to true
func TestDebugBehaviour(t *testing.T) {

	// Define test cases
	type inTest struct {
		test  int8
		debug bool
	}
	type outTest struct {
		debugInfoIncluded bool
	}

	type debugData struct {
		bidderLevelDebugAllowed    bool
		accountLevelDebugAllowed   bool
		headerOverrideDebugAllowed bool
	}

	type aTest struct {
		desc             string
		in               inTest
		out              outTest
		debugData        debugData
		generateWarnings bool
	}
	testCases := []aTest{
		{
			desc:             "test flag equals zero, ext debug flag false, no debug info expected",
			in:               inTest{test: 0, debug: false},
			out:              outTest{debugInfoIncluded: false},
			debugData:        debugData{true, true, false},
			generateWarnings: false,
		},
		{
			desc:             "test flag equals zero, ext debug flag true, debug info expected",
			in:               inTest{test: 0, debug: true},
			out:              outTest{debugInfoIncluded: true},
			debugData:        debugData{true, true, false},
			generateWarnings: false,
		},
		{
			desc:             "test flag equals 1, ext debug flag false, debug info expected",
			in:               inTest{test: 1, debug: false},
			out:              outTest{debugInfoIncluded: true},
			debugData:        debugData{true, true, false},
			generateWarnings: false,
		},
		{
			desc:             "test flag equals 1, ext debug flag true, debug info expected",
			in:               inTest{test: 1, debug: true},
			out:              outTest{debugInfoIncluded: true},
			debugData:        debugData{true, true, false},
			generateWarnings: false,
		},
		{
			desc:             "test flag not equal to 0 nor 1, ext debug flag false, no debug info expected",
			in:               inTest{test: 2, debug: false},
			out:              outTest{debugInfoIncluded: false},
			debugData:        debugData{true, true, false},
			generateWarnings: false,
		},
		{
			desc:             "test flag not equal to 0 nor 1, ext debug flag true, debug info expected",
			in:               inTest{test: -1, debug: true},
			out:              outTest{debugInfoIncluded: true},
			debugData:        debugData{true, true, false},
			generateWarnings: true,
		},
		{
			desc:             "test account level debug disabled",
			in:               inTest{test: -1, debug: true},
			out:              outTest{debugInfoIncluded: false},
			debugData:        debugData{true, false, false},
			generateWarnings: true,
		},
		{
			desc:             "test header override enabled when all other debug options are disabled",
			in:               inTest{test: -1, debug: false},
			out:              outTest{debugInfoIncluded: true},
			debugData:        debugData{false, false, true},
			generateWarnings: false,
		},
		{
			desc:             "test header override and url debug options are enabled when all other debug options are disabled",
			in:               inTest{test: -1, debug: true},
			out:              outTest{debugInfoIncluded: true},
			debugData:        debugData{false, false, true},
			generateWarnings: false,
		},
		{
			desc:             "test header override and url and bidder debug options are enabled when account debug option is disabled",
			in:               inTest{test: -1, debug: true},
			out:              outTest{debugInfoIncluded: true},
			debugData:        debugData{true, false, true},
			generateWarnings: false,
		},
		{
			desc:             "test all debug options are enabled",
			in:               inTest{test: -1, debug: true},
			out:              outTest{debugInfoIncluded: true},
			debugData:        debugData{true, true, true},
			generateWarnings: false,
		},
	}

	// Set up test
	noBidServer := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}
	server := httptest.NewServer(http.HandlerFunc(noBidServer))
	defer server.Close()

	categoriesFetcher, err := newCategoryFetcher("./test/category-mapping")
	if err != nil {
		t.Errorf("Failed to create a category Fetcher: %v", err)
	}

	bidRequest := &openrtb2.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb2.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext:    json.RawMessage(`{"appnexus": {"placementId": 1}}`),
		}},
		Site:   &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
		Device: &openrtb2.Device{UA: "curl/7.54.0", IP: "::1"},
		AT:     1,
		TMax:   500,
	}

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
		bidResponse: &adapters.BidderResponse{},
	}

	e := new(exchange)

	e.cache = &wellBehavedCache{}
	e.me = &metricsConf.DummyMetricsEngine{}
	e.gDPR = gdpr.AlwaysAllow{}
	e.currencyConverter = currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	e.categoriesFetcher = categoriesFetcher

	ctx := context.Background()

	// Run tests
	for _, test := range testCases {

		e.adapterMap = map[openrtb_ext.BidderName]adaptedBidder{
			openrtb_ext.BidderAppnexus: adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, &config.DebugInfo{Allow: test.debugData.bidderLevelDebugAllowed}),
		}

		//request level debug key
		ctx = context.WithValue(ctx, DebugContextKey, test.in.debug)

		bidRequest.Test = test.in.test

		if test.in.debug {
			bidRequest.Ext = json.RawMessage(`{"prebid":{"debug":true}}`)
		} else {
			bidRequest.Ext = nil
		}

		auctionRequest := AuctionRequest{
			BidRequest: bidRequest,
			Account:    config.Account{DebugAllow: test.debugData.accountLevelDebugAllowed},
			UserSyncs:  &emptyUsersync{},
			StartTime:  time.Now(),
		}
		if test.generateWarnings {
			var errL []error
			errL = append(errL, &errortypes.Warning{
				Message:     fmt.Sprintf("CCPA consent test warning."),
				WarningCode: errortypes.InvalidPrivacyConsentWarningCode})
			auctionRequest.Warnings = errL
		}
		debugLog := &DebugLog{}
		if test.debugData.headerOverrideDebugAllowed {
			debugLog = &DebugLog{DebugOverride: true, DebugEnabledOrOverridden: true}
		}
		// Run test
		outBidResponse, err := e.HoldAuction(ctx, auctionRequest, debugLog)

		// Assert no HoldAuction error
		assert.NoErrorf(t, err, "%s. ex.HoldAuction returned an error: %v \n", test.desc, err)
		assert.NotNilf(t, outBidResponse.Ext, "%s. outBidResponse.Ext should not be nil \n", test.desc)

		actualExt := &openrtb_ext.ExtBidResponse{}
		err = json.Unmarshal(outBidResponse.Ext, actualExt)
		assert.NoErrorf(t, err, "%s. \"ext\" JSON field could not be unmarshaled. err: \"%v\" \n outBidResponse.Ext: \"%s\" \n", test.desc, err, outBidResponse.Ext)

		assert.NotEmpty(t, actualExt.Prebid, "%s. ext.prebid should not be empty")
		assert.NotEmpty(t, actualExt.Prebid.AuctionTimestamp, "%s. ext.prebid.auctiontimestamp should not be empty when AuctionRequest.StartTime is set")
		assert.Equal(t, auctionRequest.StartTime.UnixNano()/1e+6, actualExt.Prebid.AuctionTimestamp, "%s. ext.prebid.auctiontimestamp has incorrect value")

		if test.debugData.headerOverrideDebugAllowed {
			assert.Empty(t, actualExt.Warnings, "warnings should be empty")
			assert.Empty(t, actualExt.Errors, "errors should be empty")
		}

		if test.out.debugInfoIncluded {
			assert.NotNilf(t, actualExt, "%s. ext.debug field is expected to be included in this outBidResponse.Ext and not be nil.  outBidResponse.Ext.Debug = %v \n", test.desc, actualExt.Debug)

			// Assert "Debug fields
			assert.Greater(t, len(actualExt.Debug.HttpCalls), 0, "%s. ext.debug.httpcalls array should not be empty\n", test.desc)
			assert.Equal(t, server.URL, actualExt.Debug.HttpCalls["appnexus"][0].Uri, "%s. ext.debug.httpcalls array should not be empty\n", test.desc)
			assert.NotNilf(t, actualExt.Debug.ResolvedRequest, "%s. ext.debug.resolvedrequest field is expected to be included in this outBidResponse.Ext and not be nil.  outBidResponse.Ext.Debug = %v \n", test.desc, actualExt.Debug)

			// If not nil, assert bid extension
			if test.in.debug {
				diffJson(t, test.desc, bidRequest.Ext, actualExt.Debug.ResolvedRequest.Ext)
			}
		} else if !test.debugData.bidderLevelDebugAllowed && test.debugData.accountLevelDebugAllowed {
			assert.Equal(t, len(actualExt.Debug.HttpCalls), 0, "%s. ext.debug.httpcalls array should not be empty", "With bidder level debug disable option http calls should be empty")

		} else {
			assert.Nil(t, actualExt.Debug, "%s. ext.debug.httpcalls array should not be empty", "With bidder level debug disable option http calls should be empty")
		}

		if test.out.debugInfoIncluded && !test.debugData.accountLevelDebugAllowed && !test.debugData.headerOverrideDebugAllowed {
			assert.Len(t, actualExt.Warnings, 1, "warnings should have one warning")
			assert.NotNil(t, actualExt.Warnings["general"], "general warning should be present")
			assert.Equal(t, "debug turned off for account", actualExt.Warnings["general"][0].Message, "account debug disabled message should be present")
		}

		if !test.out.debugInfoIncluded && test.in.debug && test.debugData.accountLevelDebugAllowed && !test.debugData.headerOverrideDebugAllowed {
			if test.generateWarnings {
				assert.Len(t, actualExt.Warnings, 2, "warnings should have one warning")
			} else {
				assert.Len(t, actualExt.Warnings, 1, "warnings should have one warning")
			}
			assert.NotNil(t, actualExt.Warnings["appnexus"], "bidder warning should be present")
			assert.Equal(t, "debug turned off for bidder", actualExt.Warnings["appnexus"][0].Message, "account debug disabled message should be present")
		}

		if test.generateWarnings {
			assert.NotNil(t, actualExt.Warnings["general"], "general warning should be present")
			CCPAWarningPresent := false
			for _, warn := range actualExt.Warnings["general"] {
				if warn.Code == errortypes.InvalidPrivacyConsentWarningCode {
					CCPAWarningPresent = true
					break
				}
			}
			assert.True(t, CCPAWarningPresent, "CCPA Warning should be present")
		}

	}
}

func TestTwoBiddersDebugDisabledAndEnabled(t *testing.T) {

	type testCase struct {
		bidder1DebugEnabled bool
		bidder2DebugEnabled bool
	}

	testCases := []testCase{
		{
			bidder1DebugEnabled: true, bidder2DebugEnabled: true,
		},
		{
			bidder1DebugEnabled: true, bidder2DebugEnabled: false,
		},
		{
			bidder1DebugEnabled: false, bidder2DebugEnabled: true,
		},
		{
			bidder1DebugEnabled: false, bidder2DebugEnabled: false,
		},
	}

	// Set up test
	noBidServer := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}
	server := httptest.NewServer(http.HandlerFunc(noBidServer))
	defer server.Close()

	categoriesFetcher, err := newCategoryFetcher("./test/category-mapping")
	if err != nil {
		t.Errorf("Failed to create a category Fetcher: %v", err)
	}

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
		bidResponse: &adapters.BidderResponse{},
	}

	e := new(exchange)
	e.cache = &wellBehavedCache{}
	e.me = &metricsConf.DummyMetricsEngine{}
	e.gDPR = gdpr.AlwaysAllow{}
	e.currencyConverter = currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	e.categoriesFetcher = categoriesFetcher

	debugLog := DebugLog{Enabled: true}

	for _, testCase := range testCases {
		bidRequest := &openrtb2.BidRequest{
			ID: "some-request-id",
			Imp: []openrtb2.Imp{{
				ID:     "some-impression-id",
				Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
				Ext:    json.RawMessage(`{"telaria": {"placementId": 1}, "appnexus": {"placementid": 2}}`),
			}},
			Site:   &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
			Device: &openrtb2.Device{UA: "curl/7.54.0", IP: "::1"},
			AT:     1,
			TMax:   500,
		}

		bidRequest.Ext = json.RawMessage(`{"prebid":{"debug":true}}`)

		auctionRequest := AuctionRequest{
			BidRequest: bidRequest,
			Account:    config.Account{DebugAllow: true},
			UserSyncs:  &emptyUsersync{},
			StartTime:  time.Now(),
		}

		e.adapterMap = map[openrtb_ext.BidderName]adaptedBidder{
			openrtb_ext.BidderAppnexus: adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, &config.DebugInfo{Allow: testCase.bidder1DebugEnabled}),
			openrtb_ext.BidderTelaria:  adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, &config.DebugInfo{Allow: testCase.bidder2DebugEnabled}),
		}
		// Run test
		outBidResponse, err := e.HoldAuction(context.Background(), auctionRequest, &debugLog)
		// Assert no HoldAuction err
		assert.NoErrorf(t, err, "ex.HoldAuction returned an err")
		assert.NotNilf(t, outBidResponse.Ext, "outBidResponse.Ext should not be nil")

		actualExt := &openrtb_ext.ExtBidResponse{}
		err = json.Unmarshal(outBidResponse.Ext, actualExt)
		assert.NoErrorf(t, err, "JSON field unmarshaling err. ")

		assert.NotEmpty(t, actualExt.Prebid, "ext.prebid should not be empty")
		assert.NotEmpty(t, actualExt.Prebid.AuctionTimestamp, "ext.prebid.auctiontimestamp should not be empty when AuctionRequest.StartTime is set")
		assert.Equal(t, auctionRequest.StartTime.UnixNano()/1e+6, actualExt.Prebid.AuctionTimestamp, "ext.prebid.auctiontimestamp has incorrect value")

		assert.NotNilf(t, actualExt, "ext.debug field is expected to be included in this outBidResponse.Ext and not be nil")

		// Assert "Debug fields
		if testCase.bidder1DebugEnabled {
			assert.Equal(t, server.URL, actualExt.Debug.HttpCalls["appnexus"][0].Uri, "Url for bidder with debug enabled is incorrect")
			assert.NotNilf(t, actualExt.Debug.HttpCalls["appnexus"][0].RequestBody, "ext.debug.resolvedrequest field is expected to be included in this outBidResponse.Ext and not be nil")
		}
		if testCase.bidder2DebugEnabled {
			assert.Equal(t, server.URL, actualExt.Debug.HttpCalls["telaria"][0].Uri, "Url for bidder with debug enabled is incorrect")
			assert.NotNilf(t, actualExt.Debug.HttpCalls["telaria"][0].RequestBody, "ext.debug.resolvedrequest field is expected to be included in this outBidResponse.Ext and not be nil")
		}
		if !testCase.bidder1DebugEnabled {
			assert.Nil(t, actualExt.Debug.HttpCalls["appnexus"], "ext.debug.resolvedrequest field is expected to be included in this outBidResponse.Ext and not be nil")
		}
		if !testCase.bidder2DebugEnabled {
			assert.Nil(t, actualExt.Debug.HttpCalls["telaria"], "ext.debug.resolvedrequest field is expected to be included in this outBidResponse.Ext and not be nil")
		}

		if testCase.bidder1DebugEnabled && testCase.bidder2DebugEnabled {
			assert.Equal(t, 2, len(actualExt.Debug.HttpCalls), "With bidder level debug enable option for both bidders http calls should have 2 elements")
		}
	}

}

func TestOverrideWithCustomCurrency(t *testing.T) {

	mockCurrencyClient := &fakeCurrencyRatesHttpClient{
		responseBody: `{"dataAsOf":"2018-09-12","conversions":{"USD":{"MXN":10.00}}}`,
	}
	mockCurrencyConverter := currency.NewRateConverter(
		mockCurrencyClient,
		"currency.fake.com",
		24*time.Hour,
	)

	type testIn struct {
		customCurrencyRates json.RawMessage
		bidRequestCurrency  string
	}
	type testResults struct {
		numBids         int
		bidRespPrice    float64
		bidRespCurrency string
	}

	testCases := []struct {
		desc     string
		in       testIn
		expected testResults
	}{
		{
			desc: "Blank currency field in ext. bidRequest comes with a valid currency but conversion rate was not found in PBS. Return no bids",
			in: testIn{
				customCurrencyRates: json.RawMessage(`{ "prebid": { "currency": {} } } `),
				bidRequestCurrency:  "GBP",
			},
			expected: testResults{},
		},
		{
			desc: "valid request.ext.prebid.currency, expect custom rates to override those of the currency rate server",
			in: testIn{
				customCurrencyRates: json.RawMessage(`{
						  "prebid": {
						    "currency": {
						      "rates": {
						        "USD": {
						          "MXN": 20.00,
						          "EUR": 10.95
						        }
						      }
						    }
						  }
						}`),
				bidRequestCurrency: "MXN",
			},
			expected: testResults{
				numBids:         1,
				bidRespPrice:    20.00,
				bidRespCurrency: "MXN",
			},
		},
	}

	// Init mock currency conversion service
	mockCurrencyConverter.Run()

	// Init an exchange to run an auction from
	noBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	mockAppnexusBidService := httptest.NewServer(http.HandlerFunc(noBidServer))
	defer mockAppnexusBidService.Close()

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	oneDollarBidBidder := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     mockAppnexusBidService.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
	}

	e := new(exchange)
	e.cache = &wellBehavedCache{}
	e.me = &metricsConf.DummyMetricsEngine{}
	e.gDPR = gdpr.AlwaysAllow{}
	e.currencyConverter = mockCurrencyConverter
	e.categoriesFetcher = categoriesFetcher
	e.bidIDGenerator = &mockBidIDGenerator{false, false}

	// Define mock incoming bid requeset
	mockBidRequest := &openrtb2.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb2.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext:    json.RawMessage(`{"appnexus": {"placementId": 1}}`),
		}},
		Site: &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
	}

	// Run tests
	for _, test := range testCases {

		oneDollarBidBidder.bidResponse = &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{Price: 1.00},
				},
			},
			Currency: "USD",
		}

		e.adapterMap = map[openrtb_ext.BidderName]adaptedBidder{
			openrtb_ext.BidderAppnexus: adaptBidder(oneDollarBidBidder, mockAppnexusBidService.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, nil),
		}

		// Set custom rates in extension
		mockBidRequest.Ext = test.in.customCurrencyRates

		// Set bidRequest currency list
		mockBidRequest.Cur = []string{test.in.bidRequestCurrency}

		auctionRequest := AuctionRequest{
			BidRequest: mockBidRequest,
			Account:    config.Account{},
			UserSyncs:  &emptyUsersync{},
		}

		// Run test
		outBidResponse, err := e.HoldAuction(context.Background(), auctionRequest, &DebugLog{})

		// Assertions
		assert.NoErrorf(t, err, "%s. HoldAuction error: %v \n", test.desc, err)

		if test.expected.numBids > 0 {
			// Assert out currency
			assert.Equal(t, test.expected.bidRespCurrency, outBidResponse.Cur, "Bid response currency is wrong: %s \n", test.desc)

			// Assert returned bid
			if !assert.NotNil(t, outBidResponse, "outBidResponse is nil: %s \n", test.desc) {
				return
			}
			if !assert.NotEmpty(t, outBidResponse.SeatBid, "outBidResponse.SeatBid is empty: %s", test.desc) {
				return
			}
			if !assert.NotEmpty(t, outBidResponse.SeatBid[0].Bid, "outBidResponse.SeatBid[0].Bid is empty: %s", test.desc) {
				return
			}

			// Assert returned bid price matches the currency conversion
			assert.Equal(t, test.expected.bidRespPrice, outBidResponse.SeatBid[0].Bid[0].Price, "Bid response seatBid price is wrong: %s", test.desc)
		} else {
			assert.Len(t, outBidResponse.SeatBid, 0, "outBidResponse.SeatBid should be empty: %s", test.desc)
		}
	}
}

func TestAdapterCurrency(t *testing.T) {
	fakeCurrencyClient := &fakeCurrencyRatesHttpClient{
		responseBody: `{"dataAsOf":"2018-09-12","conversions":{"USD":{"MXN":10.00}}}`,
	}
	currencyConverter := currency.NewRateConverter(
		fakeCurrencyClient,
		"currency.fake.com",
		24*time.Hour,
	)
	currencyConverter.Run()

	// Initialize Mock Bidder
	// - Response purposefully causes PBS-Core to stop processing the request, since this test is only
	//   interested in the call to MakeRequests and nothing after.
	mockBidder := &mockBidder{}
	mockBidder.On("MakeRequests", mock.Anything, mock.Anything).Return([]*adapters.RequestData(nil), []error(nil))

	// Initialize Real Exchange
	e := exchange{
		cache:             &wellBehavedCache{},
		me:                &metricsConf.DummyMetricsEngine{},
		gDPR:              gdpr.AlwaysAllow{},
		currencyConverter: currencyConverter,
		categoriesFetcher: nilCategoryFetcher{},
		bidIDGenerator:    &mockBidIDGenerator{false, false},
		adapterMap: map[openrtb_ext.BidderName]adaptedBidder{
			openrtb_ext.BidderName("foo"): adaptBidder(mockBidder, nil, &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderName("foo"), nil),
		},
	}

	// Define Bid Request
	request := &openrtb2.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb2.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext:    json.RawMessage(`{"foo": {"placementId": 1}}`),
		}},
		Site: &openrtb2.Site{
			Page: "prebid.org",
			Ext:  json.RawMessage(`{"amp":0}`),
		},
		Cur: []string{"USD"},
		Ext: json.RawMessage(`{"prebid": {"currency": {"rates": {"USD": {"MXN": 20.00}}}}}`),
	}

	// Run Auction
	auctionRequest := AuctionRequest{
		BidRequest: request,
		Account:    config.Account{},
		UserSyncs:  &emptyUsersync{},
	}
	response, err := e.HoldAuction(context.Background(), auctionRequest, &DebugLog{})
	assert.NoError(t, err)
	assert.Equal(t, "some-request-id", response.ID, "Response ID")
	assert.Empty(t, response.SeatBid, "Response Bids")
	assert.Contains(t, string(response.Ext), `"errors":{"foo":[{"code":5,"message":"The adapter failed to generate any bid requests, but also failed to generate an error explaining why"}]}`, "Response Ext")

	// Test Currency Converter Properly Passed To Adapter
	if assert.NotNil(t, mockBidder.lastExtraRequestInfo, "Currency Conversion Argument") {
		converted, err := mockBidder.lastExtraRequestInfo.ConvertCurrency(2.0, "USD", "MXN")
		assert.NoError(t, err, "Currency Conversion Error")
		assert.Equal(t, 40.0, converted, "Currency Conversion Response")
	}
}

func TestGetAuctionCurrencyRates(t *testing.T) {

	pbsRates := map[string]map[string]float64{
		"MXN": {
			"USD": 20.13,
			"EUR": 27.82,
			"JPY": 5.09, // "MXN" to "JPY" rate not found in customRates
		},
	}

	customRates := map[string]map[string]float64{
		"MXN": {
			"USD": 25.00, // different rate than in pbsRates
			"EUR": 27.82, // same as in pbsRates
			"GBP": 31.12, // not found in pbsRates at all
		},
	}

	expectedRateEngineRates := map[string]map[string]float64{
		"MXN": {
			"USD": 25.00, // rates engine will prioritize the value found in custom rates
			"EUR": 27.82, // same value in both the engine reads the custom entry first
			"JPY": 5.09,  // the engine will find it in the pbsRates conversions
			"GBP": 31.12, // the engine will find it in the custom conversions
		},
	}

	boolTrue := true
	boolFalse := false

	type testInput struct {
		pbsRates       map[string]map[string]float64
		bidExtCurrency *openrtb_ext.ExtRequestCurrency
	}
	type testOutput struct {
		constantRates  bool
		resultingRates map[string]map[string]float64
	}
	testCases := []struct {
		desc     string
		given    testInput
		expected testOutput
	}{
		{
			"valid pbsRates, valid ConversionRates, false UsePBSRates. Resulting rates identical to customRates",
			testInput{
				pbsRates: pbsRates,
				bidExtCurrency: &openrtb_ext.ExtRequestCurrency{
					ConversionRates: customRates,
					UsePBSRates:     &boolFalse,
				},
			},
			testOutput{
				resultingRates: customRates,
			},
		},
		{
			"valid pbsRates, valid ConversionRates, true UsePBSRates. Resulting rates are a mix but customRates gets priority",
			testInput{
				pbsRates: pbsRates,
				bidExtCurrency: &openrtb_ext.ExtRequestCurrency{
					ConversionRates: customRates,
					UsePBSRates:     &boolTrue,
				},
			},
			testOutput{
				resultingRates: expectedRateEngineRates,
			},
		},
		{
			"nil pbsRates, valid ConversionRates, false UsePBSRates. Resulting rates identical to customRates",
			testInput{
				pbsRates: nil,
				bidExtCurrency: &openrtb_ext.ExtRequestCurrency{
					ConversionRates: customRates,
					UsePBSRates:     &boolFalse,
				},
			},
			testOutput{
				resultingRates: customRates,
			},
		},
		{
			"nil pbsRates, valid ConversionRates, true UsePBSRates. Resulting rates identical to customRates",
			testInput{
				pbsRates: nil,
				bidExtCurrency: &openrtb_ext.ExtRequestCurrency{
					ConversionRates: customRates,
					UsePBSRates:     &boolTrue,
				},
			},
			testOutput{
				resultingRates: customRates,
			},
		},
		{
			"valid pbsRates, empty ConversionRates, false UsePBSRates. Because pbsRates cannot be used, default to constant rates",
			testInput{
				pbsRates: pbsRates,
				bidExtCurrency: &openrtb_ext.ExtRequestCurrency{
					// ConversionRates inCustomRates not initialized makes for a zero-length map
					UsePBSRates: &boolFalse,
				},
			},
			testOutput{
				constantRates: true,
			},
		},
		{
			"valid pbsRates, nil ConversionRates, UsePBSRates defaults to true. Resulting rates will be identical to pbsRates",
			testInput{
				pbsRates:       pbsRates,
				bidExtCurrency: nil,
			},
			testOutput{
				resultingRates: pbsRates,
			},
		},
		{
			"nil pbsRates, empty ConversionRates, false UsePBSRates. Default to constant rates",
			testInput{
				pbsRates: nil,
				bidExtCurrency: &openrtb_ext.ExtRequestCurrency{
					// ConversionRates inCustomRates not initialized makes for a zero-length map
					UsePBSRates: &boolFalse,
				},
			},
			testOutput{
				constantRates: true,
			},
		},
		{
			"customRates empty, UsePBSRates set to true, pbsRates are nil. Return default constant rates converter",
			testInput{
				pbsRates: nil,
				bidExtCurrency: &openrtb_ext.ExtRequestCurrency{
					// ConversionRates inCustomRates not initialized makes for a zero-length map
					UsePBSRates: &boolTrue,
				},
			},
			testOutput{
				constantRates: true,
			},
		},
		{
			"nil customRates, nil pbsRates, UsePBSRates defaults to true. Return default constant rates converter",
			testInput{
				pbsRates:       nil,
				bidExtCurrency: nil,
			},
			testOutput{
				constantRates: true,
			},
		},
	}

	for _, tc := range testCases {

		// Test setup:
		jsonPbsRates, err := json.Marshal(tc.given.pbsRates)
		if err != nil {
			t.Fatalf("Failed to marshal PBS rates: %v", err)
		}

		// Init mock currency conversion service
		mockCurrencyClient := &fakeCurrencyRatesHttpClient{
			responseBody: `{"dataAsOf":"2018-09-12","conversions":` + string(jsonPbsRates) + `}`,
		}
		mockCurrencyConverter := currency.NewRateConverter(
			mockCurrencyClient,
			"currency.fake.com",
			24*time.Hour,
		)
		mockCurrencyConverter.Run()

		e := new(exchange)
		e.currencyConverter = mockCurrencyConverter

		// Run test
		auctionRates := e.getAuctionCurrencyRates(tc.given.bidExtCurrency)

		// When fromCurrency and toCurrency are the same, a rate of 1.00 is always expected
		rate, err := auctionRates.GetRate("USD", "USD")
		assert.NoError(t, err, tc.desc)
		assert.Equal(t, float64(1), rate, tc.desc)

		// If we expect an error, assert we have one along with a conversion rate of zero
		if tc.expected.constantRates {
			rate, err := auctionRates.GetRate("USD", "MXN")
			assert.Error(t, err, tc.desc)
			assert.Equal(t, float64(0), rate, tc.desc)
		} else {
			for fromCurrency, rates := range tc.expected.resultingRates {
				for toCurrency, expectedRate := range rates {
					actualRate, err := auctionRates.GetRate(fromCurrency, toCurrency)
					assert.NoError(t, err, tc.desc)
					assert.Equal(t, expectedRate, actualRate, tc.desc)
				}
			}
		}
	}
}

func TestReturnCreativeEndToEnd(t *testing.T) {
	sampleAd := "<?xml version=\"1.0\" encoding=\"UTF-8\"?><VAST ...></VAST>"

	// Define test cases
	type aTest struct {
		desc   string
		inExt  json.RawMessage
		outAdM string
	}
	testGroups := []struct {
		groupDesc   string
		testCases   []aTest
		expectError bool
	}{
		{
			groupDesc: "Invalid or malformed bidRequest Ext, expect error in these scenarios",
			testCases: []aTest{
				{
					desc:  "Malformed ext in bidRequest",
					inExt: json.RawMessage(`malformed`),
				},
				{
					desc:  "empty cache field",
					inExt: json.RawMessage(`{"prebid":{"cache":{}}}`),
				},
			},
			expectError: true,
		},
		{
			groupDesc: "Valid bidRequest Ext but no returnCreative value specified, default to returning creative",
			testCases: []aTest{
				{
					"Nil ext in bidRequest",
					nil,
					sampleAd,
				},
				{
					"empty ext",
					json.RawMessage(``),
					sampleAd,
				},
				{
					"bids doesn't come with returnCreative value",
					json.RawMessage(`{"prebid":{"cache":{"bids":{}}}}`),
					sampleAd,
				},
				{
					"vast doesn't come with returnCreative value",
					json.RawMessage(`{"prebid":{"cache":{"vastXml":{}}}}`),
					sampleAd,
				},
			},
		},
		{
			groupDesc: "Bids field comes with returnCreative value",
			testCases: []aTest{
				{
					"Bids returnCreative set to true, return ad markup in response",
					json.RawMessage(`{"prebid":{"cache":{"bids":{"returnCreative":true}}}}`),
					sampleAd,
				},
				{
					"Bids returnCreative set to false, don't return ad markup in response",
					json.RawMessage(`{"prebid":{"cache":{"bids":{"returnCreative":false}}}}`),
					"",
				},
			},
		},
		{
			groupDesc: "Vast field comes with returnCreative value",
			testCases: []aTest{
				{
					"Vast returnCreative set to true, return ad markup in response",
					json.RawMessage(`{"prebid":{"cache":{"vastXml":{"returnCreative":true}}}}`),
					sampleAd,
				},
				{
					"Vast returnCreative set to false, don't return ad markup in response",
					json.RawMessage(`{"prebid":{"cache":{"vastXml":{"returnCreative":false}}}}`),
					"",
				},
			},
		},
		{
			groupDesc: "Both Bids and Vast come with their own returnCreative value",
			testCases: []aTest{
				{
					"Both false, expect empty AdM",
					json.RawMessage(`{"prebid":{"cache":{"bids":{"returnCreative":false},"vastXml":{"returnCreative":false}}}}`),
					"",
				},
				{
					"Bids returnCreative is true, expect valid AdM",
					json.RawMessage(`{"prebid":{"cache":{"bids":{"returnCreative":true},"vastXml":{"returnCreative":false}}}}`),
					sampleAd,
				},
				{
					"Vast returnCreative is true, expect valid AdM",
					json.RawMessage(`{"prebid":{"cache":{"bids":{"returnCreative":false},"vastXml":{"returnCreative":true}}}}`),
					sampleAd,
				},
				{
					"Both field's returnCreative set to true, expect valid AdM",
					json.RawMessage(`{"prebid":{"cache":{"bids":{"returnCreative":true},"vastXml":{"returnCreative":true}}}}`),
					sampleAd,
				},
			},
		},
	}

	// Init an exchange to run an auction from
	noBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	server := httptest.NewServer(http.HandlerFunc(noBidServer))
	defer server.Close()

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
		bidResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{AdM: sampleAd},
				},
			},
		},
	}

	e := new(exchange)
	e.adapterMap = map[openrtb_ext.BidderName]adaptedBidder{
		openrtb_ext.BidderAppnexus: adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, nil),
	}
	e.cache = &wellBehavedCache{}
	e.me = &metricsConf.DummyMetricsEngine{}
	e.gDPR = gdpr.AlwaysAllow{}
	e.currencyConverter = currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	e.categoriesFetcher = categoriesFetcher
	e.bidIDGenerator = &mockBidIDGenerator{false, false}

	// Define mock incoming bid requeset
	mockBidRequest := &openrtb2.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb2.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext:    json.RawMessage(`{"appnexus": {"placementId": 1}}`),
		}},
		Site: &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
	}

	// Run tests
	for _, testGroup := range testGroups {
		for _, test := range testGroup.testCases {
			mockBidRequest.Ext = test.inExt

			auctionRequest := AuctionRequest{
				BidRequest: mockBidRequest,
				Account:    config.Account{},
				UserSyncs:  &emptyUsersync{},
			}

			// Run test
			debugLog := DebugLog{}
			outBidResponse, err := e.HoldAuction(context.Background(), auctionRequest, &debugLog)

			// Assert return error, if any
			if testGroup.expectError {
				assert.Errorf(t, err, "HoldAuction expected to throw error for: %s - %s. \n", testGroup.groupDesc, test.desc)
				continue
			} else {
				assert.NoErrorf(t, err, "%s: %s. HoldAuction error: %v \n", testGroup.groupDesc, test.desc, err)
			}

			// Assert returned bid
			if !assert.NotNil(t, outBidResponse, "%s: %s. outBidResponse is nil \n", testGroup.groupDesc, test.desc) {
				return
			}
			if !assert.NotEmpty(t, outBidResponse.SeatBid, "%s: %s. outBidResponse.SeatBid is empty \n", testGroup.groupDesc, test.desc) {
				return
			}
			if !assert.NotEmpty(t, outBidResponse.SeatBid[0].Bid, "%s: %s. outBidResponse.SeatBid[0].Bid is empty \n", testGroup.groupDesc, test.desc) {
				return
			}
			assert.Equal(t, test.outAdM, outBidResponse.SeatBid[0].Bid[0].AdM, "Ad markup string doesn't match in: %s - %s \n", testGroup.groupDesc, test.desc)
		}
	}
}

func TestGetBidCacheInfoEndToEnd(t *testing.T) {
	testUUID := "CACHE_UUID_1234"
	testExternalCacheScheme := "https"
	testExternalCacheHost := "www.externalprebidcache.net"
	testExternalCachePath := "endpoints/cache"

	// 1) An adapter
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
			Scheme: testExternalCacheScheme,
			Host:   testExternalCacheHost,
			Path:   testExternalCachePath,
		},
	}

	adapterList := make([]openrtb_ext.BidderName, 0, 2)
	testEngine := metricsConf.NewMetricsEngine(cfg, adapterList)
	// 	2) Init new exchange with said configuration
	handlerNoBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	server := httptest.NewServer(http.HandlerFunc(handlerNoBidServer))
	defer server.Close()

	biddersInfo, err := config.LoadBidderInfoFromDisk("../static/bidder-info", cfg.Adapters, openrtb_ext.BuildBidderStringSlice())
	if err != nil {
		t.Fatal(err)
	}

	adapters, adaptersErr := BuildAdapters(server.Client(), cfg, biddersInfo, &metricsConf.DummyMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	pbc := pbc.NewClient(&http.Client{}, &cfg.CacheURL, &cfg.ExtCacheURL, testEngine)
	e := NewExchange(adapters, pbc, cfg, &metricsConf.DummyMetricsEngine{}, biddersInfo, gdpr.AlwaysAllow{}, currencyConverter, nilCategoryFetcher{}).(*exchange)
	// 	3) Build all the parameters e.buildBidResponse(ctx.Background(), liveA... ) needs
	liveAdapters := []openrtb_ext.BidderName{bidderName}

	//adapterBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid,
	bids := []*openrtb2.Bid{
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
		cacheIds: map[*openrtb2.Bid]string{
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

	//adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra,
	adapterExtra := map[openrtb_ext.BidderName]*seatResponseExtra{
		bidderName: {
			ResponseTimeMillis: 5,
			Errors: []openrtb_ext.ExtBidderMessage{
				{
					Code:    999,
					Message: "Post ib.adnxs.com/openrtb2?query1&query2: unsupported protocol scheme \"\"",
				},
			},
		},
	}
	bidRequest := &openrtb2.BidRequest{
		ID:   "some-request-id",
		TMax: 1000,
		Imp: []openrtb2.Imp{
			{
				ID:     "test-div",
				Secure: openrtb2.Int8Ptr(0),
				Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
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
		Site: &openrtb2.Site{
			Page:      "http://rubitest.com/index.html",
			Publisher: &openrtb2.Publisher{ID: "1001"},
		},
		Test: 1,
		Ext:  json.RawMessage(`{"prebid": { "cache": { "bids": {}, "vastxml": {} }, "targeting": { "pricegranularity": "med", "includewinners": true, "includebidderkeys": false } }}`),
	}

	var errList []error

	// 	4) Build bid response
	bid_resp, err := e.buildBidResponse(context.Background(), liveAdapters, adapterBids, bidRequest, adapterExtra, auc, nil, true, nil, errList)

	// 	5) Assert we have no errors and the bid response we expected
	assert.NoError(t, err, "[TestGetBidCacheInfo] buildBidResponse() threw an error")

	expectedBidResponse := &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: string(bidderName),
				Bid: []openrtb2.Bid{
					{
						Ext: json.RawMessage(`{ "prebid": { "cache": { "bids": { "cacheId": "` + testUUID + `", "url": "` + testExternalCacheScheme + `://` + testExternalCacheHost + `/` + testExternalCachePath + `?uuid=` + testUUID + `" }, "key": "", "url": "" }`),
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

	// compare cache URL
	expCacheURL, err := jsonparser.GetString(expectedBidResponse.SeatBid[0].Bid[0].Ext, "prebid", "cache", "bids", "url")
	assert.NoErrorf(t, err, "[TestGetBidCacheInfo] Error found while trying to json parse the url field from expected build response. Message: %v \n", err)

	cacheURL, err := jsonparser.GetString(bid_resp.SeatBid[0].Bid[0].Ext, "prebid", "cache", "bids", "url")
	assert.NoErrorf(t, err, "[TestGetBidCacheInfo] Error found while trying to json parse the url field from actual build response. Message: %v \n", err)

	assert.Equal(t, expCacheURL, cacheURL, "[TestGetBidCacheInfo] cacheId field in ext should equal \"%s\" \n", expCacheURL)
}

func TestBidReturnsCreative(t *testing.T) {
	sampleAd := "<?xml version=\"1.0\" encoding=\"UTF-8\"?><VAST ...></VAST>"
	sampleOpenrtbBid := &openrtb2.Bid{ID: "some-bid-id", AdM: sampleAd}

	// Define test cases
	testCases := []struct {
		description            string
		inReturnCreative       bool
		expectedCreativeMarkup string
	}{
		{
			"returnCreative set to true, expect a full creative markup string in returned bid",
			true,
			sampleAd,
		},
		{
			"returnCreative set to false, expect empty creative markup string in returned bid",
			false,
			"",
		},
	}

	// Test set up
	sampleBids := []*pbsOrtbBid{
		{
			bid:            sampleOpenrtbBid,
			bidType:        openrtb_ext.BidTypeBanner,
			bidTargets:     map[string]string{},
			generatedBidID: "randomId",
		},
	}
	sampleAuction := &auction{cacheIds: map[*openrtb2.Bid]string{sampleOpenrtbBid: "CACHE_UUID_1234"}}

	noBidHandler := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	server := httptest.NewServer(http.HandlerFunc(noBidHandler))
	defer server.Close()

	bidderImpl := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     server.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
		bidResponse: &adapters.BidderResponse{},
	}
	e := new(exchange)
	e.adapterMap = map[openrtb_ext.BidderName]adaptedBidder{
		openrtb_ext.BidderAppnexus: adaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.DummyMetricsEngine{}, openrtb_ext.BidderAppnexus, nil),
	}
	e.cache = &wellBehavedCache{}
	e.me = &metricsConf.DummyMetricsEngine{}
	e.gDPR = gdpr.AlwaysAllow{}
	e.currencyConverter = currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	//Run tests
	for _, test := range testCases {
		resultingBids, resultingErrs := e.makeBid(sampleBids, sampleAuction, test.inReturnCreative, nil)

		assert.Equal(t, 0, len(resultingErrs), "%s. Test should not return errors \n", test.description)
		assert.Equal(t, test.expectedCreativeMarkup, resultingBids[0].AdM, "%s. Ad markup string doesn't match expected \n", test.description)

		var bidExt openrtb_ext.ExtBid
		json.Unmarshal(resultingBids[0].Ext, &bidExt)
		assert.Equal(t, 0, bidExt.Prebid.DealPriority, "%s. Test should have DealPriority set to 0", test.description)
		assert.Equal(t, false, bidExt.Prebid.DealTierSatisfied, "%s. Test should have DealTierSatisfied set to false", test.description)
	}
}

func TestGetBidCacheInfo(t *testing.T) {
	bid := &openrtb2.Bid{ID: "42"}
	testCases := []struct {
		description      string
		scheme           string
		host             string
		path             string
		bid              *pbsOrtbBid
		auction          *auction
		expectedFound    bool
		expectedCacheID  string
		expectedCacheURL string
	}{
		{
			description:      "JSON Cache ID",
			scheme:           "https",
			host:             "prebid.org",
			path:             "cache",
			bid:              &pbsOrtbBid{bid: bid},
			auction:          &auction{cacheIds: map[*openrtb2.Bid]string{bid: "anyID"}},
			expectedFound:    true,
			expectedCacheID:  "anyID",
			expectedCacheURL: "https://prebid.org/cache?uuid=anyID",
		},
		{
			description:      "VAST Cache ID",
			scheme:           "https",
			host:             "prebid.org",
			path:             "cache",
			bid:              &pbsOrtbBid{bid: bid},
			auction:          &auction{vastCacheIds: map[*openrtb2.Bid]string{bid: "anyID"}},
			expectedFound:    true,
			expectedCacheID:  "anyID",
			expectedCacheURL: "https://prebid.org/cache?uuid=anyID",
		},
		{
			description:      "Cache ID Not Found",
			scheme:           "https",
			host:             "prebid.org",
			path:             "cache",
			bid:              &pbsOrtbBid{bid: bid},
			auction:          &auction{},
			expectedFound:    false,
			expectedCacheID:  "",
			expectedCacheURL: "",
		},
		{
			description:      "Scheme Not Provided",
			host:             "prebid.org",
			path:             "cache",
			bid:              &pbsOrtbBid{bid: bid},
			auction:          &auction{cacheIds: map[*openrtb2.Bid]string{bid: "anyID"}},
			expectedFound:    true,
			expectedCacheID:  "anyID",
			expectedCacheURL: "prebid.org/cache?uuid=anyID",
		},
		{
			description:      "Host And Path Not Provided - Without Scheme",
			bid:              &pbsOrtbBid{bid: bid},
			auction:          &auction{cacheIds: map[*openrtb2.Bid]string{bid: "anyID"}},
			expectedFound:    true,
			expectedCacheID:  "anyID",
			expectedCacheURL: "",
		},
		{
			description:      "Host And Path Not Provided - With Scheme",
			scheme:           "https",
			bid:              &pbsOrtbBid{bid: bid},
			auction:          &auction{cacheIds: map[*openrtb2.Bid]string{bid: "anyID"}},
			expectedFound:    true,
			expectedCacheID:  "anyID",
			expectedCacheURL: "",
		},
		{
			description:      "Nil Bid",
			scheme:           "https",
			host:             "prebid.org",
			path:             "cache",
			bid:              nil,
			auction:          &auction{cacheIds: map[*openrtb2.Bid]string{bid: "anyID"}},
			expectedFound:    false,
			expectedCacheID:  "",
			expectedCacheURL: "",
		},
		{
			description:      "Nil Embedded Bid",
			scheme:           "https",
			host:             "prebid.org",
			path:             "cache",
			bid:              &pbsOrtbBid{bid: nil},
			auction:          &auction{cacheIds: map[*openrtb2.Bid]string{bid: "anyID"}},
			expectedFound:    false,
			expectedCacheID:  "",
			expectedCacheURL: "",
		},
		{
			description:      "Nil Auction",
			scheme:           "https",
			host:             "prebid.org",
			path:             "cache",
			bid:              &pbsOrtbBid{bid: bid},
			auction:          nil,
			expectedFound:    false,
			expectedCacheID:  "",
			expectedCacheURL: "",
		},
	}

	for _, test := range testCases {
		exchange := &exchange{
			cache: &mockCache{
				scheme: test.scheme,
				host:   test.host,
				path:   test.path,
			},
		}

		cacheInfo, found := exchange.getBidCacheInfo(test.bid, test.auction)

		assert.Equal(t, test.expectedFound, found, test.description+":found")
		assert.Equal(t, test.expectedCacheID, cacheInfo.CacheId, test.description+":id")
		assert.Equal(t, test.expectedCacheURL, cacheInfo.Url, test.description+":url")
	}
}

func TestBidResponseCurrency(t *testing.T) {
	// Init objects
	cfg := &config.Configuration{Adapters: make(map[string]config.Adapter, 1)}
	cfg.Adapters["appnexus"] = config.Adapter{Endpoint: "http://ib.adnxs.com"}

	handlerNoBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	server := httptest.NewServer(http.HandlerFunc(handlerNoBidServer))
	defer server.Close()

	biddersInfo, err := config.LoadBidderInfoFromDisk("../static/bidder-info", cfg.Adapters, openrtb_ext.BuildBidderStringSlice())
	if err != nil {
		t.Fatal(err)
	}

	adapters, adaptersErr := BuildAdapters(server.Client(), cfg, biddersInfo, &metricsConf.DummyMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}

	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	e := NewExchange(adapters, nil, cfg, &metricsConf.DummyMetricsEngine{}, biddersInfo, gdpr.AlwaysAllow{}, currencyConverter, nilCategoryFetcher{}).(*exchange)

	liveAdapters := make([]openrtb_ext.BidderName, 1)
	liveAdapters[0] = "appnexus"

	bidRequest := &openrtb2.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb2.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext:    json.RawMessage(`{"appnexus": {"placementId": 10433394}}`),
		}},
		Site:   &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
		Device: &openrtb2.Device{UA: "curl/7.54.0", IP: "::1"},
		AT:     1,
		TMax:   500,
		Ext:    json.RawMessage(`{"id": "some-request-id","site": {"page": "prebid.org"},"imp": [{"id": "some-impression-id","banner": {"format": [{"w": 300,"h": 250},{"w": 300,"h": 600}]},"ext": {"appnexus": {"placementId": 10433394}}}],"tmax": 500}`),
	}

	adapterExtra := map[openrtb_ext.BidderName]*seatResponseExtra{
		"appnexus": {ResponseTimeMillis: 5},
	}

	var errList []error

	sampleBid := &openrtb2.Bid{
		ID:    "some-imp-id",
		Price: 9.517803,
		W:     300,
		H:     250,
		Ext:   nil,
	}
	aPbsOrtbBidArr := []*pbsOrtbBid{{bid: sampleBid, bidType: openrtb_ext.BidTypeBanner}}
	sampleSeatBid := []openrtb2.SeatBid{
		{
			Seat: "appnexus",
			Bid: []openrtb2.Bid{
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
	emptySeatBid := []openrtb2.SeatBid{}

	// Test cases
	type aTest struct {
		description         string
		adapterBids         map[openrtb_ext.BidderName]*pbsOrtbSeatBid
		expectedBidResponse *openrtb2.BidResponse
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
			expectedBidResponse: &openrtb2.BidResponse{
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
			expectedBidResponse: &openrtb2.BidResponse{
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
			expectedBidResponse: &openrtb2.BidResponse{
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
			expectedBidResponse: &openrtb2.BidResponse{
				ID:      "some-request-id",
				SeatBid: emptySeatBid,
				Cur:     "",
				Ext: json.RawMessage(`{"responsetimemillis":{"appnexus":5},"tmaxrequest":500}
`),
			},
		},
	}

	bidResponseExt := &openrtb_ext.ExtBidResponse{
		ResponseTimeMillis:   map[openrtb_ext.BidderName]int{openrtb_ext.BidderName("appnexus"): 5},
		RequestTimeoutMillis: 500,
	}
	// Run tests
	for i := range testCases {
		actualBidResp, err := e.buildBidResponse(context.Background(), liveAdapters, testCases[i].adapterBids, bidRequest, adapterExtra, nil, bidResponseExt, true, nil, errList)
		assert.NoError(t, err, fmt.Sprintf("[TEST_FAILED] e.buildBidResponse resturns error in test: %s Error message: %s \n", testCases[i].description, err))
		assert.Equalf(t, testCases[i].expectedBidResponse, actualBidResp, fmt.Sprintf("[TEST_FAILED] Objects must be equal for test: %s \n Expected: >>%s<< \n Actual: >>%s<< ", testCases[i].description, testCases[i].expectedBidResponse.Ext, actualBidResp.Ext))
	}
}

func TestBidResponseImpExtInfo(t *testing.T) {
	// Init objects
	cfg := &config.Configuration{Adapters: make(map[string]config.Adapter, 1)}
	cfg.Adapters["appnexus"] = config.Adapter{Endpoint: "http://ib.adnxs.com"}

	e := NewExchange(nil, nil, cfg, &metricsConf.DummyMetricsEngine{}, nil, gdpr.AlwaysAllow{}, nil, nilCategoryFetcher{}).(*exchange)

	liveAdapters := make([]openrtb_ext.BidderName, 1)
	liveAdapters[0] = "appnexus"

	bidRequest := &openrtb2.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb2.Imp{{
			ID:    "some-impression-id",
			Video: &openrtb2.Video{},
			Ext:   json.RawMessage(`{"appnexus": {"placementId": 10433394}}`),
		}},
		Ext: json.RawMessage(``),
	}

	var errList []error

	sampleBid := &openrtb2.Bid{
		ID:    "some-imp-id",
		ImpID: "some-impression-id",
		W:     300,
		H:     250,
		Ext:   nil,
	}
	aPbsOrtbBidArr := []*pbsOrtbBid{{bid: sampleBid, bidType: openrtb_ext.BidTypeVideo}}

	adapterBids := map[openrtb_ext.BidderName]*pbsOrtbSeatBid{
		openrtb_ext.BidderName("appnexus"): {
			bids: aPbsOrtbBidArr,
		},
	}

	impExtInfo := make(map[string]ImpExtInfo, 1)
	impExtInfo["some-impression-id"] = ImpExtInfo{
		true,
		[]byte(`{"video":{"h":480,"mimes":["video/mp4"]}}`)}

	expectedBidResponseExt := `{"prebid":{"type":"video"},"storedrequestattributes":{"h":480,"mimes":["video/mp4"]}}`

	actualBidResp, err := e.buildBidResponse(context.Background(), liveAdapters, adapterBids, bidRequest, nil, nil, nil, true, impExtInfo, errList)
	assert.NoError(t, err, fmt.Sprintf("imp ext info was not passed through correctly: %s", err))

	resBidExt := string(actualBidResp.SeatBid[0].Bid[0].Ext)
	assert.Equalf(t, expectedBidResponseExt, resBidExt, "Expected bid response extension is incorrect")

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
		Adapters: make(map[string]config.Adapter),
	}
	for _, bidder := range openrtb_ext.CoreBidderNames() {
		cfg.Adapters[strings.ToLower(string(bidder))] = config.Adapter{
			Endpoint: server.URL,
		}
	}
	cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAudienceNetwork))] = config.Adapter{
		Endpoint:   server.URL,
		AppSecret:  "any",
		PlatformID: "abc",
	}
	cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderBeachfront))] = config.Adapter{
		Endpoint:         server.URL,
		ExtraAdapterInfo: "{\"video_endpoint\":\"" + server.URL + "\"}",
	}

	biddersInfo, err := config.LoadBidderInfoFromDisk("../static/bidder-info", cfg.Adapters, openrtb_ext.BuildBidderStringSlice())
	if err != nil {
		t.Fatal(err)
	}

	adapters, adaptersErr := BuildAdapters(server.Client(), cfg, biddersInfo, &metricsConf.DummyMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}

	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	auctionRequest := AuctionRequest{
		BidRequest: newRaceCheckingRequest(t),
		Account:    config.Account{},
		UserSyncs:  &emptyUsersync{},
	}

	debugLog := DebugLog{}
	ex := NewExchange(adapters, &wellBehavedCache{}, cfg, &metricsConf.DummyMetricsEngine{}, biddersInfo, gdpr.AlwaysAllow{}, currencyConverter, &nilCategoryFetcher{}).(*exchange)
	_, err = ex.HoldAuction(context.Background(), auctionRequest, &debugLog)
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
func newRaceCheckingRequest(t *testing.T) *openrtb2.BidRequest {
	dnt := int8(1)
	return &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Page:   "www.some.domain.com",
			Domain: "domain.com",
			Publisher: &openrtb2.Publisher{
				ID: "some-publisher-id",
			},
		},
		Device: &openrtb2.Device{
			UA:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/67.0.3396.87 Safari/537.36",
			IFA:      "ifa",
			IP:       "132.173.230.74",
			DNT:      &dnt,
			Language: "EN",
		},
		Source: &openrtb2.Source{
			TID: "61018dc9-fa61-4c41-b7dc-f90b9ae80e87",
		},
		User: &openrtb2.User{
			ID:       "our-id",
			BuyerUID: "their-id",
			Ext:      json.RawMessage(`{"consent":"BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw"}`),
		},
		Regs: &openrtb2.Regs{
			COPPA: 1,
			Ext:   json.RawMessage(`{"gdpr":1}`),
		},
		Imp: []openrtb2.Imp{{
			ID: "some-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{{
					W: 300,
					H: 250,
				}, {
					W: 300,
					H: 600,
				}},
			},
			Ext: buildImpExt(t, "banner"),
		}, {
			Video: &openrtb2.Video{
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
		Adapters: blankAdapterConfig(openrtb_ext.CoreBidderNames()),
	}

	biddersInfo, err := config.LoadBidderInfoFromDisk("../static/bidder-info", cfg.Adapters, openrtb_ext.BuildBidderStringSlice())
	if err != nil {
		t.Fatal(err)
	}

	adapters, adaptersErr := BuildAdapters(&http.Client{}, cfg, biddersInfo, &metricsConf.DummyMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}

	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	e := NewExchange(adapters, nil, cfg, &metricsConf.DummyMetricsEngine{}, biddersInfo, gdpr.AlwaysAllow{}, currencyConverter, nilCategoryFetcher{}).(*exchange)

	chBids := make(chan *bidResponseWrapper, 1)
	panicker := func(bidderRequest BidderRequest, conversions currency.Conversions) {
		panic("panic!")
	}

	apnLabels := metrics.AdapterLabels{
		Source:      metrics.DemandWeb,
		RType:       metrics.ReqTypeORTB2Web,
		Adapter:     openrtb_ext.BidderAppnexus,
		PubID:       "test1",
		CookieFlag:  metrics.CookieFlagYes,
		AdapterBids: metrics.AdapterBidNone,
	}

	bidderRequests := []BidderRequest{
		{
			BidderName:     "bidder1",
			BidderCoreName: "appnexus",
			BidderLabels:   apnLabels,
			BidRequest: &openrtb2.BidRequest{
				ID: "b-1",
			},
		},
		{
			BidderName:     "bidder2",
			BidderCoreName: "bidder2",
			BidRequest: &openrtb2.BidRequest{
				ID: "b-2",
			},
		},
	}

	recovered := e.recoverSafely(bidderRequests, panicker, chBids)
	recovered(bidderRequests[0], nil)
}

func buildImpExt(t *testing.T, jsonFilename string) json.RawMessage {
	adapterFolders, err := ioutil.ReadDir("../adapters")
	if err != nil {
		t.Fatalf("Failed to open adapters directory: %v", err)
	}
	bidderExts := make(map[string]json.RawMessage)
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
		Adapters: make(map[string]config.Adapter),
	}
	for _, bidder := range openrtb_ext.CoreBidderNames() {
		cfg.Adapters[strings.ToLower(string(bidder))] = config.Adapter{
			Endpoint: server.URL,
		}
	}
	cfg.Adapters["audiencenetwork"] = config.Adapter{Disabled: true}

	biddersInfo, err := config.LoadBidderInfoFromDisk("../static/bidder-info", cfg.Adapters, openrtb_ext.BuildBidderStringSlice())
	if err != nil {
		t.Fatal(err)
	}

	adapters, adaptersErr := BuildAdapters(server.Client(), cfg, biddersInfo, &metricsConf.DummyMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}

	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	e := NewExchange(adapters, &mockCache{}, cfg, &metricsConf.DummyMetricsEngine{}, biddersInfo, gdpr.AlwaysAllow{}, currencyConverter, categoriesFetcher).(*exchange)

	e.adapterMap[openrtb_ext.BidderBeachfront] = panicingAdapter{}
	e.adapterMap[openrtb_ext.BidderAppnexus] = panicingAdapter{}

	request := &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Page:   "www.some.domain.com",
			Domain: "domain.com",
			Publisher: &openrtb2.Publisher{
				ID: "some-publisher-id",
			},
		},
		User: &openrtb2.User{
			ID:       "our-id",
			BuyerUID: "their-id",
			Ext:      json.RawMessage(`{"consent":"BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw"}`),
		},
		Imp: []openrtb2.Imp{{
			ID: "some-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{{
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

	auctionRequest := AuctionRequest{
		BidRequest: request,
		Account:    config.Account{},
		UserSyncs:  &emptyUsersync{},
	}
	debugLog := DebugLog{}
	_, err = e.HoldAuction(context.Background(), auctionRequest, &debugLog)
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

func TestSetDebugContextKey(t *testing.T) {
	// Test cases
	testCases := []struct {
		desc              string
		inDebugInfo       bool
		expectedDebugInfo bool
	}{
		{
			desc:              "debugInfo flag on, we expect to find DebugContextKey key in context",
			inDebugInfo:       true,
			expectedDebugInfo: true,
		},
		{
			desc:              "debugInfo flag off, we don't expect to find DebugContextKey key in context",
			inDebugInfo:       false,
			expectedDebugInfo: false,
		},
	}

	// Setup test
	ex := exchange{}

	// Run tests
	for _, test := range testCases {
		auctionCtx := ex.makeDebugContext(context.Background(), test.inDebugInfo)

		debugInfo := auctionCtx.Value(DebugContextKey)
		assert.NotNil(t, debugInfo, "%s. Flag set, `debugInfo` shouldn't be nil")
		assert.Equal(t, test.expectedDebugInfo, debugInfo.(bool), "Desc: %s. Incorrect value mapped to DebugContextKey(`debugInfo`) in the context\n", test.desc)
	}
}

// TestExchangeJSON executes tests for all the *.json files in exchangetest.
func TestExchangeJSON(t *testing.T) {
	if specFiles, err := ioutil.ReadDir("./exchangetest"); err == nil {
		for _, specFile := range specFiles {
			fileName := "./exchangetest/" + specFile.Name()
			fileDisplayName := "exchange/exchangetest/" + specFile.Name()
			t.Run(fileDisplayName, func(t *testing.T) {
				specData, err := loadFile(fileName)
				if assert.NoError(t, err, "Failed to load contents of file %s: %v", fileDisplayName, err) {
					runSpec(t, fileDisplayName, specData)
				}
			})
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

	var s struct{}
	eeac := make(map[string]struct{})
	for _, c := range []string{"FIN", "FRA", "GUF"} {
		eeac[c] = s
	}

	var gdprDefaultValue string
	if spec.AssumeGDPRApplies {
		gdprDefaultValue = "1"
	} else {
		gdprDefaultValue = "0"
	}

	privacyConfig := config.Privacy{
		CCPA: config.CCPA{
			Enforce: spec.EnforceCCPA,
		},
		LMT: config.LMT{
			Enforce: spec.EnforceLMT,
		},
		GDPR: config.GDPR{
			Enabled:         spec.GDPREnabled,
			DefaultValue:    gdprDefaultValue,
			EEACountriesMap: eeac,
		},
	}
	bidIdGenerator := &mockBidIDGenerator{}
	if spec.BidIDGenerator != nil {
		*bidIdGenerator = *spec.BidIDGenerator
	}
	ex := newExchangeForTests(t, filename, spec.OutgoingRequests, aliases, privacyConfig, bidIdGenerator)
	biddersInAuction := findBiddersInAuction(t, filename, &spec.IncomingRequest.OrtbRequest)
	debugLog := &DebugLog{}
	if spec.DebugLog != nil {
		*debugLog = *spec.DebugLog
		debugLog.Regexp = regexp.MustCompile(`[<>]`)
	}

	auctionRequest := AuctionRequest{
		BidRequest: &spec.IncomingRequest.OrtbRequest,
		Account: config.Account{
			ID:            "testaccount",
			EventsEnabled: spec.EventsEnabled,
			DebugAllow:    true,
		},
		UserSyncs: mockIdFetcher(spec.IncomingRequest.Usersyncs),
	}
	if spec.StartTime > 0 {
		auctionRequest.StartTime = time.Unix(0, spec.StartTime*1e+6)
	}
	ctx := context.Background()
	ctx = context.WithValue(ctx, DebugContextKey, true)

	bid, err := ex.HoldAuction(ctx, auctionRequest, debugLog)
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

func findBiddersInAuction(t *testing.T, context string, req *openrtb2.BidRequest) []string {
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
func extractResponseTimes(t *testing.T, context string, bid *openrtb2.BidResponse) map[string]int {
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

func newExchangeForTests(t *testing.T, filename string, expectations map[string]*bidderSpec, aliases map[string]string, privacyConfig config.Privacy, bidIDGenerator BidIDGenerator) Exchange {
	bidderAdapters := make(map[openrtb_ext.BidderName]adaptedBidder, len(expectations))
	bidderInfos := make(config.BidderInfos, len(expectations))
	for _, bidderName := range openrtb_ext.CoreBidderNames() {
		if spec, ok := expectations[string(bidderName)]; ok {
			bidderAdapters[bidderName] = &validatingBidder{
				t:             t,
				fileName:      filename,
				bidderName:    string(bidderName),
				expectations:  map[string]*bidderRequest{string(bidderName): spec.ExpectedRequest},
				mockResponses: map[string]bidderResponse{string(bidderName): spec.MockResponse},
			}
			bidderInfos[string(bidderName)] = config.BidderInfo{ModifyingVastXmlAllowed: spec.ModifyingVastXmlAllowed}
		}
	}

	for alias, coreBidder := range aliases {
		if spec, ok := expectations[alias]; ok {
			if bidder, ok := bidderAdapters[openrtb_ext.BidderName(coreBidder)]; ok {
				bidder.(*validatingBidder).expectations[alias] = spec.ExpectedRequest
				bidder.(*validatingBidder).mockResponses[alias] = spec.MockResponse
			} else {
				bidderAdapters[openrtb_ext.BidderName(coreBidder)] = &validatingBidder{
					t:             t,
					fileName:      filename,
					bidderName:    coreBidder,
					expectations:  map[string]*bidderRequest{alias: spec.ExpectedRequest},
					mockResponses: map[string]bidderResponse{alias: spec.MockResponse},
				}
			}
		}
	}

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Fatalf("Failed to create a category Fetcher: %v", error)
	}

	gdprDefaultValue := gdpr.SignalYes
	if privacyConfig.GDPR.DefaultValue == "0" {
		gdprDefaultValue = gdpr.SignalNo
	}

	return &exchange{
		adapterMap:        bidderAdapters,
		me:                metricsConf.NewMetricsEngine(&config.Configuration{}, openrtb_ext.CoreBidderNames()),
		cache:             &wellBehavedCache{},
		cacheTime:         0,
		gDPR:              &permissionsMock{allowAllBidders: true},
		currencyConverter: currency.NewRateConverter(&http.Client{}, "", time.Duration(0)),
		gdprDefaultValue:  gdprDefaultValue,
		privacyConfig:     privacyConfig,
		categoriesFetcher: categoriesFetcher,
		bidderInfo:        bidderInfos,
		externalURL:       "http://localhost",
		bidIDGenerator:    bidIDGenerator,
	}
}

type mockBidIDGenerator struct {
	GenerateBidID bool `json:"generateBidID"`
	ReturnError   bool `json:"returnError"`
}

func (big *mockBidIDGenerator) Enabled() bool {
	return big.GenerateBidID
}

func (big *mockBidIDGenerator) New() (string, error) {

	if big.ReturnError {
		err := errors.New("Test error generating bid.ext.prebid.bidid")
		return "", err
	}
	return "mock_uuid", nil

}

type fakeRandomDeduplicateBidBooleanGenerator struct {
	returnValue bool
}

func (m *fakeRandomDeduplicateBidBooleanGenerator) Generate() bool {
	return m.returnValue
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
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 20.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb2.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 30.0000, Cat: cats3, W: 1, H: 1}
	bid4 := openrtb2.Bid{ID: "bid_id4", ImpID: "imp_id4", Price: 40.0000, Cat: cats4, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_2 := pbsOrtbBid{&bid2, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 40}, nil, 0, false, ""}
	bid1_3 := pbsOrtbBid{&bid3, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30, PrimaryCategory: "AdapterOverride"}, nil, 0, false, ""}
	bid1_4 := pbsOrtbBid{&bid4, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}

	innerBids := []*pbsOrtbBid{
		&bid1_1,
		&bid1_2,
		&bid1_3,
		&bid1_4,
	}

	seatBid := pbsOrtbSeatBid{bids: innerBids, currency: "USD"}
	bidderName1 := openrtb_ext.BidderName("appnexus")

	adapterBids[bidderName1] = &seatBid

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, &requestExt, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{})

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
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 20.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb2.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 30.0000, Cat: cats3, W: 1, H: 1}
	bid4 := openrtb2.Bid{ID: "bid_id4", ImpID: "imp_id4", Price: 40.0000, Cat: cats4, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_2 := pbsOrtbBid{&bid2, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 40}, nil, 0, false, ""}
	bid1_3 := pbsOrtbBid{&bid3, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30, PrimaryCategory: "AdapterOverride"}, nil, 0, false, ""}
	bid1_4 := pbsOrtbBid{&bid4, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 50}, nil, 0, false, ""}

	innerBids := []*pbsOrtbBid{
		&bid1_1,
		&bid1_2,
		&bid1_3,
		&bid1_4,
	}

	seatBid := pbsOrtbSeatBid{bids: innerBids, currency: "USD"}
	bidderName1 := openrtb_ext.BidderName("appnexus")

	adapterBids[bidderName1] = &seatBid

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, &requestExt, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{})

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
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 20.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb2.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 30.0000, Cat: cats3, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_2 := pbsOrtbBid{&bid2, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 40}, nil, 0, false, ""}
	bid1_3 := pbsOrtbBid{&bid3, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}

	innerBids := []*pbsOrtbBid{
		&bid1_1,
		&bid1_2,
		&bid1_3,
	}

	seatBid := pbsOrtbSeatBid{bids: innerBids, currency: "USD"}
	bidderName1 := openrtb_ext.BidderName("appnexus")

	adapterBids[bidderName1] = &seatBid

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, &requestExt, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{})

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
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 20.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb2.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 30.0000, Cat: cats3, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_2 := pbsOrtbBid{&bid2, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 40}, nil, 0, false, ""}
	bid1_3 := pbsOrtbBid{&bid3, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}

	innerBids := []*pbsOrtbBid{
		&bid1_1,
		&bid1_2,
		&bid1_3,
	}

	seatBid := pbsOrtbSeatBid{bids: innerBids, currency: "USD"}
	bidderName1 := openrtb_ext.BidderName("appnexus")

	adapterBids[bidderName1] = &seatBid

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, &requestExt, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{})

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
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 15.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb2.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 20.0000, Cat: cats1, W: 1, H: 1}
	bid4 := openrtb2.Bid{ID: "bid_id4", ImpID: "imp_id4", Price: 20.0000, Cat: cats4, W: 1, H: 1}
	bid5 := openrtb2.Bid{ID: "bid_id5", ImpID: "imp_id5", Price: 20.0000, Cat: cats1, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_2 := pbsOrtbBid{&bid2, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 50}, nil, 0, false, ""}
	bid1_3 := pbsOrtbBid{&bid3, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_4 := pbsOrtbBid{&bid4, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_5 := pbsOrtbBid{&bid5, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}

	selectedBids := make(map[string]int)
	expectedCategories := map[string]string{
		"bid_id1": "10.00_Electronics_30s",
		"bid_id2": "14.00_Sports_50s",
		"bid_id3": "20.00_Electronics_30s",
		"bid_id5": "20.00_Electronics_30s",
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
			&bid1_5,
		}

		seatBid := pbsOrtbSeatBid{bids: innerBids, currency: "USD"}
		bidderName1 := openrtb_ext.BidderName("appnexus")

		adapterBids[bidderName1] = &seatBid

		bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, &requestExt, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{})

		assert.Equal(t, nil, err, "Category mapping error should be empty")
		assert.Equal(t, 3, len(rejections), "There should be 2 bid rejection messages")
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
	assert.Equal(t, 0, selectedBids["bid_id1"], "Bid 1 should be rejected on every iteration due to lower price")
	assert.NotEqual(t, 0, selectedBids["bid_id3"], "Bid 3 should be accepted at least once")
	assert.NotEqual(t, 0, selectedBids["bid_id5"], "Bid 5 should be accepted at least once")
}

func TestNoCategoryDedupe(t *testing.T) {

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequestNoBrandCat()

	targData := &targetData{
		priceGranularity: requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}
	cats4 := []string{"IAB1-2000"}
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 14.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 14.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb2.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 20.0000, Cat: cats1, W: 1, H: 1}
	bid4 := openrtb2.Bid{ID: "bid_id4", ImpID: "imp_id4", Price: 20.0000, Cat: cats4, W: 1, H: 1}
	bid5 := openrtb2.Bid{ID: "bid_id5", ImpID: "imp_id5", Price: 10.0000, Cat: cats1, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_2 := pbsOrtbBid{&bid2, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_3 := pbsOrtbBid{&bid3, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_4 := pbsOrtbBid{&bid4, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_5 := pbsOrtbBid{&bid5, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}

	selectedBids := make(map[string]int)
	expectedCategories := map[string]string{
		"bid_id1": "14.00_30s",
		"bid_id2": "14.00_30s",
		"bid_id3": "20.00_30s",
		"bid_id4": "20.00_30s",
		"bid_id5": "10.00_30s",
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
			&bid1_5,
		}

		seatBid := pbsOrtbSeatBid{bids: innerBids, currency: "USD"}
		bidderName1 := openrtb_ext.BidderName("appnexus")

		adapterBids[bidderName1] = &seatBid

		bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, &requestExt, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{})

		assert.Equal(t, nil, err, "Category mapping error should be empty")
		assert.Equal(t, 2, len(rejections), "There should be 2 bid rejection messages")
		assert.Regexpf(t, regexp.MustCompile(`bid rejected \[bid ID: bid_id(1|2)\] reason: Bid was deduplicated`), rejections[0], "Rejection message did not match expected")
		assert.Regexpf(t, regexp.MustCompile(`bid rejected \[bid ID: bid_id(3|4)\] reason: Bid was deduplicated`), rejections[1], "Rejection message did not match expected")
		assert.Equal(t, 3, len(adapterBids[bidderName1].bids), "Bidders number doesn't match")
		assert.Equal(t, 3, len(bidCategory), "Bidders category mapping doesn't match")

		for bidId, bidCat := range bidCategory {
			assert.Equal(t, expectedCategories[bidId], bidCat, "Category mapping doesn't match")
			selectedBids[bidId]++
		}
	}
	assert.Equal(t, numIterations, selectedBids["bid_id5"], "Bid 5 did not make it through every time")
	assert.NotEqual(t, 0, selectedBids["bid_id1"], "Bid 1 should be selected at least once")
	assert.NotEqual(t, 0, selectedBids["bid_id2"], "Bid 2 should be selected at least once")
	assert.NotEqual(t, 0, selectedBids["bid_id1"], "Bid 3 should be selected at least once")
	assert.NotEqual(t, 0, selectedBids["bid_id4"], "Bid 4 should be selected at least once")

}

func TestCategoryMappingBidderName(t *testing.T) {

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequest()
	requestExt.Prebid.Targeting.AppendBidderNames = true

	targData := &targetData{
		priceGranularity: requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	requestExt.Prebid.Targeting.DurationRangeSec = []int{15, 30}

	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)

	cats1 := []string{"IAB1-1"}
	cats2 := []string{"IAB1-2"}
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 10.0000, Cat: cats2, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_2 := pbsOrtbBid{&bid2, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}

	innerBids1 := []*pbsOrtbBid{
		&bid1_1,
	}
	innerBids2 := []*pbsOrtbBid{
		&bid1_2,
	}

	seatBid1 := pbsOrtbSeatBid{bids: innerBids1, currency: "USD"}
	bidderName1 := openrtb_ext.BidderName("bidder1")

	seatBid2 := pbsOrtbSeatBid{bids: innerBids2, currency: "USD"}
	bidderName2 := openrtb_ext.BidderName("bidder2")

	adapterBids[bidderName1] = &seatBid1
	adapterBids[bidderName2] = &seatBid2

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, &requestExt, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{})

	assert.NoError(t, err, "Category mapping error should be empty")
	assert.Empty(t, rejections, "There should be 0 bid rejection messages")
	assert.Equal(t, "10.00_VideoGames_30s_bidder1", bidCategory["bid_id1"], "Category mapping doesn't match")
	assert.Equal(t, "10.00_HomeDecor_30s_bidder2", bidCategory["bid_id2"], "Category mapping doesn't match")
	assert.Len(t, adapterBids[bidderName1].bids, 1, "Bidders number doesn't match")
	assert.Len(t, adapterBids[bidderName2].bids, 1, "Bidders number doesn't match")
	assert.Len(t, bidCategory, 2, "Bidders category mapping doesn't match")
}

func TestCategoryMappingBidderNameNoCategories(t *testing.T) {

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequestNoBrandCat()
	requestExt.Prebid.Targeting.AppendBidderNames = true

	targData := &targetData{
		priceGranularity: requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	requestExt.Prebid.Targeting.DurationRangeSec = []int{15, 30}

	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)

	cats1 := []string{"IAB1-1"}
	cats2 := []string{"IAB1-2"}
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 12.0000, Cat: cats2, W: 1, H: 1}

	bid1_1 := pbsOrtbBid{&bid1, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_2 := pbsOrtbBid{&bid2, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}

	innerBids1 := []*pbsOrtbBid{
		&bid1_1,
	}
	innerBids2 := []*pbsOrtbBid{
		&bid1_2,
	}

	seatBid1 := pbsOrtbSeatBid{bids: innerBids1, currency: "USD"}
	bidderName1 := openrtb_ext.BidderName("bidder1")

	seatBid2 := pbsOrtbSeatBid{bids: innerBids2, currency: "USD"}
	bidderName2 := openrtb_ext.BidderName("bidder2")

	adapterBids[bidderName1] = &seatBid1
	adapterBids[bidderName2] = &seatBid2

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, &requestExt, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{})

	assert.NoError(t, err, "Category mapping error should be empty")
	assert.Empty(t, rejections, "There should be 0 bid rejection messages")
	assert.Equal(t, "10.00_30s_bidder1", bidCategory["bid_id1"], "Category mapping doesn't match")
	assert.Equal(t, "12.00_30s_bidder2", bidCategory["bid_id2"], "Category mapping doesn't match")
	assert.Len(t, adapterBids[bidderName1].bids, 1, "Bidders number doesn't match")
	assert.Len(t, adapterBids[bidderName2].bids, 1, "Bidders number doesn't match")
	assert.Len(t, bidCategory, 2, "Bidders category mapping doesn't match")
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
		bids               []*openrtb2.Bid
		duration           int
		expectedRejections []string
		expectedCatDur     string
	}{
		{
			description: "Bid should be rejected due to not containing a category",
			reqExt:      requestExt,
			bids: []*openrtb2.Bid{
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
			bids: []*openrtb2.Bid{
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
			bids: []*openrtb2.Bid{
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
			bids: []*openrtb2.Bid{
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
				bid, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: test.duration}, nil, 0, false, ""}
			innerBids = append(innerBids, &currentBid)
		}

		seatBid := pbsOrtbSeatBid{bids: innerBids, currency: "USD"}

		adapterBids[bidderName] = &seatBid

		bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, &test.reqExt, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{})

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

func TestCategoryMappingTwoBiddersOneBidEachNoCategorySamePrice(t *testing.T) {

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequestTranslateCategories(nil)

	targData := &targetData{
		priceGranularity: requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	requestExt.Prebid.Targeting.DurationRangeSec = []int{30}
	requestExt.Prebid.Targeting.IncludeBrandCategory.WithCategory = false

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}

	bidApn1 := openrtb2.Bid{ID: "bid_idApn1", ImpID: "imp_idApn1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bidApn2 := openrtb2.Bid{ID: "bid_idApn2", ImpID: "imp_idApn2", Price: 10.0000, Cat: cats2, W: 1, H: 1}

	bid1_Apn1 := pbsOrtbBid{&bidApn1, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_Apn2 := pbsOrtbBid{&bidApn2, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}

	innerBidsApn1 := []*pbsOrtbBid{
		&bid1_Apn1,
	}

	innerBidsApn2 := []*pbsOrtbBid{
		&bid1_Apn2,
	}

	for i := 1; i < 10; i++ {
		adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)

		seatBidApn1 := pbsOrtbSeatBid{bids: innerBidsApn1, currency: "USD"}
		bidderNameApn1 := openrtb_ext.BidderName("appnexus1")

		seatBidApn2 := pbsOrtbSeatBid{bids: innerBidsApn2, currency: "USD"}
		bidderNameApn2 := openrtb_ext.BidderName("appnexus2")

		adapterBids[bidderNameApn1] = &seatBidApn1
		adapterBids[bidderNameApn2] = &seatBidApn2

		bidCategory, adapterBids, rejections, err := applyCategoryMapping(nil, &requestExt, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{})

		assert.NoError(t, err, "Category mapping error should be empty")
		assert.Len(t, rejections, 1, "There should be 1 bid rejection message")
		assert.Regexpf(t, regexp.MustCompile(`bid rejected \[bid ID: bid_idApn(1|2)\] reason: Bid was deduplicated`), rejections[0], "Rejection message did not match expected")
		assert.Len(t, bidCategory, 1, "Bidders category mapping should have only one element")

		var resultBid string
		for bidId := range bidCategory {
			resultBid = bidId
		}

		if resultBid == "bid_idApn1" {
			assert.Nil(t, seatBidApn2.bids, "Appnexus_2 seat bid should not have any bids back")
			assert.Len(t, seatBidApn1.bids, 1, "Appnexus_1 seat bid should have only one back")

		} else {
			assert.Nil(t, seatBidApn1.bids, "Appnexus_1 seat bid should not have any bids back")
			assert.Len(t, seatBidApn2.bids, 1, "Appnexus_2 seat bid should have only one back")
		}
	}
}

func TestCategoryMappingTwoBiddersManyBidsEachNoCategorySamePrice(t *testing.T) {
	// This test covers a very rare de-duplication case where bid needs to be removed from already processed bidder
	// This happens when current processing bidder has a bid that has same de-duplication key as a bid from already processed bidder
	// and already processed bid was selected to be removed

	//In this test case bids bid_idApn1_1 and bid_idApn1_2 will be removed due to hardcoded "fakeRandomDeduplicateBidBooleanGenerator{true}"

	// Also there are should be more than one bids in bidder to test how we remove single element from bids array.
	// In case there is just one bid to remove - we remove the entire bidder.

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequestTranslateCategories(nil)

	targData := &targetData{
		priceGranularity: requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	requestExt.Prebid.Targeting.DurationRangeSec = []int{30}
	requestExt.Prebid.Targeting.IncludeBrandCategory.WithCategory = false

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}

	bidApn1_1 := openrtb2.Bid{ID: "bid_idApn1_1", ImpID: "imp_idApn1_1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bidApn1_2 := openrtb2.Bid{ID: "bid_idApn1_2", ImpID: "imp_idApn1_2", Price: 20.0000, Cat: cats1, W: 1, H: 1}

	bidApn2_1 := openrtb2.Bid{ID: "bid_idApn2_1", ImpID: "imp_idApn2_1", Price: 10.0000, Cat: cats2, W: 1, H: 1}
	bidApn2_2 := openrtb2.Bid{ID: "bid_idApn2_2", ImpID: "imp_idApn2_2", Price: 20.0000, Cat: cats2, W: 1, H: 1}

	bid1_Apn1_1 := pbsOrtbBid{&bidApn1_1, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_Apn1_2 := pbsOrtbBid{&bidApn1_2, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}

	bid1_Apn2_1 := pbsOrtbBid{&bidApn2_1, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_Apn2_2 := pbsOrtbBid{&bidApn2_2, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}

	innerBidsApn1 := []*pbsOrtbBid{
		&bid1_Apn1_1,
		&bid1_Apn1_2,
	}

	innerBidsApn2 := []*pbsOrtbBid{
		&bid1_Apn2_1,
		&bid1_Apn2_2,
	}

	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid)

	seatBidApn1 := pbsOrtbSeatBid{bids: innerBidsApn1, currency: "USD"}
	bidderNameApn1 := openrtb_ext.BidderName("appnexus1")

	seatBidApn2 := pbsOrtbSeatBid{bids: innerBidsApn2, currency: "USD"}
	bidderNameApn2 := openrtb_ext.BidderName("appnexus2")

	adapterBids[bidderNameApn1] = &seatBidApn1
	adapterBids[bidderNameApn2] = &seatBidApn2

	_, adapterBids, rejections, err := applyCategoryMapping(nil, &requestExt, adapterBids, categoriesFetcher, targData, &fakeRandomDeduplicateBidBooleanGenerator{true})

	assert.NoError(t, err, "Category mapping error should be empty")

	//Total number of bids from all bidders in this case should be 2
	bidsFromFirstBidder := adapterBids[bidderNameApn1]
	bidsFromSecondBidder := adapterBids[bidderNameApn2]

	totalNumberOfbids := 0

	//due to random map order we need to identify what bidder was first
	firstBidderIndicator := true

	if bidsFromFirstBidder.bids != nil {
		totalNumberOfbids += len(bidsFromFirstBidder.bids)
	}

	if bidsFromSecondBidder.bids != nil {
		firstBidderIndicator = false
		totalNumberOfbids += len(bidsFromSecondBidder.bids)
	}

	assert.Equal(t, 2, totalNumberOfbids, "2 bids total should be returned")
	assert.Len(t, rejections, 2, "2 bids should be de-duplicated")

	if firstBidderIndicator {
		assert.Len(t, adapterBids[bidderNameApn1].bids, 2)
		assert.Len(t, adapterBids[bidderNameApn2].bids, 0)

		assert.Equal(t, "bid_idApn1_1", adapterBids[bidderNameApn1].bids[0].bid.ID, "Incorrect expected bid 1 id")
		assert.Equal(t, "bid_idApn1_2", adapterBids[bidderNameApn1].bids[1].bid.ID, "Incorrect expected bid 2 id")

		assert.Equal(t, "bid rejected [bid ID: bid_idApn2_1] reason: Bid was deduplicated", rejections[0], "Incorrect rejected bid 1")
		assert.Equal(t, "bid rejected [bid ID: bid_idApn2_2] reason: Bid was deduplicated", rejections[1], "Incorrect rejected bid 2")

	} else {
		assert.Len(t, adapterBids[bidderNameApn1].bids, 0)
		assert.Len(t, adapterBids[bidderNameApn2].bids, 2)

		assert.Equal(t, "bid_idApn2_1", adapterBids[bidderNameApn2].bids[0].bid.ID, "Incorrect expected bid 1 id")
		assert.Equal(t, "bid_idApn2_2", adapterBids[bidderNameApn2].bids[1].bid.ID, "Incorrect expected bid 2 id")

		assert.Equal(t, "bid rejected [bid ID: bid_idApn1_1] reason: Bid was deduplicated", rejections[0], "Incorrect rejected bid 1")
		assert.Equal(t, "bid rejected [bid ID: bid_idApn1_2] reason: Bid was deduplicated", rejections[1], "Incorrect rejected bid 2")

	}
}

func TestRemoveBidById(t *testing.T) {
	cats1 := []string{"IAB1-3"}

	bidApn1_1 := openrtb2.Bid{ID: "bid_idApn1_1", ImpID: "imp_idApn1_1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bidApn1_2 := openrtb2.Bid{ID: "bid_idApn1_2", ImpID: "imp_idApn1_2", Price: 20.0000, Cat: cats1, W: 1, H: 1}
	bidApn1_3 := openrtb2.Bid{ID: "bid_idApn1_3", ImpID: "imp_idApn1_3", Price: 10.0000, Cat: cats1, W: 1, H: 1}

	bid1_Apn1_1 := pbsOrtbBid{&bidApn1_1, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_Apn1_2 := pbsOrtbBid{&bidApn1_2, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}
	bid1_Apn1_3 := pbsOrtbBid{&bidApn1_3, nil, "video", nil, &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, nil, 0, false, ""}

	type aTest struct {
		desc      string
		inBidName string
		outBids   []*pbsOrtbBid
	}
	testCases := []aTest{
		{
			desc:      "remove element from the middle",
			inBidName: "bid_idApn1_2",
			outBids:   []*pbsOrtbBid{&bid1_Apn1_1, &bid1_Apn1_3},
		},
		{
			desc:      "remove element from the end",
			inBidName: "bid_idApn1_3",
			outBids:   []*pbsOrtbBid{&bid1_Apn1_1, &bid1_Apn1_2},
		},
		{
			desc:      "remove element from the beginning",
			inBidName: "bid_idApn1_1",
			outBids:   []*pbsOrtbBid{&bid1_Apn1_2, &bid1_Apn1_3},
		},
		{
			desc:      "remove element that doesn't exist",
			inBidName: "bid_idApn",
			outBids:   []*pbsOrtbBid{&bid1_Apn1_1, &bid1_Apn1_2, &bid1_Apn1_3},
		},
	}
	for _, test := range testCases {

		innerBidsApn1 := []*pbsOrtbBid{
			&bid1_Apn1_1,
			&bid1_Apn1_2,
			&bid1_Apn1_3,
		}

		seatBidApn1 := &pbsOrtbSeatBid{bids: innerBidsApn1, currency: "USD"}

		removeBidById(seatBidApn1, test.inBidName)
		assert.Len(t, seatBidApn1.bids, len(test.outBids), test.desc)
		assert.ElementsMatch(t, seatBidApn1.bids, test.outBids, "Incorrect bids in response")
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
		description               string
		dealPriority              int
		impExt                    json.RawMessage
		targ                      map[string]string
		expectedHbPbCatDur        string
		expectedDealErr           string
		expectedDealTierSatisfied bool
	}{
		{
			description:  "hb_pb_cat_dur should be modified",
			dealPriority: 5,
			impExt:       json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}`),
			targ: map[string]string{
				"hb_pb_cat_dur": "12.00_movies_30s",
			},
			expectedHbPbCatDur:        "tier5_movies_30s",
			expectedDealErr:           "",
			expectedDealTierSatisfied: true,
		},
		{
			description:  "hb_pb_cat_dur should not be modified due to priority not exceeding min",
			dealPriority: 9,
			impExt:       json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 10, "prefix": "tier"}, "placementId": 10433394}}`),
			targ: map[string]string{
				"hb_pb_cat_dur": "12.00_medicine_30s",
			},
			expectedHbPbCatDur:        "12.00_medicine_30s",
			expectedDealErr:           "",
			expectedDealTierSatisfied: false,
		},
		{
			description:  "hb_pb_cat_dur should not be modified due to invalid config",
			dealPriority: 5,
			impExt:       json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": ""}, "placementId": 10433394}}`),
			targ: map[string]string{
				"hb_pb_cat_dur": "12.00_games_30s",
			},
			expectedHbPbCatDur:        "12.00_games_30s",
			expectedDealErr:           "dealTier configuration invalid for bidder 'appnexus', imp ID 'imp_id1'",
			expectedDealTierSatisfied: false,
		},
		{
			description:  "hb_pb_cat_dur should not be modified due to deal priority of 0",
			dealPriority: 0,
			impExt:       json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}`),
			targ: map[string]string{
				"hb_pb_cat_dur": "12.00_auto_30s",
			},
			expectedHbPbCatDur:        "12.00_auto_30s",
			expectedDealErr:           "",
			expectedDealTierSatisfied: false,
		},
	}

	bidderName := openrtb_ext.BidderName("appnexus")
	for _, test := range testCases {
		bidRequest := &openrtb2.BidRequest{
			ID: "some-request-id",
			Imp: []openrtb2.Imp{
				{
					ID:  "imp_id1",
					Ext: test.impExt,
				},
			},
		}

		bid := pbsOrtbBid{&openrtb2.Bid{ID: "123456"}, nil, "video", map[string]string{}, &openrtb_ext.ExtBidPrebidVideo{}, nil, test.dealPriority, false, ""}
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
		assert.Equal(t, test.expectedDealTierSatisfied, auc.winningBidsByBidder["imp_id1"][bidderName].dealTierSatisfied, "expectedDealTierSatisfied=%v when %v", test.expectedDealTierSatisfied, test.description)
		if len(test.expectedDealErr) > 0 {
			assert.Containsf(t, dealErrs, errors.New(test.expectedDealErr), "Expected error message not found in deal errors")
		}
	}
}

func TestGetDealTiers(t *testing.T) {
	testCases := []struct {
		description string
		request     openrtb2.BidRequest
		expected    map[string]openrtb_ext.DealTierBidderMap
	}{
		{
			description: "None",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{},
			},
			expected: map[string]openrtb_ext.DealTierBidderMap{},
		},
		{
			description: "One",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp1", Ext: json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}}}`)},
				},
			},
			expected: map[string]openrtb_ext.DealTierBidderMap{
				"imp1": {openrtb_ext.BidderAppnexus: {Prefix: "tier", MinDealTier: 5}},
			},
		},
		{
			description: "Many",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp1", Ext: json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier1"}}}`)},
					{ID: "imp2", Ext: json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 8, "prefix": "tier2"}}}`)},
				},
			},
			expected: map[string]openrtb_ext.DealTierBidderMap{
				"imp1": {openrtb_ext.BidderAppnexus: {Prefix: "tier1", MinDealTier: 5}},
				"imp2": {openrtb_ext.BidderAppnexus: {Prefix: "tier2", MinDealTier: 8}},
			},
		},
		{
			description: "Many - Skips Malformed",
			request: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp1", Ext: json.RawMessage(`{"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier1"}}}`)},
					{ID: "imp2", Ext: json.RawMessage(`{"appnexus": {"dealTier": "wrong type"}}`)},
				},
			},
			expected: map[string]openrtb_ext.DealTierBidderMap{
				"imp1": {openrtb_ext.BidderAppnexus: {Prefix: "tier1", MinDealTier: 5}},
			},
		},
	}

	for _, test := range testCases {
		result := getDealTiers(&test.request)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestValidateDealTier(t *testing.T) {
	testCases := []struct {
		description    string
		dealTier       openrtb_ext.DealTier
		expectedResult bool
	}{
		{
			description:    "Valid",
			dealTier:       openrtb_ext.DealTier{Prefix: "prefix", MinDealTier: 5},
			expectedResult: true,
		},
		{
			description:    "Invalid - Empty",
			dealTier:       openrtb_ext.DealTier{},
			expectedResult: false,
		},
		{
			description:    "Invalid - Empty Prefix",
			dealTier:       openrtb_ext.DealTier{MinDealTier: 5},
			expectedResult: false,
		},
		{
			description:    "Invalid - Empty Deal Tier",
			dealTier:       openrtb_ext.DealTier{Prefix: "prefix"},
			expectedResult: false,
		},
	}

	for _, test := range testCases {
		assert.Equal(t, test.expectedResult, validateDealTier(test.dealTier), test.description)
	}
}

func TestUpdateHbPbCatDur(t *testing.T) {
	testCases := []struct {
		description               string
		targ                      map[string]string
		dealTier                  openrtb_ext.DealTier
		dealPriority              int
		expectedHbPbCatDur        string
		expectedDealTierSatisfied bool
	}{
		{
			description: "hb_pb_cat_dur should be updated with prefix and tier",
			targ: map[string]string{
				"hb_pb":         "12.00",
				"hb_pb_cat_dur": "12.00_movies_30s",
			},
			dealTier: openrtb_ext.DealTier{
				Prefix:      "tier",
				MinDealTier: 5,
			},
			dealPriority:              5,
			expectedHbPbCatDur:        "tier5_movies_30s",
			expectedDealTierSatisfied: true,
		},
		{
			description: "hb_pb_cat_dur should not be updated due to bid priority",
			targ: map[string]string{
				"hb_pb":         "12.00",
				"hb_pb_cat_dur": "12.00_auto_30s",
			},
			dealTier: openrtb_ext.DealTier{
				Prefix:      "tier",
				MinDealTier: 10,
			},
			dealPriority:              6,
			expectedHbPbCatDur:        "12.00_auto_30s",
			expectedDealTierSatisfied: false,
		},
		{
			description: "hb_pb_cat_dur should be updated with prefix and tier",
			targ: map[string]string{
				"hb_pb":         "12.00",
				"hb_pb_cat_dur": "12.00_medicine_30s",
			},
			dealTier: openrtb_ext.DealTier{
				Prefix:      "tier",
				MinDealTier: 1,
			},
			dealPriority:              7,
			expectedHbPbCatDur:        "tier7_medicine_30s",
			expectedDealTierSatisfied: true,
		},
	}

	for _, test := range testCases {
		bid := pbsOrtbBid{&openrtb2.Bid{ID: "123456"}, nil, "video", map[string]string{}, &openrtb_ext.ExtBidPrebidVideo{}, nil, test.dealPriority, false, ""}
		bidCategory := map[string]string{
			bid.bid.ID: test.targ["hb_pb_cat_dur"],
		}

		updateHbPbCatDur(&bid, test.dealTier, bidCategory)

		assert.Equal(t, test.expectedHbPbCatDur, bidCategory[bid.bid.ID], test.description)
		assert.Equal(t, test.expectedDealTierSatisfied, bid.dealTierSatisfied, test.description)
	}
}

func TestMakeBidExtJSON(t *testing.T) {

	type aTest struct {
		description        string
		ext                json.RawMessage
		extBidPrebid       openrtb_ext.ExtBidPrebid
		impExtInfo         map[string]ImpExtInfo
		expectedBidExt     string
		expectedErrMessage string
	}

	testCases := []aTest{
		{
			description:        "Valid extension, non empty extBidPrebid and valid imp ext info",
			ext:                json.RawMessage(`{"video":{"h":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"video":{"h":480,"mimes":["video/mp4"]}}`)}},
			expectedBidExt:     `{"prebid":{"type":"video"},"storedrequestattributes":{"h":480,"mimes":["video/mp4"]},"video":{"h":100}}`,
			expectedErrMessage: "",
		},
		{
			description:        "Empty extension, non empty extBidPrebid and valid imp ext info",
			ext:                nil,
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"video":{"h":480,"mimes":["video/mp4"]}}`)}},
			expectedBidExt:     `{"prebid":{"type":"video"},"storedrequestattributes":{"h":480,"mimes":["video/mp4"]}}`,
			expectedErrMessage: "",
		},
		{
			description:        "Valid extension, non empty extBidPrebid and imp ext info not found",
			ext:                json.RawMessage(`{"video":{"h":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			impExtInfo:         map[string]ImpExtInfo{"another_imp_id": {true, []byte(`{"video":{"h":480,"mimes":["video/mp4"]}}`)}},
			expectedBidExt:     `{"prebid":{"type":"video"},"video":{"h":100}}`,
			expectedErrMessage: "",
		},
		{
			description:        "Valid extension, empty extBidPrebid and valid imp ext info",
			ext:                json.RawMessage(`{"video":{"h":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{},
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"video":{"h":480,"mimes":["video/mp4"]}}`)}},
			expectedBidExt:     `{"prebid":{"type":""},"storedrequestattributes":{"h":480,"mimes":["video/mp4"]},"video":{"h":100}}`,
			expectedErrMessage: "",
		},
		{
			description:        "Valid extension, non empty extBidPrebid and empty imp ext info",
			ext:                json.RawMessage(`{"video":{"h":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			impExtInfo:         nil,
			expectedBidExt:     `{"prebid":{"type":"video"},"video":{"h":100}}`,
			expectedErrMessage: "",
		},
		{
			description:        "Valid extension, non empty extBidPrebid and valid imp ext info without video attr",
			ext:                json.RawMessage(`{"video":{"h":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"banner":{"h":480}}`)}},
			expectedBidExt:     `{"prebid":{"type":"video"},"video":{"h":100}}`,
			expectedErrMessage: "",
		},
		{
			description:        "Valid extension with prebid, non empty extBidPrebid and valid imp ext info without video attr",
			ext:                json.RawMessage(`{"prebid":{"targeting":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"banner":{"h":480}}`)}},
			expectedBidExt:     `{"prebid":{"type":"video"}}`,
			expectedErrMessage: "",
		},
		{
			description:        "Valid extension with prebid, non empty extBidPrebid and valid imp ext info with video attr",
			ext:                json.RawMessage(`{"prebid":{"targeting":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"video":{"h":480,"mimes":["video/mp4"]}}`)}},
			expectedBidExt:     `{"prebid":{"type":"video"}, "storedrequestattributes":{"h":480,"mimes":["video/mp4"]}}`,
			expectedErrMessage: "",
		},
		//Error cases
		{
			description:        "Invalid extension, valid extBidPrebid and valid imp ext info",
			ext:                json.RawMessage(`{invalid json}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"video":{"h":480,"mimes":["video/mp4"]}}`)}},
			expectedBidExt:     ``,
			expectedErrMessage: "invalid character",
		},
		{
			description:        "Valid extension, empty extBidPrebid and invalid imp ext info",
			ext:                json.RawMessage(`{"video":{"h":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{},
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"video":{!}}`)}},
			expectedBidExt:     ``,
			expectedErrMessage: "invalid character",
		},
	}

	for _, test := range testCases {
		result, err := makeBidExtJSON(test.ext, &test.extBidPrebid, test.impExtInfo, "test_imp_id")

		if test.expectedErrMessage == "" {
			assert.JSONEq(t, test.expectedBidExt, string(result), "Incorrect result")
			assert.NoError(t, err, "Error should not be returned")
		} else {
			assert.Contains(t, err.Error(), test.expectedErrMessage, "incorrect error message")
		}
	}
}

type exchangeSpec struct {
	GDPREnabled       bool                   `json:"gdpr_enabled"`
	IncomingRequest   exchangeRequest        `json:"incomingRequest"`
	OutgoingRequests  map[string]*bidderSpec `json:"outgoingRequests"`
	Response          exchangeResponse       `json:"response,omitempty"`
	EnforceCCPA       bool                   `json:"enforceCcpa"`
	EnforceLMT        bool                   `json:"enforceLmt"`
	AssumeGDPRApplies bool                   `json:"assume_gdpr_applies"`
	DebugLog          *DebugLog              `json:"debuglog,omitempty"`
	EventsEnabled     bool                   `json:"events_enabled,omitempty"`
	StartTime         int64                  `json:"start_time_ms,omitempty"`
	BidIDGenerator    *mockBidIDGenerator    `json:"bidIDGenerator,omitempty"`
}

type exchangeRequest struct {
	OrtbRequest openrtb2.BidRequest `json:"ortbRequest"`
	Usersyncs   map[string]string   `json:"usersyncs"`
}

type exchangeResponse struct {
	Bids  *openrtb2.BidResponse `json:"bids"`
	Error string                `json:"error,omitempty"`
	Ext   json.RawMessage       `json:"ext,omitempty"`
}

type bidderSpec struct {
	ExpectedRequest         *bidderRequest `json:"expectRequest"`
	MockResponse            bidderResponse `json:"mockResponse"`
	ModifyingVastXmlAllowed bool           `json:"modifyingVastXmlAllowed,omitempty"`
}

type bidderRequest struct {
	OrtbRequest   openrtb2.BidRequest `json:"ortbRequest"`
	BidAdjustment float64             `json:"bidAdjustment"`
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
	Bid  *openrtb2.Bid `json:"ortbBid,omitempty"`
	Type string        `json:"bidType,omitempty"`
}

type mockIdFetcher map[string]string

func (f mockIdFetcher) GetId(bidder openrtb_ext.BidderName) (id string, ok bool) {
	id, ok = f[string(bidder)]
	return
}

func (f mockIdFetcher) LiveSyncCount() int {
	return len(f)
}

type validatingBidder struct {
	t          *testing.T
	fileName   string
	bidderName string

	// These are maps because they may contain aliases. They should _at least_ contain an entry for bidderName.
	expectations  map[string]*bidderRequest
	mockResponses map[string]bidderResponse
}

func (b *validatingBidder) requestBid(ctx context.Context, request *openrtb2.BidRequest, name openrtb_ext.BidderName, bidAdjustment float64, conversions currency.Conversions, reqInfo *adapters.ExtraRequestInfo, accountDebugAllowed, headerDebugAllowed bool) (seatBid *pbsOrtbSeatBid, errs []error) {
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

func diffOrtbRequests(t *testing.T, description string, expected *openrtb2.BidRequest, actual *openrtb2.BidRequest) {
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

func diffOrtbResponses(t *testing.T, description string, expected *openrtb2.BidResponse, actual *openrtb2.BidResponse) {
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

func mapifySeatBids(t *testing.T, context string, seatBids []openrtb2.SeatBid) map[string]*openrtb2.SeatBid {
	seatMap := make(map[string]*openrtb2.SeatBid, len(seatBids))
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

func (c *wellBehavedCache) GetExtCacheData() (scheme string, host string, path string) {
	return "https", "www.pbcserver.com", "/pbcache/endpoint"
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

func (e *emptyUsersync) LiveSyncCount() int {
	return 0
}

type mockUsersync struct {
	syncs map[string]string
}

func (e *mockUsersync) GetId(bidder openrtb_ext.BidderName) (id string, exists bool) {
	id, exists = e.syncs[string(bidder)]
	return
}

func (e *mockUsersync) LiveSyncCount() int {
	return len(e.syncs)
}

type panicingAdapter struct{}

func (panicingAdapter) requestBid(ctx context.Context, request *openrtb2.BidRequest, name openrtb_ext.BidderName, bidAdjustment float64, conversions currency.Conversions, reqInfo *adapters.ExtraRequestInfo, accountDebugAllowed, headerDebugAllowed bool) (posb *pbsOrtbSeatBid, errs []error) {
	panic("Panic! Panic! The world is ending!")
}

func blankAdapterConfig(bidderList []openrtb_ext.BidderName) map[string]config.Adapter {
	adapters := make(map[string]config.Adapter)
	for _, b := range bidderList {
		adapters[strings.ToLower(string(b))] = config.Adapter{}
	}

	// Audience Network requires additional config to be built.
	adapters["audiencenetwork"] = config.Adapter{PlatformID: "anyID", AppSecret: "anySecret"}

	return adapters
}

type nilCategoryFetcher struct{}

func (nilCategoryFetcher) FetchCategories(ctx context.Context, primaryAdServer, publisherId, iabCategory string) (string, error) {
	return "", nil
}

// fakeCurrencyRatesHttpClient is a simple http client mock returning a constant response body
type fakeCurrencyRatesHttpClient struct {
	responseBody string
}

func (m *fakeCurrencyRatesHttpClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader(m.responseBody)),
	}, nil
}

type mockBidder struct {
	mock.Mock
	lastExtraRequestInfo *adapters.ExtraRequestInfo
}

func (m *mockBidder) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	m.lastExtraRequestInfo = reqInfo

	args := m.Called(request, reqInfo)
	return args.Get(0).([]*adapters.RequestData), args.Get(1).([]error)
}

func (m *mockBidder) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	args := m.Called(internalRequest, externalRequest, response)
	return args.Get(0).(*adapters.BidderResponse), args.Get(1).([]error)
}
