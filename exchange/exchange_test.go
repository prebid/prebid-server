package exchange

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/experiment/adscert"
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/hooks"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/metrics"
	metricsConf "github.com/prebid/prebid-server/v3/metrics/config"
	metricsConfig "github.com/prebid/prebid-server/v3/metrics/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/ortb"
	pbc "github.com/prebid/prebid-server/v3/prebid_cache_client"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/file_fetcher"
	"github.com/prebid/prebid-server/v3/stored_responses"
	"github.com/prebid/prebid-server/v3/usersync"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	jsonpatch "gopkg.in/evanphx/json-patch.v5"
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
		GDPR: config.GDPR{
			EEACountries: []string{"FIN", "FRA", "GUF"},
		},
	}

	biddersInfo, err := config.LoadBidderInfoFromDisk("../static/bidder-info")
	if err != nil {
		t.Fatal(err)
	}

	adapters, _, adaptersErr := BuildAdapters(server.Client(), cfg, biddersInfo, &metricsConf.NilMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}

	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	gdprPermsBuilder := fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder

	e := NewExchange(adapters, nil, cfg, &mockRequestValidator{}, map[string]usersync.Syncer{}, &metricsConf.NilMetricsEngine{}, biddersInfo, gdprPermsBuilder, currencyConverter, nilCategoryFetcher{}, &adscert.NilSigner{}, macros.NewStringIndexBasedReplacer(), nil, nil).(*exchange)
	for _, bidderName := range knownAdapters {
		if _, ok := e.adapterMap[bidderName]; !ok {
			if biddersInfo[string(bidderName)].IsEnabled() {
				t.Errorf("NewExchange produced an Exchange without bidder %s", bidderName)
			}
		}
	}
	if e.cacheTime != time.Duration(cfg.CacheURL.ExpectedTimeMillis)*time.Millisecond {
		t.Errorf("Bad cacheTime. Expected 20 ms, got %s", e.cacheTime.String())
	}
}

// The objective is to get to execute e.buildBidResponse(ctx.Background(), liveA... ) (*openrtb2.BidResponse, error)
// and check whether the returned request successfully prints any '&' characters as it should
// To do so, we:
//  1. Write the endpoint adapter URL with an '&' character into a new config,Configuration struct
//     as specified in https://github.com/prebid/prebid-server/issues/465
//  2. Initialize a new exchange with said configuration
//  3. Build all the parameters e.buildBidResponse(ctx.Background(), liveA... ) needs including the
//     sample request as specified in https://github.com/prebid/prebid-server/issues/465
//  4. Build a BidResponse struct using exchange.buildBidResponse(ctx.Background(), liveA... )
//  5. Assert we have no '&' characters in the response that exchange.buildBidResponse returns
func TestCharacterEscape(t *testing.T) {

	// 1) Adapter with a '& char in its endpoint property
	//    https://github.com/prebid/prebid-server/issues/465
	cfg := &config.Configuration{}
	biddersInfo := config.BidderInfos{"appnexus": config.BidderInfo{Endpoint: "http://ib.adnxs.com/openrtb2?query1&query2"}} //Note the '&' character in there

	// 	2) Init new exchange with said configuration
	//Other parameters also needed to create exchange
	handlerNoBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	server := httptest.NewServer(http.HandlerFunc(handlerNoBidServer))

	defer server.Close()

	adapters, _, adaptersErr := BuildAdapters(server.Client(), cfg, biddersInfo, &metricsConf.NilMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}

	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	gdprPermsBuilder := fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder

	e := NewExchange(adapters, nil, cfg, &mockRequestValidator{}, map[string]usersync.Syncer{}, &metricsConf.NilMetricsEngine{}, biddersInfo, gdprPermsBuilder, currencyConverter, nilCategoryFetcher{}, &adscert.NilSigner{}, macros.NewStringIndexBasedReplacer(), nil, nil).(*exchange)

	// 	3) Build all the parameters e.buildBidResponse(ctx.Background(), liveA... ) needs
	//liveAdapters []openrtb_ext.BidderName,
	liveAdapters := make([]openrtb_ext.BidderName, 1)
	liveAdapters[0] = "appnexus"

	//adapterBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid,
	adapterBids := make(map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, 1)
	adapterBids["appnexus"] = &entities.PbsOrtbSeatBid{Currency: "USD"}

	//An openrtb2.BidRequest struct as specified in https://github.com/prebid/prebid-server/issues/465
	bidRequest := &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
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
		},
	}

	//adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra,
	adapterExtra := make(map[openrtb_ext.BidderName]*seatResponseExtra, 1)
	adapterExtra["appnexus"] = &seatResponseExtra{
		ResponseTimeMillis: 5,
		Errors:             []openrtb_ext.ExtBidderMessage{{Code: 999, Message: "Post ib.adnxs.com/openrtb2?query1&query2: unsupported protocol scheme \"\""}},
	}

	var errList []error

	// 	4) Build bid response
	bidResp := e.buildBidResponse(context.Background(), liveAdapters, adapterBids, bidRequest, adapterExtra, nil, nil, true, nil, "", errList, &SeatNonBidBuilder{})

	// 	5) Assert we have no errors and one '&' character as we are supposed to
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
			Ext:    json.RawMessage(`{"prebid":{"bidder":{"appnexus": {"placementId": 1}}}}`),
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
	e.me = &metricsConf.NilMetricsEngine{}
	e.gdprPermsBuilder = fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder
	e.currencyConverter = currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	e.categoriesFetcher = categoriesFetcher
	e.requestSplitter = requestSplitter{
		me:               &metricsConf.NilMetricsEngine{},
		gdprPermsBuilder: e.gdprPermsBuilder,
	}
	ctx := context.Background()

	// Run tests
	for _, test := range testCases {

		e.adapterMap = map[openrtb_ext.BidderName]AdaptedBidder{
			openrtb_ext.BidderAppnexus: AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, &config.DebugInfo{Allow: test.debugData.bidderLevelDebugAllowed}, ""),
		}

		bidRequest.Test = test.in.test

		if test.in.debug {
			bidRequest.Ext = json.RawMessage(`{"prebid":{"debug":true}}`)
		} else {
			bidRequest.Ext = nil
		}

		auctionRequest := &AuctionRequest{
			BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: bidRequest},
			Account:           config.Account{DebugAllow: test.debugData.accountLevelDebugAllowed},
			UserSyncs:         &emptyUsersync{},
			StartTime:         time.Now(),
			HookExecutor:      &hookexecution.EmptyHookExecutor{},
			TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}
		if test.generateWarnings {
			var errL []error
			errL = append(errL, &errortypes.Warning{
				Message:     "CCPA consent test warning.",
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
		assert.False(t, auctionRequest.BidderResponseStartTime.IsZero())
		actualExt := &openrtb_ext.ExtBidResponse{}
		err = jsonutil.UnmarshalValid(outBidResponse.Ext, actualExt)
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
				actualResolvedReqExt, _, _, err := jsonparser.Get(actualExt.Debug.ResolvedRequest, "ext")
				assert.NoError(t, err, "Resolved request should have the correct format")
				assert.JSONEq(t, string(bidRequest.Ext), string(actualResolvedReqExt), test.desc)
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
			Body:    []byte(`{"key":"val"}`),
			Headers: http.Header{},
		},
		bidResponse: &adapters.BidderResponse{},
	}

	e := new(exchange)
	e.cache = &wellBehavedCache{}
	e.me = &metricsConf.NilMetricsEngine{}
	e.gdprPermsBuilder = fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder
	e.currencyConverter = currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	e.categoriesFetcher = categoriesFetcher
	e.requestSplitter = requestSplitter{
		me:               e.me,
		gdprPermsBuilder: e.gdprPermsBuilder,
	}

	debugLog := DebugLog{Enabled: true}

	for _, testCase := range testCases {
		bidRequest := &openrtb2.BidRequest{
			ID: "some-request-id",
			Imp: []openrtb2.Imp{{
				ID:     "some-impression-id",
				Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
				Ext:    json.RawMessage(`{"prebid":{"bidder":{"telaria": {"placementId": 1}, "appnexus": {"placementid": 2}}}}`),
			}},
			Site:   &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
			Device: &openrtb2.Device{UA: "curl/7.54.0", IP: "::1"},
			AT:     1,
			TMax:   500,
		}

		bidRequest.Ext = json.RawMessage(`{"prebid":{"debug":true}}`)

		auctionRequest := &AuctionRequest{
			BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: bidRequest},
			Account:           config.Account{DebugAllow: true},
			UserSyncs:         &emptyUsersync{},
			StartTime:         time.Now(),
			HookExecutor:      &hookexecution.EmptyHookExecutor{},
			TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}

		e.adapterMap = map[openrtb_ext.BidderName]AdaptedBidder{
			openrtb_ext.BidderAppnexus: AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, &config.DebugInfo{Allow: testCase.bidder1DebugEnabled}, ""),
			openrtb_ext.BidderTelaria:  AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, &config.DebugInfo{Allow: testCase.bidder2DebugEnabled}, ""),
		}
		// Run test
		outBidResponse, err := e.HoldAuction(context.Background(), auctionRequest, &debugLog)
		// Assert no HoldAuction err
		assert.NoErrorf(t, err, "ex.HoldAuction returned an err")
		assert.NotNilf(t, outBidResponse.Ext, "outBidResponse.Ext should not be nil")
		assert.False(t, auctionRequest.BidderResponseStartTime.IsZero())

		actualExt := &openrtb_ext.ExtBidResponse{}
		err = jsonutil.UnmarshalValid(outBidResponse.Ext, actualExt)
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
	mockCurrencyClient := &currency.MockCurrencyRatesHttpClient{
		ResponseBody: `{"dataAsOf":"2018-09-12","conversions":{"USD":{"MXN":10.00}}}`,
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
	e.me = &metricsConf.NilMetricsEngine{}
	e.gdprPermsBuilder = fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder
	e.currencyConverter = mockCurrencyConverter
	e.categoriesFetcher = categoriesFetcher
	e.bidIDGenerator = &fakeBidIDGenerator{GenerateBidID: false, ReturnError: false}
	e.requestSplitter = requestSplitter{
		me:               e.me,
		gdprPermsBuilder: e.gdprPermsBuilder,
	}

	// Define mock incoming bid requeset
	mockBidRequest := &openrtb2.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb2.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext:    json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placementId":1}}}}`),
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

		e.adapterMap = map[openrtb_ext.BidderName]AdaptedBidder{
			openrtb_ext.BidderAppnexus: AdaptBidder(oneDollarBidBidder, mockAppnexusBidService.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, nil, ""),
		}

		// Set custom rates in extension
		mockBidRequest.Ext = test.in.customCurrencyRates

		// Set bidRequest currency list
		mockBidRequest.Cur = []string{test.in.bidRequestCurrency}

		auctionRequest := &AuctionRequest{
			BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: mockBidRequest},
			Account:           config.Account{},
			UserSyncs:         &emptyUsersync{},
			HookExecutor:      &hookexecution.EmptyHookExecutor{},
			TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}

		// Run test
		outBidResponse, err := e.HoldAuction(context.Background(), auctionRequest, &DebugLog{})

		// Assertions
		assert.NoErrorf(t, err, "%s. HoldAuction error: %v \n", test.desc, err)
		assert.False(t, auctionRequest.BidderResponseStartTime.IsZero())

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
	mockCurrencyClient := &currency.MockCurrencyRatesHttpClient{
		ResponseBody: `{"dataAsOf":"2018-09-12","conversions":{"USD":{"MXN":10.00}}}`,
	}
	currencyConverter := currency.NewRateConverter(
		mockCurrencyClient,
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
		cache: &wellBehavedCache{},
		me:    &metricsConf.NilMetricsEngine{},
		gdprPermsBuilder: fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowAllBidders: true,
			},
		}.Builder,
		currencyConverter: currencyConverter,
		categoriesFetcher: nilCategoryFetcher{},
		bidIDGenerator:    &fakeBidIDGenerator{GenerateBidID: false, ReturnError: false},
		adapterMap: map[openrtb_ext.BidderName]AdaptedBidder{
			openrtb_ext.BidderName("appnexus"): AdaptBidder(mockBidder, nil, &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderName("appnexus"), nil, ""),
		},
	}
	e.requestSplitter = requestSplitter{
		me:               e.me,
		gdprPermsBuilder: e.gdprPermsBuilder,
	}

	// Define Bid Request
	request := &openrtb2.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb2.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext:    json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placementId":1}}}}`),
		}},
		Site: &openrtb2.Site{
			Page: "prebid.org",
			Ext:  json.RawMessage(`{"amp":0}`),
		},
		Cur: []string{"USD"},
		Ext: json.RawMessage(`{"prebid": {"currency": {"rates": {"USD": {"MXN": 20.00}}}}}`),
	}

	// Run Auction
	auctionRequest := &AuctionRequest{
		BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: request},
		Account:           config.Account{},
		UserSyncs:         &emptyUsersync{},
		HookExecutor:      &hookexecution.EmptyHookExecutor{},
		TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
	}
	response, err := e.HoldAuction(context.Background(), auctionRequest, &DebugLog{})
	assert.NoError(t, err)
	assert.Equal(t, "some-request-id", response.ID, "Response ID")
	assert.Empty(t, response.SeatBid, "Response Bids")
	assert.Contains(t, string(response.Ext), `"errors":{"appnexus":[{"code":5,"message":"The adapter failed to generate any bid requests, but also failed to generate an error explaining why"}]}`, "Response Ext")

	// Test Currency Converter Properly Passed To Adapter
	if assert.NotNil(t, mockBidder.lastExtraRequestInfo, "Currency Conversion Argument") {
		converted, err := mockBidder.lastExtraRequestInfo.ConvertCurrency(2.0, "USD", "MXN")
		assert.NoError(t, err, "Currency Conversion Error")
		assert.Equal(t, 40.0, converted, "Currency Conversion Response")
	}
}

type mockPriceFloorFetcher struct{}

func (mpf *mockPriceFloorFetcher) Fetch(configs config.AccountPriceFloors) (*openrtb_ext.PriceFloorRules, string) {
	return nil, openrtb_ext.FetchNone
}

func (mpf *mockPriceFloorFetcher) Stop() {}

func TestFloorsSignalling(t *testing.T) {
	mockCurrencyClient := &currency.MockCurrencyRatesHttpClient{
		ResponseBody: `{"dataAsOf":"2023-04-10","conversions":{"USD":{"MXN":10.00}}}`,
	}
	currencyConverter := currency.NewRateConverter(
		mockCurrencyClient,
		"currency.com",
		24*time.Hour,
	)
	currencyConverter.Run()

	// Initialize Real Exchange
	e := exchange{
		cache: &wellBehavedCache{},
		me:    &metricsConf.NilMetricsEngine{},
		gdprPermsBuilder: fakePermissionsBuilder{
			permissions: &permissionsMock{
				allowAllBidders: true,
			},
		}.Builder,
		currencyConverter: currencyConverter,
		categoriesFetcher: nilCategoryFetcher{},
		bidIDGenerator:    &fakeBidIDGenerator{GenerateBidID: false, ReturnError: false},
		priceFloorEnabled: true,
		priceFloorFetcher: &mockPriceFloorFetcher{},
	}
	e.requestSplitter = requestSplitter{
		me:               e.me,
		gdprPermsBuilder: e.gdprPermsBuilder,
	}

	type testResults struct {
		bidFloor    float64
		bidFloorCur string
		err         error
		resolvedReq string
	}

	testCases := []struct {
		desc         string
		req          *openrtb_ext.RequestWrapper
		floorsEnable bool
		expected     testResults
	}{
		{
			desc:         "no update in imp.bidfloor, floors disabled in account config",
			floorsEnable: false,
			req: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				ID: "some-request-id",
				Imp: []openrtb2.Imp{{
					ID:          "some-impression-id",
					BidFloor:    15,
					BidFloorCur: "USD",
					Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
					Ext:         json.RawMessage(`{"prebid":{}}`),
				}},
				Site: &openrtb2.Site{
					Page:   "prebid.org",
					Ext:    json.RawMessage(`{"amp":0}`),
					Domain: "www.website.com",
				},
				Cur: []string{"USD"},
				Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"USD","data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","values":{"banner|300x250|www.website.com":11,"*|*|www.test.com":15,"*|*|*":20},"Default":50,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true}}}`),
			}},
			expected: testResults{
				bidFloor:    15.00,
				bidFloorCur: "USD",
			},
		},
		{
			desc:         "no update in imp.bidfloor due to no rule matched",
			floorsEnable: true,
			req: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				ID: "some-request-id",
				Imp: []openrtb2.Imp{{
					ID:          "some-impression-id",
					BidFloor:    15,
					BidFloorCur: "USD",
					Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
					Ext:         json.RawMessage(`{"prebid":{}}`),
				}},
				Site: &openrtb2.Site{
					Page:   "prebid.org",
					Ext:    json.RawMessage(`{"amp":0}`),
					Domain: "www.website.com",
				},
				Cur: []string{"USD"},
				Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"USD","data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","values":{"banner|300x250|www.website123.com":10,"*|*|www.test.com":15},"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true}}}`),
			}},
			expected: testResults{
				bidFloor:    15.00,
				bidFloorCur: "USD",
			},
		},
		{
			desc:         "update imp.bidfloor with matched rule value",
			floorsEnable: true,
			req: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				ID: "some-request-id",
				Imp: []openrtb2.Imp{{
					ID:          "some-impression-id",
					BidFloor:    15,
					BidFloorCur: "USD",
					Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
					Ext:         json.RawMessage(`{"prebid":{}}`),
				}},
				Site: &openrtb2.Site{
					Page:   "prebid.org",
					Ext:    json.RawMessage(`{"amp":0}`),
					Domain: "www.website.com",
				},
				Cur: []string{"USD"},
				Ext: json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"USD","data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","values":{"banner|300x250|www.website.com":10,"*|*|www.test.com":15,"*|*|*":20},"Default":50,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true}}}`),
			}},
			expected: testResults{
				bidFloor:    10.00,
				bidFloorCur: "USD",
			},
		},
		{
			desc:         "update resolved request with floors details",
			floorsEnable: true,
			req: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				ID: "some-request-id",
				Imp: []openrtb2.Imp{{
					ID:          "some-impression-id",
					BidFloor:    15,
					BidFloorCur: "USD",
					Banner:      &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
					Ext:         json.RawMessage(`{"prebid":{}}`),
				}},
				Site: &openrtb2.Site{
					Page:   "prebid.org",
					Ext:    json.RawMessage(`{"amp":0}`),
					Domain: "www.website.com",
				},
				Test: 1,
				Cur:  []string{"USD"},
				Ext:  json.RawMessage(`{"prebid":{"floors":{"floormin":1,"floormincur":"USD","data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","values":{"banner|300x250|www.website.com":11,"*|*|www.test.com":15,"*|*|*":20},"Default":50,"schema":{"fields":["mediaType","size","domain"],"delimiter":"|"}}]},"enabled":true}}}`),
			}},
			expected: testResults{
				bidFloor:    11.00,
				bidFloorCur: "USD",
				resolvedReq: `{"id":"some-request-id","imp":[{"id":"some-impression-id","banner":{"format":[{"w":300,"h":250}]},"bidfloor":11,"bidfloorcur":"USD","ext":{"prebid":{"floors":{"floorrule":"banner|300x250|www.website.com","floorrulevalue":11,"floorvalue":11}}}}],"site":{"domain":"www.website.com","page":"prebid.org","ext":{"amp":0}},"test":1,"cur":["USD"],"ext":{"prebid":{"floors":{"floormin":1,"floormincur":"USD","data":{"currency":"USD","modelgroups":[{"modelversion":"model 1 from req","schema":{"fields":["mediaType","size","domain"],"delimiter":"|"},"values":{"*|*|*":20,"*|*|www.test.com":15,"banner|300x250|www.website.com":11},"default":50}]},"enabled":true,"skipped":false,"fetchstatus":"none","location":"request"}}}}`,
			},
		},
	}

	for _, test := range testCases {
		auctionRequest := &AuctionRequest{
			BidRequestWrapper: test.req,
			Account:           config.Account{DebugAllow: true, PriceFloors: config.AccountPriceFloors{Enabled: test.floorsEnable, MaxRule: 100, MaxSchemaDims: 5}},
			UserSyncs:         &emptyUsersync{},
			HookExecutor:      &hookexecution.EmptyHookExecutor{},
			TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}
		outBidResponse, err := e.HoldAuction(context.Background(), auctionRequest, &DebugLog{})

		// Assertions
		assert.Equal(t, test.expected.err, err, "Error")
		assert.Equal(t, test.expected.bidFloor, auctionRequest.BidRequestWrapper.Imp[0].BidFloor, "Floor Value")
		assert.Equal(t, test.expected.bidFloorCur, auctionRequest.BidRequestWrapper.Imp[0].BidFloorCur, "Floor Currency")

		if test.req.Test == 1 {
			actualResolvedRequest, _, _, _ := jsonparser.Get(outBidResponse.Ext, "debug", "resolvedrequest")
			assert.JSONEq(t, test.expected.resolvedReq, string(actualResolvedRequest), "Resolved request is incorrect")
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
	e.adapterMap = map[openrtb_ext.BidderName]AdaptedBidder{
		openrtb_ext.BidderAppnexus: AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, nil, ""),
	}
	e.cache = &wellBehavedCache{}
	e.me = &metricsConf.NilMetricsEngine{}
	e.gdprPermsBuilder = fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder
	e.currencyConverter = currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	e.categoriesFetcher = categoriesFetcher
	e.bidIDGenerator = &fakeBidIDGenerator{GenerateBidID: false, ReturnError: false}
	e.requestSplitter = requestSplitter{
		me:               e.me,
		gdprPermsBuilder: e.gdprPermsBuilder,
	}

	// Define mock incoming bid requeset
	mockBidRequest := &openrtb2.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb2.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext:    json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placementId":1}}}}`),
		}},
		Site: &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
	}

	// Run tests
	for _, testGroup := range testGroups {
		for _, test := range testGroup.testCases {
			mockBidRequest.Ext = test.inExt

			auctionRequest := &AuctionRequest{
				BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: mockBidRequest},
				Account:           config.Account{},
				UserSyncs:         &emptyUsersync{},
				HookExecutor:      &hookexecution.EmptyHookExecutor{},
				TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
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
				assert.False(t, auctionRequest.BidderResponseStartTime.IsZero())
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
	syncerKeys := []string{}
	var moduleStageNames map[string][]string
	testEngine := metricsConf.NewMetricsEngine(cfg, adapterList, syncerKeys, moduleStageNames)
	//	2) Init new exchange with said configuration
	handlerNoBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	server := httptest.NewServer(http.HandlerFunc(handlerNoBidServer))
	defer server.Close()

	biddersInfo, err := config.LoadBidderInfoFromDisk("../static/bidder-info")
	if err != nil {
		t.Fatal(err)
	}

	adapters, _, adaptersErr := BuildAdapters(server.Client(), cfg, biddersInfo, &metricsConf.NilMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}
	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	pbc := pbc.NewClient(&http.Client{}, &cfg.CacheURL, &cfg.ExtCacheURL, testEngine)

	gdprPermsBuilder := fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder

	e := NewExchange(adapters, pbc, cfg, &mockRequestValidator{}, map[string]usersync.Syncer{}, &metricsConf.NilMetricsEngine{}, biddersInfo, gdprPermsBuilder, currencyConverter, nilCategoryFetcher{}, &adscert.NilSigner{}, macros.NewStringIndexBasedReplacer(), nil, nil).(*exchange)
	// 	3) Build all the parameters e.buildBidResponse(ctx.Background(), liveA... ) needs
	liveAdapters := []openrtb_ext.BidderName{bidderName}

	//adapterBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid,
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
	aPbsOrtbBidArr := []*entities.PbsOrtbBid{
		{
			Bid:     bids[0],
			BidType: openrtb_ext.BidTypeBanner,
			BidTargets: map[string]string{
				"pricegranularity":  "med",
				"includewinners":    "true",
				"includebidderkeys": "false",
			},
		},
	}
	adapterBids := map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
		bidderName: {
			Bids:     aPbsOrtbBidArr,
			Currency: "USD",
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
	bidRequest := &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
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
		},
	}

	var errList []error

	// 	4) Build bid response
	bid_resp := e.buildBidResponse(context.Background(), liveAdapters, adapterBids, bidRequest, adapterExtra, auc, nil, true, nil, "", errList, &SeatNonBidBuilder{})

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
	sampleBids := []*entities.PbsOrtbBid{
		{
			Bid:            sampleOpenrtbBid,
			BidType:        openrtb_ext.BidTypeBanner,
			BidTargets:     map[string]string{},
			GeneratedBidID: "randomId",
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
	e.adapterMap = map[openrtb_ext.BidderName]AdaptedBidder{
		openrtb_ext.BidderAppnexus: AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, nil, ""),
	}
	e.cache = &wellBehavedCache{}
	e.me = &metricsConf.NilMetricsEngine{}

	e.currencyConverter = currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	//Run tests
	for _, test := range testCases {
		resultingBids, resultingErrs := e.makeBid(sampleBids, sampleAuction, test.inReturnCreative, nil, &openrtb_ext.RequestWrapper{}, nil, "", "", &SeatNonBidBuilder{})

		assert.Equal(t, 0, len(resultingErrs), "%s. Test should not return errors \n", test.description)
		assert.Equal(t, test.expectedCreativeMarkup, resultingBids[0].AdM, "%s. Ad markup string doesn't match expected \n", test.description)

		var bidExt openrtb_ext.ExtBid
		jsonutil.UnmarshalValid(resultingBids[0].Ext, &bidExt)
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
		bid              *entities.PbsOrtbBid
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
			bid:              &entities.PbsOrtbBid{Bid: bid},
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
			bid:              &entities.PbsOrtbBid{Bid: bid},
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
			bid:              &entities.PbsOrtbBid{Bid: bid},
			auction:          &auction{},
			expectedFound:    false,
			expectedCacheID:  "",
			expectedCacheURL: "",
		},
		{
			description:      "Scheme Not Provided",
			host:             "prebid.org",
			path:             "cache",
			bid:              &entities.PbsOrtbBid{Bid: bid},
			auction:          &auction{cacheIds: map[*openrtb2.Bid]string{bid: "anyID"}},
			expectedFound:    true,
			expectedCacheID:  "anyID",
			expectedCacheURL: "prebid.org/cache?uuid=anyID",
		},
		{
			description:      "Host And Path Not Provided - Without Scheme",
			bid:              &entities.PbsOrtbBid{Bid: bid},
			auction:          &auction{cacheIds: map[*openrtb2.Bid]string{bid: "anyID"}},
			expectedFound:    true,
			expectedCacheID:  "anyID",
			expectedCacheURL: "",
		},
		{
			description:      "Host And Path Not Provided - With Scheme",
			scheme:           "https",
			bid:              &entities.PbsOrtbBid{Bid: bid},
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
			bid:              &entities.PbsOrtbBid{Bid: nil},
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
			bid:              &entities.PbsOrtbBid{Bid: bid},
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
	cfg := &config.Configuration{}

	handlerNoBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	server := httptest.NewServer(http.HandlerFunc(handlerNoBidServer))
	defer server.Close()

	biddersInfo, err := config.LoadBidderInfoFromDisk("../static/bidder-info")
	if err != nil {
		t.Fatal(err)
	}

	adapters, _, adaptersErr := BuildAdapters(server.Client(), cfg, biddersInfo, &metricsConf.NilMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}

	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	gdprPermsBuilder := fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder

	e := NewExchange(adapters, nil, cfg, &mockRequestValidator{}, map[string]usersync.Syncer{}, &metricsConf.NilMetricsEngine{}, biddersInfo, gdprPermsBuilder, currencyConverter, nilCategoryFetcher{}, &adscert.NilSigner{}, macros.NewStringIndexBasedReplacer(), nil, nil).(*exchange)

	liveAdapters := make([]openrtb_ext.BidderName, 1)
	liveAdapters[0] = "appnexus"

	bidRequest := &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
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
		},
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
	aPbsOrtbBidArr := []*entities.PbsOrtbBid{{Bid: sampleBid, BidType: openrtb_ext.BidTypeBanner, OriginalBidCPM: 9.517803}}
	sampleSeatBid := []openrtb2.SeatBid{
		{
			Seat: "appnexus",
			Bid: []openrtb2.Bid{
				{
					ID:    "some-imp-id",
					Price: 9.517803,
					W:     300,
					H:     250,
					Ext:   json.RawMessage(`{"origbidcpm":9.517803,"prebid":{"meta":{},"type":"banner"}}`),
				},
			},
		},
	}
	emptySeatBid := []openrtb2.SeatBid{}

	// Test cases
	type aTest struct {
		description         string
		adapterBids         map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		expectedBidResponse *openrtb2.BidResponse
	}
	testCases := []aTest{
		{
			description: "1) Adapter to bids map comes with a non-empty currency field and non-empty bid array",
			adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				openrtb_ext.BidderName("appnexus"): {
					Bids:     aPbsOrtbBidArr,
					Currency: "USD",
				},
			},
			expectedBidResponse: &openrtb2.BidResponse{
				ID:      "some-request-id",
				SeatBid: sampleSeatBid,
				Cur:     "USD",
			},
		},
		{
			description: "2) Adapter to bids map comes with a non-empty currency field but an empty bid array",
			adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				openrtb_ext.BidderName("appnexus"): {
					Bids:     nil,
					Currency: "USD",
				},
			},
			expectedBidResponse: &openrtb2.BidResponse{
				ID:      "some-request-id",
				SeatBid: emptySeatBid,
				Cur:     "",
			},
		},
		{
			description: "3) Adapter to bids map comes with an empty currency string and a non-empty bid array",
			adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				openrtb_ext.BidderName("appnexus"): {
					Bids:     aPbsOrtbBidArr,
					Currency: "",
				},
			},
			expectedBidResponse: &openrtb2.BidResponse{
				ID:      "some-request-id",
				SeatBid: sampleSeatBid,
				Cur:     "",
			},
		},
		{
			description: "4) Adapter to bids map comes with an empty currency string and an empty bid array",
			adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				openrtb_ext.BidderName("appnexus"): {
					Bids:     nil,
					Currency: "",
				},
			},
			expectedBidResponse: &openrtb2.BidResponse{
				ID:      "some-request-id",
				SeatBid: emptySeatBid,
				Cur:     "",
			},
		},
	}

	bidResponseExt := &openrtb_ext.ExtBidResponse{
		ResponseTimeMillis:   map[openrtb_ext.BidderName]int{openrtb_ext.BidderName("appnexus"): 5},
		RequestTimeoutMillis: 500,
	}
	// Run tests
	for i := range testCases {
		actualBidResp := e.buildBidResponse(context.Background(), liveAdapters, testCases[i].adapterBids, bidRequest, adapterExtra, nil, bidResponseExt, true, nil, "", errList, &SeatNonBidBuilder{})
		assert.Equalf(t, testCases[i].expectedBidResponse, actualBidResp, fmt.Sprintf("[TEST_FAILED] Objects must be equal for test: %s \n Expected: >>%s<< \n Actual: >>%s<< ", testCases[i].description, testCases[i].expectedBidResponse.Ext, actualBidResp.Ext))
	}
}

func TestBidResponseImpExtInfo(t *testing.T) {
	// Init objects
	cfg := &config.Configuration{}

	gdprPermsBuilder := fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder

	noBidHandler := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	server := httptest.NewServer(http.HandlerFunc(noBidHandler))
	defer server.Close()

	biddersInfo := config.BidderInfos{"appnexus": config.BidderInfo{Endpoint: "http://ib.adnxs.com"}}

	adapters, _, adaptersErr := BuildAdapters(server.Client(), cfg, biddersInfo, &metricsConf.NilMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}

	e := NewExchange(adapters, nil, cfg, &mockRequestValidator{}, map[string]usersync.Syncer{}, &metricsConf.NilMetricsEngine{}, nil, gdprPermsBuilder, nil, nilCategoryFetcher{}, &adscert.NilSigner{}, macros.NewStringIndexBasedReplacer(), nil, nil).(*exchange)

	liveAdapters := make([]openrtb_ext.BidderName, 1)
	liveAdapters[0] = "appnexus"

	bidRequest := &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			ID: "some-request-id",
			Imp: []openrtb2.Imp{{
				ID:    "some-impression-id",
				Video: &openrtb2.Video{},
				Ext:   json.RawMessage(`{"appnexus": {"placementId": 10433394}}`),
			}},
			Ext: json.RawMessage(``),
		},
	}

	var errList []error

	sampleBid := &openrtb2.Bid{
		ID:    "some-imp-id",
		ImpID: "some-impression-id",
		W:     300,
		H:     250,
		Ext:   nil,
	}
	aPbsOrtbBidArr := []*entities.PbsOrtbBid{{Bid: sampleBid, BidType: openrtb_ext.BidTypeVideo, AdapterCode: openrtb_ext.BidderAppnexus}}

	adapterBids := map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
		openrtb_ext.BidderName("appnexus"): {
			Bids: aPbsOrtbBidArr,
		},
	}

	impExtInfo := make(map[string]ImpExtInfo, 1)
	impExtInfo["some-impression-id"] = ImpExtInfo{
		true,
		[]byte(`{"video":{"h":480,"mimes":["video/mp4"]}}`),
		json.RawMessage(`{"imp_passthrough_val":1}`)}

	expectedBidResponseExt := `{"origbidcpm":0,"prebid":{"meta":{"adaptercode":"appnexus"},"type":"video","passthrough":{"imp_passthrough_val":1}},"storedrequestattributes":{"h":480,"mimes":["video/mp4"]}}`

	actualBidResp := e.buildBidResponse(context.Background(), liveAdapters, adapterBids, bidRequest, nil, nil, nil, true, impExtInfo, "", errList, &SeatNonBidBuilder{})

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

	cfg := &config.Configuration{}

	biddersInfo, err := config.LoadBidderInfoFromDisk("../static/bidder-info")
	if err != nil {
		t.Fatal(err)
	}

	adapters, _, adaptersErr := BuildAdapters(server.Client(), cfg, biddersInfo, &metricsConf.NilMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}

	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	auctionRequest := &AuctionRequest{
		BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: getTestBuildRequest(t)},
		Account:           config.Account{},
		UserSyncs:         &emptyUsersync{},
		HookExecutor:      &hookexecution.EmptyHookExecutor{},
		TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
	}

	debugLog := DebugLog{}

	gdprPermsBuilder := fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder

	ex := NewExchange(adapters, &wellBehavedCache{}, cfg, &mockRequestValidator{}, map[string]usersync.Syncer{}, &metricsConf.NilMetricsEngine{}, biddersInfo, gdprPermsBuilder, currencyConverter, &nilCategoryFetcher{}, &adscert.NilSigner{}, macros.NewStringIndexBasedReplacer(), nil, nil).(*exchange)
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

func getTestBuildRequest(t *testing.T) *openrtb2.BidRequest {
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
			Ext: json.RawMessage(`{"prebid":{"bidder":{"appnexus": {"placementId": 1}}}}`),
		}, {
			Video: &openrtb2.Video{
				MIMEs:       []string{"video/mp4"},
				MinDuration: 1,
				MaxDuration: 300,
				W:           ptrutil.ToPtr[int64](300),
				H:           ptrutil.ToPtr[int64](600),
			},
			Ext: json.RawMessage(`{"prebid":{"bidder":{"appnexus": {"placementId": 1}}}}`),
		}},
	}
}

func TestPanicRecovery(t *testing.T) {
	cfg := &config.Configuration{
		CacheURL: config.Cache{
			ExpectedTimeMillis: 20,
		},
	}

	biddersInfo, err := config.LoadBidderInfoFromDisk("../static/bidder-info")
	if err != nil {
		t.Fatal(err)
	}

	adapters, _, adaptersErr := BuildAdapters(&http.Client{}, cfg, biddersInfo, &metricsConf.NilMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}

	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	gdprPermsBuilder := fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder

	e := NewExchange(adapters, nil, cfg, &mockRequestValidator{}, map[string]usersync.Syncer{}, &metricsConf.NilMetricsEngine{}, biddersInfo, gdprPermsBuilder, currencyConverter, nilCategoryFetcher{}, &adscert.NilSigner{}, macros.NewStringIndexBasedReplacer(), nil, nil).(*exchange)

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

// TestPanicRecoveryHighLevel calls HoldAuction with a panicingAdapter{}
func TestPanicRecoveryHighLevel(t *testing.T) {
	noBidServer := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}
	server := httptest.NewServer(http.HandlerFunc(noBidServer))
	defer server.Close()

	cfg := &config.Configuration{}

	biddersInfo, err := config.LoadBidderInfoFromDisk("../static/bidder-info")
	if err != nil {
		t.Fatal(err)
	}

	adapters, _, adaptersErr := BuildAdapters(server.Client(), cfg, biddersInfo, &metricsConf.NilMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}

	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	gdprPermsBuilder := fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder
	e := NewExchange(adapters, &mockCache{}, cfg, &mockRequestValidator{}, map[string]usersync.Syncer{}, &metricsConf.NilMetricsEngine{}, biddersInfo, gdprPermsBuilder, currencyConverter, categoriesFetcher, &adscert.NilSigner{}, macros.NewStringIndexBasedReplacer(), nil, nil).(*exchange)

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
			Ext: json.RawMessage(`{"ext_field": "value"}`),
		}},
	}

	auctionRequest := &AuctionRequest{
		BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: request},
		Account:           config.Account{},
		UserSyncs:         &emptyUsersync{},
		HookExecutor:      &hookexecution.EmptyHookExecutor{},
		TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
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

// TestExchangeJSON executes tests for all the *.json files in exchangetest.
func TestExchangeJSON(t *testing.T) {
	if specFiles, err := os.ReadDir("./exchangetest"); err == nil {
		for _, specFile := range specFiles {
			if !strings.HasSuffix(specFile.Name(), ".json") {
				continue
			}

			fileName := "./exchangetest/" + specFile.Name()
			fileDisplayName := "exchange/exchangetest/" + specFile.Name()

			t.Run(fileDisplayName, func(t *testing.T) {
				specData, err := loadFile(fileName)
				if assert.NoError(t, err, "Failed to load contents of file %s: %v", fileDisplayName, err) {
					assert.NotPanics(t, func() { runSpec(t, fileDisplayName, specData) }, fileDisplayName)
				}
			})
		}
	}
}

// LoadFile reads and parses a file as a test case. If something goes wrong, it returns an error.
func loadFile(filename string) (*exchangeSpec, error) {
	specData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file %s: %v", filename, err)
	}

	var spec exchangeSpec
	if err := jsonutil.UnmarshalValid(specData, &spec); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal JSON from file: %v", err)
	}

	return &spec, nil
}

func runSpec(t *testing.T, filename string, spec *exchangeSpec) {
	aliases, err := parseRequestAliases(spec.IncomingRequest.OrtbRequest)
	if err != nil {
		t.Fatalf("%s: Failed to parse aliases", filename)
	}

	var s struct{}
	eeacMap := make(map[string]struct{})
	eeac := []string{"FIN", "FRA", "GUF"}
	for _, c := range eeac {
		eeacMap[c] = s
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
			EEACountries:    eeac,
			EEACountriesMap: eeacMap,
			TCF2: config.TCF2{
				Enabled: spec.GDPREnabled,
			},
		},
	}
	bidIdGenerator := &fakeBidIDGenerator{}
	if spec.BidIDGenerator != nil {
		bidIdGenerator = spec.BidIDGenerator
	}
	ex := newExchangeForTests(t, filename, aliases, privacyConfig, bidIdGenerator, spec)
	biddersInAuction := findBiddersInAuction(t, filename, &spec.IncomingRequest.OrtbRequest)
	debugLog := &DebugLog{}
	if spec.DebugLog != nil {
		*debugLog = *spec.DebugLog
		debugLog.Regexp = regexp.MustCompile(`[<>]`)
	}

	// Passthrough JSON Testing
	impExtInfoMap := make(map[string]ImpExtInfo)
	if spec.PassthroughFlag {
		impPassthrough, impID, err := getInfoFromImp(&openrtb_ext.RequestWrapper{BidRequest: &spec.IncomingRequest.OrtbRequest})
		if err != nil {
			t.Errorf("%s: Exchange returned an unexpected error. Got %s", filename, err.Error())
		}
		impExtInfoMap[impID] = ImpExtInfo{Passthrough: impPassthrough}
	}

	// Imp Setting for Bid Validation
	if spec.HostConfigBidValidation.SecureMarkup == config.ValidationEnforce || spec.HostConfigBidValidation.SecureMarkup == config.ValidationWarn {
		_, impID, err := getInfoFromImp(&openrtb_ext.RequestWrapper{BidRequest: &spec.IncomingRequest.OrtbRequest})
		if err != nil {
			t.Errorf("%s: Exchange returned an unexpected error. Got %s", filename, err.Error())
		}
		impExtInfoMap[impID] = ImpExtInfo{}
	}

	if spec.AccountPrivacy.DSA != nil && len(spec.AccountPrivacy.DSA.Default) > 0 {
		if err := jsonutil.Unmarshal([]byte(spec.AccountPrivacy.DSA.Default), &spec.AccountPrivacy.DSA.DefaultUnpacked); err != nil {
			t.Errorf("%s: Exchange returned an unexpected error. Got %s", filename, err.Error())
		}
	}

	activityControl := privacy.NewActivityControl(&spec.AccountPrivacy)

	auctionRequest := &AuctionRequest{
		BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: &spec.IncomingRequest.OrtbRequest},
		Account: config.Account{
			ID: "testaccount",
			Events: config.Events{
				Enabled: spec.EventsEnabled,
			},
			DebugAllow:  true,
			PriceFloors: config.AccountPriceFloors{Enabled: spec.AccountFloorsEnabled, EnforceDealFloors: spec.AccountEnforceDealFloors},
			Privacy:     spec.AccountPrivacy,
			Validations: spec.AccountConfigBidValidation,
			GDPR:        config.AccountGDPR{EEACountries: spec.AccountEEACountries},
		},
		UserSyncs:     mockIdFetcher(spec.IncomingRequest.Usersyncs),
		ImpExtInfoMap: impExtInfoMap,
		HookExecutor:  &hookexecution.EmptyHookExecutor{},
		TCF2Config:    gdpr.NewTCF2Config(privacyConfig.GDPR.TCF2, config.AccountGDPR{}),
		Activities:    activityControl,
	}

	if spec.MultiBid != nil {
		auctionRequest.Account.DefaultBidLimit = spec.MultiBid.AccountMaxBid

		requestExt := &openrtb_ext.ExtRequest{}
		err := jsonutil.UnmarshalValid(spec.IncomingRequest.OrtbRequest.Ext, requestExt)
		assert.NoError(t, err, "invalid request ext")
		validatedMultiBids, errs := openrtb_ext.ValidateAndBuildExtMultiBid(&requestExt.Prebid)
		for _, err := range errs { // same as in validateRequestExt().
			auctionRequest.Warnings = append(auctionRequest.Warnings, &errortypes.Warning{
				WarningCode: errortypes.MultiBidWarningCode,
				Message:     err.Error(),
			})
		}

		requestExt.Prebid.MultiBid = validatedMultiBids
		updateReqExt, err := jsonutil.Marshal(requestExt)
		assert.NoError(t, err, "invalid request ext")
		auctionRequest.BidRequestWrapper.Ext = updateReqExt
	}

	if spec.StartTime > 0 {
		auctionRequest.StartTime = time.Unix(0, spec.StartTime*1e+6)
	}
	if spec.RequestType != nil {
		auctionRequest.RequestType = *spec.RequestType
	}
	ctx := context.Background()

	aucResponse, err := ex.HoldAuction(ctx, auctionRequest, debugLog)
	var bid *openrtb2.BidResponse
	var bidExt *openrtb_ext.ExtBidResponse
	if aucResponse != nil {
		bid = aucResponse.BidResponse
		bidExt = aucResponse.ExtBidResponse
	}
	if len(spec.Response.Error) > 0 && spec.Response.Bids == nil {
		if err.Error() != spec.Response.Error {
			t.Errorf("%s: Exchange returned different errors. Expected %s, got %s", filename, spec.Response.Error, err.Error())
		}
		return
	}
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
		assert.JSONEq(t, string(bid.Ext), string(spec.Response.Ext), "Debug info modified")
	}

	if spec.PassthroughFlag || (spec.MultiBid != nil && spec.MultiBid.AssertMultiBidWarnings) {
		expectedPassthough := ""
		actualPassthrough := ""
		actualBidRespExt := &openrtb_ext.ExtBidResponse{}
		if bid.Ext != nil {
			if err := jsonutil.UnmarshalValid(bid.Ext, actualBidRespExt); err != nil {
				assert.NoError(t, err, fmt.Sprintf("Error when unmarshalling: %s", err))
			}
			if actualBidRespExt.Prebid != nil {
				actualPassthrough = string(actualBidRespExt.Prebid.Passthrough)
			}
		}
		expectedBidRespExt := &openrtb_ext.ExtBidResponse{}
		if spec.Response.Ext != nil {
			if err := jsonutil.UnmarshalValid(spec.Response.Ext, expectedBidRespExt); err != nil {
				assert.NoError(t, err, fmt.Sprintf("Error when unmarshalling: %s", err))
			}
			if expectedBidRespExt.Prebid != nil {
				expectedPassthough = string(expectedBidRespExt.Prebid.Passthrough)
			}
		}

		if spec.MultiBid != nil && spec.MultiBid.AssertMultiBidWarnings {
			assert.Equal(t, expectedBidRespExt.Warnings, actualBidRespExt.Warnings, "Expected same multi-bid warnings")
		}

		if spec.PassthroughFlag {
			// special handling since JSONEq fails if either parameters is an empty string instead of json
			if expectedPassthough == "" || actualPassthrough == "" {
				assert.Equal(t, expectedPassthough, actualPassthrough, "Expected bid response extension is incorrect")
			} else {
				assert.JSONEq(t, expectedPassthough, actualPassthrough, "Expected bid response extension is incorrect")
			}
		}

	}

	if spec.FledgeEnabled {
		assert.JSONEq(t, string(spec.Response.Ext), string(bid.Ext), "ext mismatch")
	}

	expectedBidRespExt := &openrtb_ext.ExtBidResponse{}
	if spec.Response.Ext != nil {
		if err := jsonutil.UnmarshalValid(spec.Response.Ext, expectedBidRespExt); err != nil {
			assert.NoError(t, err, fmt.Sprintf("Error when unmarshalling: %s", err))
		}
	}
	if spec.HostConfigBidValidation.BannerCreativeMaxSize == config.ValidationEnforce || spec.HostConfigBidValidation.SecureMarkup == config.ValidationEnforce {
		actualBidRespExt := &openrtb_ext.ExtBidResponse{}
		if bid.Ext != nil {
			if err := jsonutil.UnmarshalValid(bid.Ext, actualBidRespExt); err != nil {
				assert.NoError(t, err, fmt.Sprintf("Error when unmarshalling: %s", err))
			}
		}
		assert.Equal(t, expectedBidRespExt.Errors, actualBidRespExt.Errors, "Expected errors from response ext do not match")
	}
	if expectedBidRespExt.Prebid != nil {
		assert.ElementsMatch(t, expectedBidRespExt.Prebid.SeatNonBid, bidExt.Prebid.SeatNonBid, "Expected seatNonBids from response ext do not match")
	}
}

func findBiddersInAuction(t *testing.T, context string, req *openrtb2.BidRequest) []string {

	if splitImps, err := splitImps(req.Imp, &mockRequestValidator{}, nil, false, nil); err != nil {
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
		if err := jsonutil.UnmarshalValid(data, &responseTimes); err != nil {
			t.Errorf("%s: Failed to unmarshal ext.responsetimemillis into map[string]int: %v", context, err)
			return nil
		}

		// Delete the response times so that they don't appear in the JSON, because they can't be tested reliably anyway.
		// If there's no other ext, just delete it altogether.
		bid.Ext = jsonparser.Delete(bid.Ext, "responsetimemillis")
		if jsonpatch.Equal(bid.Ext, []byte("{}")) {
			bid.Ext = nil
		}
		return responseTimes
	}
}

func newExchangeForTests(t *testing.T, filename string, aliases map[string]string, privacyConfig config.Privacy, bidIDGenerator BidIDGenerator, exSpec *exchangeSpec) Exchange {
	bidderAdapters := make(map[openrtb_ext.BidderName]AdaptedBidder, len(exSpec.OutgoingRequests))
	bidderInfos := make(config.BidderInfos, len(exSpec.OutgoingRequests))
	for _, bidderName := range openrtb_ext.CoreBidderNames() {
		if spec, ok := exSpec.OutgoingRequests[string(bidderName)]; ok {
			bidderAdapters[bidderName] = &validatingBidder{
				t:             t,
				fileName:      filename,
				bidderName:    string(bidderName),
				expectations:  map[string]*bidderRequest{string(bidderName): spec.ExpectedRequest},
				mockResponses: map[string]bidderResponse{string(bidderName): spec.MockResponse},
			}
			ortbVersion, _ := exSpec.ORTBVersion[string(bidderName)]
			bidderInfos[string(bidderName)] = config.BidderInfo{
				ModifyingVastXmlAllowed: spec.ModifyingVastXmlAllowed,
				OpenRTB:                 &config.OpenRTBInfo{Version: ortbVersion},
			}
		} else {
			bidderInfos[string(bidderName)] = config.BidderInfo{}
		}
	}

	for alias, coreBidder := range aliases {
		if spec, ok := exSpec.OutgoingRequests[alias]; ok {
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

	gdprPermsBuilder := fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder

	bidderToSyncerKey := map[string]string{}
	for _, bidderName := range openrtb_ext.CoreBidderNames() {
		bidderToSyncerKey[string(bidderName)] = string(bidderName)
	}

	gdprDefaultValue := gdpr.SignalYes
	if privacyConfig.GDPR.DefaultValue == "0" {
		gdprDefaultValue = gdpr.SignalNo
	}

	var hostSChainNode *openrtb2.SupplyChainNode
	if exSpec.HostSChainFlag {
		hostSChainNode = &openrtb2.SupplyChainNode{
			ASI: "pbshostcompany.com", SID: "00001", RID: "BidRequest", HP: openrtb2.Int8Ptr(1),
		}
	}

	metricsEngine := metricsConf.NewMetricsEngine(&config.Configuration{}, openrtb_ext.CoreBidderNames(), nil, nil)
	paramValidator, err := openrtb_ext.NewBidderParamsValidator("../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to create params validator: %v", error)
	}
	bidderMap := GetActiveBidders(bidderInfos)
	requestValidator := ortb.NewRequestValidator(bidderMap, map[string]string{}, paramValidator)
	requestSplitter := requestSplitter{
		bidderToSyncerKey: bidderToSyncerKey,
		me:                metricsEngine,
		privacyConfig:     privacyConfig,
		gdprPermsBuilder:  gdprPermsBuilder,
		hostSChainNode:    hostSChainNode,
		bidderInfo:        bidderInfos,
		requestValidator:  requestValidator,
	}

	return &exchange{
		adapterMap:               bidderAdapters,
		me:                       metricsEngine,
		cache:                    &wellBehavedCache{},
		cacheTime:                0,
		currencyConverter:        currency.NewRateConverter(&http.Client{}, "", time.Duration(0)),
		gdprDefaultValue:         gdprDefaultValue,
		gdprPermsBuilder:         gdprPermsBuilder,
		privacyConfig:            privacyConfig,
		categoriesFetcher:        categoriesFetcher,
		bidderInfo:               bidderInfos,
		bidderToSyncerKey:        bidderToSyncerKey,
		externalURL:              "http://localhost",
		bidIDGenerator:           bidIDGenerator,
		hostSChainNode:           hostSChainNode,
		server:                   config.Server{ExternalUrl: exSpec.Server.ExternalUrl, GvlID: exSpec.Server.GvlID, DataCenter: exSpec.Server.DataCenter},
		bidValidationEnforcement: exSpec.HostConfigBidValidation,
		requestSplitter:          requestSplitter,
		priceFloorEnabled:        exSpec.FloorsEnabled,
		priceFloorFetcher:        &mockPriceFloorFetcher{},
	}
}

type fakeBidIDGenerator struct {
	GenerateBidID bool `json:"generateBidID"`
	ReturnError   bool `json:"returnError"`
	bidCount      map[string]int
}

func (f *fakeBidIDGenerator) Enabled() bool {
	return f.GenerateBidID
}

func (f *fakeBidIDGenerator) New(bidder string) (string, error) {
	if f.ReturnError {
		return "", errors.New("Test error generating bid.ext.prebid.bidid")
	}

	if f.bidCount == nil {
		f.bidCount = make(map[string]int)
	}

	f.bidCount[bidder] += 1
	return fmt.Sprintf("bid-%v-%v", bidder, f.bidCount[bidder]), nil
}

type fakeBooleanGenerator struct {
	value bool
}

func (f *fakeBooleanGenerator) Generate() bool {
	return f.value
}

func newExtRequest() openrtb_ext.ExtRequest {
	priceGran := openrtb_ext.PriceGranularity{
		Precision: ptrutil.ToPtr(2),
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
		PriceGranularity:     &priceGran,
		IncludeWinners:       ptrutil.ToPtr(true),
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
		Precision: ptrutil.ToPtr(2),
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
		PriceGranularity:     &priceGran,
		IncludeWinners:       ptrutil.ToPtr(true),
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
		priceGranularity: *requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	requestExt.Prebid.Targeting.DurationRangeSec = []int{15, 30, 50}

	adapterBids := make(map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid)

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}
	cats3 := []string{"IAB1-1000"}
	cats4 := []string{"IAB1-2000"}
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 20.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb2.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 30.0000, Cat: cats3, W: 1, H: 1}
	bid4 := openrtb2.Bid{ID: "bid_id4", ImpID: "imp_id4", Price: 40.0000, Cat: cats4, W: 1, H: 1}

	bid1_1 := entities.PbsOrtbBid{Bid: &bid1, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_2 := entities.PbsOrtbBid{Bid: &bid2, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 40}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 20.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_3 := entities.PbsOrtbBid{Bid: &bid3, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30, PrimaryCategory: "AdapterOverride"}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 30.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_4 := entities.PbsOrtbBid{Bid: &bid4, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 40.0000, OriginalBidCur: "USD", TargetBidderCode: ""}

	innerBids := []*entities.PbsOrtbBid{
		&bid1_1,
		&bid1_2,
		&bid1_3,
		&bid1_4,
	}

	seatBid := entities.PbsOrtbSeatBid{Bids: innerBids, Currency: "USD"}
	bidderName1 := openrtb_ext.BidderName("appnexus")

	adapterBids[bidderName1] = &seatBid

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(context.TODO(), *requestExt.Prebid.Targeting, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{}, &SeatNonBidBuilder{})

	assert.Equal(t, nil, err, "Category mapping error should be empty")
	assert.Equal(t, 1, len(rejections), "There should be 1 bid rejection message")
	assert.Equal(t, "bid rejected [bid ID: bid_id4] reason: Category mapping file for primary ad server: 'freewheel', publisher: '' not found", rejections[0], "Rejection message did not match expected")
	assert.Equal(t, "10.00_Electronics_30s", bidCategory["bid_id1"], "Category mapping doesn't match")
	assert.Equal(t, "20.00_Sports_50s", bidCategory["bid_id2"], "Category mapping doesn't match")
	assert.Equal(t, "20.00_AdapterOverride_30s", bidCategory["bid_id3"], "Category mapping override from adapter didn't take")
	assert.Equal(t, 3, len(adapterBids[bidderName1].Bids), "Bidders number doesn't match")
	assert.Equal(t, 3, len(bidCategory), "Bidders category mapping doesn't match")
}

func TestCategoryMappingNoIncludeBrandCategory(t *testing.T) {

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequestNoBrandCat()

	targData := &targetData{
		priceGranularity: *requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}
	requestExt.Prebid.Targeting.DurationRangeSec = []int{15, 30, 40, 50}

	adapterBids := make(map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid)

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}
	cats3 := []string{"IAB1-1000"}
	cats4 := []string{"IAB1-2000"}
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 20.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb2.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 30.0000, Cat: cats3, W: 1, H: 1}
	bid4 := openrtb2.Bid{ID: "bid_id4", ImpID: "imp_id4", Price: 40.0000, Cat: cats4, W: 1, H: 1}

	bid1_1 := entities.PbsOrtbBid{Bid: &bid1, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_2 := entities.PbsOrtbBid{Bid: &bid2, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 40}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 20.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_3 := entities.PbsOrtbBid{Bid: &bid3, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30, PrimaryCategory: "AdapterOverride"}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 30.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_4 := entities.PbsOrtbBid{Bid: &bid4, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 50}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 40.0000, OriginalBidCur: "USD", TargetBidderCode: ""}

	innerBids := []*entities.PbsOrtbBid{
		&bid1_1,
		&bid1_2,
		&bid1_3,
		&bid1_4,
	}

	seatBid := entities.PbsOrtbSeatBid{Bids: innerBids, Currency: "USD"}
	bidderName1 := openrtb_ext.BidderName("appnexus")

	adapterBids[bidderName1] = &seatBid

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(context.TODO(), *requestExt.Prebid.Targeting, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{}, &SeatNonBidBuilder{})

	assert.Equal(t, nil, err, "Category mapping error should be empty")
	assert.Empty(t, rejections, "There should be no bid rejection messages")
	assert.Equal(t, "10.00_30s", bidCategory["bid_id1"], "Category mapping doesn't match")
	assert.Equal(t, "20.00_40s", bidCategory["bid_id2"], "Category mapping doesn't match")
	assert.Equal(t, "20.00_30s", bidCategory["bid_id3"], "Category mapping doesn't match")
	assert.Equal(t, "20.00_50s", bidCategory["bid_id4"], "Category mapping doesn't match")
	assert.Equal(t, 4, len(adapterBids[bidderName1].Bids), "Bidders number doesn't match")
	assert.Equal(t, 4, len(bidCategory), "Bidders category mapping doesn't match")
}

func TestCategoryMappingTranslateCategoriesNil(t *testing.T) {

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequestTranslateCategories(nil)

	targData := &targetData{
		priceGranularity: *requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	requestExt.Prebid.Targeting.DurationRangeSec = []int{15, 30, 50}

	adapterBids := make(map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid)

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}
	cats3 := []string{"IAB1-1000"}
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 20.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb2.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 30.0000, Cat: cats3, W: 1, H: 1}

	bid1_1 := entities.PbsOrtbBid{Bid: &bid1, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_2 := entities.PbsOrtbBid{Bid: &bid2, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 40}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 20.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_3 := entities.PbsOrtbBid{Bid: &bid3, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 30.0000, OriginalBidCur: "USD", TargetBidderCode: ""}

	innerBids := []*entities.PbsOrtbBid{
		&bid1_1,
		&bid1_2,
		&bid1_3,
	}

	seatBid := entities.PbsOrtbSeatBid{Bids: innerBids, Currency: "USD"}
	bidderName1 := openrtb_ext.BidderName("appnexus")

	adapterBids[bidderName1] = &seatBid

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(context.TODO(), *requestExt.Prebid.Targeting, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{}, &SeatNonBidBuilder{})

	assert.Equal(t, nil, err, "Category mapping error should be empty")
	assert.Equal(t, 1, len(rejections), "There should be 1 bid rejection message")
	assert.Equal(t, "bid rejected [bid ID: bid_id3] reason: Category mapping file for primary ad server: 'freewheel', publisher: '' not found", rejections[0], "Rejection message did not match expected")
	assert.Equal(t, "10.00_Electronics_30s", bidCategory["bid_id1"], "Category mapping doesn't match")
	assert.Equal(t, "20.00_Sports_50s", bidCategory["bid_id2"], "Category mapping doesn't match")
	assert.Equal(t, 2, len(adapterBids[bidderName1].Bids), "Bidders number doesn't match")
	assert.Equal(t, 2, len(bidCategory), "Bidders category mapping doesn't match")
}

func newExtRequestTranslateCategories(translateCategories *bool) openrtb_ext.ExtRequest {
	priceGran := openrtb_ext.PriceGranularity{
		Precision: ptrutil.ToPtr(2),
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
		PriceGranularity:     &priceGran,
		IncludeWinners:       ptrutil.ToPtr(true),
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
		priceGranularity: *requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	requestExt.Prebid.Targeting.DurationRangeSec = []int{15, 30, 50}

	adapterBids := make(map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid)

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}
	cats3 := []string{"IAB1-1000"}
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 20.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb2.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 30.0000, Cat: cats3, W: 1, H: 1}

	bid1_1 := entities.PbsOrtbBid{Bid: &bid1, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_2 := entities.PbsOrtbBid{Bid: &bid2, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 40}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 20.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_3 := entities.PbsOrtbBid{Bid: &bid3, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 30.0000, OriginalBidCur: "USD", TargetBidderCode: ""}

	innerBids := []*entities.PbsOrtbBid{
		&bid1_1,
		&bid1_2,
		&bid1_3,
	}

	seatBid := entities.PbsOrtbSeatBid{Bids: innerBids, Currency: "USD"}
	bidderName1 := openrtb_ext.BidderName("appnexus")

	adapterBids[bidderName1] = &seatBid

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(context.TODO(), *requestExt.Prebid.Targeting, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{}, &SeatNonBidBuilder{})

	assert.Equal(t, nil, err, "Category mapping error should be empty")
	assert.Empty(t, rejections, "There should be no bid rejection messages")
	assert.Equal(t, "10.00_IAB1-3_30s", bidCategory["bid_id1"], "Category should not be translated")
	assert.Equal(t, "20.00_IAB1-4_50s", bidCategory["bid_id2"], "Category should not be translated")
	assert.Equal(t, "20.00_IAB1-1000_30s", bidCategory["bid_id3"], "Bid should not be rejected")
	assert.Equal(t, 3, len(adapterBids[bidderName1].Bids), "Bidders number doesn't match")
	assert.Equal(t, 3, len(bidCategory), "Bidders category mapping doesn't match")
}

func TestCategoryDedupe(t *testing.T) {
	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}
	requestExt := newExtRequest()
	targData := &targetData{
		priceGranularity: *requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	// bid3 and bid5 will be same price, category, and duration so one of them should be removed based on the dedupe generator
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: []string{"IAB1-3"}, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 15.0000, Cat: []string{"IAB1-4"}, W: 1, H: 1}
	bid3 := openrtb2.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 20.0000, Cat: []string{"IAB1-3"}, W: 1, H: 1}
	bid4 := openrtb2.Bid{ID: "bid_id4", ImpID: "imp_id4", Price: 20.0000, Cat: []string{"IAB1-INVALID"}, W: 1, H: 1}
	bid5 := openrtb2.Bid{ID: "bid_id5", ImpID: "imp_id5", Price: 20.0000, Cat: []string{"IAB1-3"}, W: 1, H: 1}

	bid1_1 := entities.PbsOrtbBid{Bid: &bid1, BidType: "video", BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, OriginalBidCPM: 10.0000, OriginalBidCur: "USD"}
	bid1_2 := entities.PbsOrtbBid{Bid: &bid2, BidType: "video", BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 50}, OriginalBidCPM: 15.0000, OriginalBidCur: "USD"}
	bid1_3 := entities.PbsOrtbBid{Bid: &bid3, BidType: "video", BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, OriginalBidCPM: 20.0000, OriginalBidCur: "USD"}
	bid1_4 := entities.PbsOrtbBid{Bid: &bid4, BidType: "video", BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, OriginalBidCPM: 20.0000, OriginalBidCur: "USD"}
	bid1_5 := entities.PbsOrtbBid{Bid: &bid5, BidType: "video", BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, OriginalBidCPM: 20.0000, OriginalBidCur: "USD"}

	bidderName1 := openrtb_ext.BidderName("appnexus")

	tests := []struct {
		name                 string
		dedupeGeneratorValue bool
		expectedBids         []*entities.PbsOrtbBid
		expectedCategories   map[string]string
	}{
		{
			name:                 "bid_id5_selected_over_bid_id3",
			dedupeGeneratorValue: true,
			expectedBids:         []*entities.PbsOrtbBid{&bid1_2, &bid1_5},
			expectedCategories: map[string]string{
				"bid_id2": "14.00_Sports_50s",
				"bid_id5": "20.00_Electronics_30s",
			},
		},
		{
			name:                 "bid_id3_selected_over_bid_id5",
			dedupeGeneratorValue: false,
			expectedBids:         []*entities.PbsOrtbBid{&bid1_2, &bid1_3},
			expectedCategories: map[string]string{
				"bid_id2": "14.00_Sports_50s",
				"bid_id3": "20.00_Electronics_30s",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapterBids := map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				bidderName1: {
					Bids: []*entities.PbsOrtbBid{
						&bid1_1,
						&bid1_2,
						&bid1_3,
						&bid1_4,
						&bid1_5,
					},
					Currency: "USD",
				},
			}
			deduplicateGenerator := fakeBooleanGenerator{value: tt.dedupeGeneratorValue}
			bidCategory, adapterBids, rejections, err := applyCategoryMapping(context.TODO(), *requestExt.Prebid.Targeting, adapterBids, categoriesFetcher, targData, &deduplicateGenerator, &SeatNonBidBuilder{})

			assert.Nil(t, err)
			assert.Equal(t, 3, len(rejections))
			assert.Equal(t, adapterBids[bidderName1].Bids, tt.expectedBids)
			assert.Equal(t, bidCategory, tt.expectedCategories)
		})
	}
}

func TestNoCategoryDedupe(t *testing.T) {

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	requestExt := newExtRequestNoBrandCat()

	targData := &targetData{
		priceGranularity: *requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	adapterBids := make(map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid)

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}
	cats4 := []string{"IAB1-2000"}
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 14.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 14.0000, Cat: cats2, W: 1, H: 1}
	bid3 := openrtb2.Bid{ID: "bid_id3", ImpID: "imp_id3", Price: 20.0000, Cat: cats1, W: 1, H: 1}
	bid4 := openrtb2.Bid{ID: "bid_id4", ImpID: "imp_id4", Price: 20.0000, Cat: cats4, W: 1, H: 1}
	bid5 := openrtb2.Bid{ID: "bid_id5", ImpID: "imp_id5", Price: 10.0000, Cat: cats1, W: 1, H: 1}

	bid1_1 := entities.PbsOrtbBid{Bid: &bid1, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 14.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_2 := entities.PbsOrtbBid{Bid: &bid2, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 14.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_3 := entities.PbsOrtbBid{Bid: &bid3, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 20.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_4 := entities.PbsOrtbBid{Bid: &bid4, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 20.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_5 := entities.PbsOrtbBid{Bid: &bid5, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}

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
		innerBids := []*entities.PbsOrtbBid{
			&bid1_1,
			&bid1_2,
			&bid1_3,
			&bid1_4,
			&bid1_5,
		}

		seatBid := entities.PbsOrtbSeatBid{Bids: innerBids, Currency: "USD"}
		bidderName1 := openrtb_ext.BidderName("appnexus")

		adapterBids[bidderName1] = &seatBid

		bidCategory, adapterBids, rejections, err := applyCategoryMapping(context.TODO(), *requestExt.Prebid.Targeting, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{}, &SeatNonBidBuilder{})

		assert.Equal(t, nil, err, "Category mapping error should be empty")
		assert.Equal(t, 2, len(rejections), "There should be 2 bid rejection messages")
		assert.Regexpf(t, regexp.MustCompile(`bid rejected \[bid ID: bid_id(1|2)\] reason: Bid was deduplicated`), rejections[0], "Rejection message did not match expected")
		assert.Regexpf(t, regexp.MustCompile(`bid rejected \[bid ID: bid_id(3|4)\] reason: Bid was deduplicated`), rejections[1], "Rejection message did not match expected")
		assert.Equal(t, 3, len(adapterBids[bidderName1].Bids), "Bidders number doesn't match")
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
		priceGranularity: *requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	requestExt.Prebid.Targeting.DurationRangeSec = []int{15, 30}

	adapterBids := make(map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid)

	cats1 := []string{"IAB1-1"}
	cats2 := []string{"IAB1-2"}
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 10.0000, Cat: cats2, W: 1, H: 1}

	bid1_1 := entities.PbsOrtbBid{Bid: &bid1, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_2 := entities.PbsOrtbBid{Bid: &bid2, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}

	innerBids1 := []*entities.PbsOrtbBid{
		&bid1_1,
	}
	innerBids2 := []*entities.PbsOrtbBid{
		&bid1_2,
	}

	seatBid1 := entities.PbsOrtbSeatBid{Bids: innerBids1, Currency: "USD"}
	bidderName1 := openrtb_ext.BidderName("bidder1")

	seatBid2 := entities.PbsOrtbSeatBid{Bids: innerBids2, Currency: "USD"}
	bidderName2 := openrtb_ext.BidderName("bidder2")

	adapterBids[bidderName1] = &seatBid1
	adapterBids[bidderName2] = &seatBid2

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(context.TODO(), *requestExt.Prebid.Targeting, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{}, &SeatNonBidBuilder{})

	assert.NoError(t, err, "Category mapping error should be empty")
	assert.Empty(t, rejections, "There should be 0 bid rejection messages")
	assert.Equal(t, "10.00_VideoGames_30s_bidder1", bidCategory["bid_id1"], "Category mapping doesn't match")
	assert.Equal(t, "10.00_HomeDecor_30s_bidder2", bidCategory["bid_id2"], "Category mapping doesn't match")
	assert.Len(t, adapterBids[bidderName1].Bids, 1, "Bidders number doesn't match")
	assert.Len(t, adapterBids[bidderName2].Bids, 1, "Bidders number doesn't match")
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
		priceGranularity: *requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	requestExt.Prebid.Targeting.DurationRangeSec = []int{30, 10, 25, 5, 20, 50}

	adapterBids := make(map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid)

	cats1 := []string{"IAB1-1"}
	cats2 := []string{"IAB1-2"}
	bid1 := openrtb2.Bid{ID: "bid_id1", ImpID: "imp_id1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bid2 := openrtb2.Bid{ID: "bid_id2", ImpID: "imp_id2", Price: 12.0000, Cat: cats2, W: 1, H: 1}

	bid1_1 := entities.PbsOrtbBid{Bid: &bid1, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 17}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_2 := entities.PbsOrtbBid{Bid: &bid2, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 8}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 12.0000, OriginalBidCur: "USD", TargetBidderCode: ""}

	innerBids1 := []*entities.PbsOrtbBid{
		&bid1_1,
	}
	innerBids2 := []*entities.PbsOrtbBid{
		&bid1_2,
	}

	seatBid1 := entities.PbsOrtbSeatBid{Bids: innerBids1, Currency: "USD"}
	bidderName1 := openrtb_ext.BidderName("bidder1")

	seatBid2 := entities.PbsOrtbSeatBid{Bids: innerBids2, Currency: "USD"}
	bidderName2 := openrtb_ext.BidderName("bidder2")

	adapterBids[bidderName1] = &seatBid1
	adapterBids[bidderName2] = &seatBid2

	bidCategory, adapterBids, rejections, err := applyCategoryMapping(context.TODO(), *requestExt.Prebid.Targeting, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{}, &SeatNonBidBuilder{})

	assert.NoError(t, err, "Category mapping error should be empty")
	assert.Empty(t, rejections, "There should be 0 bid rejection messages")
	assert.Equal(t, "10.00_20s_bidder1", bidCategory["bid_id1"], "Category mapping doesn't match")
	assert.Equal(t, "12.00_10s_bidder2", bidCategory["bid_id2"], "Category mapping doesn't match")
	assert.Len(t, adapterBids[bidderName1].Bids, 1, "Bidders number doesn't match")
	assert.Len(t, adapterBids[bidderName2].Bids, 1, "Bidders number doesn't match")
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
		priceGranularity: *requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	invalidReqExt := newExtRequest()
	invalidReqExt.Prebid.Targeting.DurationRangeSec = []int{15, 30, 50}
	invalidReqExt.Prebid.Targeting.IncludeBrandCategory.PrimaryAdServer = 2
	invalidReqExt.Prebid.Targeting.IncludeBrandCategory.Publisher = "some_publisher"

	adapterBids := make(map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid)
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
				"bid rejected [bid ID: bid_id1] reason: bid duration exceeds maximum allowed",
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
		innerBids := []*entities.PbsOrtbBid{}
		for _, bid := range test.bids {
			currentBid := entities.PbsOrtbBid{
				Bid: bid, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: test.duration}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
			innerBids = append(innerBids, &currentBid)
		}

		seatBid := entities.PbsOrtbSeatBid{Bids: innerBids, Currency: "USD"}

		adapterBids[bidderName] = &seatBid

		bidCategory, adapterBids, rejections, err := applyCategoryMapping(context.TODO(), *test.reqExt.Prebid.Targeting, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{}, &SeatNonBidBuilder{})

		if len(test.expectedCatDur) > 0 {
			// Bid deduplication case
			assert.Equal(t, 1, len(adapterBids[bidderName].Bids), "Bidders number doesn't match")
			assert.Equal(t, 1, len(bidCategory), "Bidders category mapping doesn't match")
			assert.Equal(t, test.expectedCatDur, bidCategory["bid_id1"], "Bid category did not contain expected hb_pb_cat_dur")
		} else {
			assert.Empty(t, adapterBids[bidderName].Bids, "Bidders number doesn't match")
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
		priceGranularity: *requestExt.Prebid.Targeting.PriceGranularity,
		includeWinners:   true,
	}

	requestExt.Prebid.Targeting.DurationRangeSec = []int{30}
	requestExt.Prebid.Targeting.IncludeBrandCategory.WithCategory = false

	cats1 := []string{"IAB1-3"}
	cats2 := []string{"IAB1-4"}

	bidApn1 := openrtb2.Bid{ID: "bid_idApn1", ImpID: "imp_idApn1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bidApn2 := openrtb2.Bid{ID: "bid_idApn2", ImpID: "imp_idApn2", Price: 10.0000, Cat: cats2, W: 1, H: 1}

	bid1_Apn1 := entities.PbsOrtbBid{Bid: &bidApn1, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_Apn2 := entities.PbsOrtbBid{Bid: &bidApn2, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}

	innerBidsApn1 := []*entities.PbsOrtbBid{
		&bid1_Apn1,
	}

	innerBidsApn2 := []*entities.PbsOrtbBid{
		&bid1_Apn2,
	}

	for i := 1; i < 10; i++ {
		adapterBids := make(map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid)

		seatBidApn1 := entities.PbsOrtbSeatBid{Bids: innerBidsApn1, Currency: "USD"}
		bidderNameApn1 := openrtb_ext.BidderName("appnexus1")

		seatBidApn2 := entities.PbsOrtbSeatBid{Bids: innerBidsApn2, Currency: "USD"}
		bidderNameApn2 := openrtb_ext.BidderName("appnexus2")

		adapterBids[bidderNameApn1] = &seatBidApn1
		adapterBids[bidderNameApn2] = &seatBidApn2

		bidCategory, _, rejections, err := applyCategoryMapping(context.TODO(), *requestExt.Prebid.Targeting, adapterBids, categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{}, &SeatNonBidBuilder{})

		assert.NoError(t, err, "Category mapping error should be empty")
		assert.Len(t, rejections, 1, "There should be 1 bid rejection message")
		assert.Regexpf(t, regexp.MustCompile(`bid rejected \[bid ID: bid_idApn(1|2)\] reason: Bid was deduplicated`), rejections[0], "Rejection message did not match expected")
		assert.Len(t, bidCategory, 1, "Bidders category mapping should have only one element")

		var resultBid string
		for bidId := range bidCategory {
			resultBid = bidId
		}

		if resultBid == "bid_idApn1" {
			assert.Nil(t, seatBidApn2.Bids, "Appnexus_2 seat bid should not have any bids back")
			assert.Len(t, seatBidApn1.Bids, 1, "Appnexus_1 seat bid should have only one back")

		} else {
			assert.Nil(t, seatBidApn1.Bids, "Appnexus_1 seat bid should not have any bids back")
			assert.Len(t, seatBidApn2.Bids, 1, "Appnexus_2 seat bid should have only one back")
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
		priceGranularity: *requestExt.Prebid.Targeting.PriceGranularity,
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

	bid1_Apn1_1 := entities.PbsOrtbBid{Bid: &bidApn1_1, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_Apn1_2 := entities.PbsOrtbBid{Bid: &bidApn1_2, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 20.0000, OriginalBidCur: "USD", TargetBidderCode: ""}

	bid1_Apn2_1 := entities.PbsOrtbBid{Bid: &bidApn2_1, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_Apn2_2 := entities.PbsOrtbBid{Bid: &bidApn2_2, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 20.0000, OriginalBidCur: "USD", TargetBidderCode: ""}

	innerBidsApn1 := []*entities.PbsOrtbBid{
		&bid1_Apn1_1,
		&bid1_Apn1_2,
	}

	innerBidsApn2 := []*entities.PbsOrtbBid{
		&bid1_Apn2_1,
		&bid1_Apn2_2,
	}

	adapterBids := make(map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid)

	seatBidApn1 := entities.PbsOrtbSeatBid{Bids: innerBidsApn1, Currency: "USD"}
	bidderNameApn1 := openrtb_ext.BidderName("appnexus1")

	seatBidApn2 := entities.PbsOrtbSeatBid{Bids: innerBidsApn2, Currency: "USD"}
	bidderNameApn2 := openrtb_ext.BidderName("appnexus2")

	adapterBids[bidderNameApn1] = &seatBidApn1
	adapterBids[bidderNameApn2] = &seatBidApn2

	_, adapterBids, rejections, err := applyCategoryMapping(context.TODO(), *requestExt.Prebid.Targeting, adapterBids, categoriesFetcher, targData, &fakeBooleanGenerator{value: true}, &SeatNonBidBuilder{})

	assert.NoError(t, err, "Category mapping error should be empty")

	//Total number of bids from all bidders in this case should be 2
	bidsFromFirstBidder := adapterBids[bidderNameApn1]
	bidsFromSecondBidder := adapterBids[bidderNameApn2]

	totalNumberOfbids := 0

	//due to random map order we need to identify what bidder was first
	firstBidderIndicator := true

	if bidsFromFirstBidder.Bids != nil {
		totalNumberOfbids += len(bidsFromFirstBidder.Bids)
	}

	if bidsFromSecondBidder.Bids != nil {
		firstBidderIndicator = false
		totalNumberOfbids += len(bidsFromSecondBidder.Bids)
	}

	assert.Equal(t, 2, totalNumberOfbids, "2 bids total should be returned")
	assert.Len(t, rejections, 2, "2 bids should be de-duplicated")

	if firstBidderIndicator {
		assert.Len(t, adapterBids[bidderNameApn1].Bids, 2)
		assert.Len(t, adapterBids[bidderNameApn2].Bids, 0)

		assert.Equal(t, "bid_idApn1_1", adapterBids[bidderNameApn1].Bids[0].Bid.ID, "Incorrect expected bid 1 id")
		assert.Equal(t, "bid_idApn1_2", adapterBids[bidderNameApn1].Bids[1].Bid.ID, "Incorrect expected bid 2 id")

		assert.Equal(t, "bid rejected [bid ID: bid_idApn2_1] reason: Bid was deduplicated", rejections[0], "Incorrect rejected bid 1")
		assert.Equal(t, "bid rejected [bid ID: bid_idApn2_2] reason: Bid was deduplicated", rejections[1], "Incorrect rejected bid 2")

	} else {
		assert.Len(t, adapterBids[bidderNameApn1].Bids, 0)
		assert.Len(t, adapterBids[bidderNameApn2].Bids, 2)

		assert.Equal(t, "bid_idApn2_1", adapterBids[bidderNameApn2].Bids[0].Bid.ID, "Incorrect expected bid 1 id")
		assert.Equal(t, "bid_idApn2_2", adapterBids[bidderNameApn2].Bids[1].Bid.ID, "Incorrect expected bid 2 id")

		assert.Equal(t, "bid rejected [bid ID: bid_idApn1_1] reason: Bid was deduplicated", rejections[0], "Incorrect rejected bid 1")
		assert.Equal(t, "bid rejected [bid ID: bid_idApn1_2] reason: Bid was deduplicated", rejections[1], "Incorrect rejected bid 2")

	}
}

func TestRemoveBidById(t *testing.T) {
	cats1 := []string{"IAB1-3"}

	bidApn1_1 := openrtb2.Bid{ID: "bid_idApn1_1", ImpID: "imp_idApn1_1", Price: 10.0000, Cat: cats1, W: 1, H: 1}
	bidApn1_2 := openrtb2.Bid{ID: "bid_idApn1_2", ImpID: "imp_idApn1_2", Price: 20.0000, Cat: cats1, W: 1, H: 1}
	bidApn1_3 := openrtb2.Bid{ID: "bid_idApn1_3", ImpID: "imp_idApn1_3", Price: 10.0000, Cat: cats1, W: 1, H: 1}

	bid1_Apn1_1 := entities.PbsOrtbBid{Bid: &bidApn1_1, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_Apn1_2 := entities.PbsOrtbBid{Bid: &bidApn1_2, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 20.0000, OriginalBidCur: "USD", TargetBidderCode: ""}
	bid1_Apn1_3 := entities.PbsOrtbBid{Bid: &bidApn1_3, BidMeta: nil, BidType: "video", BidTargets: nil, BidVideo: &openrtb_ext.ExtBidPrebidVideo{Duration: 30}, BidEvents: nil, BidFloors: nil, DealPriority: 0, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 10.0000, OriginalBidCur: "USD", TargetBidderCode: ""}

	type aTest struct {
		desc      string
		inBidName string
		outBids   []*entities.PbsOrtbBid
	}
	testCases := []aTest{
		{
			desc:      "remove element from the middle",
			inBidName: "bid_idApn1_2",
			outBids:   []*entities.PbsOrtbBid{&bid1_Apn1_1, &bid1_Apn1_3},
		},
		{
			desc:      "remove element from the end",
			inBidName: "bid_idApn1_3",
			outBids:   []*entities.PbsOrtbBid{&bid1_Apn1_1, &bid1_Apn1_2},
		},
		{
			desc:      "remove element from the beginning",
			inBidName: "bid_idApn1_1",
			outBids:   []*entities.PbsOrtbBid{&bid1_Apn1_2, &bid1_Apn1_3},
		},
		{
			desc:      "remove element that doesn't exist",
			inBidName: "bid_idApn",
			outBids:   []*entities.PbsOrtbBid{&bid1_Apn1_1, &bid1_Apn1_2, &bid1_Apn1_3},
		},
	}
	for _, test := range testCases {

		innerBidsApn1 := []*entities.PbsOrtbBid{
			&bid1_Apn1_1,
			&bid1_Apn1_2,
			&bid1_Apn1_3,
		}

		seatBidApn1 := &entities.PbsOrtbSeatBid{Bids: innerBidsApn1, Currency: "USD"}

		removeBidById(seatBidApn1, test.inBidName)
		assert.Len(t, seatBidApn1.Bids, len(test.outBids), test.desc)
		assert.ElementsMatch(t, seatBidApn1.Bids, test.outBids, "Incorrect bids in response")
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
	type testInput struct {
		dealPriority int
		impExt       json.RawMessage
		targ         map[string]string
		bidderName   openrtb_ext.BidderName
	}

	type testOutput struct {
		hbPbCatDur        string
		dealErr           string
		dealTierSatisfied bool
	}

	testCases := []struct {
		description string
		in          testInput
		expected    testOutput
	}{
		{
			description: "hb_pb_cat_dur should be modified",
			in: testInput{
				dealPriority: 5,
				impExt:       json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}}}`),
				targ: map[string]string{
					"hb_pb_cat_dur": "12.00_movies_30s",
				},
				bidderName: openrtb_ext.BidderName("appnexus"),
			},
			expected: testOutput{
				hbPbCatDur:        "tier5_movies_30s",
				dealErr:           "",
				dealTierSatisfied: true,
			},
		},
		{
			description: "hb_pb_cat_dur should be modified even with a mixed case bidder in the impExt",
			in: testInput{
				dealPriority: 5,
				impExt:       json.RawMessage(`{"prebid": {"bidder": {"APPnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}}}`),
				targ: map[string]string{
					"hb_pb_cat_dur": "12.00_movies_30s",
				},
				bidderName: openrtb_ext.BidderName("appnexus"),
			},
			expected: testOutput{
				hbPbCatDur:        "tier5_movies_30s",
				dealErr:           "",
				dealTierSatisfied: true,
			},
		},
		{
			description: "hb_pb_cat_dur should be modified even with a mixed case bidder in the winningBidsByBidder map",
			in: testInput{
				dealPriority: 5,
				impExt:       json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}}}`),
				targ: map[string]string{
					"hb_pb_cat_dur": "12.00_movies_30s",
				},
				bidderName: openrtb_ext.BidderName("APPnexus"),
			},
			expected: testOutput{
				hbPbCatDur:        "tier5_movies_30s",
				dealErr:           "",
				dealTierSatisfied: true,
			},
		},
		{
			description: "hb_pb_cat_dur should not be modified due to unknown bidder in the winningBidsByBidder map",
			in: testInput{
				dealPriority: 9,
				impExt:       json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 10, "prefix": "tier"}, "placementId": 10433394}}}}`),
				targ: map[string]string{
					"hb_pb_cat_dur": "12.00_medicine_30s",
				},
				bidderName: openrtb_ext.BidderName("unknown"),
			},
			expected: testOutput{
				hbPbCatDur:        "12.00_medicine_30s",
				dealErr:           "",
				dealTierSatisfied: false,
			},
		},
		{
			description: "hb_pb_cat_dur should not be modified due to priority not exceeding min",
			in: testInput{
				dealPriority: 9,
				impExt:       json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 10, "prefix": "tier"}, "placementId": 10433394}}}}`),
				targ: map[string]string{
					"hb_pb_cat_dur": "12.00_medicine_30s",
				},
				bidderName: openrtb_ext.BidderName("appnexus"),
			},
			expected: testOutput{
				hbPbCatDur:        "12.00_medicine_30s",
				dealErr:           "",
				dealTierSatisfied: false,
			},
		},
		{
			description: "hb_pb_cat_dur should not be modified due to invalid config",
			in: testInput{
				dealPriority: 5,
				impExt:       json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": ""}, "placementId": 10433394}}}}`),
				targ: map[string]string{
					"hb_pb_cat_dur": "12.00_games_30s",
				},
				bidderName: openrtb_ext.BidderName("appnexus"),
			},
			expected: testOutput{
				hbPbCatDur:        "12.00_games_30s",
				dealErr:           "dealTier configuration invalid for bidder 'appnexus', imp ID 'imp_id1'",
				dealTierSatisfied: false,
			},
		},
		{
			description: "hb_pb_cat_dur should not be modified due to deal priority of 0",
			in: testInput{
				dealPriority: 0,
				impExt:       json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}}}`),
				targ: map[string]string{
					"hb_pb_cat_dur": "12.00_auto_30s",
				},
				bidderName: openrtb_ext.BidderName("appnexus"),
			},
			expected: testOutput{
				hbPbCatDur:        "12.00_auto_30s",
				dealErr:           "",
				dealTierSatisfied: false,
			},
		},
	}

	for _, test := range testCases {
		bidRequest := &openrtb2.BidRequest{
			ID: "some-request-id",
			Imp: []openrtb2.Imp{
				{
					ID:  "imp_id1",
					Ext: test.in.impExt,
				},
			},
		}

		bid := entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "123456"}, BidMeta: nil, BidType: "video", BidTargets: map[string]string{}, BidVideo: &openrtb_ext.ExtBidPrebidVideo{}, BidEvents: nil, BidFloors: nil, DealPriority: test.in.dealPriority, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 0, OriginalBidCur: "USD", TargetBidderCode: ""}
		bidCategory := map[string]string{
			bid.Bid.ID: test.in.targ["hb_pb_cat_dur"],
		}

		auc := &auction{
			allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
				"imp_id1": {
					test.in.bidderName: {&bid},
				},
			},
		}

		dealErrs := applyDealSupport(bidRequest, auc, bidCategory, nil)

		assert.Equal(t, test.expected.hbPbCatDur, bidCategory[auc.allBidsByBidder["imp_id1"][test.in.bidderName][0].Bid.ID], test.description)
		assert.Equal(t, test.expected.dealTierSatisfied, auc.allBidsByBidder["imp_id1"][test.in.bidderName][0].DealTierSatisfied, "expected.dealTierSatisfied=%v when %v", test.expected.dealTierSatisfied, test.description)
		if len(test.expected.dealErr) > 0 {
			assert.Containsf(t, dealErrs, errors.New(test.expected.dealErr), "Expected error message not found in deal errors")
		}
	}
}

func TestApplyDealSupportMultiBid(t *testing.T) {
	type args struct {
		bidRequest  *openrtb2.BidRequest
		auc         *auction
		bidCategory map[string]string
		multiBid    map[string]openrtb_ext.ExtMultiBid
	}
	type want struct {
		errs                      []error
		expectedHbPbCatDur        map[string]map[string][]string
		expectedDealTierSatisfied map[string]map[string][]bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "multibid disabled, hb_pb_cat_dur should be modified only for first bid",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					ID: "some-request-id",
					Imp: []openrtb2.Imp{
						{
							ID:  "imp_id1",
							Ext: json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}}}`),
						},
						{
							ID:  "imp_id1",
							Ext: json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}}}`),
						},
					},
				},
				auc: &auction{
					allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
						"imp_id1": {
							openrtb_ext.BidderName("appnexus"): {
								&entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "123456"}, BidMeta: nil, BidType: "video", BidTargets: map[string]string{}, BidVideo: &openrtb_ext.ExtBidPrebidVideo{}, BidEvents: nil, BidFloors: nil, DealPriority: 5, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 0, OriginalBidCur: "USD", TargetBidderCode: ""},
								&entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "789101"}, BidMeta: nil, BidType: "video", BidTargets: map[string]string{}, BidVideo: &openrtb_ext.ExtBidPrebidVideo{}, BidEvents: nil, BidFloors: nil, DealPriority: 5, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 0, OriginalBidCur: "USD", TargetBidderCode: ""},
							},
						},
					},
				},
				bidCategory: map[string]string{
					"123456": "12.00_movies_30s",
					"789101": "12.00_movies_30s",
				},
				multiBid: nil,
			},
			want: want{
				errs: []error{},
				expectedHbPbCatDur: map[string]map[string][]string{
					"imp_id1": {
						"appnexus": []string{"tier5_movies_30s", "12.00_movies_30s"},
					},
				},
				expectedDealTierSatisfied: map[string]map[string][]bool{
					"imp_id1": {
						"appnexus": []bool{true, false},
					},
				},
			},
		},
		{
			name: "multibid enabled, hb_pb_cat_dur should be modified for all winning bids",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					ID: "some-request-id",
					Imp: []openrtb2.Imp{
						{
							ID:  "imp_id1",
							Ext: json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}}}`),
						},
						{
							ID:  "imp_id1",
							Ext: json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}}}`),
						},
					},
				},
				auc: &auction{
					allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
						"imp_id1": {
							openrtb_ext.BidderName("appnexus"): {
								&entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "123456"}, BidMeta: nil, BidType: "video", BidTargets: map[string]string{}, BidVideo: &openrtb_ext.ExtBidPrebidVideo{}, BidEvents: nil, BidFloors: nil, DealPriority: 5, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 0, OriginalBidCur: "USD", TargetBidderCode: ""},
								&entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "789101"}, BidMeta: nil, BidType: "video", BidTargets: map[string]string{}, BidVideo: &openrtb_ext.ExtBidPrebidVideo{}, BidEvents: nil, BidFloors: nil, DealPriority: 5, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 0, OriginalBidCur: "USD", TargetBidderCode: ""},
							},
						},
					},
				},
				bidCategory: map[string]string{
					"123456": "12.00_movies_30s",
					"789101": "12.00_movies_30s",
				},
				multiBid: map[string]openrtb_ext.ExtMultiBid{
					"appnexus": {
						TargetBidderCodePrefix: "appN",
						MaxBids:                ptrutil.ToPtr(2),
					},
				},
			},
			want: want{
				errs: []error{},
				expectedHbPbCatDur: map[string]map[string][]string{
					"imp_id1": {
						"appnexus": []string{"tier5_movies_30s", "tier5_movies_30s"},
					},
				},
				expectedDealTierSatisfied: map[string]map[string][]bool{
					"imp_id1": {
						"appnexus": []bool{true, true},
					},
				},
			},
		},
		{
			name: "multibid enabled but TargetBidderCodePrefix not defined, hb_pb_cat_dur should be modified only for first bid",
			args: args{
				bidRequest: &openrtb2.BidRequest{
					ID: "some-request-id",
					Imp: []openrtb2.Imp{
						{
							ID:  "imp_id1",
							Ext: json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}}}`),
						},
						{
							ID:  "imp_id1",
							Ext: json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}, "placementId": 10433394}}}}`),
						},
					},
				},
				auc: &auction{
					allBidsByBidder: map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
						"imp_id1": {
							openrtb_ext.BidderName("appnexus"): {
								&entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "123456"}, BidMeta: nil, BidType: "video", BidTargets: map[string]string{}, BidVideo: &openrtb_ext.ExtBidPrebidVideo{}, BidEvents: nil, BidFloors: nil, DealPriority: 5, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 0, OriginalBidCur: "USD", TargetBidderCode: ""},
								&entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "789101"}, BidMeta: nil, BidType: "video", BidTargets: map[string]string{}, BidVideo: &openrtb_ext.ExtBidPrebidVideo{}, BidEvents: nil, BidFloors: nil, DealPriority: 5, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 0, OriginalBidCur: "USD", TargetBidderCode: ""},
							},
						},
					},
				},
				bidCategory: map[string]string{
					"123456": "12.00_movies_30s",
					"789101": "12.00_movies_30s",
				},
				multiBid: map[string]openrtb_ext.ExtMultiBid{
					"appnexus": {
						MaxBids: ptrutil.ToPtr(2),
					},
				},
			},
			want: want{
				errs: []error{},
				expectedHbPbCatDur: map[string]map[string][]string{
					"imp_id1": {
						"appnexus": []string{"tier5_movies_30s", "12.00_movies_30s"},
					},
				},
				expectedDealTierSatisfied: map[string]map[string][]bool{
					"imp_id1": {
						"appnexus": []bool{true, false},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := applyDealSupport(tt.args.bidRequest, tt.args.auc, tt.args.bidCategory, tt.args.multiBid)
			assert.Equal(t, tt.want.errs, errs)

			for impID, topBidsPerImp := range tt.args.auc.allBidsByBidder {
				for bidder, topBidsPerBidder := range topBidsPerImp {
					for i, topBid := range topBidsPerBidder {
						assert.Equal(t, tt.want.expectedHbPbCatDur[impID][bidder.String()][i], tt.args.bidCategory[topBid.Bid.ID], tt.name)
						assert.Equal(t, tt.want.expectedDealTierSatisfied[impID][bidder.String()][i], topBid.DealTierSatisfied, tt.name)
					}
				}
			}
		})
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
					{ID: "imp1", Ext: json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier"}}}}}`)},
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
					{ID: "imp1", Ext: json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier1"}}}}}`)},
					{ID: "imp2", Ext: json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 8, "prefix": "tier2"}}}}}`)},
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
					{ID: "imp1", Ext: json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": {"minDealTier": 5, "prefix": "tier1"}}}}}`)},
					{ID: "imp2", Ext: json.RawMessage(`{"prebid": {"bidder": {"appnexus": {"dealTier": "wrong type"}}}}`)},
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
		bid := entities.PbsOrtbBid{Bid: &openrtb2.Bid{ID: "123456"}, BidMeta: nil, BidType: "video", BidTargets: map[string]string{}, BidVideo: &openrtb_ext.ExtBidPrebidVideo{}, BidEvents: nil, BidFloors: nil, DealPriority: test.dealPriority, DealTierSatisfied: false, GeneratedBidID: "", OriginalBidCPM: 0, OriginalBidCur: "USD", TargetBidderCode: ""}
		bidCategory := map[string]string{
			bid.Bid.ID: test.targ["hb_pb_cat_dur"],
		}

		updateHbPbCatDur(&bid, test.dealTier, bidCategory)

		assert.Equal(t, test.expectedHbPbCatDur, bidCategory[bid.Bid.ID], test.description)
		assert.Equal(t, test.expectedDealTierSatisfied, bid.DealTierSatisfied, test.description)
	}
}

func TestMakeBidExtJSON(t *testing.T) {

	type aTest struct {
		description        string
		ext                json.RawMessage
		extBidPrebid       openrtb_ext.ExtBidPrebid
		impExtInfo         map[string]ImpExtInfo
		origbidcpm         float64
		origbidcur         string
		expectedBidExt     string
		expectedErrMessage string
	}

	testCases := []aTest{
		{
			description:        "Valid extension, non empty extBidPrebid, valid imp ext info, meta from adapter",
			ext:                json.RawMessage(`{"video":{"h":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video"), Meta: &openrtb_ext.ExtBidPrebidMeta{BrandName: "foo"}, Passthrough: nil},
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"video":{"h":480,"mimes":["video/mp4"]}}`), json.RawMessage(`{"imp_passthrough_val": 1}`)}},
			origbidcpm:         10.0000,
			origbidcur:         "USD",
			expectedBidExt:     `{"prebid":{"meta": {"brandName": "foo","adaptercode": "adapter"}, "passthrough":{"imp_passthrough_val":1}, "type":"video"}, "storedrequestattributes":{"h":480,"mimes":["video/mp4"]},"video":{"h":100}, "origbidcpm": 10, "origbidcur": "USD"}`,
			expectedErrMessage: "",
		},
		{
			description:        "Valid extension, non empty extBidPrebid, valid imp ext info, meta from response, imp passthrough is nil",
			ext:                json.RawMessage(`{"video":{"h":100},"prebid":{"meta": {"brandName": "foo"}}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"video":{"h":480,"mimes":["video/mp4"]}}`), nil}},
			origbidcpm:         10.0000,
			origbidcur:         "USD",
			expectedBidExt:     `{"prebid":{"meta": {"brandName": "foo","adaptercode": "adapter"}, "type":"video"},"storedrequestattributes":{"h":480,"mimes":["video/mp4"]},"video":{"h":100}, "origbidcpm": 10, "origbidcur": "USD"}`,
			expectedErrMessage: "",
		},
		{
			description:        "Empty extension, non empty extBidPrebid and valid imp ext info",
			ext:                nil,
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"video":{"h":480,"mimes":["video/mp4"]}}`), json.RawMessage(`{"imp_passthrough_val": 1}`)}},
			origbidcpm:         0,
			expectedBidExt:     `{"origbidcpm": 0,"prebid":{"meta":{"adaptercode": "adapter"},"passthrough":{"imp_passthrough_val":1}, "type":"video"},"storedrequestattributes":{"h":480,"mimes":["video/mp4"]}}`,
			expectedErrMessage: "",
		},
		{
			description:        "Valid extension, non empty extBidPrebid and imp ext info not found",
			ext:                json.RawMessage(`{"video":{"h":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			impExtInfo:         map[string]ImpExtInfo{"another_imp_id": {true, []byte(`{"video":{"h":480,"mimes":["video/mp4"]}}`), json.RawMessage(`{"imp_passthrough_val": 1}`)}},
			origbidcpm:         10.0000,
			origbidcur:         "USD",
			expectedBidExt:     `{"prebid":{"meta":{"adaptercode": "adapter"},"type":"video"},"video":{"h":100}, "origbidcpm": 10, "origbidcur": "USD"}`,
			expectedErrMessage: "",
		},
		{
			description:        "Valid extension, empty extBidPrebid and valid imp ext info",
			ext:                json.RawMessage(`{"video":{"h":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{},
			origbidcpm:         10.0000,
			origbidcur:         "USD",
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"video":{"h":480,"mimes":["video/mp4"]}}`), json.RawMessage(`{"imp_passthrough_val": 1}`)}},
			expectedBidExt:     `{"prebid":{"meta":{"adaptercode": "adapter"},"passthrough":{"imp_passthrough_val":1}},"storedrequestattributes":{"h":480,"mimes":["video/mp4"]},"video":{"h":100}, "origbidcpm": 10, "origbidcur": "USD"}`,
			expectedErrMessage: "",
		},
		{
			description:        "Valid extension, non empty extBidPrebid and empty imp ext info",
			ext:                json.RawMessage(`{"video":{"h":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			origbidcpm:         10.0000,
			origbidcur:         "USD",
			impExtInfo:         nil,
			expectedBidExt:     `{"prebid":{"meta":{"adaptercode": "adapter"},"type":"video"},"video":{"h":100}, "origbidcpm": 10, "origbidcur": "USD"}`,
			expectedErrMessage: "",
		},
		{
			description:        "Valid extension, non empty extBidPrebid and valid imp ext info without video attr",
			ext:                json.RawMessage(`{"video":{"h":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			origbidcpm:         10.0000,
			origbidcur:         "USD",
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"banner":{"h":480}}`), json.RawMessage(`{"imp_passthrough_val": 1}`)}},
			expectedBidExt:     `{"prebid":{"meta":{"adaptercode": "adapter"},"passthrough":{"imp_passthrough_val":1}, "type":"video"},"video":{"h":100}, "origbidcpm": 10, "origbidcur": "USD"}`,
			expectedErrMessage: "",
		},
		{
			description:        "Valid extension with prebid, non empty extBidPrebid and valid imp ext info without video attr",
			ext:                json.RawMessage(`{"prebid":{"targeting":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			origbidcpm:         10.0000,
			origbidcur:         "USD",
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"banner":{"h":480}}`), json.RawMessage(`{"imp_passthrough_val": 1}`)}},
			expectedBidExt:     `{"prebid":{"meta":{"adaptercode": "adapter"},"passthrough":{"imp_passthrough_val":1}, "type":"video"}, "origbidcpm": 10, "origbidcur": "USD"}`,
			expectedErrMessage: "",
		},
		{
			description:        "Valid extension with prebid, non empty extBidPrebid and valid imp ext info with video attr",
			ext:                json.RawMessage(`{"prebid":{"targeting":100}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			origbidcpm:         10.0000,
			origbidcur:         "USD",
			impExtInfo:         map[string]ImpExtInfo{"test_imp_id": {true, []byte(`{"video":{"h":480,"mimes":["video/mp4"]}}`), json.RawMessage(`{"imp_passthrough_val": 1}`)}},
			expectedBidExt:     `{"prebid":{"meta":{"adaptercode": "adapter"},"passthrough":{"imp_passthrough_val":1}, "type":"video"}, "storedrequestattributes":{"h":480,"mimes":["video/mp4"]}, "origbidcpm": 10, "origbidcur": "USD"}`,
			expectedErrMessage: "",
		},
		{
			description:        "Meta - Defined By Bid - Nil Extension",
			ext:                nil,
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("banner"), Meta: &openrtb_ext.ExtBidPrebidMeta{BrandName: "foo"}},
			impExtInfo:         map[string]ImpExtInfo{},
			origbidcpm:         0,
			origbidcur:         "USD",
			expectedBidExt:     `{"origbidcpm": 0,"prebid":{"meta":{"brandName":"foo","adaptercode": "adapter"},"type":"banner"}, "origbidcur": "USD"}`,
			expectedErrMessage: "",
		},
		{
			description:        "Meta - Defined By Bid - Empty Extension",
			ext:                json.RawMessage(`{}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("banner"), Meta: &openrtb_ext.ExtBidPrebidMeta{BrandName: "foo"}},
			impExtInfo:         nil,
			origbidcpm:         0,
			origbidcur:         "USD",
			expectedBidExt:     `{"origbidcpm": 0,"prebid":{"meta":{"brandName":"foo","adaptercode": "adapter"},"type":"banner"}, "origbidcur": "USD"}`,
			expectedErrMessage: "",
		},
		{
			description:        "Meta - Defined By Bid - Existing Extension Overwritten",
			ext:                json.RawMessage(`{"prebid":{"meta":{"brandName":"notfoo", "brandId": 42}}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("banner"), Meta: &openrtb_ext.ExtBidPrebidMeta{BrandName: "foo"}},
			impExtInfo:         nil,
			origbidcpm:         10.0000,
			origbidcur:         "USD",
			expectedBidExt:     `{"prebid":{"meta":{"brandName":"foo","adaptercode": "adapter"},"type":"banner"}, "origbidcpm": 10, "origbidcur": "USD"}`,
			expectedErrMessage: "",
		},
		{
			description:        "Meta - Not Defined By Bid - Persists From Bid Ext",
			ext:                json.RawMessage(`{"prebid":{"meta":{"brandName":"foo"}}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("banner")},
			impExtInfo:         nil,
			origbidcpm:         10.0000,
			origbidcur:         "USD",
			expectedBidExt:     `{"prebid":{"meta":{"brandName":"foo","adaptercode": "adapter"},"type":"banner"}, "origbidcpm": 10, "origbidcur": "USD"}`,
			expectedErrMessage: "",
		},
		{
			description:        "Meta - Not Defined By Bid - Persists From Bid Ext - Invalid Fields Ignored",
			ext:                json.RawMessage(`{"prebid":{"meta":{"brandName":"foo","unknown":"value"}}}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("banner")},
			impExtInfo:         nil,
			origbidcpm:         -1,
			origbidcur:         "USD",
			expectedBidExt:     `{"prebid":{"meta":{"brandName":"foo","adaptercode":"adapter"},"type":"banner"}, "origbidcur": "USD"}`,
			expectedErrMessage: "",
		},
		{
			description:        "Meta - Not Defined",
			ext:                nil,
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("banner")},
			impExtInfo:         nil,
			origbidcpm:         0,
			origbidcur:         "USD",
			expectedBidExt:     `{"origbidcpm": 0,"prebid":{"type":"banner","meta":{"adaptercode":"adapter"}}, "origbidcur": "USD"}`,
			expectedErrMessage: "",
		},
		//Error cases
		{
			description:        "Invalid extension, valid extBidPrebid and valid imp ext info",
			ext:                json.RawMessage(`{invalid json}`),
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("video")},
			expectedBidExt:     ``,
			expectedErrMessage: "expects \" or n, but found i",
		},
		{
			description:        "Meta - Invalid",
			ext:                json.RawMessage(`{"prebid":{"meta":{"brandId":"foo"}}}`), // brandId should be an int, but is a string in this test case
			extBidPrebid:       openrtb_ext.ExtBidPrebid{Type: openrtb_ext.BidType("banner")},
			impExtInfo:         nil,
			expectedErrMessage: "error validating response from server, cannot unmarshal openrtb_ext.ExtBidPrebidMeta.BrandID: unexpected character: \xff",
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			var adapter openrtb_ext.BidderName = "adapter"
			result, err := makeBidExtJSON(test.ext, &test.extBidPrebid, test.impExtInfo, "test_imp_id", test.origbidcpm, test.origbidcur, adapter)

			if test.expectedErrMessage == "" {
				assert.JSONEq(t, test.expectedBidExt, string(result), "Incorrect result")
				assert.NoError(t, err, "Error should not be returned")
			} else {
				assert.Contains(t, err.Error(), test.expectedErrMessage, "incorrect error message")
			}
		})
	}
}

func TestStoredAuctionResponses(t *testing.T) {
	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	e := new(exchange)
	e.cache = &wellBehavedCache{}
	e.me = &metricsConf.NilMetricsEngine{}
	e.categoriesFetcher = categoriesFetcher
	e.bidIDGenerator = &fakeBidIDGenerator{GenerateBidID: false, ReturnError: false}
	e.currencyConverter = currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	e.gdprPermsBuilder = fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder

	// Define mock incoming bid requeset
	mockBidRequest := &openrtb2.BidRequest{
		ID: "request-id",
		Imp: []openrtb2.Imp{{
			ID:    "impression-id",
			Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](400), H: ptrutil.ToPtr[int64](300)},
		}},
	}

	expectedBidResponse := &openrtb2.BidResponse{
		ID: "request-id",
		SeatBid: []openrtb2.SeatBid{
			{
				Bid: []openrtb2.Bid{
					{ID: "bid_id", ImpID: "impression-id", Ext: json.RawMessage(`{"origbidcpm":0,"prebid":{"meta":{},"type":"video"}}`)},
				},
				Seat: "appnexus",
			},
		},
	}

	testCases := []struct {
		desc              string
		storedAuctionResp map[string]json.RawMessage
		errorExpected     bool
	}{
		{
			desc: "Single imp with valid stored response",
			storedAuctionResp: map[string]json.RawMessage{
				"impression-id": json.RawMessage(`[{"bid": [{"id": "bid_id", "ext": {"prebid": {"type": "video"}}}],"seat": "appnexus"}]`),
			},
			errorExpected: false,
		},
		{
			desc: "Single imp with invalid stored response",
			storedAuctionResp: map[string]json.RawMessage{
				"impression-id": json.RawMessage(`[}]`),
			},
			errorExpected: true,
		},
	}

	for _, test := range testCases {

		auctionRequest := &AuctionRequest{
			BidRequestWrapper:      &openrtb_ext.RequestWrapper{BidRequest: mockBidRequest},
			Account:                config.Account{},
			UserSyncs:              &emptyUsersync{},
			StoredAuctionResponses: test.storedAuctionResp,
			HookExecutor:           &hookexecution.EmptyHookExecutor{},
			TCF2Config:             gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}
		// Run test
		outBidResponse, err := e.HoldAuction(context.Background(), auctionRequest, &DebugLog{})
		if test.errorExpected {
			assert.Error(t, err, "Error should be returned")
		} else {
			assert.NoErrorf(t, err, "%s. HoldAuction error: %v \n", test.desc, err)
			outBidResponse.Ext = nil
			assert.Equal(t, expectedBidResponse, outBidResponse.BidResponse, "Incorrect stored auction response")
		}

	}
}

func TestBuildStoredAuctionResponses(t *testing.T) {

	type testIn struct {
		StoredAuctionResponses map[string]json.RawMessage
	}
	type testResults struct {
		adapterBids  map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		fledge       *openrtb_ext.Fledge
		liveAdapters []openrtb_ext.BidderName
	}

	testCases := []struct {
		desc         string
		in           testIn
		expected     testResults
		errorMessage string
	}{
		{
			desc: "Single imp with single stored response bid",
			in: testIn{
				StoredAuctionResponses: map[string]json.RawMessage{
					"impression-id": json.RawMessage(`[{"bid": [{"id": "bid_id", "ext": {"prebid": {"type": "native"}}}],"seat": "appnexus"}]`),
				},
			},
			expected: testResults{
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					openrtb_ext.BidderName("appnexus"): {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid:     &openrtb2.Bid{ID: "bid_id", ImpID: "impression-id", Ext: []byte(`{"prebid": {"type": "native"}}`)},
								BidType: openrtb_ext.BidTypeNative,
							},
						},
					},
				},
				liveAdapters: []openrtb_ext.BidderName{openrtb_ext.BidderName("appnexus")},
			},
		},
		{
			desc: "Single imp with single stored response bid with incorrect bid type",
			in: testIn{
				StoredAuctionResponses: map[string]json.RawMessage{
					"impression-id": json.RawMessage(`[{"bid": [{"id": "bid_id", "ext": {"prebid": {"type": "incorrect"}}}],"seat": "appnexus"}]`),
				},
			},
			expected: testResults{
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					openrtb_ext.BidderName("appnexus"): {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid:     &openrtb2.Bid{ID: "bid_id", ImpID: "impression-id", Ext: []byte(`{"prebid": {"type": "native"}}`)},
								BidType: openrtb_ext.BidTypeNative,
							},
						},
					},
				},
				liveAdapters: []openrtb_ext.BidderName{openrtb_ext.BidderName("appnexus")},
			},
			errorMessage: "Failed to parse bid mediatype for impression \"impression-id\", invalid BidType: incorrect",
		},
		{
			desc: "Single imp with multiple bids in stored response one bidder",
			in: testIn{
				StoredAuctionResponses: map[string]json.RawMessage{
					"impression-id": json.RawMessage(`[{"bid": [{"id": "bid_id1", "ext": {"prebid": {"type": "native"}}}, {"id": "bid_id2", "ext": {"prebid": {"type": "video"}}}],"seat": "appnexus"}]`),
				},
			},
			expected: testResults{
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					openrtb_ext.BidderName("appnexus"): {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "bid_id1", ImpID: "impression-id", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
							{Bid: &openrtb2.Bid{ID: "bid_id2", ImpID: "impression-id", Ext: []byte(`{"prebid": {"type": "video"}}`)}, BidType: openrtb_ext.BidTypeVideo},
						},
					},
				},
				liveAdapters: []openrtb_ext.BidderName{openrtb_ext.BidderName("appnexus")},
			},
		},
		{
			desc: "Single imp with multiple bids in stored response two bidders",
			in: testIn{
				StoredAuctionResponses: map[string]json.RawMessage{
					"impression-id": json.RawMessage(`[{"bid": [{"id": "apn_id1", "ext": {"prebid": {"type": "native"}}}, {"id": "apn_id2", "ext": {"prebid": {"type": "native"}}}],"seat": "appnexus"}, {"bid": [{"id": "rubicon_id1", "ext": {"prebid": {"type": "banner"}}}, {"id": "rubicon_id2", "ext": {"prebid": {"type": "banner"}}}],"seat": "rubicon"}]`),
				},
			},
			expected: testResults{
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					openrtb_ext.BidderName("appnexus"): {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "apn_id1", ImpID: "impression-id", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
							{Bid: &openrtb2.Bid{ID: "apn_id2", ImpID: "impression-id", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
						},
					},
					openrtb_ext.BidderName("rubicon"): {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "rubicon_id1", ImpID: "impression-id", Ext: []byte(`{"prebid": {"type": "banner"}}`)}, BidType: openrtb_ext.BidTypeBanner},
							{Bid: &openrtb2.Bid{ID: "rubicon_id2", ImpID: "impression-id", Ext: []byte(`{"prebid": {"type": "banner"}}`)}, BidType: openrtb_ext.BidTypeBanner},
						},
					},
				},
				liveAdapters: []openrtb_ext.BidderName{openrtb_ext.BidderName("appnexus"), openrtb_ext.BidderName("rubicon")},
			},
		},
		{
			desc: "Two imps with two bids in stored response two bidders, different bids number",
			in: testIn{
				StoredAuctionResponses: map[string]json.RawMessage{
					"impression-id1": json.RawMessage(`[{"bid": [{"id": "apn_id1", "ext": {"prebid": {"type": "native"}}}, {"id": "apn_id2", "ext": {"prebid": {"type": "native"}}}],"seat": "appnexus"}]`),
					"impression-id2": json.RawMessage(`[{"bid": [{"id": "apn_id1", "ext": {"prebid": {"type": "native"}}}, {"id": "apn_id2", "ext": {"prebid": {"type": "native"}}}],"seat": "appnexus"}, {"bid": [{"id": "rubicon_id1", "ext": {"prebid": {"type": "native"}}}, {"id": "rubicon_id2", "ext": {"prebid": {"type": "native"}}}],"seat": "rubicon"}]`),
				},
			},
			expected: testResults{
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					openrtb_ext.BidderName("appnexus"): {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "apn_id1", ImpID: "impression-id1", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
							{Bid: &openrtb2.Bid{ID: "apn_id2", ImpID: "impression-id1", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
							{Bid: &openrtb2.Bid{ID: "apn_id1", ImpID: "impression-id2", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
							{Bid: &openrtb2.Bid{ID: "apn_id2", ImpID: "impression-id2", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
						},
					},
					openrtb_ext.BidderName("rubicon"): {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "rubicon_id1", ImpID: "impression-id2", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
							{Bid: &openrtb2.Bid{ID: "rubicon_id2", ImpID: "impression-id2", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
						},
					},
				},
				liveAdapters: []openrtb_ext.BidderName{openrtb_ext.BidderName("appnexus"), openrtb_ext.BidderName("rubicon")},
			},
		},
		{
			desc: "Two imps with two bids in stored response two bidders",
			in: testIn{
				StoredAuctionResponses: map[string]json.RawMessage{
					"impression-id1": json.RawMessage(`[{"bid": [{"id": "apn_id1", "ext": {"prebid": {"type": "native"}}}, {"id": "apn_id2", "ext": {"prebid": {"type": "native"}}}],"seat": "appnexus"}, {"bid": [{"id": "rubicon_id1", "ext": {"prebid": {"type": "native"}}}, {"id": "rubicon_id2", "ext": {"prebid": {"type": "native"}}}],"seat": "rubicon"}]`),
					"impression-id2": json.RawMessage(`[{"bid": [{"id": "apn_id1", "ext": {"prebid": {"type": "native"}}}, {"id": "apn_id2", "ext": {"prebid": {"type": "native"}}}],"seat": "appnexus"}, {"bid": [{"id": "rubicon_id1", "ext": {"prebid": {"type": "native"}}}, {"id": "rubicon_id2", "ext": {"prebid": {"type": "native"}}}],"seat": "rubicon"}]`),
				},
			},
			expected: testResults{
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					openrtb_ext.BidderName("appnexus"): {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "apn_id1", ImpID: "impression-id1", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
							{Bid: &openrtb2.Bid{ID: "apn_id2", ImpID: "impression-id1", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
							{Bid: &openrtb2.Bid{ID: "apn_id1", ImpID: "impression-id2", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
							{Bid: &openrtb2.Bid{ID: "apn_id2", ImpID: "impression-id2", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
						},
					},
					openrtb_ext.BidderName("rubicon"): {
						Bids: []*entities.PbsOrtbBid{
							{Bid: &openrtb2.Bid{ID: "rubicon_id1", ImpID: "impression-id1", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
							{Bid: &openrtb2.Bid{ID: "rubicon_id2", ImpID: "impression-id1", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
							{Bid: &openrtb2.Bid{ID: "rubicon_id1", ImpID: "impression-id2", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
							{Bid: &openrtb2.Bid{ID: "rubicon_id2", ImpID: "impression-id2", Ext: []byte(`{"prebid": {"type": "native"}}`)}, BidType: openrtb_ext.BidTypeNative},
						},
					},
				},
				liveAdapters: []openrtb_ext.BidderName{openrtb_ext.BidderName("appnexus"), openrtb_ext.BidderName("rubicon")},
			},
		},
		{
			desc: "Fledge in stored response bid",
			in: testIn{
				StoredAuctionResponses: map[string]json.RawMessage{
					"impression-id": json.RawMessage(`[{"bid": [],"seat": "openx", "ext": {"prebid": {"fledge": {"auctionconfigs": [{"impid": "1", "bidder": "openx", "adapter": "openx", "config": [1,2,3]}]}}}}]`),
				},
			},
			expected: testResults{
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					openrtb_ext.BidderName("openx"): {
						Bids: []*entities.PbsOrtbBid{},
					},
				},
				liveAdapters: []openrtb_ext.BidderName{openrtb_ext.BidderName("openx")},
				fledge: &openrtb_ext.Fledge{
					AuctionConfigs: []*openrtb_ext.FledgeAuctionConfig{
						{
							ImpId:   "impression-id",
							Bidder:  "openx",
							Adapter: "openx",
							Config:  json.RawMessage("[1,2,3]"),
						},
					},
				},
			},
		},
		{
			desc: "Single imp with single stored response bid with bid.mtype",
			in: testIn{
				StoredAuctionResponses: map[string]json.RawMessage{
					"impression-id": json.RawMessage(`[{"bid": [{"id": "bid_id", "mtype": 2, "ext": {"prebid": {"type": "native"}}}],"seat": "appnexus"}]`),
				},
			},
			expected: testResults{
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					openrtb_ext.BidderName("appnexus"): {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid:     &openrtb2.Bid{ID: "bid_id", ImpID: "impression-id", MType: 2, Ext: []byte(`{"prebid": {"type": "native"}}`)},
								BidType: openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
				liveAdapters: []openrtb_ext.BidderName{openrtb_ext.BidderName("appnexus")},
			},
		},
		{
			desc: "Multiple imps with multiple stored response bid with bid.mtype and different types",
			in: testIn{
				StoredAuctionResponses: map[string]json.RawMessage{
					"impression-id1": json.RawMessage(`[{"bid": [{"id": "bid_id", "mtype": 1, "ext": {"prebid": {"type": "native"}}}],"seat": "appnexus"}]`),
					"impression-id2": json.RawMessage(`[{"bid": [{"id": "bid_id", "mtype": 2, "ext": {"prebid": {"type": "native"}}}],"seat": "appnexus"}]`),
					"impression-id3": json.RawMessage(`[{"bid": [{"id": "bid_id", "mtype": 3, "ext": {"prebid": {"type": "native"}}}],"seat": "appnexus"}]`),
					"impression-id4": json.RawMessage(`[{"bid": [{"id": "bid_id", "mtype": 4, "ext": {"prebid": {"type": "native"}}}],"seat": "appnexus"}]`),
				},
			},
			expected: testResults{
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					openrtb_ext.BidderName("appnexus"): {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid:     &openrtb2.Bid{ID: "bid_id", ImpID: "impression-id1", MType: 1, Ext: []byte(`{"prebid": {"type": "native"}}`)},
								BidType: openrtb_ext.BidTypeBanner,
							},
							{
								Bid:     &openrtb2.Bid{ID: "bid_id", ImpID: "impression-id2", MType: 2, Ext: []byte(`{"prebid": {"type": "native"}}`)},
								BidType: openrtb_ext.BidTypeVideo,
							},
							{
								Bid:     &openrtb2.Bid{ID: "bid_id", ImpID: "impression-id3", MType: 3, Ext: []byte(`{"prebid": {"type": "native"}}`)},
								BidType: openrtb_ext.BidTypeAudio,
							},
							{
								Bid:     &openrtb2.Bid{ID: "bid_id", ImpID: "impression-id4", MType: 4, Ext: []byte(`{"prebid": {"type": "native"}}`)},
								BidType: openrtb_ext.BidTypeNative,
							},
						},
					},
				},
				liveAdapters: []openrtb_ext.BidderName{openrtb_ext.BidderName("appnexus")},
			},
		},
		{
			desc: "Single imp with single stored response bid with incorrect bid.mtype",
			in: testIn{
				StoredAuctionResponses: map[string]json.RawMessage{
					"impression-id": json.RawMessage(`[{"bid": [{"id": "bid_id", "mtype": 10, "ext": {"prebid": {"type": "native"}}}],"seat": "appnexus"}]`),
				},
			},
			expected: testResults{
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					openrtb_ext.BidderName("appnexus"): {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid:     &openrtb2.Bid{ID: "bid_id", ImpID: "impression-id", MType: 2, Ext: []byte(`{"prebid": {"type": "native"}}`)},
								BidType: openrtb_ext.BidTypeVideo,
							},
						},
					},
				},
				liveAdapters: []openrtb_ext.BidderName{openrtb_ext.BidderName("appnexus")},
			},
			errorMessage: "Failed to parse bid mType for impression \"impression-id\"",
		},
	}
	for _, test := range testCases {

		bids, fledge, adapters, err := buildStoredAuctionResponse(test.in.StoredAuctionResponses)
		if len(test.errorMessage) > 0 {
			assert.Equal(t, test.errorMessage, err.Error(), " incorrect expected error")
		} else {
			assert.NoErrorf(t, err, "%s. HoldAuction error: %v \n", test.desc, err)

			assert.ElementsMatch(t, test.expected.liveAdapters, adapters, "Incorrect adapter list")
			assert.Equal(t, fledge, test.expected.fledge, "Incorrect FLEDGE response")

			for _, bidderName := range test.expected.liveAdapters {
				assert.ElementsMatch(t, test.expected.adapterBids[bidderName].Bids, bids[bidderName].Bids, "Incorrect bids")
			}
		}
	}
}

func TestAuctionDebugEnabled(t *testing.T) {
	categoriesFetcher, err := newCategoryFetcher("./test/category-mapping")
	assert.NoError(t, err, "error should be nil")
	e := new(exchange)
	e.cache = &wellBehavedCache{}
	e.me = &metricsConf.NilMetricsEngine{}
	e.categoriesFetcher = categoriesFetcher
	e.bidIDGenerator = &fakeBidIDGenerator{GenerateBidID: false, ReturnError: false}
	e.currencyConverter = currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	e.gdprPermsBuilder = fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder
	e.requestSplitter = requestSplitter{
		me:               e.me,
		gdprPermsBuilder: e.gdprPermsBuilder,
	}

	ctx := context.Background()

	bidRequest := &openrtb2.BidRequest{
		ID:   "some-request-id",
		Test: 1,
	}

	auctionRequest := &AuctionRequest{
		BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: bidRequest},
		Account:           config.Account{DebugAllow: false},
		UserSyncs:         &emptyUsersync{},
		StartTime:         time.Now(),
		RequestType:       metrics.ReqTypeORTB2Web,
		HookExecutor:      &hookexecution.EmptyHookExecutor{},
		TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
	}

	debugLog := &DebugLog{DebugOverride: true, DebugEnabledOrOverridden: true}
	resp, err := e.HoldAuction(ctx, auctionRequest, debugLog)

	assert.NoError(t, err, "error should be nil")

	expectedResolvedRequest := `{"id":"some-request-id","imp":null,"test":1}`
	actualResolvedRequest, _, _, err := jsonparser.Get(resp.Ext, "debug", "resolvedrequest")
	assert.NoError(t, err, "error should be nil")
	assert.NotNil(t, actualResolvedRequest, "actualResolvedRequest should not be nil")
	assert.JSONEq(t, expectedResolvedRequest, string(actualResolvedRequest), "Resolved request is incorrect")

}

func TestPassExperimentConfigsToHoldAuction(t *testing.T) {
	noBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	server := httptest.NewServer(http.HandlerFunc(noBidServer))
	defer server.Close()

	cfg := &config.Configuration{}

	biddersInfo, err := config.LoadBidderInfoFromDisk("../static/bidder-info")
	if err != nil {
		t.Fatal(err)
	}
	biddersInfo["appnexus"] = config.BidderInfo{
		Endpoint: "test.com",
		Capabilities: &config.CapabilitiesInfo{
			Site: &config.PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo},
			},
		},
		Experiment: config.BidderInfoExperiment{AdsCert: config.BidderAdsCert{Enabled: true}}}

	signer := MockSigner{}

	adapters, _, adaptersErr := BuildAdapters(server.Client(), cfg, biddersInfo, &metricsConf.NilMetricsEngine{})
	if adaptersErr != nil {
		t.Fatalf("Error intializing adapters: %v", adaptersErr)
	}

	currencyConverter := currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	gdprPermsBuilder := fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder

	e := NewExchange(adapters, nil, cfg, &mockRequestValidator{}, map[string]usersync.Syncer{}, &metricsConf.NilMetricsEngine{}, biddersInfo, gdprPermsBuilder, currencyConverter, nilCategoryFetcher{}, &signer, macros.NewStringIndexBasedReplacer(), nil, nil).(*exchange)

	// Define mock incoming bid requeset
	mockBidRequest := &openrtb2.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb2.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext:    json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placementId":1}}}}`),
		}},
		Site: &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
		Ext:  json.RawMessage(`{"prebid":{"experiment":{"adscert":{"enabled": true}}}}`),
	}

	auctionRequest := &AuctionRequest{
		BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: mockBidRequest},
		Account:           config.Account{},
		UserSyncs:         &emptyUsersync{},
		HookExecutor:      &hookexecution.EmptyHookExecutor{},
		TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
	}

	debugLog := DebugLog{}
	_, err = e.HoldAuction(context.Background(), auctionRequest, &debugLog)

	assert.NoError(t, err, "unexpected error occured")
	assert.Equal(t, "test.com", signer.data, "incorrect signer data")
}

func TestCallSignHeader(t *testing.T) {
	type aTest struct {
		description    string
		experiment     openrtb_ext.Experiment
		bidderInfo     config.BidderInfo
		expectedResult bool
	}
	var nilExperiment openrtb_ext.Experiment

	testCases := []aTest{
		{
			description:    "both experiment.adsCert enabled for request and for bidder ",
			experiment:     openrtb_ext.Experiment{AdsCert: &openrtb_ext.AdsCert{Enabled: true}},
			bidderInfo:     config.BidderInfo{Experiment: config.BidderInfoExperiment{AdsCert: config.BidderAdsCert{Enabled: true}}},
			expectedResult: true,
		},
		{
			description:    "experiment is not defined in request, bidder config adsCert enabled",
			experiment:     nilExperiment,
			bidderInfo:     config.BidderInfo{Experiment: config.BidderInfoExperiment{AdsCert: config.BidderAdsCert{Enabled: true}}},
			expectedResult: false,
		},
		{
			description:    "experiment.adsCert is not defined in request, bidder config adsCert enabled",
			experiment:     openrtb_ext.Experiment{AdsCert: nil},
			bidderInfo:     config.BidderInfo{Experiment: config.BidderInfoExperiment{AdsCert: config.BidderAdsCert{Enabled: true}}},
			expectedResult: false,
		},
		{
			description:    "experiment.adsCert is disabled in request, bidder config adsCert enabled",
			experiment:     openrtb_ext.Experiment{AdsCert: &openrtb_ext.AdsCert{Enabled: false}},
			bidderInfo:     config.BidderInfo{Experiment: config.BidderInfoExperiment{AdsCert: config.BidderAdsCert{Enabled: true}}},
			expectedResult: false,
		},
		{
			description:    "experiment.adsCert is enabled in request, bidder config adsCert disabled",
			experiment:     openrtb_ext.Experiment{AdsCert: &openrtb_ext.AdsCert{Enabled: true}},
			bidderInfo:     config.BidderInfo{Experiment: config.BidderInfoExperiment{AdsCert: config.BidderAdsCert{Enabled: false}}},
			expectedResult: false,
		},
		{
			description:    "experiment.adsCert is disabled in request, bidder config adsCert disabled",
			experiment:     openrtb_ext.Experiment{AdsCert: &openrtb_ext.AdsCert{Enabled: false}},
			bidderInfo:     config.BidderInfo{Experiment: config.BidderInfoExperiment{AdsCert: config.BidderAdsCert{Enabled: false}}},
			expectedResult: false,
		},
	}
	for _, test := range testCases {
		result := isAdsCertEnabled(&test.experiment, test.bidderInfo)
		assert.Equal(t, test.expectedResult, result, "incorrect result returned")
	}

}

func TestValidateBannerCreativeSize(t *testing.T) {
	exchange := exchange{bidValidationEnforcement: config.Validations{MaxCreativeWidth: 100, MaxCreativeHeight: 100},
		me: metricsConf.NewMetricsEngine(&config.Configuration{}, openrtb_ext.CoreBidderNames(), nil, nil),
	}
	testCases := []struct {
		description                 string
		givenBid                    *entities.PbsOrtbBid
		givenBidResponseExt         *openrtb_ext.ExtBidResponse
		givenBidderName             string
		givenPubID                  string
		expectedBannerCreativeValid bool
	}{
		{
			description:                 "The dimensions are invalid, both values bigger than the max",
			givenBid:                    &entities.PbsOrtbBid{Bid: &openrtb2.Bid{W: 200, H: 200}},
			givenBidResponseExt:         &openrtb_ext.ExtBidResponse{Errors: make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage)},
			givenBidderName:             "bidder",
			givenPubID:                  "1",
			expectedBannerCreativeValid: false,
		},
		{
			description:                 "The width is invalid, height is valid, the dimensions as a whole are invalid",
			givenBid:                    &entities.PbsOrtbBid{Bid: &openrtb2.Bid{W: 200, H: 50}},
			givenBidResponseExt:         &openrtb_ext.ExtBidResponse{Errors: make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage)},
			givenBidderName:             "bidder",
			givenPubID:                  "1",
			expectedBannerCreativeValid: false,
		},
		{
			description:                 "The width is valid, height is invalid, the dimensions as a whole are invalid",
			givenBid:                    &entities.PbsOrtbBid{Bid: &openrtb2.Bid{W: 50, H: 200}},
			givenBidResponseExt:         &openrtb_ext.ExtBidResponse{Errors: make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage)},
			givenBidderName:             "bidder",
			givenPubID:                  "1",
			expectedBannerCreativeValid: false,
		},
		{
			description:                 "Both width and height are valid, the dimensions are valid",
			givenBid:                    &entities.PbsOrtbBid{Bid: &openrtb2.Bid{W: 50, H: 50}},
			givenBidResponseExt:         &openrtb_ext.ExtBidResponse{Errors: make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage)},
			givenBidderName:             "bidder",
			givenPubID:                  "1",
			expectedBannerCreativeValid: true,
		},
	}
	for _, test := range testCases {
		acutalBannerCreativeValid := exchange.validateBannerCreativeSize(test.givenBid, test.givenBidResponseExt, openrtb_ext.BidderName(test.givenBidderName), test.givenPubID, "enforce")
		assert.Equal(t, test.expectedBannerCreativeValid, acutalBannerCreativeValid)
	}
}

func TestValidateBidAdM(t *testing.T) {
	exchange := exchange{bidValidationEnforcement: config.Validations{MaxCreativeWidth: 100, MaxCreativeHeight: 100},
		me: metricsConf.NewMetricsEngine(&config.Configuration{}, openrtb_ext.CoreBidderNames(), nil, nil),
	}
	testCases := []struct {
		description         string
		givenBid            *entities.PbsOrtbBid
		givenBidResponseExt *openrtb_ext.ExtBidResponse
		givenBidderName     string
		givenPubID          string
		expectedBidAdMValid bool
	}{
		{
			description:         "The adm of the bid contains insecure string and no secure string, adm is invalid",
			givenBid:            &entities.PbsOrtbBid{Bid: &openrtb2.Bid{AdM: "http://domain.com/invalid"}},
			givenBidResponseExt: &openrtb_ext.ExtBidResponse{Errors: make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage)},
			givenBidderName:     "bidder",
			givenPubID:          "1",
			expectedBidAdMValid: false,
		},
		{
			description:         "The adm has both an insecure and secure string defined and therefore the adm is valid",
			givenBid:            &entities.PbsOrtbBid{Bid: &openrtb2.Bid{AdM: "http://www.foo.com https://www.bar.com"}},
			givenBidResponseExt: &openrtb_ext.ExtBidResponse{Errors: make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage)},
			givenBidderName:     "bidder",
			givenPubID:          "1",
			expectedBidAdMValid: true,
		},
		{
			description:         "The adm has both an insecure and secure string defined and therefore the adm is valid",
			givenBid:            &entities.PbsOrtbBid{Bid: &openrtb2.Bid{AdM: "http%3A//www.foo.com https%3A//www.bar.com"}},
			givenBidResponseExt: &openrtb_ext.ExtBidResponse{Errors: make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage)},
			givenBidderName:     "bidder",
			givenPubID:          "1",
			expectedBidAdMValid: true,
		},
		{
			description:         "The adm of the bid are valid with a secure string",
			givenBid:            &entities.PbsOrtbBid{Bid: &openrtb2.Bid{AdM: "https://domain.com/valid"}},
			givenBidResponseExt: &openrtb_ext.ExtBidResponse{Errors: make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage)},
			givenBidderName:     "bidder",
			givenPubID:          "1",
			expectedBidAdMValid: true,
		},
	}
	for _, test := range testCases {
		actualBidAdMValid := exchange.validateBidAdM(test.givenBid, test.givenBidResponseExt, openrtb_ext.BidderName(test.givenBidderName), test.givenPubID, "enforce")
		assert.Equal(t, test.expectedBidAdMValid, actualBidAdMValid)

	}
}

func TestMakeBidWithValidation(t *testing.T) {
	sampleAd := "<?xml version=\"1.0\" encoding=\"UTF-8\"?><VAST ...></VAST>"
	sampleOpenrtbBid := &openrtb2.Bid{ID: "some-bid-id", AdM: sampleAd}

	testCases := []struct {
		name                     string
		givenBidRequestExt       json.RawMessage
		givenValidations         config.Validations
		givenBids                []*entities.PbsOrtbBid
		givenSeat                openrtb_ext.BidderName
		expectedNumOfBids        int
		expectedNonBids          *SeatNonBidBuilder
		expectedNumDebugErrors   int
		expectedNumDebugWarnings int
	}{
		{
			name:               "One_of_two_bids_is_invalid_based_on_DSA_object_presence",
			givenBidRequestExt: json.RawMessage(`{"dsa": {"dsarequired": 2}}`),
			givenValidations:   config.Validations{},
			givenBids:          []*entities.PbsOrtbBid{{Bid: &openrtb2.Bid{Ext: json.RawMessage(`{"dsa": {"adrender":1}}`)}}, {Bid: &openrtb2.Bid{}}},
			givenSeat:          "pubmatic",
			expectedNumOfBids:  1,
			expectedNonBids: &SeatNonBidBuilder{
				"pubmatic": {
					{
						StatusCode: 300,
						Ext: &openrtb_ext.NonBidExt{
							Prebid: openrtb_ext.ExtResponseNonBidPrebid{
								Bid: openrtb_ext.NonBidObject{},
							},
						},
					},
				},
			},
			expectedNumDebugWarnings: 1,
		},
		{
			name:              "Creative_size_validation_enforced,_one_of_two_bids_has_invalid_dimensions",
			givenValidations:  config.Validations{BannerCreativeMaxSize: config.ValidationEnforce, MaxCreativeWidth: 100, MaxCreativeHeight: 100},
			givenBids:         []*entities.PbsOrtbBid{{Bid: &openrtb2.Bid{W: 200, H: 200}, BidType: openrtb_ext.BidTypeBanner}, {Bid: &openrtb2.Bid{W: 50, H: 50}, BidType: openrtb_ext.BidTypeBanner}},
			givenSeat:         "pubmatic",
			expectedNumOfBids: 1,
			expectedNonBids: &SeatNonBidBuilder{
				"pubmatic": {
					{
						StatusCode: 351,
						Ext: &openrtb_ext.NonBidExt{
							Prebid: openrtb_ext.ExtResponseNonBidPrebid{
								Bid: openrtb_ext.NonBidObject{
									W: 200,
									H: 200,
								},
							},
						},
					},
				},
			},
			expectedNumDebugErrors: 1,
		},
		{
			name:                   "Creative_size_validation_warned,_one_of_two_bids_has_invalid_dimensions",
			givenValidations:       config.Validations{BannerCreativeMaxSize: config.ValidationWarn, MaxCreativeWidth: 100, MaxCreativeHeight: 100},
			givenBids:              []*entities.PbsOrtbBid{{Bid: &openrtb2.Bid{W: 200, H: 200}, BidType: openrtb_ext.BidTypeBanner}, {Bid: &openrtb2.Bid{W: 50, H: 50}, BidType: openrtb_ext.BidTypeBanner}},
			givenSeat:              "pubmatic",
			expectedNumOfBids:      2,
			expectedNonBids:        &SeatNonBidBuilder{},
			expectedNumDebugErrors: 1,
		},
		{
			name:              "AdM_validation_enforced,_one_of_two_bids_has_invalid_AdM",
			givenValidations:  config.Validations{SecureMarkup: config.ValidationEnforce},
			givenBids:         []*entities.PbsOrtbBid{{Bid: &openrtb2.Bid{AdM: "http://domain.com/invalid", ImpID: "1"}, BidType: openrtb_ext.BidTypeBanner}, {Bid: &openrtb2.Bid{AdM: "https://domain.com/valid", ImpID: "2"}, BidType: openrtb_ext.BidTypeBanner}},
			givenSeat:         "pubmatic",
			expectedNumOfBids: 1,
			expectedNonBids: &SeatNonBidBuilder{
				"pubmatic": {
					{
						ImpId:      "1",
						StatusCode: 352,
						Ext: &openrtb_ext.NonBidExt{
							Prebid: openrtb_ext.ExtResponseNonBidPrebid{
								Bid: openrtb_ext.NonBidObject{},
							},
						},
					},
				},
			},
			expectedNumDebugErrors: 1,
		},
		{
			name:                   "AdM_validation_warned,_one_of_two_bids_has_invalid_AdM",
			givenValidations:       config.Validations{SecureMarkup: config.ValidationWarn},
			givenBids:              []*entities.PbsOrtbBid{{Bid: &openrtb2.Bid{AdM: "http://domain.com/invalid", ImpID: "1"}, BidType: openrtb_ext.BidTypeBanner}, {Bid: &openrtb2.Bid{AdM: "https://domain.com/valid", ImpID: "2"}, BidType: openrtb_ext.BidTypeBanner}},
			givenSeat:              "pubmatic",
			expectedNumOfBids:      2,
			expectedNonBids:        &SeatNonBidBuilder{},
			expectedNumDebugErrors: 1,
		},
		{
			name:              "Adm_validation_skipped,_creative_size_validation_enforced,_one_of_two_bids_has_invalid_AdM",
			givenValidations:  config.Validations{SecureMarkup: config.ValidationSkip, BannerCreativeMaxSize: config.ValidationEnforce},
			givenBids:         []*entities.PbsOrtbBid{{Bid: &openrtb2.Bid{AdM: "http://domain.com/invalid"}, BidType: openrtb_ext.BidTypeBanner}, {Bid: &openrtb2.Bid{AdM: "https://domain.com/valid"}, BidType: openrtb_ext.BidTypeBanner}},
			givenSeat:         "pubmatic",
			expectedNumOfBids: 2,
			expectedNonBids:   &SeatNonBidBuilder{},
		},
		{
			name:              "Creative_size_validation_skipped,_Adm_Validation_enforced,_one_of_two_bids_has_invalid_dimensions",
			givenValidations:  config.Validations{BannerCreativeMaxSize: config.ValidationSkip, MaxCreativeWidth: 100, MaxCreativeHeight: 100},
			givenBids:         []*entities.PbsOrtbBid{{Bid: &openrtb2.Bid{W: 200, H: 200}, BidType: openrtb_ext.BidTypeBanner}, {Bid: &openrtb2.Bid{W: 50, H: 50}, BidType: openrtb_ext.BidTypeBanner}},
			givenSeat:         "pubmatic",
			expectedNumOfBids: 2,
			expectedNonBids:   &SeatNonBidBuilder{},
		},
	}

	// Test set up
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
	e.adapterMap = map[openrtb_ext.BidderName]AdaptedBidder{
		openrtb_ext.BidderAppnexus: AdaptBidder(bidderImpl, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, nil, ""),
	}
	e.cache = &wellBehavedCache{}
	e.me = &metricsConf.NilMetricsEngine{}

	e.currencyConverter = currency.NewRateConverter(&http.Client{}, "", time.Duration(0))

	ImpExtInfoMap := make(map[string]ImpExtInfo)
	ImpExtInfoMap["1"] = ImpExtInfo{}
	ImpExtInfoMap["2"] = ImpExtInfo{}

	//Run tests
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			bidExtResponse := &openrtb_ext.ExtBidResponse{
				Errors:   make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage),
				Warnings: make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage),
			}
			bidRequest := &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: &openrtb2.Regs{
						Ext: test.givenBidRequestExt,
					},
				},
			}
			e.bidValidationEnforcement = test.givenValidations
			sampleBids := test.givenBids
			nonBids := &SeatNonBidBuilder{}
			resultingBids, resultingErrs := e.makeBid(sampleBids, sampleAuction, true, ImpExtInfoMap, bidRequest, bidExtResponse, test.givenSeat, "", nonBids)

			assert.Equal(t, 0, len(resultingErrs))
			assert.Equal(t, test.expectedNumOfBids, len(resultingBids))
			assert.Equal(t, test.expectedNonBids, nonBids)
			assert.Equal(t, test.expectedNumDebugErrors, len(bidExtResponse.Errors))
			assert.Equal(t, test.expectedNumDebugWarnings, len(bidExtResponse.Warnings))
		})
	}
}

func TestSetBidValidationStatus(t *testing.T) {
	testCases := []struct {
		description  string
		givenHost    config.Validations
		givenAccount config.Validations
		expected     config.Validations
	}{
		{
			description:  "Host configuration is different than account, account setting should be preferred (enforce)",
			givenHost:    config.Validations{BannerCreativeMaxSize: config.ValidationSkip, SecureMarkup: config.ValidationSkip},
			givenAccount: config.Validations{BannerCreativeMaxSize: config.ValidationEnforce, SecureMarkup: config.ValidationEnforce},
			expected:     config.Validations{BannerCreativeMaxSize: config.ValidationEnforce, SecureMarkup: config.ValidationSkip},
		},
		{
			description:  "Host configuration is different than account, account setting should be preferred (warn)",
			givenHost:    config.Validations{BannerCreativeMaxSize: config.ValidationEnforce, SecureMarkup: config.ValidationEnforce},
			givenAccount: config.Validations{BannerCreativeMaxSize: config.ValidationWarn, SecureMarkup: config.ValidationWarn},
			expected:     config.Validations{BannerCreativeMaxSize: config.ValidationWarn, SecureMarkup: config.ValidationEnforce},
		},
		{
			description:  "Host configuration is different than account, account setting should be preferred (skip)",
			givenHost:    config.Validations{BannerCreativeMaxSize: config.ValidationWarn, SecureMarkup: config.ValidationWarn},
			givenAccount: config.Validations{BannerCreativeMaxSize: config.ValidationSkip, SecureMarkup: config.ValidationSkip},
			expected:     config.Validations{BannerCreativeMaxSize: config.ValidationSkip, SecureMarkup: config.ValidationWarn},
		},
		{
			description:  "No account confiugration given, host confg should be preferred",
			givenHost:    config.Validations{BannerCreativeMaxSize: config.ValidationSkip, SecureMarkup: config.ValidationSkip},
			givenAccount: config.Validations{},
			expected:     config.Validations{BannerCreativeMaxSize: config.ValidationSkip, SecureMarkup: config.ValidationSkip},
		},
	}
	for _, test := range testCases {
		test.givenHost.SetBannerCreativeMaxSize(test.givenAccount)
		assert.Equal(t, test.expected, test.givenHost)
	}
}

/*
TestOverrideConfigAlternateBidderCodesWithRequestValues makes sure that the correct alternabiddercodes list is forwarded to the adapters and only the approved bids are returned in auction response.

1. request.ext.prebid.alternatebiddercodes has priority over the content of config.Account.Alternatebiddercodes.

2. request is updated with config.Account.Alternatebiddercodes values if request.ext.prebid.alternatebiddercodes is empty or not specified.

3. request.ext.prebid.alternatebiddercodes is given priority over config.Account.Alternatebiddercodes if both are specified.
*/
func TestOverrideConfigAlternateBidderCodesWithRequestValues(t *testing.T) {
	type testIn struct {
		config     config.Configuration
		requestExt json.RawMessage
	}
	type testResults struct {
		expectedSeats []string
	}

	testCases := []struct {
		desc     string
		in       testIn
		expected testResults
	}{
		{
			desc: "alternatebiddercode defined neither in config nor in the request",
			in: testIn{
				config: config.Configuration{},
			},
			expected: testResults{
				expectedSeats: []string{"pubmatic"},
			},
		},
		{
			desc: "alternatebiddercode defined in config and not in request",
			in: testIn{
				config: config.Configuration{
					AccountDefaults: config.Account{
						AlternateBidderCodes: &openrtb_ext.ExtAlternateBidderCodes{
							Enabled: true,
							Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
								"pubmatic": {
									Enabled:            true,
									AllowedBidderCodes: []string{"groupm"},
								},
							},
						},
					},
				},
				requestExt: json.RawMessage(`{}`),
			},
			expected: testResults{
				expectedSeats: []string{"pubmatic", "groupm"},
			},
		},
		{
			desc: "alternatebiddercode defined in request and not in config",
			in: testIn{
				requestExt: json.RawMessage(`{"prebid": {"alternatebiddercodes": {"enabled": true, "bidders": {"pubmatic": {"enabled": true, "allowedbiddercodes": ["appnexus"]}}}}}`),
			},
			expected: testResults{
				expectedSeats: []string{"pubmatic", "appnexus"},
			},
		},
		{
			desc: "alternatebiddercode defined in both config and in request",
			in: testIn{
				config: config.Configuration{
					AccountDefaults: config.Account{
						AlternateBidderCodes: &openrtb_ext.ExtAlternateBidderCodes{
							Enabled: true,
							Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
								"pubmatic": {
									Enabled:            true,
									AllowedBidderCodes: []string{"groupm"},
								},
							},
						},
					},
				},
				requestExt: json.RawMessage(`{"prebid": {"alternatebiddercodes": {"enabled": true, "bidders": {"pubmatic": {"enabled": true, "allowedbiddercodes": ["ix"]}}}}}`),
			},
			expected: testResults{
				expectedSeats: []string{"pubmatic", "ix"},
			},
		},
	}

	// Init an exchange to run an auction from
	noBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }
	mockPubMaticBidService := httptest.NewServer(http.HandlerFunc(noBidServer))
	defer mockPubMaticBidService.Close()

	categoriesFetcher, error := newCategoryFetcher("./test/category-mapping")
	if error != nil {
		t.Errorf("Failed to create a category Fetcher: %v", error)
	}

	mockBidderRequestResponse := &goodSingleBidder{
		httpRequest: &adapters.RequestData{
			Method:  "POST",
			Uri:     mockPubMaticBidService.URL,
			Body:    []byte("{\"key\":\"val\"}"),
			Headers: http.Header{},
		},
		bidResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{Bid: &openrtb2.Bid{ID: "1"}, Seat: ""},
				{Bid: &openrtb2.Bid{ID: "2"}, Seat: "pubmatic"},
				{Bid: &openrtb2.Bid{ID: "3"}, Seat: "appnexus"},
				{Bid: &openrtb2.Bid{ID: "4"}, Seat: "groupm"},
				{Bid: &openrtb2.Bid{ID: "5"}, Seat: "ix"},
			},
			Currency: "USD",
		},
	}

	e := new(exchange)
	e.cache = &wellBehavedCache{}
	e.me = &metricsConf.NilMetricsEngine{}
	e.gdprPermsBuilder = fakePermissionsBuilder{
		permissions: &permissionsMock{
			allowAllBidders: true,
		},
	}.Builder
	e.currencyConverter = currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	e.categoriesFetcher = categoriesFetcher
	e.bidIDGenerator = &fakeBidIDGenerator{GenerateBidID: false, ReturnError: false}
	e.requestSplitter = requestSplitter{
		me:               e.me,
		gdprPermsBuilder: e.gdprPermsBuilder,
	}

	// Define mock incoming bid requeset
	mockBidRequest := &openrtb2.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb2.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext:    json.RawMessage(`{"prebid":{"bidder":{"pubmatic": {"publisherId": 1}}}}`),
		}},
		Site: &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
	}

	// Run tests
	for _, test := range testCases {
		e.adapterMap = map[openrtb_ext.BidderName]AdaptedBidder{
			openrtb_ext.BidderPubmatic: AdaptBidder(mockBidderRequestResponse, mockPubMaticBidService.Client(), &test.in.config, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderPubmatic, nil, ""),
		}

		mockBidRequest.Ext = test.in.requestExt

		auctionRequest := &AuctionRequest{
			BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: mockBidRequest},
			Account:           test.in.config.AccountDefaults,
			UserSyncs:         &emptyUsersync{},
			HookExecutor:      &hookexecution.EmptyHookExecutor{},
			TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
		}

		// Run test
		outBidResponse, err := e.HoldAuction(context.Background(), auctionRequest, &DebugLog{})

		// Assertions
		assert.NoErrorf(t, err, "%s. HoldAuction error: %v \n", test.desc, err)
		assert.NotNil(t, outBidResponse)
		assert.False(t, auctionRequest.BidderResponseStartTime.IsZero())

		// So 2 seatBids are expected as,
		// the default "" and "pubmatic" bids will be in one seat and the extra-bids "groupm"/"appnexus"/"ix" in another seat.
		assert.Len(t, outBidResponse.SeatBid, len(test.expected.expectedSeats), "%s. seatbid count miss-match\n", test.desc)

		for i, seatBid := range outBidResponse.SeatBid {
			assert.Contains(t, test.expected.expectedSeats, seatBid.Seat, "%s. unexpected seatbid\n", test.desc)

			if seatBid.Seat == string(openrtb_ext.BidderPubmatic) {
				assert.Len(t, outBidResponse.SeatBid[i].Bid, 2, "%s. unexpected bid count\n", test.desc)
			} else {
				assert.Len(t, outBidResponse.SeatBid[i].Bid, 1, "%s. unexpected bid count\n", test.desc)
			}
		}
	}
}

func TestGetAllBids(t *testing.T) {
	noBidServer := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
	server := httptest.NewServer(http.HandlerFunc(noBidServer))
	defer server.Close()

	type testIn struct {
		bidderRequests                 []BidderRequest
		bidAdjustments                 map[string]float64
		conversions                    currency.Conversions
		accountDebugAllowed            bool
		globalPrivacyControlHeader     string
		headerDebugAllowed             bool
		alternateBidderCodes           openrtb_ext.ExtAlternateBidderCodes
		experiment                     *openrtb_ext.Experiment
		hookExecutor                   hookexecution.StageExecutor
		pbsRequestStartTime            time.Time
		bidAdjustmentRules             map[string][]openrtb_ext.Adjustment
		tmaxAdjustments                *TmaxAdjustmentsPreprocessed
		adapterMap                     map[openrtb_ext.BidderName]AdaptedBidder
		liveAdaptersPreferredMediaType map[openrtb_ext.BidderName]openrtb_ext.BidType
	}
	type testResults struct {
		adapterBids   map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		adapterExtra  map[openrtb_ext.BidderName]*seatResponseExtra
		extraRespInfo extraAuctionResponseInfo
	}
	testCases := []struct {
		desc     string
		in       testIn
		expected testResults
	}{
		{
			desc: "alternateBidderCodes-config-absent: pubmatic bidder returns bids with 'pubmatic' and 'groupm' seats",
			in: testIn{
				bidderRequests: []BidderRequest{
					{
						BidderName:     "pubmatic",
						BidderCoreName: "pubmatic",
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{{
								ID: "some-impression-id",
							}},
						},
					},
				},
				conversions:         &currency.ConstantRates{},
				hookExecutor:        hookexecution.EmptyHookExecutor{},
				pbsRequestStartTime: time.Now(),
				adapterMap: map[openrtb_ext.BidderName]AdaptedBidder{
					openrtb_ext.BidderPubmatic: AdaptBidder(&goodSingleBidder{
						httpRequest: &adapters.RequestData{
							Method: "POST",
							Uri:    server.URL,
						},
						bidResponse: &adapters.BidderResponse{
							Bids: []*adapters.TypedBid{
								{Bid: &openrtb2.Bid{ID: "1"}, Seat: "pubmatic"},
								{Bid: &openrtb2.Bid{ID: "2"}, Seat: "groupm"},
							},
						},
					}, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderPubmatic, nil, ""),
				},
			},
			expected: testResults{
				extraRespInfo: extraAuctionResponseInfo{
					bidsFound: true,
				},
				adapterExtra: map[openrtb_ext.BidderName]*seatResponseExtra{
					"pubmatic": {
						Warnings: []openrtb_ext.ExtBidderMessage{
							{
								Code:    errortypes.AlternateBidderCodeWarningCode,
								Message: `alternateBidderCodes disabled for "pubmatic", rejecting bids for "groupm"`,
							},
						},
					},
				},
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID: "1",
								},
								OriginalBidCur: "USD",
								AdapterCode:    openrtb_ext.BidderPubmatic,
							},
						},
						Currency:  "USD",
						Seat:      "pubmatic",
						HttpCalls: []*openrtb_ext.ExtHttpCall{},
					},
				},
			},
		},
		{
			desc: "alternateBidderCodes-enabled: pubmatic bidder returns bids with 'pubmatic' and 'groupm' seats",
			in: testIn{
				bidderRequests: []BidderRequest{
					{
						BidderName:     "pubmatic",
						BidderCoreName: "pubmatic",
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{{
								ID: "some-impression-id",
							}},
						},
					},
				},
				conversions: &currency.ConstantRates{},
				alternateBidderCodes: openrtb_ext.ExtAlternateBidderCodes{
					Enabled: true,
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						"pubmatic": {
							Enabled:            true,
							AllowedBidderCodes: []string{"groupm"},
						},
					},
				},
				hookExecutor:        hookexecution.EmptyHookExecutor{},
				pbsRequestStartTime: time.Now(),
				adapterMap: map[openrtb_ext.BidderName]AdaptedBidder{
					openrtb_ext.BidderPubmatic: AdaptBidder(&goodSingleBidder{
						httpRequest: &adapters.RequestData{
							Method: "POST",
							Uri:    server.URL,
						},
						bidResponse: &adapters.BidderResponse{
							Bids: []*adapters.TypedBid{
								{Bid: &openrtb2.Bid{ID: "1"}, Seat: "pubmatic"},
								{Bid: &openrtb2.Bid{ID: "2"}, Seat: "groupm"},
							},
						},
					}, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderPubmatic, nil, ""),
				},
			},
			expected: testResults{
				extraRespInfo: extraAuctionResponseInfo{
					bidsFound: true,
				},
				adapterExtra: map[openrtb_ext.BidderName]*seatResponseExtra{
					"pubmatic": {
						Warnings: []openrtb_ext.ExtBidderMessage{},
					},
				},
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"pubmatic": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID: "1",
								},
								OriginalBidCur: "USD",
								AdapterCode:    openrtb_ext.BidderPubmatic,
							},
						},
						Currency:  "USD",
						Seat:      "pubmatic",
						HttpCalls: []*openrtb_ext.ExtHttpCall{},
					},
					"groupm": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID: "2",
								},
								OriginalBidCur: "USD",
								AdapterCode:    openrtb_ext.BidderPubmatic,
							},
						},
						Currency:  "USD",
						Seat:      "groupm",
						HttpCalls: []*openrtb_ext.ExtHttpCall{},
					},
				},
			},
		},
		{
			desc: "alternateBidderCodes-enabled: pubmatic bidder returns bids with only 'groupm' seat",
			in: testIn{
				bidderRequests: []BidderRequest{
					{
						BidderName:     "pubmatic",
						BidderCoreName: "pubmatic",
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{{
								ID: "some-impression-id",
							}},
						},
					},
				},
				conversions: &currency.ConstantRates{},
				alternateBidderCodes: openrtb_ext.ExtAlternateBidderCodes{
					Enabled: true,
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						"pubmatic": {
							Enabled:            true,
							AllowedBidderCodes: []string{"groupm"},
						},
					},
				},
				hookExecutor:        hookexecution.EmptyHookExecutor{},
				pbsRequestStartTime: time.Now(),
				adapterMap: map[openrtb_ext.BidderName]AdaptedBidder{
					openrtb_ext.BidderPubmatic: AdaptBidder(&goodSingleBidder{
						httpRequest: &adapters.RequestData{
							Method: "POST",
							Uri:    server.URL,
						},
						bidResponse: &adapters.BidderResponse{
							Bids: []*adapters.TypedBid{
								{Bid: &openrtb2.Bid{ID: "2"}, Seat: "groupm"},
							},
						},
					}, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderPubmatic, nil, ""),
				},
			},
			expected: testResults{
				extraRespInfo: extraAuctionResponseInfo{
					bidsFound: true,
				},
				adapterExtra: map[openrtb_ext.BidderName]*seatResponseExtra{
					"pubmatic": {
						Warnings: []openrtb_ext.ExtBidderMessage{},
					},
				},
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
					"groupm": {
						Bids: []*entities.PbsOrtbBid{
							{
								Bid: &openrtb2.Bid{
									ID: "2",
								},
								OriginalBidCur: "USD",
								AdapterCode:    openrtb_ext.BidderPubmatic,
							},
						},
						Currency:  "USD",
						Seat:      "groupm",
						HttpCalls: []*openrtb_ext.ExtHttpCall{},
					},
				},
			},
		},
		{
			desc: "bidder responded with empty bid",
			in: testIn{
				bidderRequests: []BidderRequest{
					{
						BidderName:     "pubmatic",
						BidderCoreName: "pubmatic",
						BidRequest: &openrtb2.BidRequest{
							ID: "some-request-id",
							Imp: []openrtb2.Imp{{
								ID: "some-impression-id",
							}},
						},
					},
				},
				conversions: &currency.ConstantRates{},
				alternateBidderCodes: openrtb_ext.ExtAlternateBidderCodes{
					Enabled: true,
					Bidders: map[string]openrtb_ext.ExtAdapterAlternateBidderCodes{
						"pubmatic": {
							Enabled:            true,
							AllowedBidderCodes: []string{"groupm"},
						},
					},
				},
				hookExecutor:        hookexecution.EmptyHookExecutor{},
				pbsRequestStartTime: time.Now(),
				adapterMap: map[openrtb_ext.BidderName]AdaptedBidder{
					openrtb_ext.BidderPubmatic: AdaptBidder(&goodSingleBidder{
						httpRequest: &adapters.RequestData{
							Method: "POST",
							Uri:    server.URL,
						},
						bidResponse: &adapters.BidderResponse{},
					}, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderPubmatic, nil, ""),
				},
			},
			expected: testResults{
				extraRespInfo: extraAuctionResponseInfo{
					bidsFound: false,
				},
				adapterExtra: map[openrtb_ext.BidderName]*seatResponseExtra{
					"pubmatic": {
						Warnings: []openrtb_ext.ExtBidderMessage{},
					},
				},
				adapterBids: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{},
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			e := exchange{
				cache: &wellBehavedCache{},
				me:    &metricsConf.NilMetricsEngine{},
				gdprPermsBuilder: fakePermissionsBuilder{
					permissions: &permissionsMock{
						allowAllBidders: true,
					},
				}.Builder,
				adapterMap: test.in.adapterMap,
			}

			adapterBids, adapterExtra, extraRespInfo := e.getAllBids(context.Background(), test.in.bidderRequests, test.in.bidAdjustments,
				test.in.conversions, test.in.accountDebugAllowed, test.in.globalPrivacyControlHeader, test.in.headerDebugAllowed, test.in.alternateBidderCodes, test.in.experiment,
				test.in.hookExecutor, test.in.pbsRequestStartTime, test.in.bidAdjustmentRules, test.in.tmaxAdjustments, false, test.in.liveAdaptersPreferredMediaType)

			assert.Equalf(t, test.expected.extraRespInfo.bidsFound, extraRespInfo.bidsFound, "extraRespInfo.bidsFound mismatch")
			assert.Equalf(t, test.expected.adapterBids, adapterBids, "adapterBids mismatch")
			assert.Equalf(t, len(test.expected.adapterExtra), len(adapterExtra), "adapterExtra length mismatch")
			for adapter, extra := range test.expected.adapterExtra {
				assert.Equalf(t, extra.Warnings, adapterExtra[adapter].Warnings, "adapterExtra.Warnings mismatch for adapter [%s]", adapter)
			}
		})
	}
}

type MockSigner struct {
	data string
}

func (ms *MockSigner) Sign(destinationURL string, body []byte) (string, error) {
	ms.data = destinationURL
	return "mock data", nil
}

type exchangeSpec struct {
	GDPREnabled                bool                   `json:"gdpr_enabled"`
	FloorsEnabled              bool                   `json:"floors_enabled"`
	IncomingRequest            exchangeRequest        `json:"incomingRequest"`
	OutgoingRequests           map[string]*bidderSpec `json:"outgoingRequests"`
	Response                   exchangeResponse       `json:"response,omitempty"`
	EnforceCCPA                bool                   `json:"enforceCcpa"`
	EnforceLMT                 bool                   `json:"enforceLmt"`
	AssumeGDPRApplies          bool                   `json:"assume_gdpr_applies"`
	DebugLog                   *DebugLog              `json:"debuglog,omitempty"`
	EventsEnabled              bool                   `json:"events_enabled,omitempty"`
	StartTime                  int64                  `json:"start_time_ms,omitempty"`
	BidIDGenerator             *fakeBidIDGenerator    `json:"bidIDGenerator,omitempty"`
	RequestType                *metrics.RequestType   `json:"requestType,omitempty"`
	PassthroughFlag            bool                   `json:"passthrough_flag,omitempty"`
	HostSChainFlag             bool                   `json:"host_schain_flag,omitempty"`
	HostConfigBidValidation    config.Validations     `json:"host_bid_validations"`
	AccountConfigBidValidation config.Validations     `json:"account_bid_validations"`
	AccountFloorsEnabled       bool                   `json:"account_floors_enabled"`
	AccountEnforceDealFloors   bool                   `json:"account_enforce_deal_floors"`
	FledgeEnabled              bool                   `json:"fledge_enabled,omitempty"`
	MultiBid                   *multiBidSpec          `json:"multiBid,omitempty"`
	Server                     exchangeServer         `json:"server,omitempty"`
	AccountPrivacy             config.AccountPrivacy  `json:"accountPrivacy,omitempty"`
	ORTBVersion                map[string]string      `json:"ortbversion"`
	AccountEEACountries        []string               `json:"account_eea_countries"`
}

type multiBidSpec struct {
	AccountMaxBid          int  `json:"default_bid_limit"`
	AssertMultiBidWarnings bool `json:"assert_multi_bid_warnings"`
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

type exchangeServer struct {
	ExternalUrl string `json:"externalURL"`
	GvlID       int    `json:"gvlID"`
	DataCenter  string `json:"dataCenter"`
}

type bidderSpec struct {
	ExpectedRequest         *bidderRequest `json:"expectRequest"`
	MockResponse            bidderResponse `json:"mockResponse"`
	ModifyingVastXmlAllowed bool           `json:"modifyingVastXmlAllowed,omitempty"`
}

type bidderRequest struct {
	OrtbRequest    openrtb2.BidRequest `json:"ortbRequest"`
	BidAdjustments map[string]float64  `json:"bidAdjustments"`
}

type bidderResponse struct {
	SeatBids  []*bidderSeatBid           `json:"pbsSeatBids,omitempty"`
	Errors    []string                   `json:"errors,omitempty"`
	HttpCalls []*openrtb_ext.ExtHttpCall `json:"httpCalls,omitempty"`
}

// bidderSeatBid is basically a subset of entities.PbsOrtbSeatBid from exchange/bidder.go.
// The only real reason I'm not reusing that type is because I don't want people to think that the
// JSON property tags on those types are contracts in prod.
type bidderSeatBid struct {
	Bids                 []bidderBid                        `json:"pbsBids,omitempty"`
	Seat                 string                             `json:"seat"`
	Currency             string                             `json:"currency"`
	FledgeAuctionConfigs []*openrtb_ext.FledgeAuctionConfig `json:"fledgeAuctionConfigs,omitempty"`
}

// bidderBid is basically a subset of entities.PbsOrtbBid from exchange/bidder.go.
// See the comment on bidderSeatBid for more info.
type bidderBid struct {
	Bid      *openrtb2.Bid                  `json:"ortbBid,omitempty"`
	Type     string                         `json:"bidType,omitempty"`
	BidVideo *openrtb_ext.ExtBidPrebidVideo `json:"bidVideo,omitempty"`
	Meta     *openrtb_ext.ExtBidPrebidMeta  `json:"bidMeta,omitempty"`
}

type mockIdFetcher map[string]string

func (f mockIdFetcher) GetUID(key string) (uid string, exists bool, notExpired bool) {
	uid, exists = f[string(key)]
	return
}

func (f mockIdFetcher) HasAnyLiveSyncs() bool {
	return len(f) > 0
}

type validatingBidder struct {
	t          *testing.T
	fileName   string
	bidderName string

	// These are maps because they may contain aliases. They should _at least_ contain an entry for bidderName.
	expectations  map[string]*bidderRequest
	mockResponses map[string]bidderResponse
}

func (b *validatingBidder) requestBid(ctx context.Context, bidderRequest BidderRequest, conversions currency.Conversions, reqInfo *adapters.ExtraRequestInfo, adsCertSigner adscert.Signer, bidRequestOptions bidRequestOptions, alternateBidderCodes openrtb_ext.ExtAlternateBidderCodes, executor hookexecution.StageExecutor, ruleToAdjustments openrtb_ext.AdjustmentsByDealID) (seatBids []*entities.PbsOrtbSeatBid, extaInfo extraBidderRespInfo, errs []error) {
	if expectedRequest, ok := b.expectations[string(bidderRequest.BidderName)]; ok {
		if expectedRequest != nil {
			if !reflect.DeepEqual(expectedRequest.BidAdjustments, bidRequestOptions.bidAdjustments) {
				b.t.Errorf("%s: Bidder %s got wrong bid adjustment. Expected %v, got %v", b.fileName, bidderRequest.BidderName, expectedRequest.BidAdjustments, bidRequestOptions.bidAdjustments)
			}
			diffOrtbRequests(b.t, fmt.Sprintf("Request to %s in %s", string(bidderRequest.BidderName), b.fileName), &expectedRequest.OrtbRequest, bidderRequest.BidRequest)
		}
	} else {
		b.t.Errorf("%s: Bidder %s got unexpected request for alias %s. No input assertions.", b.fileName, b.bidderName, bidderRequest.BidderName)
	}

	if mockResponse, ok := b.mockResponses[string(bidderRequest.BidderName)]; ok {
		if len(mockResponse.SeatBids) != 0 {
			for _, mockSeatBid := range mockResponse.SeatBids {
				var bids []*entities.PbsOrtbBid

				if len(mockSeatBid.Bids) != 0 {
					bids = make([]*entities.PbsOrtbBid, len(mockSeatBid.Bids))
					for i := 0; i < len(bids); i++ {
						bids[i] = &entities.PbsOrtbBid{
							OriginalBidCPM: mockSeatBid.Bids[i].Bid.Price,
							Bid:            mockSeatBid.Bids[i].Bid,
							BidVideo:       mockSeatBid.Bids[i].BidVideo,
							BidType:        openrtb_ext.BidType(mockSeatBid.Bids[i].Type),
							BidMeta:        mockSeatBid.Bids[i].Meta,
						}
					}
				}

				seatBids = append(seatBids, &entities.PbsOrtbSeatBid{
					Bids:                 bids,
					HttpCalls:            mockResponse.HttpCalls,
					Seat:                 mockSeatBid.Seat,
					Currency:             mockSeatBid.Currency,
					FledgeAuctionConfigs: mockSeatBid.FledgeAuctionConfigs,
				})
			}
		} else {
			seatBids = []*entities.PbsOrtbSeatBid{{
				Bids:      nil,
				HttpCalls: mockResponse.HttpCalls,
				Seat:      string(bidderRequest.BidderName),
			}}
		}

		for _, err := range mockResponse.Errors {
			errs = append(errs, errors.New(err))
		}
	} else {
		b.t.Errorf("%s: Bidder %s got unexpected request for alias %s. No mock responses.", b.fileName, b.bidderName, bidderRequest.BidderName)
	}

	return
}

type capturingRequestBidder struct {
	req *openrtb2.BidRequest
}

func (b *capturingRequestBidder) requestBid(ctx context.Context, bidderRequest BidderRequest, conversions currency.Conversions, reqInfo *adapters.ExtraRequestInfo, adsCertSigner adscert.Signer, bidRequestOptions bidRequestOptions, alternateBidderCodes openrtb_ext.ExtAlternateBidderCodes, executor hookexecution.StageExecutor, ruleToAdjustments openrtb_ext.AdjustmentsByDealID) (seatBid []*entities.PbsOrtbSeatBid, errs []error) {
	b.req = bidderRequest.BidRequest
	return []*entities.PbsOrtbSeatBid{{}}, nil
}

func diffOrtbRequests(t *testing.T, description string, expected *openrtb2.BidRequest, actual *openrtb2.BidRequest) {
	t.Helper()
	actualJSON, err := jsonutil.Marshal(actual)
	if err != nil {
		t.Fatalf("%s failed to marshal actual BidRequest into JSON. %v", description, err)
	}

	expectedJSON, err := jsonutil.Marshal(expected)
	if err != nil {
		t.Fatalf("%s failed to marshal expected BidRequest into JSON. %v", description, err)
	}

	assert.JSONEq(t, string(expectedJSON), string(actualJSON), description)
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
	actualJSON, err := jsonutil.Marshal(actualSeats)
	if err != nil {
		t.Fatalf("%s failed to marshal actual BidResponse into JSON. %v", description, err)
	}

	expectedJSON, err := jsonutil.Marshal(expectedSeats)
	if err != nil {
		t.Fatalf("%s failed to marshal expected BidResponse into JSON. %v", description, err)
	}
	assert.JSONEq(t, string(expectedJSON), string(actualJSON), description)
}

func mapifySeatBids(t *testing.T, context string, seatBids []openrtb2.SeatBid) map[string]*openrtb2.SeatBid {
	seatMap := make(map[string]*openrtb2.SeatBid, len(seatBids))
	for i := 0; i < len(seatBids); i++ {
		seatName := seatBids[i].Seat
		if _, ok := seatMap[seatName]; ok {
			t.Fatalf("%s: Contains duplicate Seat: %s", context, seatName)
		} else {
			// The sequence of extra bids for same seat from different bidder is not guaranteed as we randomize the list of adapters
			// This is w.r.t changes at exchange.go#561 (club bids from different bidders for same extra-bid)
			sort.Slice(seatBids[i].Bid, func(x, y int) bool {
				return isNewWinningBid(&seatBids[i].Bid[x], &seatBids[i].Bid[y], true)
			})
			seatMap[seatName] = &seatBids[i]
		}
	}

	return seatMap
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

func (e *emptyUsersync) GetUID(key string) (uid string, exists bool, notExpired bool) {
	return "", false, false
}

func (e *emptyUsersync) HasAnyLiveSyncs() bool {
	return false
}

type panicingAdapter struct{}

func (panicingAdapter) requestBid(ctx context.Context, bidderRequest BidderRequest, conversions currency.Conversions, reqInfo *adapters.ExtraRequestInfo, adsCertSigner adscert.Signer, bidRequestMetadata bidRequestOptions, alternateBidderCodes openrtb_ext.ExtAlternateBidderCodes, executor hookexecution.StageExecutor, ruleToAdjustments openrtb_ext.AdjustmentsByDealID) (posb []*entities.PbsOrtbSeatBid, extraInfo extraBidderRespInfo, errs []error) {
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

func parseRequestAliases(r openrtb2.BidRequest) (map[string]string, error) {
	if len(r.Ext) == 0 {
		return nil, nil
	}

	ext := struct {
		Prebid struct {
			Aliases map[string]string `json:"aliases"`
		} `json:"prebid"`
	}{}

	if err := jsonutil.Unmarshal(r.Ext, &ext); err != nil {
		return nil, err
	}

	return ext.Prebid.Aliases, nil
}

func getInfoFromImp(req *openrtb_ext.RequestWrapper) (json.RawMessage, string, error) {
	bidRequest := req.BidRequest
	imp := bidRequest.Imp[0]
	impID := imp.ID

	var bidderExts map[string]json.RawMessage
	if err := jsonutil.UnmarshalValid(imp.Ext, &bidderExts); err != nil {
		return nil, "", err
	}

	var extPrebid openrtb_ext.ExtImpPrebid
	if bidderExts[openrtb_ext.PrebidExtKey] != nil {
		if err := jsonutil.UnmarshalValid(bidderExts[openrtb_ext.PrebidExtKey], &extPrebid); err != nil {
			return nil, "", err
		}
	}
	return extPrebid.Passthrough, impID, nil
}

func TestModulesCanBeExecutedForMultipleBiddersSimultaneously(t *testing.T) {
	noBidServer := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	}
	server := httptest.NewServer(http.HandlerFunc(noBidServer))
	defer server.Close()

	reqBdy := []byte(`{"key":"val"}`)

	bidderImplAppnexus := &goodSingleBidder{
		httpRequest: &adapters.RequestData{Method: http.MethodPost, Uri: server.URL, Body: reqBdy, Headers: http.Header{}},
		bidResponse: &adapters.BidderResponse{},
	}
	bidderImplTelaria := &goodSingleBidder{
		httpRequest: &adapters.RequestData{Method: http.MethodPost, Uri: server.URL, Body: reqBdy, Headers: http.Header{}},
		bidResponse: &adapters.BidderResponse{},
	}
	bidderImpl33Across := &goodSingleBidder{
		httpRequest: &adapters.RequestData{Method: http.MethodPost, Uri: server.URL, Body: reqBdy, Headers: http.Header{}},
		bidResponse: &adapters.BidderResponse{},
	}
	bidderImplAax := &goodSingleBidder{
		httpRequest: &adapters.RequestData{Method: http.MethodPost, Uri: server.URL, Body: reqBdy, Headers: http.Header{}},
		bidResponse: &adapters.BidderResponse{},
	}

	e := new(exchange)
	e.me = &metricsConf.NilMetricsEngine{}
	e.currencyConverter = currency.NewRateConverter(&http.Client{}, "", time.Duration(0))
	e.requestSplitter = requestSplitter{
		me:               e.me,
		gdprPermsBuilder: e.gdprPermsBuilder,
	}

	bidRequest := &openrtb2.BidRequest{
		ID: "some-request-id",
		Imp: []openrtb2.Imp{{
			ID:     "some-impression-id",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}, {W: 300, H: 600}}},
			Ext: json.RawMessage(
				`{"prebid":{"bidder":{"telaria": {"placementId": 1}, "appnexus": {"placementid": 2}, "33across": {"placementId": 3}, "aax": {"placementid": 4}}}}`,
			),
		}},
		Site:   &openrtb2.Site{Page: "prebid.org", Ext: json.RawMessage(`{"amp":0}`)},
		Device: &openrtb2.Device{UA: "curl/7.54.0", IP: "::1"},
		AT:     1,
		TMax:   500,
	}

	exec := hookexecution.NewHookExecutor(TestApplyHookMutationsBuilder{}, "/openrtb2/auction", &metricsConfig.NilMetricsEngine{})

	auctionRequest := &AuctionRequest{
		BidRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: bidRequest},
		Account:           config.Account{DebugAllow: true},
		UserSyncs:         &emptyUsersync{},
		StartTime:         time.Now(),
		HookExecutor:      exec,
		TCF2Config:        gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
	}

	e.adapterMap = map[openrtb_ext.BidderName]AdaptedBidder{
		openrtb_ext.BidderAppnexus: AdaptBidder(bidderImplAppnexus, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAppnexus, &config.DebugInfo{}, ""),
		openrtb_ext.BidderTelaria:  AdaptBidder(bidderImplTelaria, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderTelaria, &config.DebugInfo{}, ""),
		openrtb_ext.Bidder33Across: AdaptBidder(bidderImpl33Across, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.Bidder33Across, &config.DebugInfo{}, ""),
		openrtb_ext.BidderAax:      AdaptBidder(bidderImplAax, server.Client(), &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderAax, &config.DebugInfo{}, ""),
	}
	// Run test
	_, err := e.HoldAuction(context.Background(), auctionRequest, &DebugLog{})
	// Assert no HoldAuction err
	assert.NoErrorf(t, err, "ex.HoldAuction returned an err")
	assert.False(t, auctionRequest.BidderResponseStartTime.IsZero())

	// check stage outcomes
	assert.Equal(t, len(exec.GetOutcomes()), len(e.adapterMap), "stage outcomes append operation failed")
	//check that all modules were applied and logged
	for _, sto := range exec.GetOutcomes() {
		assert.Equal(t, 2, len(sto.Groups), "not all groups were executed")
		for _, group := range sto.Groups {
			assert.Equal(t, 5, len(group.InvocationResults), "not all module hooks were applied")
			for _, r := range group.InvocationResults {
				assert.Equal(t, "success", string(r.Status), fmt.Sprintf("Module %s hook %s completed unsuccessfully", r.HookID.ModuleCode, r.HookID.HookImplCode))
			}
		}
	}
}

type TestApplyHookMutationsBuilder struct {
	hooks.EmptyPlanBuilder
}

func (e TestApplyHookMutationsBuilder) PlanForBidderRequestStage(_ string, _ *config.Account) hooks.Plan[hookstage.BidderRequest] {
	return hooks.Plan[hookstage.BidderRequest]{
		hooks.Group[hookstage.BidderRequest]{
			Timeout: 100 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.BidderRequest]{
				{Module: "foobar1", Code: "foo1", Hook: mockUpdateBidRequestHook{}},
				{Module: "foobar2", Code: "foo2", Hook: mockUpdateBidRequestHook{}},
				{Module: "foobar3", Code: "foo3", Hook: mockUpdateBidRequestHook{}},
				{Module: "foobar4", Code: "foo4", Hook: mockUpdateBidRequestHook{}},
				{Module: "foobar5", Code: "foo5", Hook: mockUpdateBidRequestHook{}},
			},
		},
		hooks.Group[hookstage.BidderRequest]{
			Timeout: 100 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.BidderRequest]{
				{Module: "foobar6", Code: "foo6", Hook: mockUpdateBidRequestHook{}},
				{Module: "foobar7", Code: "foo7", Hook: mockUpdateBidRequestHook{}},
				{Module: "foobar8", Code: "foo8", Hook: mockUpdateBidRequestHook{}},
				{Module: "foobar9", Code: "foo9", Hook: mockUpdateBidRequestHook{}},
				{Module: "foobar10", Code: "foo10", Hook: mockUpdateBidRequestHook{}},
			},
		},
	}
}

type mockUpdateBidRequestHook struct{}

func (e mockUpdateBidRequestHook) HandleBidderRequestHook(_ context.Context, mctx hookstage.ModuleInvocationContext, _ hookstage.BidderRequestPayload) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	time.Sleep(50 * time.Millisecond)
	c := hookstage.ChangeSet[hookstage.BidderRequestPayload]{}
	c.AddMutation(
		func(payload hookstage.BidderRequestPayload) (hookstage.BidderRequestPayload, error) {
			site := ptrutil.Clone(payload.Request.Site)
			site.Name = "test"
			payload.Request.Site = site
			return payload, nil
		}, hookstage.MutationUpdate, "bidRequest", "site.name",
	).AddMutation(
		func(payload hookstage.BidderRequestPayload) (hookstage.BidderRequestPayload, error) {
			site := ptrutil.Clone(payload.Request.Site)
			site.Domain = "test.com"
			payload.Request.Site = site
			return payload, nil
		}, hookstage.MutationUpdate, "bidRequest", "site.domain",
	)

	mctx.ModuleContext = map[string]interface{}{"some-ctx": "some-ctx"}

	return hookstage.HookResult[hookstage.BidderRequestPayload]{ChangeSet: c, ModuleContext: mctx.ModuleContext}, nil
}

func TestNilAuctionRequest(t *testing.T) {
	ex := &exchange{}
	response, err := ex.HoldAuction(context.Background(), nil, &DebugLog{})
	assert.Nil(t, response)
	assert.Nil(t, err)
}

func TestSelectNewDuration(t *testing.T) {
	type testInput struct {
		dur       int
		durRanges []int
	}
	type testOutput struct {
		dur int
		err error
	}
	testCases := []struct {
		desc     string
		in       testInput
		expected testOutput
	}{
		{
			desc: "nil duration range array, don't expect error",
			in: testInput{
				dur:       1,
				durRanges: nil,
			},
			expected: testOutput{1, nil},
		},
		{
			desc: "empty duration range array, don't expect error",
			in: testInput{
				dur:       1,
				durRanges: []int{},
			},
			expected: testOutput{1, nil},
		},
		{
			desc: "all duration range array elements less than duration, expect error",
			in: testInput{
				dur:       5,
				durRanges: []int{-1, 0, 1, 2, 3},
			},
			expected: testOutput{5, errors.New("bid duration exceeds maximum allowed")},
		},
		{
			desc: "all duration range array elements greater than duration, expect smallest element in durRanges and nil error",
			in: testInput{
				dur:       5,
				durRanges: []int{9, math.MaxInt32, 8},
			},
			expected: testOutput{8, nil},
		},
		{
			desc: "some array elements greater than duration, expect the value greater than dur that is closest in value.",
			in: testInput{
				dur:       5,
				durRanges: []int{math.MaxInt32, -3, 7, 2},
			},
			expected: testOutput{7, nil},
		},
		{
			desc: "an entry in the duration range array is equal to duration, expect its value in return.",
			in: testInput{
				dur:       5,
				durRanges: []int{-3, math.MaxInt32, 5, 7},
			},
			expected: testOutput{5, nil},
		},
	}
	for _, tc := range testCases {
		newDur, err := findDurationRange(tc.in.dur, tc.in.durRanges)

		assert.Equal(t, tc.expected.dur, newDur, tc.desc)
		assert.Equal(t, tc.expected.err, err, tc.desc)
	}
}

func TestSetSeatNonBid(t *testing.T) {
	type args struct {
		bidResponseExt *openrtb_ext.ExtBidResponse
		seatNonBids    SeatNonBidBuilder
	}
	tests := []struct {
		name string
		args args
		want *openrtb_ext.ExtBidResponse
	}{
		{
			name: "empty-seatNonBidsMap",
			args: args{seatNonBids: SeatNonBidBuilder{}, bidResponseExt: nil},
			want: nil,
		},
		{
			name: "nil-bidResponseExt",
			args: args{seatNonBids: SeatNonBidBuilder{"key": nil}, bidResponseExt: nil},
			want: &openrtb_ext.ExtBidResponse{
				Prebid: &openrtb_ext.ExtResponsePrebid{
					SeatNonBid: []openrtb_ext.SeatNonBid{{
						Seat: "key",
					}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := setSeatNonBid(tt.args.bidResponseExt, tt.args.seatNonBids); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("setSeatNonBid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildMultiBidMap(t *testing.T) {
	type testCase struct {
		desc     string
		inPrebid *openrtb_ext.ExtRequestPrebid
		expected map[string]openrtb_ext.ExtMultiBid
	}
	testGroups := []struct {
		groupDesc string
		tests     []testCase
	}{
		{
			groupDesc: "Nil or empty tests",
			tests: []testCase{
				{
					desc:     "prebid nil, expect nil map",
					inPrebid: nil,
					expected: nil,
				},
				{
					desc:     "prebid.MultiBid nil, expect nil map",
					inPrebid: &openrtb_ext.ExtRequestPrebid{},
					expected: nil,
				},
				{
					desc: "not-nil prebid.MultiBid is empty, expect empty map",
					inPrebid: &openrtb_ext.ExtRequestPrebid{
						MultiBid: []*openrtb_ext.ExtMultiBid{},
					},
					expected: map[string]openrtb_ext.ExtMultiBid{},
				},
			},
		},
		{
			groupDesc: "prebid.MultiBid.Bidder tests",
			tests: []testCase{
				{
					desc: "Lowercase prebid.MultiBid.Bidder is found in the BidderName list, entry is mapped",
					inPrebid: &openrtb_ext.ExtRequestPrebid{
						MultiBid: []*openrtb_ext.ExtMultiBid{
							{Bidder: "appnexus"},
						},
					},
					expected: map[string]openrtb_ext.ExtMultiBid{
						"appnexus": {Bidder: "appnexus"},
					},
				},
				{
					desc: "Uppercase prebid.MultiBid.Bidder is found in the BidderName list, entry is mapped",
					inPrebid: &openrtb_ext.ExtRequestPrebid{
						MultiBid: []*openrtb_ext.ExtMultiBid{
							{Bidder: "APPNEXUS"},
						},
					},
					expected: map[string]openrtb_ext.ExtMultiBid{
						"appnexus": {Bidder: "APPNEXUS"},
					},
				},
				{
					desc: "Lowercase prebid.MultiBid.Bidder is not found in the BidderName list, expect empty map",
					inPrebid: &openrtb_ext.ExtRequestPrebid{
						MultiBid: []*openrtb_ext.ExtMultiBid{
							{Bidder: "unknown"},
						},
					},
					expected: map[string]openrtb_ext.ExtMultiBid{},
				},
				{
					desc: "Mixed-case prebid.MultiBid.Bidder is not found in the BidderName list, expect empty map",
					inPrebid: &openrtb_ext.ExtRequestPrebid{
						MultiBid: []*openrtb_ext.ExtMultiBid{
							{Bidder: "UnknownBidder"},
						},
					},
					expected: map[string]openrtb_ext.ExtMultiBid{},
				},
				{
					desc: "Different-cased prebid.MultiBid.Bidder entries that refer to the same adapter are found in the BidderName list are mapped once",
					inPrebid: &openrtb_ext.ExtRequestPrebid{
						MultiBid: []*openrtb_ext.ExtMultiBid{
							{Bidder: "AppNexus"},
							{Bidder: "appnexus"},
						},
					},
					expected: map[string]openrtb_ext.ExtMultiBid{
						"appnexus": {Bidder: "appnexus"},
					},
				},
			},
		},
		{
			groupDesc: "prebid.MultiBid.Bidders tests",
			tests: []testCase{
				{
					desc: "Lowercase prebid.MultiBid.Bidder is found in the BidderName list, entry is mapped",
					inPrebid: &openrtb_ext.ExtRequestPrebid{
						MultiBid: []*openrtb_ext.ExtMultiBid{
							{Bidders: []string{"appnexus"}},
						},
					},
					expected: map[string]openrtb_ext.ExtMultiBid{
						"appnexus": {
							Bidders: []string{"appnexus"},
						},
					},
				},
				{
					desc: "Lowercase prebid.MultiBid.Bidder is not found in the BidderName list, expect empty map",
					inPrebid: &openrtb_ext.ExtRequestPrebid{
						MultiBid: []*openrtb_ext.ExtMultiBid{
							{Bidders: []string{"unknown"}},
						},
					},
					expected: map[string]openrtb_ext.ExtMultiBid{},
				},
				{
					desc: "Mixed-case prebid.MultiBid.Bidder is not found in the BidderName list, expect empty map",
					inPrebid: &openrtb_ext.ExtRequestPrebid{
						MultiBid: []*openrtb_ext.ExtMultiBid{
							{Bidders: []string{"UnknownBidder"}},
						},
					},
					expected: map[string]openrtb_ext.ExtMultiBid{},
				},
				{
					desc: "Different-cased prebid.MultiBid.Bidder entries that refer to the same adapter are found in the BidderName list are mapped once",
					inPrebid: &openrtb_ext.ExtRequestPrebid{
						MultiBid: []*openrtb_ext.ExtMultiBid{
							{Bidders: []string{"AppNexus", "appnexus", "UnknownBidder"}},
						},
					},
					expected: map[string]openrtb_ext.ExtMultiBid{
						"appnexus": {
							Bidders: []string{"AppNexus", "appnexus", "UnknownBidder"},
						},
					},
				},
			},
		},
		{
			groupDesc: "prebid.MultiBid.Bidder and prebid.MultiBid.Bidders entries in tests",
			tests: []testCase{
				{
					desc: "prebid.MultiBid.Bidder found, ignore entries in prebid.MultiBid.Bidders, even if its unknown",
					inPrebid: &openrtb_ext.ExtRequestPrebid{
						MultiBid: []*openrtb_ext.ExtMultiBid{
							{
								Bidder:  "UnknownBidder",
								Bidders: []string{"appnexus", "rubicon", "pubmatic"},
							},
						},
					},
					expected: map[string]openrtb_ext.ExtMultiBid{},
				},
				{
					desc: "prebid.MultiBid.Bidder found in one entry, prebid.MultiBid.Bidders in another. Add all to map",
					inPrebid: &openrtb_ext.ExtRequestPrebid{
						MultiBid: []*openrtb_ext.ExtMultiBid{
							{
								Bidder:  "pubmatic",
								Bidders: []string{"appnexus", "rubicon", "UnknownBidder"},
							},
							{
								Bidders: []string{"UnknownBidder", "appnexus", "rubicon"},
							},
						},
					},
					expected: map[string]openrtb_ext.ExtMultiBid{
						"pubmatic": {
							Bidder:  "pubmatic",
							Bidders: []string{"appnexus", "rubicon", "UnknownBidder"},
						},
						"appnexus": {
							Bidders: []string{"UnknownBidder", "appnexus", "rubicon"},
						},
						"rubicon": {
							Bidders: []string{"UnknownBidder", "appnexus", "rubicon"},
						},
					},
				},
			},
		},
	}
	for _, group := range testGroups {
		for _, tc := range group.tests {
			t.Run(group.groupDesc+tc.desc, func(t *testing.T) {
				multiBidMap := buildMultiBidMap(tc.inPrebid)
				assert.Equal(t, tc.expected, multiBidMap, tc.desc)
			})
		}
	}
}

func TestBidsToUpdate(t *testing.T) {
	type testInput struct {
		multiBid map[string]openrtb_ext.ExtMultiBid
		bidder   string
	}
	testCases := []struct {
		desc     string
		in       testInput
		expected int
	}{
		{
			desc:     "Empty multibid map. Expect openrtb_ext.DefaultBidLimit",
			in:       testInput{},
			expected: openrtb_ext.DefaultBidLimit,
		},
		{
			desc: "Empty bidder. Expect openrtb_ext.DefaultBidLimit",
			in: testInput{
				multiBid: map[string]openrtb_ext.ExtMultiBid{
					"appnexus": {
						Bidder:  "appnexus",
						MaxBids: ptrutil.ToPtr(2),
					},
				},
			},
			expected: openrtb_ext.DefaultBidLimit,
		},
		{
			desc: "bidder finds a match in multibid map but TargetBidderCodePrefix is empty. Expect openrtb_ext.DefaultBidLimit",
			in: testInput{
				multiBid: map[string]openrtb_ext.ExtMultiBid{
					"appnexus": {
						Bidder:  "appnexus",
						MaxBids: ptrutil.ToPtr(2),
					},
				},
				bidder: "appnexus",
			},
			expected: openrtb_ext.DefaultBidLimit,
		},
		{
			desc: "multibid element with non-empty TargetBidderCodePrefix matches bidder. Expect MaxBids value",
			in: testInput{
				multiBid: map[string]openrtb_ext.ExtMultiBid{
					"appnexus": {
						Bidder:                 "appnexus",
						MaxBids:                ptrutil.ToPtr(2),
						TargetBidderCodePrefix: "aPrefix",
					},
				},
				bidder: "appnexus",
			},
			expected: 2,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.expected, bidsToUpdate(tc.in.multiBid, tc.in.bidder), tc.desc)
		})
	}
}

func TestGetBidderPreferredMediaType(t *testing.T) {
	tests := []struct {
		name                string
		prebid              *openrtb_ext.ExtRequestPrebid
		account             *config.Account
		liveAdapters        []openrtb_ext.BidderName
		singleFormatBidders map[openrtb_ext.BidderName]struct{}
		expected            openrtb_ext.PreferredMediaType
	}{
		{
			name:         "Nil account and request preferred media type",
			prebid:       nil,
			account:      &config.Account{},
			liveAdapters: []openrtb_ext.BidderName{"bidderA"},
			singleFormatBidders: map[openrtb_ext.BidderName]struct{}{
				"bidderA": {},
			},
			expected: openrtb_ext.PreferredMediaType{},
		},
		{
			name:   "Account preferred media type only",
			prebid: nil,
			account: &config.Account{
				PreferredMediaType: map[openrtb_ext.BidderName]openrtb_ext.BidType{
					"bidderA": openrtb_ext.BidTypeBanner,
				},
			},
			liveAdapters: []openrtb_ext.BidderName{"bidderA"},
			singleFormatBidders: map[openrtb_ext.BidderName]struct{}{
				"bidderA": {},
			},
			expected: openrtb_ext.PreferredMediaType{
				"bidderA": openrtb_ext.BidTypeBanner,
			},
		},
		{
			name: "Request preferred media type only",
			prebid: &openrtb_ext.ExtRequestPrebid{
				BidderControls: map[openrtb_ext.BidderName]openrtb_ext.BidderControl{
					"bidderB": {PreferredMediaType: openrtb_ext.BidTypeVideo},
				},
			},
			account:      &config.Account{},
			liveAdapters: []openrtb_ext.BidderName{"bidderB"},
			singleFormatBidders: map[openrtb_ext.BidderName]struct{}{
				"bidderB": {},
			},
			expected: openrtb_ext.PreferredMediaType{
				"bidderB": openrtb_ext.BidTypeVideo,
			},
		},
		{
			name: "Account and request preferred media type",
			prebid: &openrtb_ext.ExtRequestPrebid{
				BidderControls: map[openrtb_ext.BidderName]openrtb_ext.BidderControl{
					"bidderB": {PreferredMediaType: openrtb_ext.BidTypeVideo},
				},
			},
			account: &config.Account{
				PreferredMediaType: map[openrtb_ext.BidderName]openrtb_ext.BidType{
					"bidderA": openrtb_ext.BidTypeBanner,
				},
			},
			liveAdapters: []openrtb_ext.BidderName{"bidderA", "bidderB", "bidderC"},
			singleFormatBidders: map[openrtb_ext.BidderName]struct{}{
				"bidderA": {},
				"bidderB": {},
			},
			expected: openrtb_ext.PreferredMediaType{
				"bidderA": openrtb_ext.BidTypeBanner,
				"bidderB": openrtb_ext.BidTypeVideo,
			},
		},
		{
			name: "Request overrides account preferred media type",
			prebid: &openrtb_ext.ExtRequestPrebid{
				BidderControls: map[openrtb_ext.BidderName]openrtb_ext.BidderControl{
					"bidderA": {PreferredMediaType: openrtb_ext.BidTypeVideo},
				},
			},
			account: &config.Account{
				PreferredMediaType: map[openrtb_ext.BidderName]openrtb_ext.BidType{
					"bidderA": openrtb_ext.BidTypeBanner,
				},
			},
			liveAdapters: []openrtb_ext.BidderName{"bidderA"},
			singleFormatBidders: map[openrtb_ext.BidderName]struct{}{
				"bidderA": {},
			},
			expected: openrtb_ext.PreferredMediaType{
				"bidderA": openrtb_ext.BidTypeVideo,
			},
		},
		{
			name: "Bidder not in singleFormatBidders",
			prebid: &openrtb_ext.ExtRequestPrebid{
				BidderControls: map[openrtb_ext.BidderName]openrtb_ext.BidderControl{
					"bidderA": {PreferredMediaType: openrtb_ext.BidTypeVideo},
				},
			},
			account: &config.Account{
				PreferredMediaType: map[openrtb_ext.BidderName]openrtb_ext.BidType{
					"bidderA": openrtb_ext.BidTypeBanner,
				},
			},
			liveAdapters: []openrtb_ext.BidderName{"bidderA"},
			singleFormatBidders: map[openrtb_ext.BidderName]struct{}{
				"bidderB": {},
			},
			expected: openrtb_ext.PreferredMediaType{},
		},
		{
			name: "No bidders in singleFormatBidders",
			prebid: &openrtb_ext.ExtRequestPrebid{
				BidderControls: map[openrtb_ext.BidderName]openrtb_ext.BidderControl{
					"bidderA": {PreferredMediaType: openrtb_ext.BidTypeVideo},
				},
			},
			account: &config.Account{
				PreferredMediaType: map[openrtb_ext.BidderName]openrtb_ext.BidType{
					"bidderA": openrtb_ext.BidTypeBanner,
				},
			},
			liveAdapters:        []openrtb_ext.BidderName{"bidderA"},
			singleFormatBidders: make(map[openrtb_ext.BidderName]struct{}),
			expected:            nil,
		},
		{
			name: "Different bidders in singleFormatBidders than liveAdapters",
			prebid: &openrtb_ext.ExtRequestPrebid{
				BidderControls: map[openrtb_ext.BidderName]openrtb_ext.BidderControl{
					"bidderA": {PreferredMediaType: openrtb_ext.BidTypeVideo},
				},
			},
			account: &config.Account{
				PreferredMediaType: map[openrtb_ext.BidderName]openrtb_ext.BidType{
					"bidderC": openrtb_ext.BidTypeBanner,
					"bidderD": openrtb_ext.BidTypeVideo,
				},
			},
			liveAdapters: []openrtb_ext.BidderName{"bidderA", "bidderB"},
			singleFormatBidders: map[openrtb_ext.BidderName]struct{}{
				"bidderC": {},
				"bidderD": {},
			},
			expected: openrtb_ext.PreferredMediaType{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBidderPreferredMediaTypeMap(tt.prebid, tt.account, tt.liveAdapters, tt.singleFormatBidders)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsEEACountry(t *testing.T) {
	eeaCountries := []string{"FRA", "DEU", "ITA", "ESP", "NLD"}

	tests := []struct {
		name     string
		country  string
		eeaList  []string
		expected bool
	}{
		{
			name:     "Country_in_EEA",
			country:  "FRA",
			eeaList:  eeaCountries,
			expected: true,
		},
		{
			name:     "Country_in_EEA_lowercase",
			country:  "fra",
			eeaList:  eeaCountries,
			expected: true,
		},
		{
			name:     "Country_not_in_EEA",
			country:  "USA",
			eeaList:  eeaCountries,
			expected: false,
		},
		{
			name:     "Empty_country_string",
			country:  "",
			eeaList:  eeaCountries,
			expected: false,
		},
		{
			name:     "EEA_list_is_empty",
			country:  "FRA",
			eeaList:  []string{},
			expected: false,
		},
		{
			name:     "EEA_list_is_nil",
			country:  "FRA",
			eeaList:  nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEEACountry(tt.country, tt.eeaList)
			assert.Equal(t, tt.expected, result)
		})
	}
}

type mockRequestValidator struct {
	errors []error
}

func (mrv *mockRequestValidator) ValidateImp(imp *openrtb_ext.ImpWrapper, cfg ortb.ValidationConfig, index int, aliases map[string]string, hasStoredResponses bool, storedBidResponses stored_responses.ImpBidderStoredResp) []error {
	return mrv.errors
}
