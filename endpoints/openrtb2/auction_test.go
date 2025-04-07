package openrtb2

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	jsoniter "github.com/json-iterator/go"
	"github.com/julienschmidt/httprouter"
	gpplib "github.com/prebid/go-gpp"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/analytics"
	analyticsBuild "github.com/prebid/prebid-server/v3/analytics/build"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/exchange"
	"github.com/prebid/prebid-server/v3/hooks"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/metrics"
	metricsConfig "github.com/prebid/prebid-server/v3/metrics/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/ortb"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/v3/stored_responses"
	"github.com/prebid/prebid-server/v3/util/iputil"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

const jsonFileExtension string = ".json"

func TestMain(m *testing.M) {
	jsoniter.RegisterExtension(&jsonutil.RawMessageExtension{})
	os.Exit(m.Run())
}

func TestJsonSampleRequests(t *testing.T) {
	testSuites := []struct {
		description          string
		sampleRequestsSubDir string
	}{
		{
			"Assert 200s on all bidRequests from exemplary folder",
			"valid-whole/exemplary",
		},
		{
			"Asserts we return 200s on well-formed Native requests.",
			"valid-native",
		},
		{
			"Asserts we return 400s on requests that are not supposed to pass validation",
			"invalid-whole",
		},
		{
			"Asserts we return 400s on requests with Native requests that don't pass validation",
			"invalid-native",
		},
		{
			"Makes sure we handle aliased bidders properly",
			"aliased",
		},
		{
			"Makes sure we handle alternate bidder codes properly",
			"alternate-bidder-code",
		},
		{
			"Asserts we return 500s on requests referencing accounts with malformed configs.",
			"account-malformed",
		},
		{
			"Asserts we return 503s on requests with blocked apps.",
			"blocked",
		},
		{
			"Assert that requests that come with no user id nor app id return error if the `AccountRequired` field in the `config.Configuration` structure is set to true",
			"account-required/no-account",
		},
		{
			"Assert requests that come with a valid user id or app id when account is required",
			"account-required/with-account",
		},
		{
			"Tests diagnostic messages for invalid stored requests",
			"invalid-stored",
		},
		{
			"Make sure requests with disabled bidders will fail",
			"disabled/bad",
		},
		{
			"There are both disabled and non-disabled bidders, we expect a 200",
			"disabled/good",
		},
		{
			"Assert we correctly use the server conversion rates when needed",
			"currency-conversion/server-rates/valid",
		},
		{
			"Assert we correctly throw an error when no conversion rate was found in the server conversions map",
			"currency-conversion/server-rates/errors",
		},
		{
			"Assert we correctly use request-defined custom currency rates when present in root.ext",
			"currency-conversion/custom-rates/valid",
		},
		{
			"Assert we correctly validate request-defined custom currency rates when present in root.ext",
			"currency-conversion/custom-rates/errors",
		},
		{
			"Assert request with ad server targeting is processing correctly",
			"adservertargeting",
		},
		{
			"Assert request with bid adjustments defined is processing correctly",
			"bidadjustments",
		},
	}

	for _, tc := range testSuites {
		err := filepath.WalkDir(filepath.Join("sample-requests", tc.sampleRequestsSubDir), func(path string, info fs.DirEntry, err error) error {
			// According to documentation, needed to avoid panics
			if err != nil {
				return err
			}

			// Test suite will traverse the directory tree recursively and will only consider files with `json` extension
			if !info.IsDir() && filepath.Ext(info.Name()) == jsonFileExtension {
				t.Run(tc.description, func(t *testing.T) {
					runJsonBasedTest(t, path, tc.description)
				})
			}

			return nil
		})
		assert.NoError(t, err, "Test case %s. Error reading files from directory %s \n", tc.description, tc.sampleRequestsSubDir)
	}
}

func runJsonBasedTest(t *testing.T, filename, desc string) {
	t.Helper()

	fileData, err := os.ReadFile(filename)
	if !assert.NoError(t, err, "Test case %s. Error reading file %s \n", desc, filename) {
		return
	}

	// Retrieve test case input and expected output from JSON file
	test, err := parseTestData(fileData, filename)
	if !assert.NoError(t, err) {
		return
	}

	// Build endpoint for testing. If no error, run test case
	cfg := &config.Configuration{MaxRequestSize: maxSize}
	if test.Config != nil {
		cfg.BlockedApps = test.Config.BlockedApps
		cfg.BlockedAppsLookup = test.Config.getBlockedAppLookup()
		cfg.AccountRequired = test.Config.AccountRequired
		cfg.AccountDefaults.PreferredMediaType = test.Config.PreferredMediaType
	}
	cfg.MarshalAccountDefaults()
	test.endpointType = OPENRTB_ENDPOINT

	auctionEndpointHandler, ex, mockBidServers, mockCurrencyRatesServer, err := buildTestEndpoint(test, cfg)
	if assert.NoError(t, err) {
		assert.NotPanics(t, func() { runEndToEndTest(t, auctionEndpointHandler, test, fileData, filename) }, filename)
	}

	if test.ExpectedValidatedBidReq != nil {
		// compare as json to ignore whitespace and ext field ordering
		actualJson, err := jsonutil.Marshal(ex.actualValidatedBidReq)
		if assert.NoError(t, err, "Error converting actual bid request to json. Test file: %s", filename) {
			assert.JSONEq(t, string(test.ExpectedValidatedBidReq), string(actualJson), "Not the expected validated request. Test file: %s", filename)
		}
	}
	if test.ExpectedMockBidderRequests != nil {
		for bidder, req := range test.ExpectedMockBidderRequests {
			a, ok := ex.adapters[openrtb_ext.BidderName(bidder)]
			if !ok {
				t.Fatalf("Unexpected bidder %s has an expected mock bidder request. Test file: %s", bidder, filename)
			}
			aa := a.(*exchange.BidderAdapter)
			ma := aa.Bidder.(*adapters.InfoAwareBidder).Bidder.(*mockAdapter)
			assert.JSONEq(t, string(req), string(ma.requestData[0]), "Not the expected mock bidder request for bidder %s. Test file: %s", bidder, filename)
		}
	}

	// Close servers regardless if the test case was run or not
	for _, mockBidServer := range mockBidServers {
		mockBidServer.Close()
	}
	mockCurrencyRatesServer.Close()
}

func runEndToEndTest(t *testing.T, auctionEndpointHandler httprouter.Handle, test testCase, fileData []byte, testFile string) {
	t.Helper()

	// Hit the auction endpoint with the test case configuration and mockBidRequest
	request := httptest.NewRequest("POST", "/openrtb2/auction", bytes.NewReader(test.BidRequest))
	recorder := httptest.NewRecorder()
	auctionEndpointHandler(recorder, request, nil)

	// Assertions
	actualCode := recorder.Code
	actualJsonBidResponse := recorder.Body.String()
	assert.Equal(t, test.ExpectedReturnCode, actualCode, "Test failed. Filename: %s \n", testFile)

	// Either assert bid response or expected error
	if len(test.ExpectedErrorMessage) > 0 {
		assert.Contains(t, actualJsonBidResponse, test.ExpectedErrorMessage, "Actual: %s \nExpected: %s. Filename: %s \n", actualJsonBidResponse, test.ExpectedErrorMessage, testFile)
	}

	if len(test.ExpectedBidResponse) > 0 {
		var expectedBidResponse openrtb2.BidResponse
		var actualBidResponse openrtb2.BidResponse
		var err error

		err = jsonutil.Unmarshal(test.ExpectedBidResponse, &expectedBidResponse)
		if assert.NoError(t, err, "Could not unmarshal expected bidResponse taken from test file.\n Test file: %s\n Error:%s\n", testFile, err) {
			err = jsonutil.UnmarshalValid([]byte(actualJsonBidResponse), &actualBidResponse)
			if assert.NoError(t, err, "Could not unmarshal actual bidResponse from auction.\n Test file: %s\n Error:%s\n actualJsonBidResponse: %s", testFile, err, actualJsonBidResponse) {
				assertBidResponseEqual(t, testFile, expectedBidResponse, actualBidResponse)
			}
		}
	}
}

func compareWarnings(t *testing.T, expectedBidResponseExt, actualBidResponseExt []byte, warnPath string) {
	expectedWarnings, _, _, err := jsonparser.Get(expectedBidResponseExt, warnPath)
	if err != nil && err != jsonparser.KeyPathNotFoundError {
		assert.Fail(t, "error getting data from response extension")
	}
	if len(expectedWarnings) > 0 {
		actualWarnings, _, _, err := jsonparser.Get(actualBidResponseExt, warnPath)
		if err != nil && err != jsonparser.KeyPathNotFoundError {
			assert.Fail(t, "error getting data from response extension")
		}

		var expectedWarn []openrtb_ext.ExtBidderMessage
		err = jsonutil.UnmarshalValid(expectedWarnings, &expectedWarn)
		if err != nil {
			assert.Fail(t, "error unmarshalling expected warnings data from response extension")
		}

		var actualWarn []openrtb_ext.ExtBidderMessage
		err = jsonutil.UnmarshalValid(actualWarnings, &actualWarn)
		if err != nil {
			assert.Fail(t, "error unmarshalling actual warnings data from response extension")
		}

		// warnings from different bidders may be returned in different order.
		assert.Equal(t, len(expectedWarn), len(actualWarn), "incorrect warnings number")
		for i, expWarn := range expectedWarn {
			actualWarning := actualWarn[i]
			assert.Contains(t, actualWarning.Message, expWarn.Message, "incorrect warning")
		}
	}
}

// Once unmarshalled, bidResponse objects can't simply be compared with an `assert.Equalf()` call
// because tests fail if the elements inside the `bidResponse.SeatBid` and `bidResponse.SeatBid.Bid`
// arrays, if any, are not listed in the exact same order in the actual version and in the expected version.
func assertBidResponseEqual(t *testing.T, testFile string, expectedBidResponse openrtb2.BidResponse, actualBidResponse openrtb2.BidResponse) {

	//Assert non-array BidResponse fields
	assert.Equalf(t, expectedBidResponse.ID, actualBidResponse.ID, "BidResponse.ID doesn't match expected. Test: %s\n", testFile)
	assert.Equalf(t, expectedBidResponse.Cur, actualBidResponse.Cur, "BidResponse.Cur doesn't match expected. Test: %s\n", testFile)

	if len(expectedBidResponse.Ext) > 0 {
		compareWarnings(t, expectedBidResponse.Ext, actualBidResponse.Ext, "warnings.general")
	}

	//Assert []SeatBid and their Bid elements independently of their order
	assert.Len(t, actualBidResponse.SeatBid, len(expectedBidResponse.SeatBid), "BidResponse.SeatBid is expected to contain %d elements but contains %d. Test: %s\n", len(expectedBidResponse.SeatBid), len(actualBidResponse.SeatBid), testFile)

	//Given that bidResponses have the same length, compare them in an order-independent way using maps
	var actualSeatBidsMap map[string]openrtb2.SeatBid = make(map[string]openrtb2.SeatBid, 0)
	for _, seatBid := range actualBidResponse.SeatBid {
		actualSeatBidsMap[seatBid.Seat] = seatBid
	}

	var expectedSeatBidsMap map[string]openrtb2.SeatBid = make(map[string]openrtb2.SeatBid, 0)
	for _, seatBid := range expectedBidResponse.SeatBid {
		expectedSeatBidsMap[seatBid.Seat] = seatBid
	}

	for bidderName, expectedSeatBid := range expectedSeatBidsMap {
		if !assert.Contains(t, actualSeatBidsMap, bidderName, "BidResponse.SeatBid[%s] was not found as expected. Test: %s\n", bidderName, testFile) {
			continue
		}

		//Assert non-array SeatBid fields
		assert.Equalf(t, expectedSeatBid.Seat, actualSeatBidsMap[bidderName].Seat, "actualSeatBidsMap[%s].Seat doesn't match expected. Test: %s\n", bidderName, testFile)
		assert.Equalf(t, expectedSeatBid.Group, actualSeatBidsMap[bidderName].Group, "actualSeatBidsMap[%s].Group doesn't match expected. Test: %s\n", bidderName, testFile)
		assert.Equalf(t, expectedSeatBid.Ext, actualSeatBidsMap[bidderName].Ext, "actualSeatBidsMap[%s].Ext doesn't match expected. Test: %s\n", bidderName, testFile)

		// Assert Bid arrays
		assert.Len(t, actualSeatBidsMap[bidderName].Bid, len(expectedSeatBid.Bid), "BidResponse.SeatBid[].Bid array is expected to contain %d elements but has %d. Test: %s\n", len(expectedSeatBid.Bid), len(actualSeatBidsMap[bidderName].Bid), testFile)
		// Given that actualSeatBidsMap[bidderName].Bid and expectedSeatBid.Bid have the same length, compare them in an order-independent way using maps
		var expectedBidMap map[string]openrtb2.Bid = make(map[string]openrtb2.Bid, 0)
		for _, bid := range expectedSeatBid.Bid {
			expectedBidMap[bid.ID] = bid
		}

		var actualBidMap map[string]openrtb2.Bid = make(map[string]openrtb2.Bid, 0)
		for _, bid := range actualSeatBidsMap[bidderName].Bid {
			actualBidMap[bid.ID] = bid
		}

		for bidID, expectedBid := range expectedBidMap {
			if !assert.Contains(t, actualBidMap, bidID, "BidResponse.SeatBid[%s].Bid[%s].ID doesn't match expected. Test: %s\n", bidderName, bidID, testFile) {
				continue
			}
			assert.Equalf(t, expectedBid.ImpID, actualBidMap[bidID].ImpID, "BidResponse.SeatBid[%s].Bid[%s].ImpID doesn't match expected. Test: %s\n", bidderName, bidID, testFile)
			assert.Equalf(t, expectedBid.Price, actualBidMap[bidID].Price, "BidResponse.SeatBid[%s].Bid[%s].Price doesn't match expected. Test: %s\n", bidderName, bidID, testFile)

			if len(expectedBid.Ext) > 0 {
				assert.JSONEq(t, string(expectedBid.Ext), string(actualBidMap[bidID].Ext), "Incorrect bid extension")
			}
		}
	}
}

func TestBidRequestAssert(t *testing.T) {
	appnexusB1 := openrtb2.Bid{ID: "appnexus-bid-1", Price: 5.00}
	appnexusB2 := openrtb2.Bid{ID: "appnexus-bid-2", Price: 7.00}
	rubiconB1 := openrtb2.Bid{ID: "rubicon-bid-1", Price: 1.50}
	rubiconB2 := openrtb2.Bid{ID: "rubicon-bid-2", Price: 4.00}

	sampleSeatBids := []openrtb2.SeatBid{
		{
			Seat: "appnexus-bids",
			Bid:  []openrtb2.Bid{appnexusB1, appnexusB2},
		},
		{
			Seat: "rubicon-bids",
			Bid:  []openrtb2.Bid{rubiconB1, rubiconB2},
		},
	}

	testSuites := []struct {
		description         string
		expectedBidResponse openrtb2.BidResponse
		actualBidResponse   openrtb2.BidResponse
	}{
		{
			"identical SeatBids, exact same SeatBid and Bid arrays order",
			openrtb2.BidResponse{ID: "anId", BidID: "bidId", SeatBid: sampleSeatBids},
			openrtb2.BidResponse{ID: "anId", BidID: "bidId", SeatBid: sampleSeatBids},
		},
		{
			"identical SeatBids but Seatbid array elements come in different order",
			openrtb2.BidResponse{ID: "anId", BidID: "bidId", SeatBid: sampleSeatBids},
			openrtb2.BidResponse{ID: "anId", BidID: "bidId",
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: "rubicon-bids",
						Bid:  []openrtb2.Bid{rubiconB1, rubiconB2},
					},
					{
						Seat: "appnexus-bids",
						Bid:  []openrtb2.Bid{appnexusB1, appnexusB2},
					},
				},
			},
		},
		{
			"SeatBids seem to be identical except for the different order of Bid array elements in one of them",
			openrtb2.BidResponse{ID: "anId", BidID: "bidId", SeatBid: sampleSeatBids},
			openrtb2.BidResponse{ID: "anId", BidID: "bidId",
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: "appnexus-bids",
						Bid:  []openrtb2.Bid{appnexusB2, appnexusB1},
					},
					{
						Seat: "rubicon-bids",
						Bid:  []openrtb2.Bid{rubiconB1, rubiconB2},
					},
				},
			},
		},
		{
			"Both SeatBid elements and bid elements come in different order",
			openrtb2.BidResponse{ID: "anId", BidID: "bidId", SeatBid: sampleSeatBids},
			openrtb2.BidResponse{ID: "anId", BidID: "bidId",
				SeatBid: []openrtb2.SeatBid{
					{
						Seat: "rubicon-bids",
						Bid:  []openrtb2.Bid{rubiconB2, rubiconB1},
					},
					{
						Seat: "appnexus-bids",
						Bid:  []openrtb2.Bid{appnexusB2, appnexusB1},
					},
				},
			},
		},
	}

	for _, test := range testSuites {
		assertBidResponseEqual(t, test.description, test.expectedBidResponse, test.actualBidResponse)
	}
}

// TestExplicitUserId makes sure that the cookie's ID doesn't override an explicit value sent in the request.
func TestExplicitUserId(t *testing.T) {
	cookieName := "userid"
	mockId := "12345"
	cfg := &config.Configuration{
		MaxRequestSize: maxSize,
		HostCookie: config.HostCookie{
			CookieName: cookieName,
		},
	}
	ex := &mockExchange{}

	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(`{
"id": "some-request-id",
		"site": {
			"page": "test.somepage.com"
		},
		"user": {
			"id": "explicit"
		},
		"imp": [
			{
				"id": "my-imp-id",
				"banner": {
					"format": [
						{
							"w": 300,
							"h": 600
						}
					]
				},
				"pmp": {
					"deals": [
						{
							"id": "some-deal-id"
						}
					]
				},
				"ext": {
					"appnexus": {
						"placementId": 12883451
					}
				}
			}
		]
	}`))
	request.AddCookie(&http.Cookie{
		Name:  cookieName,
		Value: mockId,
	})

	endpoint, _ := NewEndpoint(
		fakeUUIDGenerator{},
		ex,
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		cfg,
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
	)

	endpoint(httptest.NewRecorder(), request, nil)

	if ex.lastRequest == nil {
		t.Fatalf("The request never made it into the Exchange.")
	}

	if ex.lastRequest.User == nil {
		t.Fatalf("The exchange should have received a request with a non-nil user.")
	}

	if ex.lastRequest.User.ID != "explicit" {
		t.Errorf("Bad User ID. Expected explicit, got %s", ex.lastRequest.User.ID)
	}
}

// TestBadAliasRequests() reuses two requests that would fail anyway.  Here, we
// take advantage of our knowledge that processStoredRequests() in auction.go
// processes aliases before it processes stored imps.  Changing that order
// would probably cause this test to fail.
func TestBadAliasRequests(t *testing.T) {
	doBadAliasRequest(t, "sample-requests/invalid-stored/bad_stored_imp.json", "Invalid request: Invalid JSON Document\n")
	doBadAliasRequest(t, "sample-requests/invalid-stored/bad_incoming_imp.json", "Invalid request: Invalid JSON Document\n")
}

// doBadAliasRequest() is a customized variation of doRequest(), above
func doBadAliasRequest(t *testing.T, filename string, expectMsg string) {
	t.Helper()
	fileData := readFile(t, filename)
	testBidRequest, _, _, err := jsonparser.Get(fileData, "mockBidRequest")
	assert.NoError(t, err, "Error jsonparsing root.mockBidRequest from file %s. Desc: %v.", filename, err)

	// aliasJSON lacks a comma after the "appnexus" entry so is bad JSON
	aliasJSON := []byte(`{"ext":{"prebid":{"aliases": {"test1": "appnexus" "test2": "rubicon", "test3": "openx"}}}}`)

	bidderInfos := getBidderInfos(nil, openrtb_ext.CoreBidderNames())

	bidderMap := exchange.GetActiveBidders(bidderInfos)
	disabledBidders := exchange.GetDisabledBidderWarningMessages(bidderInfos)

	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	endpoint, _ := NewEndpoint(
		fakeUUIDGenerator{},
		&nobidExchange{},
		ortb.NewRequestValidator(bidderMap, disabledBidders, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		disabledBidders,
		aliasJSON,
		bidderMap,
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
	)

	request := httptest.NewRequest("POST", "/openrtb2/auction", bytes.NewReader(testBidRequest))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	assertResponseCode(t, filename, recorder.Code, http.StatusBadRequest, recorder.Body.String())
	assert.Equal(t, string(expectMsg), recorder.Body.String(), "file %s had bad response body", filename)

}

func newParamsValidator(t *testing.T) openrtb_ext.BidderParamValidator {
	paramValidator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Error creating the param validator: %v", err)
	}
	return paramValidator
}

func assertResponseCode(t *testing.T, filename string, actual int, expected int, msg string) {
	t.Helper()
	if actual != expected {
		t.Errorf("Expected a %d response from %v. Got %d: %s", expected, filename, actual, msg)
	}
}

func getRequestPayload(t *testing.T, example []byte) []byte {
	t.Helper()
	if value, _, _, err := jsonparser.Get(example, "requestPayload"); err != nil {
		t.Fatalf("Error parsing root.requestPayload from request: %v.", err)
	} else {
		return value
	}
	return nil
}

// TestNilExchange makes sure we fail when given nil for the Exchange.
func TestNilExchange(t *testing.T) {
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	_, err := NewEndpoint(
		fakeUUIDGenerator{},
		nil,
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
	)

	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil Exchange.")
	}
}

// TestExchangeError makes sure we return a 500 if the exchange auction fails.
func TestExchangeError(t *testing.T) {
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	endpoint, _ := NewEndpoint(
		fakeUUIDGenerator{},
		&brokenExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
	)

	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d. Got %d. Input was: %s", http.StatusInternalServerError, recorder.Code, validRequest(t, "site.json"))
	}
}

// TestUserAgentSetting makes sure we read the User-Agent header if it wasn't defined on the request.
func TestUserAgentSetting(t *testing.T) {
	httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	httpReq.Header.Set("User-Agent", "foo")
	bidReq := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}}

	setUAImplicitly(httpReq, bidReq)

	if bidReq.Device == nil {
		t.Fatal("bidrequest.device should have been set implicitly.")
	}
	if bidReq.Device.UA != "foo" {
		t.Errorf("bidrequest.device.ua should have been \"foo\". Got %s", bidReq.Device.UA)
	}
}

// TestUserAgentOverride makes sure that the explicit UA from the request takes precedence.
func TestUserAgentOverride(t *testing.T) {
	httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	httpReq.Header.Set("User-Agent", "foo")
	bidReq := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
		Device: &openrtb2.Device{
			UA: "bar",
		},
	}}

	setUAImplicitly(httpReq, bidReq)

	if bidReq.Device.UA != "bar" {
		t.Errorf("bidrequest.device.ua should have been \"bar\". Got %s", bidReq.Device.UA)
	}
}

func TestAuctionTypeDefault(t *testing.T) {
	bidReq := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}}
	setAuctionTypeImplicitly(bidReq)

	if bidReq.AT != 1 {
		t.Errorf("Expected request.at to be 1. Got %d", bidReq.AT)
	}
}

func TestImplicitIPsEndToEnd(t *testing.T) {
	testCases := []struct {
		description         string
		reqJSONFile         string
		xForwardedForHeader string
		privateNetworksIPv4 []net.IPNet
		privateNetworksIPv6 []net.IPNet
		expectedDeviceIPv4  string
		expectedDeviceIPv6  string
	}{
		{
			description:         "IPv4",
			reqJSONFile:         "site.json",
			xForwardedForHeader: "1.1.1.1",
			expectedDeviceIPv4:  "1.1.1.1",
		},
		{
			description:         "IPv6",
			reqJSONFile:         "site.json",
			xForwardedForHeader: "1111::",
			expectedDeviceIPv6:  "1111::",
		},
		{
			description:         "IPv4 - Defined In Request",
			reqJSONFile:         "site-has-ipv4.json",
			xForwardedForHeader: "1.1.1.1",
			expectedDeviceIPv4:  "8.8.8.8", // Hardcoded value in test file.
		},
		{
			description:         "IPv6 - Defined In Request",
			reqJSONFile:         "site-has-ipv6.json",
			xForwardedForHeader: "1111::",
			expectedDeviceIPv6:  "8888::", // Hardcoded value in test file.
		},
		{
			description:         "IPv4 - Defined In Request - Private Network",
			reqJSONFile:         "site-has-ipv4.json",
			xForwardedForHeader: "1.1.1.1",
			privateNetworksIPv4: []net.IPNet{{IP: net.IP{8, 8, 8, 0}, Mask: net.CIDRMask(24, 32)}}, // Hardcoded value in test file.
			expectedDeviceIPv4:  "1.1.1.1",
		},
		{
			description:         "IPv6 - Defined In Request - Private Network",
			reqJSONFile:         "site-has-ipv6.json",
			xForwardedForHeader: "1111::",
			privateNetworksIPv6: []net.IPNet{{IP: net.ParseIP("8800::"), Mask: net.CIDRMask(8, 128)}}, // Hardcoded value in test file.
			expectedDeviceIPv6:  "1111::",
		},
	}

	for _, test := range testCases {
		exchange := &nobidExchange{}
		cfg := &config.Configuration{
			MaxRequestSize: maxSize,
			RequestValidation: config.RequestValidation{
				IPv4PrivateNetworksParsed: test.privateNetworksIPv4,
				IPv6PrivateNetworksParsed: test.privateNetworksIPv6,
			},
		}
		endpoint, _ := NewEndpoint(
			fakeUUIDGenerator{},
			exchange,
			ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
			&mockStoredReqFetcher{},
			empty_fetcher.EmptyFetcher{},
			cfg,
			&metricsConfig.NilMetricsEngine{},
			analyticsBuild.New(&config.Analytics{}),
			map[string]string{},
			[]byte{},
			openrtb_ext.BuildBidderMap(),
			empty_fetcher.EmptyFetcher{},
			hooks.EmptyPlanBuilder{},
			nil,
		)

		httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, test.reqJSONFile)))
		httpReq.Header.Set("X-Forwarded-For", test.xForwardedForHeader)

		endpoint(httptest.NewRecorder(), httpReq, nil)

		result := exchange.gotRequest
		if !assert.NotEmpty(t, result, test.description+"Request received by the exchange.") {
			t.FailNow()
		}
		assert.Equal(t, test.expectedDeviceIPv4, result.Device.IP, test.description+":ipv4")
		assert.Equal(t, test.expectedDeviceIPv6, result.Device.IPv6, test.description+":ipv6")
	}
}

func TestImplicitDNT(t *testing.T) {
	var (
		disabled int8 = 0
		enabled  int8 = 1
	)
	testCases := []struct {
		description     string
		dntHeader       string
		request         openrtb2.BidRequest
		expectedRequest openrtb2.BidRequest
	}{
		{
			description:     "Device Missing - Not Set In Header",
			dntHeader:       "",
			request:         openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{},
		},
		{
			description: "Device Missing - Set To 0 In Header",
			dntHeader:   "0",
			request:     openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{
				Device: &openrtb2.Device{
					DNT: &disabled,
				},
			},
		},
		{
			description: "Device Missing - Set To 1 In Header",
			dntHeader:   "1",
			request:     openrtb2.BidRequest{},
			expectedRequest: openrtb2.BidRequest{
				Device: &openrtb2.Device{
					DNT: &enabled,
				},
			},
		},
		{
			description: "Not Set In Request - Not Set In Header",
			dntHeader:   "",
			request: openrtb2.BidRequest{
				Device: &openrtb2.Device{},
			},
			expectedRequest: openrtb2.BidRequest{
				Device: &openrtb2.Device{},
			},
		},
		{
			description: "Not Set In Request - Set To 0 In Header",
			dntHeader:   "0",
			request: openrtb2.BidRequest{
				Device: &openrtb2.Device{},
			},
			expectedRequest: openrtb2.BidRequest{
				Device: &openrtb2.Device{
					DNT: &disabled,
				},
			},
		},
		{
			description: "Not Set In Request - Set To 1 In Header",
			dntHeader:   "1",
			request: openrtb2.BidRequest{
				Device: &openrtb2.Device{},
			},
			expectedRequest: openrtb2.BidRequest{
				Device: &openrtb2.Device{
					DNT: &enabled,
				},
			},
		},
		{
			description: "Set In Request - Not Set In Header",
			dntHeader:   "",
			request: openrtb2.BidRequest{
				Device: &openrtb2.Device{
					DNT: &enabled,
				},
			},
			expectedRequest: openrtb2.BidRequest{
				Device: &openrtb2.Device{
					DNT: &enabled,
				},
			},
		},
		{
			description: "Set In Request - Set To 0 In Header",
			dntHeader:   "0",
			request: openrtb2.BidRequest{
				Device: &openrtb2.Device{
					DNT: &enabled,
				},
			},
			expectedRequest: openrtb2.BidRequest{
				Device: &openrtb2.Device{
					DNT: &enabled,
				},
			},
		},
		{
			description: "Set In Request - Set To 1 In Header",
			dntHeader:   "1",
			request: openrtb2.BidRequest{
				Device: &openrtb2.Device{
					DNT: &enabled,
				},
			},
			expectedRequest: openrtb2.BidRequest{
				Device: &openrtb2.Device{
					DNT: &enabled,
				},
			},
		},
	}

	for _, test := range testCases {
		httpReq := httptest.NewRequest("POST", "/openrtb2/auction", nil)
		httpReq.Header.Set("DNT", test.dntHeader)
		reqWrapper := &openrtb_ext.RequestWrapper{BidRequest: &test.request}
		setDoNotTrackImplicitly(httpReq, reqWrapper)
		assert.Equal(t, test.expectedRequest, *reqWrapper.BidRequest, test.description)
	}
}

func TestImplicitDNTEndToEnd(t *testing.T) {
	var (
		disabled int8 = 0
		enabled  int8 = 1
	)
	testCases := []struct {
		description string
		reqJSONFile string
		dntHeader   string
		expectedDNT *int8
	}{
		{
			description: "Not Set In Request - Not Set In Header",
			reqJSONFile: "site.json",
			dntHeader:   "",
			expectedDNT: nil,
		},
		{
			description: "Not Set In Request - Set To 0 In Header",
			reqJSONFile: "site.json",
			dntHeader:   "0",
			expectedDNT: &disabled,
		},
		{
			description: "Not Set In Request - Set To 1 In Header",
			reqJSONFile: "site.json",
			dntHeader:   "1",
			expectedDNT: &enabled,
		},
		{
			description: "Set In Request - Not Set In Header",
			reqJSONFile: "site-has-dnt.json",
			dntHeader:   "",
			expectedDNT: &enabled, // Hardcoded value in test file.
		},
		{
			description: "Set In Request - Not Overwritten By Header",
			reqJSONFile: "site-has-dnt.json",
			dntHeader:   "0",
			expectedDNT: &enabled, // Hardcoded value in test file.
		},
	}

	for _, test := range testCases {
		exchange := &nobidExchange{}
		endpoint, _ := NewEndpoint(
			fakeUUIDGenerator{},
			exchange,
			ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
			&mockStoredReqFetcher{},
			empty_fetcher.EmptyFetcher{},
			&config.Configuration{MaxRequestSize: maxSize},
			&metricsConfig.NilMetricsEngine{},
			analyticsBuild.New(&config.Analytics{}),
			map[string]string{},
			[]byte{},
			openrtb_ext.BuildBidderMap(),
			empty_fetcher.EmptyFetcher{},
			hooks.EmptyPlanBuilder{},
			nil,
		)

		httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, test.reqJSONFile)))
		httpReq.Header.Set("DNT", test.dntHeader)

		endpoint(httptest.NewRecorder(), httpReq, nil)

		result := exchange.gotRequest
		if !assert.NotEmpty(t, result, test.description+"Request received by the exchange.") {
			t.FailNow()
		}
		assert.Equal(t, test.expectedDNT, result.Device.DNT, test.description+":dnt")
	}
}

func TestReferer(t *testing.T) {
	testCases := []struct {
		description             string
		givenReferer            string
		expectedDomain          string
		expectedPage            string
		expectedPublisherDomain string
		bidReq                  *openrtb_ext.RequestWrapper
	}{
		{
			description:             "site.page/domain are unchanged when site.page/domain and http referer are not set",
			expectedDomain:          "",
			expectedPage:            "",
			expectedPublisherDomain: "",
			bidReq: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{},
				},
			}},
		},
		{
			description:             "site.page/domain are derived from referer when neither is set and http referer is set",
			givenReferer:            "https://test.somepage.com",
			expectedDomain:          "test.somepage.com",
			expectedPublisherDomain: "somepage.com",
			expectedPage:            "https://test.somepage.com",
			bidReq: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Publisher: &openrtb2.Publisher{},
				},
			}},
		},
		{
			description:             "site.domain is derived from site.page when site.page is set and http referer is not set",
			expectedDomain:          "test.somepage.com",
			expectedPublisherDomain: "somepage.com",
			expectedPage:            "https://test.somepage.com",
			bidReq: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Page:      "https://test.somepage.com",
					Publisher: &openrtb2.Publisher{},
				},
			}},
		},
		{
			description:             "site.domain is derived from http referer when site.page and http referer are set",
			givenReferer:            "http://test.com",
			expectedDomain:          "test.com",
			expectedPublisherDomain: "test.com",
			expectedPage:            "https://test.somepage.com",
			bidReq: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Page:      "https://test.somepage.com",
					Publisher: &openrtb2.Publisher{},
				},
			}},
		},
		{
			description:             "site.page/domain are unchanged when site.page/domain and http referer are set",
			givenReferer:            "http://test.com",
			expectedDomain:          "some.domain.com",
			expectedPublisherDomain: "test.com",
			expectedPage:            "https://test.somepage.com",
			bidReq: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Domain:    "some.domain.com",
					Page:      "https://test.somepage.com",
					Publisher: &openrtb2.Publisher{},
				},
			}},
		},
		{
			description:             "Publisher domain shouldn't be overrwriten if already set",
			expectedDomain:          "test.somepage.com",
			expectedPublisherDomain: "differentpage.com",
			expectedPage:            "https://test.somepage.com",
			bidReq: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Domain: "",
					Page:   "https://test.somepage.com",
					Publisher: &openrtb2.Publisher{
						Domain: "differentpage.com",
					},
				},
			}},
		},
		{
			description:             "request.site is nil",
			givenReferer:            "http://test.com",
			expectedDomain:          "test.com",
			expectedPublisherDomain: "test.com",
			expectedPage:            "http://test.com",
			bidReq:                  &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}},
		},
	}

	for _, test := range testCases {
		httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
		httpReq.Header.Set("Referer", test.givenReferer)

		setSiteImplicitly(httpReq, test.bidReq)

		assert.NotNil(t, test.bidReq.Site, test.description)
		assert.Equal(t, test.expectedDomain, test.bidReq.Site.Domain, test.description)
		assert.Equal(t, test.expectedPage, test.bidReq.Site.Page, test.description)
		assert.Equal(t, test.expectedPublisherDomain, test.bidReq.Site.Publisher.Domain, test.description)
	}
}

func TestParseImpInfoSingleImpression(t *testing.T) {

	expectedRes := []ImpExtPrebidData{
		{
			Imp:          json.RawMessage(`{"video":{"h":300,"w":200},"ext": {"prebid": {"storedrequest": {"id": "1"},"options": {"echovideoattrs": true}}}}`),
			ImpExtPrebid: openrtb_ext.ExtImpPrebid{StoredRequest: &openrtb_ext.ExtStoredRequest{ID: "1"}, Options: &openrtb_ext.Options{EchoVideoAttrs: true}},
		},
		{
			Imp:          json.RawMessage(`{"id": "adUnit2","ext": {"prebid": {"storedrequest": {"id": "1"},"options": {"echovideoattrs": true}},"appnexus": {"placementId": "def","trafficSourceCode": "mysite.com","reserve": null},"rubicon": null}}`),
			ImpExtPrebid: openrtb_ext.ExtImpPrebid{StoredRequest: &openrtb_ext.ExtStoredRequest{ID: "1"}, Options: &openrtb_ext.Options{EchoVideoAttrs: true}},
		},
		{
			Imp:          json.RawMessage(`{"ext": {"prebid": {"storedrequest": {"id": "2"},"options": {"echovideoattrs": false}}}}`),
			ImpExtPrebid: openrtb_ext.ExtImpPrebid{StoredRequest: &openrtb_ext.ExtStoredRequest{ID: "2"}, Options: &openrtb_ext.Options{EchoVideoAttrs: false}},
		},
		{
			//in this case impression doesn't have storedrequest so we don't expect any data about this imp will be returned
			Imp:          json.RawMessage(`{"id": "some-static-imp","video":{"mimes":["video/mp4"]},"ext": {"appnexus": {"placementId": "abc","position": "below"}}}`),
			ImpExtPrebid: openrtb_ext.ExtImpPrebid{},
		},
		{
			Imp:          json.RawMessage(`{"id":"my-imp-id", "video":{"h":300, "w":200}, "ext":{"prebid":{"storedrequest": {"id": "6"}}}}`),
			ImpExtPrebid: openrtb_ext.ExtImpPrebid{StoredRequest: &openrtb_ext.ExtStoredRequest{ID: "6"}},
		},
	}

	for i, requestData := range testStoredRequests {
		impInfo, errs := parseImpInfo([]byte(requestData))
		assert.Len(t, errs, 0, "No errors should be returned")
		assert.JSONEq(t, string(expectedRes[i].Imp), string(impInfo[0].Imp), "Incorrect impression data")
		assert.Equal(t, expectedRes[i].ImpExtPrebid, impInfo[0].ImpExtPrebid, "Incorrect impression ext prebid data")

	}
}

func TestParseImpInfoMultipleImpressions(t *testing.T) {

	inputData := []byte(`{
		"id": "ThisID",
		"imp": [
			{
				"id": "imp1",
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						},
						"options": {
							"echovideoattrs": true
						}
					}
				}
			},
			{
				"id": "imp2",
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "2"
						},
						"options": {
							"echovideoattrs": false
						}
					}
				}
			},
			{
				"id": "imp3"
			}
		]
	}`)

	expectedRes := []ImpExtPrebidData{
		{
			Imp:          json.RawMessage(`{"id": "imp1","ext": {"prebid": {"storedrequest": {"id": "1"},"options": {"echovideoattrs": true}}}}`),
			ImpExtPrebid: openrtb_ext.ExtImpPrebid{StoredRequest: &openrtb_ext.ExtStoredRequest{ID: "1"}, Options: &openrtb_ext.Options{EchoVideoAttrs: true}},
		},
		{
			Imp:          json.RawMessage(`{"id": "imp2","ext": {"prebid": {"storedrequest": {"id": "2"},"options": {"echovideoattrs": false}}}}`),
			ImpExtPrebid: openrtb_ext.ExtImpPrebid{StoredRequest: &openrtb_ext.ExtStoredRequest{ID: "2"}, Options: &openrtb_ext.Options{EchoVideoAttrs: false}},
		},
		{
			Imp:          json.RawMessage(`{"id": "imp3"}`),
			ImpExtPrebid: openrtb_ext.ExtImpPrebid{},
		},
	}

	impInfo, errs := parseImpInfo([]byte(inputData))
	assert.Len(t, errs, 0, "No errors should be returned")
	for i, res := range expectedRes {
		assert.JSONEq(t, string(res.Imp), string(impInfo[i].Imp), "Incorrect impression data")
		assert.Equal(t, res.ImpExtPrebid, impInfo[i].ImpExtPrebid, "Incorrect impression ext prebid data")
	}
}

// Test the stored request functionality
func TestStoredRequests(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	testStoreVideoAttr := []bool{true, true, false, false, false}

	for i, requestData := range testStoredRequests {
		impInfo, errs := parseImpInfo([]byte(requestData))
		assert.Len(t, errs, 0, "No errors should be returned")
		storedBidRequestId, hasStoredBidRequest, storedRequests, storedImps, errs := deps.getStoredRequests(context.Background(), json.RawMessage(requestData), impInfo)
		assert.Len(t, errs, 0, "No errors should be returned")
		newRequest, impExtInfoMap, errList := deps.processStoredRequests(json.RawMessage(requestData), impInfo, storedRequests, storedImps, storedBidRequestId, hasStoredBidRequest)
		if len(errList) != 0 {
			for _, err := range errList {
				if err != nil {
					t.Errorf("processStoredRequests Error: %s", err.Error())
				} else {
					t.Error("processStoredRequests Error: received nil error")
				}
			}
		}
		expectJson := json.RawMessage(testFinalRequests[i])
		assert.JSONEq(t, string(expectJson), string(newRequest), "Incorrect result request %d", i)
		expectedImp := testStoredImpIds[i]
		expectedStoredImp := json.RawMessage(testStoredImps[i])
		if len(impExtInfoMap[expectedImp].StoredImp) > 0 {
			assert.JSONEq(t, string(expectedStoredImp), string(impExtInfoMap[expectedImp].StoredImp), "Incorrect expected stored imp %d", i)

		}
		assert.Equalf(t, testStoreVideoAttr[i], impExtInfoMap[expectedImp].EchoVideoAttrs, "EchoVideoAttrs value is incorrect")
	}
}

func TestMergeBidderParams(t *testing.T) {
	testCases := []struct {
		description         string
		givenRequest        openrtb2.BidRequest
		expectedRequestImps []openrtb2.Imp
	}{
		{
			description: "No Request Params",
			givenRequest: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1}}`)}},
			},
			expectedRequestImps: []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1}}`)}},
		},
		{
			description: "No Request Params - Empty Object",
			givenRequest: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1}}`)}},
				Ext: json.RawMessage(`{"prebid":{"bidderparams":{}}}`),
			},
			expectedRequestImps: []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1}}`)}},
		},
		{
			description: "Malformed Request Params",
			givenRequest: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1}}`)}},
				Ext: json.RawMessage(`{"prebid":{"bidderparams":malformed}}`),
			},
			expectedRequestImps: []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1}}`)}},
		},
		{
			description: "No Imps",
			givenRequest: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{},
				Ext: json.RawMessage(`{"prebid":{"bidderparams":{"bidder1":{"b":2}}}}`),
			},
			expectedRequestImps: []openrtb2.Imp{},
		},
		{
			description: "One Imp - imp.ext Modified",
			givenRequest: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1}}`)}},
				Ext: json.RawMessage(`{"prebid":{"bidderparams":{"bidder1":{"b":2}}}}`),
			},
			expectedRequestImps: []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1,"b":2}}`)}},
		},
		{
			description: "One Imp - imp.ext.prebid.bidder Modified",
			givenRequest: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"prebid":{"bidder":{"bidder1":{"a":1}}}}`)}},
				Ext: json.RawMessage(`{"prebid":{"bidderparams":{"bidder1":{"b":2}}}}`),
			},
			expectedRequestImps: []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"prebid":{"bidder":{"bidder1":{"a":1,"b":2}}}}`)}},
		},
		{
			description: "One Imp - imp.ext + imp.ext.prebid.bidder Modified - Different Bidders",
			givenRequest: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1},"prebid":{"bidder":{"bidder2":{"a":"one"}}}}`)}},
				Ext: json.RawMessage(`{"prebid":{"bidderparams":{"bidder1":{"b":2},"bidder2":{"b":"two"}}}}`),
			},
			expectedRequestImps: []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1,"b":2},"prebid":{"bidder":{"bidder2":{"a":"one","b":"two"}}}}`)}},
		},
		{
			description: "One Imp - imp.ext + imp.ext.prebid.bidder Modified - Same Bidder",
			givenRequest: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1},"prebid":{"bidder":{"bidder1":{"a":"one"}}}}`)}},
				Ext: json.RawMessage(`{"prebid":{"bidderparams":{"bidder1":{"b":2}}}}`),
			},
			expectedRequestImps: []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1,"b":2},"prebid":{"bidder":{"bidder1":{"a":"one","b":2}}}}`)}},
		},
		{
			description: "One Imp - No imp.ext",
			givenRequest: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1"}},
				Ext: json.RawMessage(`{"prebid":{"bidderparams":{"bidder1":{"b":2}}}}`),
			},
			expectedRequestImps: []openrtb2.Imp{{ID: "1"}},
		},
		{
			description: "Multiple Imps - Modified Mixed",
			givenRequest: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1},"prebid":{"bidder":{"bidder1":{"a":"one"}}}}`)},
					{ID: "2", Ext: json.RawMessage(`{"bidder2":{"a":1,"b":"existing"},"prebid":{"bidder":{"bidder2":{"a":"one"}}}}`)}},
				Ext: json.RawMessage(`{"prebid":{"bidderparams":{"bidder1":{"b":2},"bidder2":{"b":"two"}}}}`),
			},
			expectedRequestImps: []openrtb2.Imp{
				{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1,"b":2},"prebid":{"bidder":{"bidder1":{"a":"one","b":2}}}}`)},
				{ID: "2", Ext: json.RawMessage(`{"bidder2":{"a":1,"b":"existing"},"prebid":{"bidder":{"bidder2":{"a":"one","b":"two"}}}}`)}},
		},
		{
			description: "Multiple Imps - None Modified Mixed",
			givenRequest: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1},"prebid":{"bidder":{"bidder2":{"a":"one"}}}}`)},
					{ID: "2", Ext: json.RawMessage(`{"bidder1":{"a":2},"prebid":{"bidder":{"bidder2":{"a":"two"}}}}`)}},
				Ext: json.RawMessage(`{"prebid":{"bidderparams":{"bidder3":{"c":3}}}}`),
			},
			expectedRequestImps: []openrtb2.Imp{
				{ID: "1", Ext: json.RawMessage(`{"bidder1":{"a":1},"prebid":{"bidder":{"bidder2":{"a":"one"}}}}`)},
				{ID: "2", Ext: json.RawMessage(`{"bidder1":{"a":2},"prebid":{"bidder":{"bidder2":{"a":"two"}}}}`)}},
		},
		{
			description: "Multiple Imps - One Malformed",
			givenRequest: openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "1", Ext: json.RawMessage(`malformed`)},
					{ID: "2", Ext: json.RawMessage(`{"bidder2":{"a":1,"b":"existing"},"prebid":{"bidder":{"bidder2":{"a":"one"}}}}`)}},
				Ext: json.RawMessage(`{"prebid":{"bidderparams":{"bidder1":{"b":2},"bidder2":{"b":"two"}}}}`),
			},
			expectedRequestImps: []openrtb2.Imp{
				{ID: "1", Ext: json.RawMessage(`malformed`)},
				{ID: "2", Ext: json.RawMessage(`{"bidder2":{"a":1,"b":"existing"},"prebid":{"bidder":{"bidder2":{"a":"one","b":"two"}}}}`)}},
		},
	}

	for _, test := range testCases {
		w := &openrtb_ext.RequestWrapper{BidRequest: &test.givenRequest}
		actualErr := mergeBidderParams(w)

		// errors are only possible from the marshal operation, which is not testable
		assert.NoError(t, actualErr, test.description+":err")

		// rebuild request before asserting value
		assert.NoError(t, w.RebuildRequest(), test.description+":rebuild_request")

		assert.Equal(t, test.givenRequest.Imp, test.expectedRequestImps, test.description+":imps")
	}
}

func TestMergeBidderParamsImpExt(t *testing.T) {
	testCases := []struct {
		description       string
		givenImpExt       map[string]json.RawMessage
		givenReqExtParams map[string]map[string]json.RawMessage
		expectedModified  bool
		expectedImpExt    map[string]json.RawMessage
	}{
		{
			description:       "One Bidder - Modified (no collision)",
			givenImpExt:       map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1}`)},
			givenReqExtParams: map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:  true,
			expectedImpExt:    map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`)},
		},
		{
			description:       "One Bidder - Modified (imp.ext bidder empty)",
			givenImpExt:       map[string]json.RawMessage{"bidder1": json.RawMessage(`{}`)},
			givenReqExtParams: map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:  true,
			expectedImpExt:    map[string]json.RawMessage{"bidder1": json.RawMessage(`{"b":2}`)},
		},
		{
			description:       "One Bidder - Not Modified (imp.ext bidder not defined)",
			givenImpExt:       map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`)},
			givenReqExtParams: map[string]map[string]json.RawMessage{"bidder-not-defined": {"b": json.RawMessage(`4`)}},
			expectedModified:  false,
			expectedImpExt:    map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`)},
		},
		{
			description:       "One Bidder - Not Modified (imp.ext bidder nil)",
			givenImpExt:       map[string]json.RawMessage{"bidder1": nil},
			givenReqExtParams: map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`4`)}},
			expectedModified:  false,
			expectedImpExt:    map[string]json.RawMessage{"bidder1": nil},
		},
		{
			description:       "One Bidder - Not Modified (imp.ext wins)",
			givenImpExt:       map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`)},
			givenReqExtParams: map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`4`)}},
			expectedModified:  false,
			expectedImpExt:    map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`)},
		},
		{
			description:       "One Bidder - Not Modified (reserved bidder ignored)",
			givenImpExt:       map[string]json.RawMessage{"gpid": json.RawMessage(`{"a":1}`)},
			givenReqExtParams: map[string]map[string]json.RawMessage{"gpid": {"b": json.RawMessage(`2`)}},
			expectedModified:  false,
			expectedImpExt:    map[string]json.RawMessage{"gpid": json.RawMessage(`{"a":1}`)},
		},
		{
			description:       "One Bidder - Not Modified (reserved bidder ignored - not embedded object)",
			givenImpExt:       map[string]json.RawMessage{"gpid": json.RawMessage(`1`)},
			givenReqExtParams: map[string]map[string]json.RawMessage{"gpid": {"b": json.RawMessage(`2`)}},
			expectedModified:  false,
			expectedImpExt:    map[string]json.RawMessage{"gpid": json.RawMessage(`1`)},
		},
		{
			description:       "One Bidder - Not Modified (malformed ignored)",
			givenImpExt:       map[string]json.RawMessage{"bidder1": json.RawMessage(`malformed`)},
			givenReqExtParams: map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:  false,
			expectedImpExt:    map[string]json.RawMessage{"bidder1": json.RawMessage(`malformed`)},
		},
		{
			description:       "Multiple Bidders - Mixed",
			givenImpExt:       map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1}`), "bidder2": json.RawMessage(`{"a":"one","b":"two"}`)},
			givenReqExtParams: map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}, "bidder2": {"b": json.RawMessage(`"three"`)}},
			expectedModified:  true,
			expectedImpExt:    map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`), "bidder2": json.RawMessage(`{"a":"one","b":"two"}`)},
		},
		{
			description:       "Multiple Bidders - None Modified",
			givenImpExt:       map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1}`), "bidder2": json.RawMessage(`{"a":"one","b":"two"}`)},
			givenReqExtParams: map[string]map[string]json.RawMessage{},
			expectedModified:  false,
			expectedImpExt:    map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1}`), "bidder2": json.RawMessage(`{"a":"one","b":"two"}`)},
		},
	}

	for _, test := range testCases {
		impExt := openrtb_ext.CreateImpExtForTesting(test.givenImpExt, nil)

		err := mergeBidderParamsImpExt(&impExt, test.givenReqExtParams)

		// errors are only possible from the marshal operation, which is not testable
		assert.NoError(t, err, test.description+":err")

		assert.Equal(t, test.expectedModified, impExt.Dirty(), test.description+":modified")
		assert.Equal(t, test.expectedImpExt, impExt.GetExt(), test.description+":imp.ext")
	}
}

func TestMergeBidderParamsImpExtPrebid(t *testing.T) {
	testCases := []struct {
		description          string
		givenImpExtPrebid    *openrtb_ext.ExtImpPrebid
		givenReqExtParams    map[string]map[string]json.RawMessage
		expectedModified     bool
		expectedImpExtPrebid *openrtb_ext.ExtImpPrebid
	}{
		{
			description:          "No Prebid Section",
			givenImpExtPrebid:    nil,
			givenReqExtParams:    map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:     false,
			expectedImpExtPrebid: nil,
		},
		{
			description:          "No Prebid Bidder Section",
			givenImpExtPrebid:    &openrtb_ext.ExtImpPrebid{Bidder: nil},
			givenReqExtParams:    map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:     false,
			expectedImpExtPrebid: &openrtb_ext.ExtImpPrebid{Bidder: nil},
		},
		{
			description:          "Empty Prebid Bidder Section",
			givenImpExtPrebid:    &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{}},
			givenReqExtParams:    map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:     false,
			expectedImpExtPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{}},
		},
		{
			description:          "One Bidder - Modified (no collision)",
			givenImpExtPrebid:    &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1}`)}},
			givenReqExtParams:    map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:     true,
			expectedImpExtPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`)}},
		},
		{
			description:          "One Bidder - Modified (imp.ext bidder empty)",
			givenImpExtPrebid:    &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{}`)}},
			givenReqExtParams:    map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:     true,
			expectedImpExtPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"b":2}`)}},
		},
		{
			description:          "One Bidder - Not Modified (imp.ext wins)",
			givenImpExtPrebid:    &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`)}},
			givenReqExtParams:    map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`4`)}},
			expectedModified:     false,
			expectedImpExtPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`)}},
		},
		{
			description:          "One Bidder - Not Modified (imp.ext bidder not defined)",
			givenImpExtPrebid:    &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`)}},
			givenReqExtParams:    map[string]map[string]json.RawMessage{"bidder-not-defined": {"b": json.RawMessage(`4`)}},
			expectedModified:     false,
			expectedImpExtPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`)}},
		},
		{
			description:          "One Bidder - Not Modified (imp.ext bidder nil)",
			givenImpExtPrebid:    &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": nil}},
			givenReqExtParams:    map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`4`)}},
			expectedModified:     false,
			expectedImpExtPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": nil}},
		},
		{
			description:          "One Bidder - Not Modified (malformed ignored)",
			givenImpExtPrebid:    &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`malformed`)}},
			givenReqExtParams:    map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:     false,
			expectedImpExtPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`malformed`)}},
		},
		{
			description:          "Multiple Bidders - Mixed",
			givenImpExtPrebid:    &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1}`), "bidder2": json.RawMessage(`{"a":"one","b":"two"}`)}},
			givenReqExtParams:    map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}, "bidder2": {"b": json.RawMessage(`"three"`)}},
			expectedModified:     true,
			expectedImpExtPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`), "bidder2": json.RawMessage(`{"a":"one","b":"two"}`)}},
		},
		{
			description:          "Multiple Bidders - None Modified",
			givenImpExtPrebid:    &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1}`), "bidder2": json.RawMessage(`{"a":"one","b":"two"}`)}},
			givenReqExtParams:    map[string]map[string]json.RawMessage{},
			expectedModified:     false,
			expectedImpExtPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1}`), "bidder2": json.RawMessage(`{"a":"one","b":"two"}`)}},
		},
	}

	for _, test := range testCases {
		impExt := openrtb_ext.CreateImpExtForTesting(map[string]json.RawMessage{}, test.givenImpExtPrebid)

		err := mergeBidderParamsImpExtPrebid(&impExt, test.givenReqExtParams)

		// errors are only possible from the marshal operation, which is not testable
		assert.NoError(t, err, test.description+":err")

		assert.Equal(t, test.expectedModified, impExt.Dirty(), test.description+":modified")
		assert.Equal(t, test.expectedImpExtPrebid, impExt.GetPrebid(), test.description+":imp.ext.prebid")
	}
}

func TestValidateExactlyOneInventoryType(t *testing.T) {

	testCases := []struct {
		description         string
		givenRequestWrapper *openrtb_ext.RequestWrapper
		expectedError       error
	}{
		{
			description:         "None provided - invalid",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}},
			expectedError:       errors.New("One of request.site or request.app or request.dooh must be defined"),
		},
		{
			description: "Only site provided",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				Site: &openrtb2.Site{},
			}},
			expectedError: nil,
		},
		{
			description: "Only app provided",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				App: &openrtb2.App{},
			}},
			expectedError: nil,
		},
		{
			description: "Only dooh provided",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				DOOH: &openrtb2.DOOH{},
			}},
			expectedError: nil,
		},
		{
			description: "Two provided (site+app) - invalid",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				Site: &openrtb2.Site{},
				App:  &openrtb2.App{},
			}},
			expectedError: errors.New("No more than one of request.site or request.app or request.dooh can be defined"),
		},
		{
			description: "Two provided (site+dooh) - invalid",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				Site: &openrtb2.Site{},
				DOOH: &openrtb2.DOOH{},
			}},
			expectedError: errors.New("No more than one of request.site or request.app or request.dooh can be defined"),
		},
		{
			description: "Two provided (app+dooh) - invalid",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				App:  &openrtb2.App{},
				DOOH: &openrtb2.DOOH{},
			}},
			expectedError: errors.New("No more than one of request.site or request.app or request.dooh can be defined"),
		},
		{
			description: "Three provided - invalid",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
				Site: &openrtb2.Site{},
				App:  &openrtb2.App{},
				DOOH: &openrtb2.DOOH{},
			}},
			expectedError: errors.New("No more than one of request.site or request.app or request.dooh can be defined"),
		},
	}

	for _, test := range testCases {
		error := validateExactlyOneInventoryType(test.givenRequestWrapper)
		assert.Equalf(t, test.expectedError, error, "Error doesn't match: %s\n", test.description)
	}

}

func TestValidateRequest(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	testCases := []struct {
		description           string
		givenIsAmp            bool
		givenRequestWrapper   *openrtb_ext.RequestWrapper
		givenHttpRequest      *http.Request
		givenAccount          *config.Account
		expectedErrorList     []error
		expectedChannelObject *openrtb_ext.ExtRequestPrebidChannel
	}{
		{
			description: "No errors in bid request with request.ext.prebid.channel info, expect validate request to throw no errors",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus":{"placementId": 12345678}}`),
						},
					},
					Ext: []byte(`{"prebid":{"channel": {"name": "nameOfChannel", "version": "1.0"}}}`),
				},
			},
			givenIsAmp:            false,
			expectedErrorList:     []error{},
			expectedChannelObject: &openrtb_ext.ExtRequestPrebidChannel{Name: "nameOfChannel", Version: "1.0"},
		},
		{
			description: "Error in bid request with request.ext.prebid.channel.name being blank, expect validate request to return error",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus":{"placementId": 12345678}}`),
						},
					},
					Ext: []byte(`{"prebid":{"channel": {"name": "", "version": ""}}}`),
				},
			},
			givenIsAmp:        false,
			expectedErrorList: []error{errors.New("ext.prebid.channel.name can't be empty")},
		},
		{
			description: "AliasGVLID validation error",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus":{"placementId": 12345678}}`),
						},
					},
					Ext: []byte(`{"prebid":{"aliases":{"yahoossp":"appnexus"}, "aliasgvlids":{"pubmatic1":1}}}`),
				},
			},
			givenIsAmp:            false,
			expectedErrorList:     []error{errors.New("request.ext.prebid.aliasgvlids. vendorId 1 refers to unknown bidder alias: pubmatic1")},
			expectedChannelObject: &openrtb_ext.ExtRequestPrebidChannel{Name: appChannel, Version: ""},
		},
		{
			description: "AliasGVLID validation error as vendorID < 1",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus":{"placementId": 12345678}}`),
						},
					},
					Ext: []byte(`{"prebid":{"aliases":{"yahoossp":"appnexus"}, "aliasgvlids":{"yahoossp":0}}}`),
				},
			},
			givenIsAmp:            false,
			expectedErrorList:     []error{errors.New("request.ext.prebid.aliasgvlids. Invalid vendorId 0 for alias: yahoossp. Choose a different vendorId, or remove this entry.")},
			expectedChannelObject: &openrtb_ext.ExtRequestPrebidChannel{Name: appChannel, Version: ""},
		},
		{
			description: "No errors in bid request with request.ext.prebid but no channel info, expect validate request to throw no errors and fill channel with app",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus":{"placementId": 12345678}}`),
						},
					},
					Ext: []byte(`{"prebid":{"aliases":{"yahoossp":"appnexus"}, "aliasgvlids":{"yahoossp":1}}}`),
				},
			},
			givenIsAmp:            false,
			expectedErrorList:     []error{},
			expectedChannelObject: &openrtb_ext.ExtRequestPrebidChannel{Name: appChannel, Version: ""},
		},
		{
			description: "Minimum required site attributes missing",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:   "Some-ID",
					Site: &openrtb2.Site{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus":{"placementId": 12345678}}`),
						},
					},
				},
			},
			expectedErrorList: []error{
				errors.New("request.site should include at least one of request.site.id or request.site.page."),
			},
		},
		{
			description: "Minimum required DOOH attributes missing",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:   "Some-ID",
					DOOH: &openrtb2.DOOH{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus":{"placementId": 12345678}}`),
						},
					},
				},
			},
			expectedErrorList: []error{
				errors.New("request.dooh should include at least one of request.dooh.id or request.dooh.venuetype."),
			},
		},
	}

	for _, test := range testCases {
		errorList := deps.validateRequest(test.givenAccount, test.givenHttpRequest, test.givenRequestWrapper, test.givenIsAmp, false, nil, false)
		assert.Equalf(t, test.expectedErrorList, errorList, "Error doesn't match: %s\n", test.description)

		if len(errorList) == 0 {
			requestExt, err := test.givenRequestWrapper.GetRequestExt()
			assert.Empty(t, err, test.description)
			requestPrebid := requestExt.GetPrebid()

			assert.Equalf(t, test.expectedChannelObject, requestPrebid.Channel, "Channel information isn't correct: %s\n", test.description)
		}
	}
}

func TestValidateRequestExt(t *testing.T) {
	testCases := []struct {
		description     string
		givenRequestExt json.RawMessage
		expectedErrors  []string
	}{
		{
			description:     "nil",
			givenRequestExt: nil,
		},
		{
			description:     "prebid - nil",
			givenRequestExt: json.RawMessage(`{}`),
		},
		{
			description:     "prebid - empty",
			givenRequestExt: json.RawMessage(`{"prebid":{}}`),
		},
		{
			description:     "prebid cache - empty",
			givenRequestExt: json.RawMessage(`{"prebid":{"cache":{}}}`),
			expectedErrors:  []string{`request.ext is invalid: request.ext.prebid.cache requires one of the "bids" or "vastxml" properties`},
		},
		{
			description:     "prebid cache - bids - null",
			givenRequestExt: json.RawMessage(`{"prebid":{"cache":{"bids":null}}}`),
			expectedErrors:  []string{`request.ext is invalid: request.ext.prebid.cache requires one of the "bids" or "vastxml" properties`},
		},
		{
			description:     "prebid cache - bids - wrong type",
			givenRequestExt: json.RawMessage(`{"prebid":{"cache":{"bids":true}}}`),
			expectedErrors:  []string{"cannot unmarshal openrtb_ext.ExtRequestPrebidCache.Bids: expect { or n, but found t"},
		},
		{
			description:     "prebid cache - bids - provided",
			givenRequestExt: json.RawMessage(`{"prebid":{"cache":{"bids":{}}}}`),
		},
		{
			description:     "prebid cache - vastxml - null",
			givenRequestExt: json.RawMessage(`{"prebid": {"cache": {"vastxml": null}}}`),
			expectedErrors:  []string{`request.ext is invalid: request.ext.prebid.cache requires one of the "bids" or "vastxml" properties`},
		},
		{
			description:     "prebid cache - vastxml - wrong type",
			givenRequestExt: json.RawMessage(`{"prebid":{"cache":{"vastxml":true}}}`),
			expectedErrors:  []string{"cannot unmarshal openrtb_ext.ExtRequestPrebidCache.VastXML: expect { or n, but found t"},
		},
		{
			description:     "prebid cache - vastxml - provided",
			givenRequestExt: json.RawMessage(`{"prebid":{"cache":{"vastxml":{}}}}`),
		},
		{
			description:     "prebid cache - bids + vastxml - provided",
			givenRequestExt: json.RawMessage(`{"prebid":{"cache":{"bids":{},"vastxml":{}}}}`),
		},
		{
			description:     "prebid price granularity invalid",
			givenRequestExt: json.RawMessage(`{"prebid":{"targeting":{"pricegranularity":{"precision":-1,"ranges":[{"min":0,"max":20,"increment":0.1}]}}}}`),
			expectedErrors:  []string{"Price granularity error: precision must be non-negative"},
		},
		{
			description:     "prebid native media type price granualrity valid",
			givenRequestExt: json.RawMessage(`{"prebid":{"targeting":{"mediatypepricegranularity":{"native":{"precision":3,"ranges":[{"max":20,"increment":4.5}]}}}}}`),
		},
		{
			description:     "valid multibid",
			givenRequestExt: json.RawMessage(`{"prebid": {"multibid": [{"Bidder": "pubmatic", "MaxBids": 2}]}}`),
		},
		{
			description:     "multibid with invalid entries",
			givenRequestExt: json.RawMessage(`{"prebid": {"multibid": [{"Bidder": "pubmatic"}, {"Bidder": "pubmatic", "MaxBids": 2}, {"Bidders": ["pubmatic"], "MaxBids": 3}]}}`),
			expectedErrors: []string{
				`maxBids not defined for {Bidder:pubmatic, Bidders:[], MaxBids:<nil>, TargetBidderCodePrefix:}`,
				`multiBid already defined for pubmatic, ignoring this instance {Bidder:, Bidders:[pubmatic], MaxBids:3, TargetBidderCodePrefix:}`,
			},
		},
	}

	for _, test := range testCases {
		w := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Ext: test.givenRequestExt}}
		errs := validateRequestExt(w)

		if len(test.expectedErrors) > 0 {
			for i, expectedError := range test.expectedErrors {
				assert.EqualError(t, errs[i], expectedError, test.description)
			}
		} else {
			assert.Nil(t, errs, test.description)
		}
	}
}

func TestValidateTargeting(t *testing.T) {
	testCases := []struct {
		name           string
		givenTargeting *openrtb_ext.ExtRequestTargeting
		expectedError  error
	}{
		{
			name:           "nil",
			givenTargeting: nil,
			expectedError:  nil,
		},
		{
			name: "pricegranularity-ranges-out-of-order",
			givenTargeting: &openrtb_ext.ExtRequestTargeting{
				PriceGranularity: &openrtb_ext.PriceGranularity{
					Precision: ptrutil.ToPtr(2),
					Ranges: []openrtb_ext.GranularityRange{
						{Min: 1.0, Max: 2.0, Increment: 0.2},
						{Min: 0.0, Max: 1.0, Increment: 0.5},
					},
				},
			},
			expectedError: errors.New(`Price granularity error: range list must be ordered with increasing "max"`),
		},
		{
			name: "mediatypepricegranularity-nil",
			givenTargeting: &openrtb_ext.ExtRequestTargeting{
				MediaTypePriceGranularity: nil,
			},
			expectedError: nil,
		},
		{
			name: "mediatypepricegranularity-video-ok",
			givenTargeting: &openrtb_ext.ExtRequestTargeting{
				MediaTypePriceGranularity: &openrtb_ext.MediaTypePriceGranularity{
					Video: &openrtb_ext.PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges: []openrtb_ext.GranularityRange{
							{Min: 0.0, Max: 10.0, Increment: 1},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "mediatypepricegranularity-banner-ok",
			givenTargeting: &openrtb_ext.ExtRequestTargeting{
				MediaTypePriceGranularity: &openrtb_ext.MediaTypePriceGranularity{
					Banner: &openrtb_ext.PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges: []openrtb_ext.GranularityRange{
							{Min: 0.0, Max: 10.0, Increment: 1},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "mediatypepricegranularity-native-ok",
			givenTargeting: &openrtb_ext.ExtRequestTargeting{
				MediaTypePriceGranularity: &openrtb_ext.MediaTypePriceGranularity{
					Native: &openrtb_ext.PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges: []openrtb_ext.GranularityRange{
							{Min: 0.0, Max: 20.0, Increment: 1},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "mediatypepricegranularity-video+banner-ok",
			givenTargeting: &openrtb_ext.ExtRequestTargeting{
				MediaTypePriceGranularity: &openrtb_ext.MediaTypePriceGranularity{
					Banner: &openrtb_ext.PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges: []openrtb_ext.GranularityRange{
							{Min: 0.0, Max: 10.0, Increment: 1},
						},
					},
					Video: &openrtb_ext.PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges: []openrtb_ext.GranularityRange{
							{Min: 0.0, Max: 10.0, Increment: 1},
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "mediatypepricegranularity-video-invalid",
			givenTargeting: &openrtb_ext.ExtRequestTargeting{
				MediaTypePriceGranularity: &openrtb_ext.MediaTypePriceGranularity{
					Video: &openrtb_ext.PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges: []openrtb_ext.GranularityRange{
							{Min: 0.0, Max: 10.0, Increment: -1},
						},
					},
				},
			},
			expectedError: errors.New("Price granularity error: increment must be a nonzero positive number"),
		},
		{
			name: "mediatypepricegranularity-banner-invalid",
			givenTargeting: &openrtb_ext.ExtRequestTargeting{
				MediaTypePriceGranularity: &openrtb_ext.MediaTypePriceGranularity{
					Banner: &openrtb_ext.PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges: []openrtb_ext.GranularityRange{
							{Min: 0.0, Max: 0.0, Increment: 1},
						},
					},
				},
			},
			expectedError: errors.New("Price granularity error: range list must be ordered with increasing \"max\""),
		},
		{
			name: "mediatypepricegranularity-native-invalid",
			givenTargeting: &openrtb_ext.ExtRequestTargeting{
				MediaTypePriceGranularity: &openrtb_ext.MediaTypePriceGranularity{
					Native: &openrtb_ext.PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges: []openrtb_ext.GranularityRange{
							{Min: 0.0, Max: 0.0, Increment: 1},
						},
					},
				},
			},
			expectedError: errors.New("Price granularity error: range list must be ordered with increasing \"max\""),
		},
		{
			name: "mediatypepricegranularity-video-ok-banner-invalid",
			givenTargeting: &openrtb_ext.ExtRequestTargeting{
				MediaTypePriceGranularity: &openrtb_ext.MediaTypePriceGranularity{
					Banner: &openrtb_ext.PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges: []openrtb_ext.GranularityRange{
							{Min: 0.0, Max: 10.0, Increment: -1},
						},
					},
					Video: &openrtb_ext.PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges: []openrtb_ext.GranularityRange{
							{Min: 0.0, Max: 0.0, Increment: 1},
						},
					},
				},
			},
			expectedError: errors.New("Price granularity error: range list must be ordered with increasing \"max\""),
		},
		{
			name: "mediatypepricegranularity-native-invalid-banner-ok",
			givenTargeting: &openrtb_ext.ExtRequestTargeting{
				MediaTypePriceGranularity: &openrtb_ext.MediaTypePriceGranularity{
					Native: &openrtb_ext.PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges: []openrtb_ext.GranularityRange{
							{Min: 0.0, Max: 10.0, Increment: -1},
						},
					},
					Video: &openrtb_ext.PriceGranularity{
						Precision: ptrutil.ToPtr(2),
						Ranges: []openrtb_ext.GranularityRange{
							{Min: 0.0, Max: 0.0, Increment: 1},
						},
					},
				},
			},
			expectedError: errors.New("Price granularity error: range list must be ordered with increasing \"max\""),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expectedError, validateTargeting(tc.givenTargeting))
		})
	}
}

func TestValidatePriceGranularity(t *testing.T) {
	testCases := []struct {
		description           string
		givenPriceGranularity *openrtb_ext.PriceGranularity
		expectedError         error
	}{
		{
			description: "Precision is nil",
			givenPriceGranularity: &openrtb_ext.PriceGranularity{
				Precision: nil,
			},
			expectedError: errors.New("Price granularity error: precision is required"),
		},
		{
			description: "Precision is negative",
			givenPriceGranularity: &openrtb_ext.PriceGranularity{
				Precision: ptrutil.ToPtr(-1),
			},
			expectedError: errors.New("Price granularity error: precision must be non-negative"),
		},
		{
			description: "Precision is too big",
			givenPriceGranularity: &openrtb_ext.PriceGranularity{
				Precision: ptrutil.ToPtr(20),
			},
			expectedError: errors.New("Price granularity error: precision of more than 15 significant figures is not supported"),
		},
		{
			description: "price granularity ranges out of order",
			givenPriceGranularity: &openrtb_ext.PriceGranularity{
				Precision: ptrutil.ToPtr(2),
				Ranges: []openrtb_ext.GranularityRange{
					{Min: 1.0, Max: 2.0, Increment: 0.2},
					{Min: 0.0, Max: 1.0, Increment: 0.5},
				},
			},
			expectedError: errors.New(`Price granularity error: range list must be ordered with increasing "max"`),
		},
		{
			description: "price granularity negative increment",
			givenPriceGranularity: &openrtb_ext.PriceGranularity{
				Precision: ptrutil.ToPtr(2),
				Ranges: []openrtb_ext.GranularityRange{
					{Min: 0.0, Max: 1.0, Increment: -0.1},
				},
			},
			expectedError: errors.New("Price granularity error: increment must be a nonzero positive number"),
		},
		{
			description: "price granularity correct",
			givenPriceGranularity: &openrtb_ext.PriceGranularity{
				Precision: ptrutil.ToPtr(2),
				Ranges: []openrtb_ext.GranularityRange{
					{Min: 0.0, Max: 10.0, Increment: 1},
				},
			},
			expectedError: nil,
		},
		{
			description: "price granularity with correct precision and ranges not specified",
			givenPriceGranularity: &openrtb_ext.PriceGranularity{
				Precision: ptrutil.ToPtr(2),
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			assert.Equal(t, tc.expectedError, validatePriceGranularity(tc.givenPriceGranularity))
		})
	}
}

func TestValidateOrFillChannel(t *testing.T) {
	testCases := []struct {
		description           string
		givenIsAmp            bool
		givenRequestWrapper   *openrtb_ext.RequestWrapper
		expectedError         error
		expectedChannelObject *openrtb_ext.ExtRequestPrebidChannel
	}{
		{
			description: "No request.ext info in app request, so we expect channel name to be set to app",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{App: &openrtb2.App{}},
			},
			givenIsAmp:            false,
			expectedError:         nil,
			expectedChannelObject: &openrtb_ext.ExtRequestPrebidChannel{Name: appChannel, Version: ""},
		},
		{
			description: "No request.ext info in amp request, so we expect channel name to be set to amp",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{},
			},
			givenIsAmp:            true,
			expectedError:         nil,
			expectedChannelObject: &openrtb_ext.ExtRequestPrebidChannel{Name: ampChannel, Version: ""},
		},
		{
			description: "Channel object in request with populated name/version, we expect same name/version in object that's created",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"channel": {"name": "video", "version": "1.0"}}}`)},
			},
			givenIsAmp:            false,
			expectedError:         nil,
			expectedChannelObject: &openrtb_ext.ExtRequestPrebidChannel{Name: "video", Version: "1.0"},
		},
		{
			description: "No channel object in site request, expect nil",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Site: &openrtb2.Site{}, Ext: []byte(`{"prebid":{}}`)},
			},
			givenIsAmp:            false,
			expectedError:         nil,
			expectedChannelObject: nil,
		},
		{
			description: "No channel name given in channel object, we expect error to be thrown",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{App: &openrtb2.App{}, Ext: []byte(`{"prebid":{"channel": {"name": "", "version": ""}}}`)},
			},
			givenIsAmp:            false,
			expectedError:         errors.New("ext.prebid.channel.name can't be empty"),
			expectedChannelObject: nil,
		},
		{
			description: "App request, has request.ext, no request.ext.prebid, expect channel name to be filled with app",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{App: &openrtb2.App{}, Ext: []byte(`{}`)},
			},
			givenIsAmp:            false,
			expectedError:         nil,
			expectedChannelObject: &openrtb_ext.ExtRequestPrebidChannel{Name: appChannel, Version: ""},
		},
		{
			description: "App request, has request.ext.prebid, but no channel object, expect channel name to be filled with app",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{App: &openrtb2.App{}, Ext: []byte(`{"prebid":{}}`)},
			},
			givenIsAmp:            false,
			expectedError:         nil,
			expectedChannelObject: &openrtb_ext.ExtRequestPrebidChannel{Name: appChannel, Version: ""},
		},
		{
			description: "Amp request, has request.ext, no request.ext.prebid, expect channel name to be filled with amp",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{}`)},
			},
			givenIsAmp:            true,
			expectedError:         nil,
			expectedChannelObject: &openrtb_ext.ExtRequestPrebidChannel{Name: ampChannel, Version: ""},
		},
		{
			description: "Amp request, has request.ext.prebid, but no channel object, expect channel name to be filled with amp",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{}}`)},
			},
			givenIsAmp:            true,
			expectedError:         nil,
			expectedChannelObject: &openrtb_ext.ExtRequestPrebidChannel{Name: ampChannel, Version: ""},
		},
	}

	for _, test := range testCases {
		err := validateOrFillChannel(test.givenRequestWrapper, test.givenIsAmp)
		assert.Equalf(t, test.expectedError, err, "Error doesn't match: %s\n", test.description)

		if err == nil {
			requestExt, err := test.givenRequestWrapper.GetRequestExt()
			assert.Empty(t, err, test.description)
			requestPrebid := requestExt.GetPrebid()

			assert.Equalf(t, test.expectedChannelObject, requestPrebid.Channel, "Channel information isn't correct: %s\n", test.description)
		}
	}
}

func TestSetIntegrationType(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	testCases := []struct {
		description             string
		givenRequestWrapper     *openrtb_ext.RequestWrapper
		givenAccount            *config.Account
		expectedIntegrationType string
	}{
		{
			description: "Request has integration type defined, expect that same integration type",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"integration": "TestIntegrationType"}}`)},
			},
			givenAccount:            &config.Account{DefaultIntegration: "TestDefaultIntegration"},
			expectedIntegrationType: "TestIntegrationType",
		},
		{
			description: "Request doesn't have request.ext.prebid path, expect default integration value",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(``)},
			},
			givenAccount:            &config.Account{DefaultIntegration: "TestDefaultIntegration"},
			expectedIntegrationType: "TestDefaultIntegration",
		},
		{
			description: "Request has blank integration in request, expect default integration value ",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid":{"integration": ""}}`)},
			},
			givenAccount:            &config.Account{DefaultIntegration: "TestDefaultIntegration"},
			expectedIntegrationType: "TestDefaultIntegration",
		},
	}

	for _, test := range testCases {
		err := deps.setIntegrationType(test.givenRequestWrapper, test.givenAccount)
		assert.Empty(t, err, test.description)
		integrationTypeFromReq, err2 := getIntegrationFromRequest(test.givenRequestWrapper)
		assert.Empty(t, err2, test.description)
		assert.Equalf(t, test.expectedIntegrationType, integrationTypeFromReq, "Integration type information isn't correct: %s\n", test.description)
	}
}

func TestStoredRequestGenerateUuid(t *testing.T) {
	uuid := "foo"

	deps := &endpointDeps{
		fakeUUIDGenerator{id: "foo", err: nil},
		&nobidExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	req := &openrtb2.BidRequest{}

	testCases := []struct {
		description            string
		givenRawData           string
		givenGenerateRequestID bool
		expectedID             string
		expectedCur            string
	}{
		{
			description:            "GenerateRequestID is true, rawData is an app request and has stored bid request we should generate uuid",
			givenRawData:           testBidRequests[2],
			givenGenerateRequestID: true,
			expectedID:             uuid,
		},
		{
			description:            "GenerateRequestID is true, rawData is a site request, has stored bid, and stored bidrequestID is not the macro {{UUID}}, we should not generate uuid",
			givenRawData:           testBidRequests[3],
			givenGenerateRequestID: true,
			expectedID:             "ThisID",
		},
		{
			description:            "GenerateRequestID is false, rawData is an app request and has stored bid, and stored bidrequestID is the macro {{UUID}}, so we should generate uuid",
			givenRawData:           testBidRequests[4],
			givenGenerateRequestID: false,
			expectedID:             uuid,
		},
		{
			description:            "GenerateRequestID is true, rawData is an app request, but no stored bid, we should not generate uuid",
			givenRawData:           testBidRequests[0],
			givenGenerateRequestID: true,
			expectedID:             "ThisID",
		},
		{
			description:            "GenerateRequestID is false and macro ID is not present, so we should not generate uuid",
			givenRawData:           testBidRequests[0],
			givenGenerateRequestID: false,
			expectedID:             "ThisID",
		},
		{
			description:            "GenerateRequestID is false, and rawData is a site request, and macro {{UUID}} is present, we should generate uuid",
			givenRawData:           testBidRequests[1],
			givenGenerateRequestID: false,
			expectedID:             uuid,
		},
		{
			description:            "Macro ID {{UUID}} case sensitivity check meaning a macro that is lowercase {{uuid}} shouldn't generate a uuid",
			givenRawData:           testBidRequests[2],
			givenGenerateRequestID: false,
			expectedID:             "ThisID",
		},
		{
			description:            "Test to check that stored requests are being merged properly when UUID isn't being generated",
			givenRawData:           testBidRequests[5],
			givenGenerateRequestID: false,
			expectedID:             "ThisID",
			expectedCur:            "USD",
		},
	}

	for _, test := range testCases {
		deps.cfg.GenerateRequestID = test.givenGenerateRequestID
		impInfo, errs := parseImpInfo([]byte(test.givenRawData))
		assert.Empty(t, errs, test.description)
		storedBidRequestId, hasStoredBidRequest, storedRequests, storedImps, errs := deps.getStoredRequests(context.Background(), json.RawMessage(test.givenRawData), impInfo)
		assert.Empty(t, errs, test.description)
		newRequest, _, errList := deps.processStoredRequests(json.RawMessage(test.givenRawData), impInfo, storedRequests, storedImps, storedBidRequestId, hasStoredBidRequest)
		assert.Empty(t, errList, test.description)

		if err := jsonutil.UnmarshalValid(newRequest, req); err != nil {
			t.Errorf("processStoredRequests Error: %s", err.Error())
		}
		if test.expectedCur != "" {
			assert.Equalf(t, test.expectedCur, req.Cur[0], "The stored request wasn't merged properly: %s\n", test.description)
		}
		assert.Equalf(t, test.expectedID, req.ID, "The Bid Request ID is incorrect: %s\n", test.description)
	}
}

// TestOversizedRequest makes sure we behave properly when the request size exceeds the configured max.
func TestOversizedRequest(t *testing.T) {
	reqBody := validRequest(t, "site.json")
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(len(reqBody) - 1)},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps.Auction(recorder, req, nil)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Endpoint should return a 400 if the request exceeds the size max.")
	}

	if bytesRead, err := req.Body.Read(make([]byte, 1)); bytesRead != 0 || err != io.EOF {
		t.Errorf("The request body should still be fully read.")
	}
}

// TestRequestSizeEdgeCase makes sure we behave properly when the request size *equals* the configured max.
func TestRequestSizeEdgeCase(t *testing.T) {
	reqBody := validRequest(t, "site.json")
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(len(reqBody))},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(reqBody))
	recorder := httptest.NewRecorder()

	deps.Auction(recorder, req, nil)

	if recorder.Code != http.StatusOK {
		t.Errorf("Endpoint should return a 200 if the request equals the size max.")
	}

	if bytesRead, err := req.Body.Read(make([]byte, 1)); bytesRead != 0 || err != io.EOF {
		t.Errorf("The request body should have been read to completion.")
	}
}

// TestNoEncoding prevents #231.
func TestNoEncoding(t *testing.T) {
	endpoint, _ := NewEndpoint(
		fakeUUIDGenerator{},
		&mockExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
	)
	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if !strings.Contains(recorder.Body.String(), "<script></script>") {
		t.Errorf("The Response from the exchange should not be html-encoded")
	}
}

// TestTimeoutParser makes sure we parse tmax properly.
func TestTimeoutParser(t *testing.T) {
	reqJson := json.RawMessage(`{"tmax":22}`)
	timeout := parseTimeout(reqJson, 11*time.Millisecond)
	if timeout != 22*time.Millisecond {
		t.Errorf("Failed to parse tmax properly. Expected %d, got %d", 22*time.Millisecond, timeout)
	}
}

func TestImplicitAMPNoExt(t *testing.T) {
	httpReq, err := http.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	if !assert.NoError(t, err) {
		return
	}

	reqWrapper := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
		Site: &openrtb2.Site{},
	}}

	setSiteImplicitly(httpReq, reqWrapper)

	assert.NoError(t, reqWrapper.RebuildRequest())
	assert.JSONEq(t, `{"amp":0}`, string(reqWrapper.Site.Ext))
}

func TestImplicitAMPOtherExt(t *testing.T) {
	httpReq, err := http.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	if !assert.NoError(t, err) {
		return
	}

	reqWrapper := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Ext: json.RawMessage(`{"other":true}`),
		},
	}}

	setSiteImplicitly(httpReq, reqWrapper)

	assert.NoError(t, reqWrapper.RebuildRequest())
	assert.JSONEq(t, `{"amp":0,"other":true}`, string(reqWrapper.Site.Ext))
}

func TestExplicitAMP(t *testing.T) {
	httpReq, err := http.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site-amp.json")))
	if !assert.NoError(t, err) {
		return
	}

	bidReq := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Ext: json.RawMessage(`{"amp":1}`),
		},
	}}
	setSiteImplicitly(httpReq, bidReq)
	assert.JSONEq(t, `{"amp":1}`, string(bidReq.Site.Ext))
}

// TestContentType prevents #328
func TestContentType(t *testing.T) {
	endpoint, _ := NewEndpoint(
		fakeUUIDGenerator{},
		&mockExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
	)
	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type should be application/json. Got %s", recorder.Header().Get("Content-Type"))
	}
}

func validRequest(t *testing.T, filename string) string {
	requestData, err := os.ReadFile("sample-requests/valid-whole/supplementary/" + filename)
	if err != nil {
		t.Fatalf("Failed to fetch a valid request: %v", err)
	}
	testBidRequest, _, _, err := jsonparser.Get(requestData, "mockBidRequest")
	assert.NoError(t, err, "Error jsonparsing root.mockBidRequest from file %s. Desc: %v.", filename, err)

	return string(testBidRequest)
}

func TestCurrencyTrunc(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	ui := int64(1)
	req := openrtb2.BidRequest{
		ID: "anyRequestID",
		Imp: []openrtb2.Imp{
			{
				ID: "anyImpID",
				Banner: &openrtb2.Banner{
					W: &ui,
					H: &ui,
				},
				Ext: json.RawMessage(`{"appnexus": {"placementId": 5667}}`),
			},
		},
		Site: &openrtb2.Site{
			ID: "anySiteID",
		},
		Cur: []string{"USD", "EUR"},
	}

	errL := deps.validateRequest(nil, nil, &openrtb_ext.RequestWrapper{BidRequest: &req}, false, false, nil, false)

	expectedError := errortypes.Warning{Message: "A prebid request can only process one currency. Taking the first currency in the list, USD, as the active currency"}
	assert.ElementsMatch(t, errL, []error{&expectedError})
}

func TestCCPAInvalid(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	ui := int64(1)
	req := openrtb2.BidRequest{
		ID: "anyRequestID",
		Imp: []openrtb2.Imp{
			{
				ID: "anyImpID",
				Banner: &openrtb2.Banner{
					W: &ui,
					H: &ui,
				},
				Ext: json.RawMessage(`{"appnexus": {"placementId": 5667}}`),
			},
		},
		Site: &openrtb2.Site{
			ID: "anySiteID",
		},
		Regs: &openrtb2.Regs{
			USPrivacy: "invalid by length",
		},
	}

	errL := deps.validateRequest(nil, nil, &openrtb_ext.RequestWrapper{BidRequest: &req}, false, false, nil, false)

	expectedWarning := errortypes.Warning{
		Message:     "CCPA consent is invalid and will be ignored. (request.regs.ext.us_privacy must contain 4 characters)",
		WarningCode: errortypes.InvalidPrivacyConsentWarningCode}
	assert.ElementsMatch(t, errL, []error{&expectedWarning})
}

func TestNoSaleInvalid(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	ui := int64(1)
	req := openrtb2.BidRequest{
		ID: "anyRequestID",
		Imp: []openrtb2.Imp{
			{
				ID: "anyImpID",
				Banner: &openrtb2.Banner{
					W: &ui,
					H: &ui,
				},
				Ext: json.RawMessage(`{"appnexus": {"placementId": 5667}}`),
			},
		},
		Site: &openrtb2.Site{
			ID: "anySiteID",
		},
		Regs: &openrtb2.Regs{
			Ext: json.RawMessage(`{"us_privacy": "1NYN"}`),
		},
		Ext: json.RawMessage(`{"prebid": {"nosale": ["*", "appnexus"]} }`),
	}

	errL := deps.validateRequest(nil, nil, &openrtb_ext.RequestWrapper{BidRequest: &req}, false, false, nil, false)

	expectedError := errors.New("request.ext.prebid.nosale is invalid: can only specify all bidders if no other bidders are provided")
	assert.ElementsMatch(t, errL, []error{expectedError})
}

func TestValidateSourceTID(t *testing.T) {
	cfg := &config.Configuration{
		AutoGenSourceTID: true,
	}

	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		cfg,
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	ui := int64(1)
	req := openrtb2.BidRequest{
		ID: "anyRequestID",
		Imp: []openrtb2.Imp{
			{
				ID: "anyImpID",
				Banner: &openrtb2.Banner{
					W: &ui,
					H: &ui,
				},
				Ext: json.RawMessage(`{"appnexus": {"placementId": 5667}}`),
			},
		},
		Site: &openrtb2.Site{
			ID: "anySiteID",
		},
	}

	deps.validateRequest(nil, nil, &openrtb_ext.RequestWrapper{BidRequest: &req}, false, false, nil, false)
	assert.NotEmpty(t, req.Source.TID, "Expected req.Source.TID to be filled with a randomly generated UID")
}

func TestSChainInvalid(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	ui := int64(1)
	req := openrtb2.BidRequest{
		ID: "anyRequestID",
		Imp: []openrtb2.Imp{
			{
				ID: "anyImpID",
				Banner: &openrtb2.Banner{
					W: &ui,
					H: &ui,
				},
				Ext: json.RawMessage(`{"appnexus": {"placementId": 5667}}`),
			},
		},
		Site: &openrtb2.Site{
			ID: "anySiteID",
		},
		Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],"schain":{"complete":1,"nodes":[{"asi":"directseller1.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}, {"bidders":["appnexus"],"schain":{"complete":1,"nodes":[{"asi":"directseller2.com","sid":"00002","rid":"BidRequest2","hp":1}],"ver":"1.0"}}]}}`),
	}

	errL := deps.validateRequest(nil, nil, &openrtb_ext.RequestWrapper{BidRequest: &req}, false, false, nil, false)

	expectedError := errors.New("request.ext.prebid.schains contains multiple schains for bidder appnexus; it must contain no more than one per bidder.")
	assert.ElementsMatch(t, errL, []error{expectedError})
}

func TestSearchAccountID(t *testing.T) {
	// Correctness for lookup within Publisher object left to TestGetAccountID
	// This however tests the expected lookup paths in outer site, app and dooh
	testCases := []struct {
		description       string
		request           []byte
		expectedAccID     string
		expectedError     error
		expectedIsAppReq  bool
		expectedIsSiteReq bool
		expectedIsDOOHReq bool
	}{
		{
			description:       "No publisher available",
			request:           []byte(`{}`),
			expectedAccID:     "",
			expectedError:     nil,
			expectedIsAppReq:  false,
			expectedIsDOOHReq: false,
		},
		{
			description:       "Publisher.ID doesn't exist",
			request:           []byte(`{"site":{"publisher":{}}}`),
			expectedAccID:     "",
			expectedError:     nil,
			expectedIsAppReq:  false,
			expectedIsDOOHReq: false,
		},
		{
			description:       "Publisher.ID not a string",
			request:           []byte(`{"site":{"publisher":{"id":42}}}`),
			expectedAccID:     "",
			expectedError:     errors.New("site.publisher.id must be a string"),
			expectedIsAppReq:  false,
			expectedIsDOOHReq: false,
		},
		{
			description:       "Publisher available in request.site",
			request:           []byte(`{"site":{"publisher":{"id":"42"}}}`),
			expectedAccID:     "42",
			expectedError:     nil,
			expectedIsAppReq:  false,
			expectedIsDOOHReq: false,
		},
		{
			description:       "Publisher available in request.app",
			request:           []byte(`{"app":{"publisher":{"id":"42"}}}`),
			expectedAccID:     "42",
			expectedError:     nil,
			expectedIsAppReq:  true,
			expectedIsDOOHReq: false,
		},
		{
			description:       "Publisher available in request.dooh",
			request:           []byte(`{"dooh":{"publisher":{"id":"42"}}}`),
			expectedAccID:     "42",
			expectedError:     nil,
			expectedIsAppReq:  false,
			expectedIsDOOHReq: true,
		},
	}

	for _, test := range testCases {
		accountId, isAppReq, isDOOHReq, err := searchAccountId(test.request)
		assert.Equal(t, test.expectedAccID, accountId, "searchAccountID should return expected account ID for test case: %s", test.description)
		assert.Equal(t, test.expectedIsAppReq, isAppReq, "searchAccountID should return expected isAppReq for test case: %s", test.description)
		assert.Equal(t, test.expectedIsDOOHReq, isDOOHReq, "searchAccountID should return expected isDOOHReq for test case: %s", test.description)
		assert.Equal(t, test.expectedError, err, "searchAccountID should return expected error for test case: %s", test.description)
	}

}

func TestGetAccountID(t *testing.T) {
	testPubID := "test-pub"
	testParentAccount := "test-account"
	testPubExt := openrtb_ext.ExtPublisher{
		Prebid: &openrtb_ext.ExtPublisherPrebid{
			ParentAccount: &testParentAccount,
		},
	}
	testPubExtJSON, err := jsonutil.Marshal(testPubExt)
	assert.NoError(t, err)

	testCases := []struct {
		description   string
		pub           *openrtb2.Publisher
		expectedAccID string
	}{
		{
			description: "Publisher.ID and Publisher.Ext.Prebid.ParentAccount both present",
			pub: &openrtb2.Publisher{
				ID:  testPubID,
				Ext: testPubExtJSON,
			},
			expectedAccID: testParentAccount,
		},
		{
			description: "Only Publisher.Ext.Prebid.ParentAccount present",
			pub: &openrtb2.Publisher{
				ID:  "",
				Ext: testPubExtJSON,
			},
			expectedAccID: testParentAccount,
		},
		{
			description: "Only Publisher.ID present",
			pub: &openrtb2.Publisher{
				ID: testPubID,
			},
			expectedAccID: testPubID,
		},
		{
			description:   "Neither Publisher.ID or Publisher.Ext.Prebid.ParentAccount present",
			pub:           &openrtb2.Publisher{},
			expectedAccID: metrics.PublisherUnknown,
		},
		{
			description:   "Publisher is nil",
			pub:           nil,
			expectedAccID: metrics.PublisherUnknown,
		},
	}

	for _, test := range testCases {
		acc := getAccountID(test.pub)
		assert.Equal(t, test.expectedAccID, acc, "getAccountID should return expected account for test case: %s", test.description)
	}
}

func TestSanitizeRequest(t *testing.T) {
	testCases := []struct {
		description  string
		req          *openrtb2.BidRequest
		ipValidator  iputil.IPValidator
		expectedIPv4 string
		expectedIPv6 string
	}{
		{
			description: "Empty",
			req: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					IP:   "",
					IPv6: "",
				},
			},
			expectedIPv4: "",
			expectedIPv6: "",
		},
		{
			description: "Valid",
			req: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					IP:   "1.1.1.1",
					IPv6: "1111::",
				},
			},
			ipValidator:  hardcodedResponseIPValidator{response: true},
			expectedIPv4: "1.1.1.1",
			expectedIPv6: "1111::",
		},
		{
			description: "Invalid",
			req: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					IP:   "1.1.1.1",
					IPv6: "1111::",
				},
			},
			ipValidator:  hardcodedResponseIPValidator{response: false},
			expectedIPv4: "",
			expectedIPv6: "",
		},
		{
			description: "Invalid - Wrong IP Types",
			req: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					IP:   "1111::",
					IPv6: "1.1.1.1",
				},
			},
			ipValidator:  hardcodedResponseIPValidator{response: true},
			expectedIPv4: "",
			expectedIPv6: "",
		},
		{
			description: "Malformed",
			req: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					IP:   "malformed",
					IPv6: "malformed",
				},
			},
			expectedIPv4: "",
			expectedIPv6: "",
		},
	}

	for _, test := range testCases {
		bidReq := &openrtb_ext.RequestWrapper{BidRequest: test.req}

		sanitizeRequest(bidReq, test.ipValidator)
		assert.Equal(t, test.expectedIPv4, test.req.Device.IP, test.description+":ipv4")
		assert.Equal(t, test.expectedIPv6, test.req.Device.IPv6, test.description+":ipv6")
	}
}

func TestValidateAndFillSourceTID(t *testing.T) {
	testTID := "some-tid"
	testCases := []struct {
		description         string
		req                 *openrtb_ext.RequestWrapper
		generateRequestID   bool
		hasStoredBidRequest bool
		isAmp               bool
		expectRandImpTID    bool
		expectRandSourceTID bool
		expectSourceTid     *string
		expectImpTid        *string
	}{
		{
			description: "req source.tid not set, expect random value",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:     "1",
					Imp:    []openrtb2.Imp{{ID: "1"}},
					Source: &openrtb2.Source{},
				},
			},
			generateRequestID:   false,
			hasStoredBidRequest: false,
			isAmp:               false,
			expectRandSourceTID: true,
			expectRandImpTID:    false,
		},
		{
			description: "req source.tid set to {{UUID}}, expect to be replaced by random value",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:     "1",
					Imp:    []openrtb2.Imp{{ID: "1"}},
					Source: &openrtb2.Source{TID: "{{UUID}}"},
				},
			},
			generateRequestID:   false,
			hasStoredBidRequest: false,
			isAmp:               false,
			expectRandSourceTID: true,
			expectRandImpTID:    false,
		},
		{
			description: "req source.tid is set, isAmp = true, generateRequestID = true, expect to be replaced by random value",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:     "1",
					Imp:    []openrtb2.Imp{{ID: "1"}},
					Source: &openrtb2.Source{TID: "test-tid"},
				},
			},
			generateRequestID:   true,
			hasStoredBidRequest: false,
			isAmp:               true,
			expectRandSourceTID: true,
			expectRandImpTID:    false,
		},
		{
			description: "req source.tid is set,  hasStoredBidRequest = true, generateRequestID = true, expect to be replaced by random value",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:     "1",
					Imp:    []openrtb2.Imp{{ID: "1"}},
					Source: &openrtb2.Source{TID: "test-tid"},
				},
			},
			generateRequestID:   true,
			hasStoredBidRequest: true,
			isAmp:               false,
			expectRandSourceTID: true,
			expectRandImpTID:    false,
		},
		{
			description: "req source.tid is set,  hasStoredBidRequest = true, generateRequestID = false, expect NOT to be replaced by random value",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:     "1",
					Imp:    []openrtb2.Imp{{ID: "1"}},
					Source: &openrtb2.Source{TID: testTID},
				},
			},
			generateRequestID:   false,
			hasStoredBidRequest: true,
			isAmp:               false,
			expectRandSourceTID: false,
			expectRandImpTID:    false,
			expectSourceTid:     &testTID,
		},
		{
			description: "req imp.ext.tid not set, expect random value",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:     "1",
					Imp:    []openrtb2.Imp{{ID: "1"}},
					Source: &openrtb2.Source{},
				},
			},
			generateRequestID:   false,
			hasStoredBidRequest: false,
			isAmp:               false,
			expectRandSourceTID: false,
			expectRandImpTID:    true,
		},
		{
			description: "req imp.ext.tid set to {{UUID}}, expect random value",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:     "1",
					Imp:    []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"tid": "{{UUID}}"}`)}},
					Source: &openrtb2.Source{},
				},
			},
			generateRequestID:   false,
			hasStoredBidRequest: false,
			isAmp:               false,
			expectRandSourceTID: false,
			expectRandImpTID:    true,
		},
		{
			description: "req imp.tid is set,  hasStoredBidRequest = true, generateRequestID = true, expect to be replaced by random value",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:     "1",
					Imp:    []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"tid": "some-tid"}`)}},
					Source: &openrtb2.Source{TID: "test-tid"},
				},
			},
			generateRequestID:   true,
			hasStoredBidRequest: true,
			isAmp:               false,
			expectRandSourceTID: false,
			expectRandImpTID:    true,
		},
		{
			description: "req imp.tid is set,  isAmp = true, generateRequestID = true, expect to be replaced by random value",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:     "1",
					Imp:    []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"tid": "some-tid"}`)}},
					Source: &openrtb2.Source{TID: "test-tid"},
				},
			},
			generateRequestID:   true,
			hasStoredBidRequest: false,
			isAmp:               true,
			expectRandSourceTID: false,
			expectRandImpTID:    true,
		},
		{
			description: "req imp.tid is set,  hasStoredBidRequest = true, generateRequestID = false, expect NOT to be replaced by random value",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:     "1",
					Imp:    []openrtb2.Imp{{ID: "1", Ext: json.RawMessage(`{"tid": "some-tid"}`)}},
					Source: &openrtb2.Source{TID: testTID},
				},
			},
			generateRequestID:   false,
			hasStoredBidRequest: true,
			isAmp:               false,
			expectRandSourceTID: false,
			expectRandImpTID:    false,
			expectImpTid:        &testTID,
		},
	}

	for _, test := range testCases {
		_ = validateAndFillSourceTID(test.req, test.generateRequestID, test.hasStoredBidRequest, test.isAmp)
		impWrapper := &openrtb_ext.ImpWrapper{}
		impWrapper.Imp = &test.req.Imp[0]
		ie, _ := impWrapper.GetImpExt()
		impTID := ie.GetTid()
		if test.expectRandSourceTID {
			assert.NotEmpty(t, test.req.Source.TID, test.description)
		} else if test.expectRandImpTID {
			assert.NotEqual(t, testTID, impTID, test.description)
			assert.NotEmpty(t, impTID, test.description)
		} else if test.expectSourceTid != nil {
			assert.Equal(t, test.req.Source.TID, *test.expectSourceTid, test.description)
		} else if test.expectImpTid != nil {
			assert.Equal(t, impTID, *test.expectImpTid, test.description)
		}
	}
}

func TestEidPermissionsInvalid(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	ui := int64(1)
	req := openrtb2.BidRequest{
		ID: "anyRequestID",
		Imp: []openrtb2.Imp{
			{
				ID: "anyImpID",
				Banner: &openrtb2.Banner{
					W: &ui,
					H: &ui,
				},
				Ext: json.RawMessage(`{"appnexus": {"placementId": 5667}}`),
			},
		},
		Site: &openrtb2.Site{
			ID: "anySiteID",
		},
		Ext: json.RawMessage(`{"prebid": {"data": {"eidpermissions": [{"source":"a", "bidders":[]}]} } }`),
	}

	errL := deps.validateRequest(nil, nil, &openrtb_ext.RequestWrapper{BidRequest: &req}, false, false, nil, false)

	expectedError := errors.New(`request.ext.prebid.data.eidpermissions[0] missing or empty required field: "bidders"`)
	assert.ElementsMatch(t, errL, []error{expectedError})
}

func TestValidateEidPermissions(t *testing.T) {
	knownBidders := map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")}
	knownAliases := map[string]string{"b": "b"}

	testCases := []struct {
		name          string
		request       *openrtb_ext.ExtRequest
		expectedError error
	}{
		{
			name:          "valid-empty-ext",
			request:       &openrtb_ext.ExtRequest{},
			expectedError: nil,
		},
		{
			name:          "valid-nil-ext.prebid.data",
			request:       &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{}},
			expectedError: nil,
		},
		{
			name:          "valid-empty-ext.prebid.data",
			request:       &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{}}},
			expectedError: nil,
		},
		{
			name:          "valid-nil-ext.prebid.data.eidpermissions",
			request:       &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: nil}}},
			expectedError: nil,
		},
		{
			name:          "valid-none",
			request:       &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{}}}},
			expectedError: nil,
		},
		{
			name: "valid-one",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"a"}},
			}}}},
			expectedError: nil,
		},
		{
			name: "valid-one-case-insensitive",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"A"}},
			}}}},
			expectedError: nil,
		},
		{
			name: "valid-many",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"a"}},
				{Source: "sourceB", Bidders: []string{"a"}},
			}}}},
			expectedError: nil,
		},
		{
			name: "invalid-missing-source",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"a"}},
				{Bidders: []string{"a"}},
			}}}},
			expectedError: errors.New(`request.ext.prebid.data.eidpermissions[1] missing required field: "source"`),
		},
		{
			name: "invalid-duplicate-source",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"a"}},
				{Source: "sourceA", Bidders: []string{"a"}},
			}}}},
			expectedError: errors.New(`request.ext.prebid.data.eidpermissions[1] duplicate entry with field: "source"`),
		},
		{
			name: "invalid-missing-bidders-nil",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"a"}},
				{Source: "sourceB"},
			}}}},
			expectedError: errors.New(`request.ext.prebid.data.eidpermissions[1] missing or empty required field: "bidders"`),
		},
		{
			name: "invalid-missing-bidders-empty",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"a"}},
				{Source: "sourceB", Bidders: []string{}},
			}}}},
			expectedError: errors.New(`request.ext.prebid.data.eidpermissions[1] missing or empty required field: "bidders"`),
		},
		{
			name: "invalid-invalid-bidders",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"a"}},
				{Source: "sourceB", Bidders: []string{"z"}},
			}}}},
			expectedError: errors.New(`request.ext.prebid.data.eidpermissions[1] contains unrecognized bidder "z"`),
		},
		{
			name: "invalid-alias-case-sensitive",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"B"}},
			}}}},
			expectedError: errors.New(`request.ext.prebid.data.eidpermissions[0] contains unrecognized bidder "B"`),
		},
	}

	endpoint := &endpointDeps{bidderMap: knownBidders, normalizeBidderName: fakeNormalizeBidderName}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := endpoint.validateEidPermissions(test.request.Prebid.Data, knownAliases)
			assert.Equal(t, test.expectedError, result)
		})
	}
}

func TestValidateBidders(t *testing.T) {
	testCases := []struct {
		description   string
		bidders       []string
		knownBidders  map[string]openrtb_ext.BidderName
		knownAliases  map[string]string
		expectedError error
	}{
		{
			description:   "Valid - No Bidders",
			bidders:       []string{},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")},
			knownAliases:  map[string]string{"c": "c"},
			expectedError: nil,
		},
		{
			description:   "Valid - All Bidders",
			bidders:       []string{"*"},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")},
			knownAliases:  map[string]string{"c": "c"},
			expectedError: nil,
		},
		{
			description:   "Valid - One Core Bidder",
			bidders:       []string{"a"},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")},
			knownAliases:  map[string]string{"c": "c"},
			expectedError: nil,
		},
		{
			description:   "Valid - One Core Bidder - Case Insensitive",
			bidders:       []string{"A"},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")},
			knownAliases:  map[string]string{"c": "c"},
			expectedError: nil,
		},
		{
			description:   "Valid - Many Core Bidders",
			bidders:       []string{"a", "b"},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a"), "b": openrtb_ext.BidderName("b")},
			knownAliases:  map[string]string{"c": "c"},
			expectedError: nil,
		},
		{
			description:   "Valid - One Alias Bidder",
			bidders:       []string{"c"},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")},
			knownAliases:  map[string]string{"c": "c"},
			expectedError: nil,
		},
		{
			description:   "Valid - One Alias Bidder - Case Sensitive",
			bidders:       []string{"C"},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")},
			knownAliases:  map[string]string{"c": "c"},
			expectedError: errors.New(`unrecognized bidder "C"`),
		},
		{
			description:   "Valid - Many Alias Bidders",
			bidders:       []string{"c", "d"},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")},
			knownAliases:  map[string]string{"c": "c", "d": "d"},
			expectedError: nil,
		},
		{
			description:   "Valid - Mixed Core + Alias Bidders",
			bidders:       []string{"a", "c"},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")},
			knownAliases:  map[string]string{"c": "c"},
			expectedError: nil,
		},
		{
			description:   "Invalid - Unknown Bidder",
			bidders:       []string{"z"},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")},
			knownAliases:  map[string]string{"c": "c"},
			expectedError: errors.New(`unrecognized bidder "z"`),
		},
		{
			description:   "Invalid - Unknown Bidder With Known Bidders",
			bidders:       []string{"a", "c", "z"},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")},
			knownAliases:  map[string]string{"c": "c"},
			expectedError: errors.New(`unrecognized bidder "z"`),
		},
		{
			description:   "Invalid - All Bidders With Known Bidder",
			bidders:       []string{"*", "a"},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")},
			knownAliases:  map[string]string{"c": "c"},
			expectedError: errors.New(`bidder wildcard "*" mixed with specific bidders`),
		},
		{
			description:   "Invalid - Returns First Error - All Bidders",
			bidders:       []string{"*", "z"},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")},
			knownAliases:  map[string]string{"c": "c"},
			expectedError: errors.New(`bidder wildcard "*" mixed with specific bidders`),
		},
		{
			description:   "Invalid - Returns First Error - Unknown Bidder",
			bidders:       []string{"z", "*"},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")},
			knownAliases:  map[string]string{"c": "c"},
			expectedError: errors.New(`unrecognized bidder "z"`),
		},
	}

	endpoint := &endpointDeps{normalizeBidderName: fakeNormalizeBidderName}
	for _, test := range testCases {
		result := endpoint.validateBidders(test.bidders, test.knownBidders, test.knownAliases)
		assert.Equal(t, test.expectedError, result, test.description)
	}
}

func TestIOS14EndToEnd(t *testing.T) {
	exchange := &nobidExchange{}

	endpoint, _ := NewEndpoint(
		fakeUUIDGenerator{},
		exchange,
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
	)

	httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "app-ios140-no-ifa.json")))

	endpoint(httptest.NewRecorder(), httpReq, nil)

	result := exchange.gotRequest
	if !assert.NotEmpty(t, result, "request received by the exchange.") {
		t.FailNow()
	}

	var lmtOne int8 = 1
	assert.Equal(t, &lmtOne, result.Device.Lmt)
}

func TestAuctionWarnings(t *testing.T) {
	testCases := []struct {
		name            string
		file            string
		expectedWarning string
	}{
		{
			name:            "us-privacy-invalid",
			file:            "us-privacy-invalid.json",
			expectedWarning: "CCPA consent is invalid and will be ignored. (request.regs.ext.us_privacy must contain 4 characters)",
		},
		{
			name:            "us-privacy-signals-conflict",
			file:            "us-privacy-conflict.json",
			expectedWarning: "regs.us_privacy consent does not match uspv1 in GPP, using regs.gpp",
		},
		{
			name:            "empty-gppsid-array-conflicts-with-regs-gdpr", // gdpr set to 1, an empty non-nil gpp_sid array doesn't match
			file:            "empty-gppsid-conflict.json",
			expectedWarning: "regs.gdpr signal conflicts with GPP (regs.gpp_sid) and will be ignored",
		},
		{
			name:            "gdpr-signals-conflict", // gdpr signals do not match
			file:            "gdpr-conflict.json",
			expectedWarning: "regs.gdpr signal conflicts with GPP (regs.gpp_sid) and will be ignored",
		},
		{
			name:            "gdpr-signals-conflict2", // gdpr consent strings do not match
			file:            "gdpr-conflict2.json",
			expectedWarning: "user.consent GDPR string conflicts with GPP (regs.gpp) GDPR string, using regs.gpp",
		},
	}
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&warningsCheckExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			reqBody := validRequest(t, test.file)
			req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(reqBody))
			recorder := httptest.NewRecorder()

			deps.Auction(recorder, req, nil)

			if recorder.Code != http.StatusOK {
				t.Errorf("Endpoint should return a 200")
			}
			warnings := deps.ex.(*warningsCheckExchange).auctionRequest.Warnings
			if !assert.Len(t, warnings, 1, "One warning should be returned from exchange") {
				t.FailNow()
			}
			actualWarning := warnings[0].(*errortypes.Warning)
			assert.Equal(t, test.expectedWarning, actualWarning.Message, "Warning message is incorrect")

			assert.Equal(t, errortypes.InvalidPrivacyConsentWarningCode, actualWarning.WarningCode, "Warning code is incorrect")
		})
	}
}

func TestParseRequestParseImpInfoError(t *testing.T) {
	reqBody := validRequest(t, "imp-info-invalid.json")
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&warningsCheckExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(len(reqBody))},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	hookExecutor := hookexecution.NewHookExecutor(deps.hookExecutionPlanBuilder, hookexecution.EndpointAuction, deps.metricsEngine)

	req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(reqBody))

	resReq, impExtInfoMap, _, _, _, _, errL := deps.parseRequest(req, &metrics.Labels{}, hookExecutor)

	assert.Nil(t, resReq, "Result request should be nil due to incorrect imp")
	assert.Nil(t, impExtInfoMap, "Impression info map should be nil due to incorrect imp")
	assert.Len(t, errL, 1, "One error should be returned")
	assert.Contains(t, errL[0].Error(), "cannot unmarshal openrtb_ext.Options.EchoVideoAttrs", "Incorrect error message")
}

func TestParseGzipedRequest(t *testing.T) {
	testCases :=
		[]struct {
			desc           string
			reqContentEnc  string
			maxReqSize     int64
			compressionCfg config.Compression
			expectedErr    string
		}{
			{
				desc:           "Gzip compression enabled, request size exceeds max request size",
				reqContentEnc:  "gzip",
				maxReqSize:     10,
				compressionCfg: config.Compression{Request: config.CompressionInfo{GZIP: true}},
				expectedErr:    "request size exceeded max size of 10 bytes.",
			},
			{
				desc:           "Gzip compression enabled, request size is within max request size",
				reqContentEnc:  "gzip",
				maxReqSize:     2000,
				compressionCfg: config.Compression{Request: config.CompressionInfo{GZIP: true}},
				expectedErr:    "",
			},
			{
				desc:           "Gzip compression enabled, request size is within max request size, content-encoding value not in lower case",
				reqContentEnc:  "GZIP",
				maxReqSize:     2000,
				compressionCfg: config.Compression{Request: config.CompressionInfo{GZIP: true}},
				expectedErr:    "",
			},
			{
				desc:           "Request is Gzip compressed, but Gzip compression is disabled",
				reqContentEnc:  "gzip",
				compressionCfg: config.Compression{Request: config.CompressionInfo{GZIP: false}},
				expectedErr:    "Content-Encoding of type gzip is not supported",
			},
			{
				desc:           "Request is not Gzip compressed, but Gzip compression is enabled",
				reqContentEnc:  "",
				maxReqSize:     2000,
				compressionCfg: config.Compression{Request: config.CompressionInfo{GZIP: true}},
				expectedErr:    "",
			},
		}

	reqBody := []byte(validRequest(t, "site.json"))
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&warningsCheckExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(50), Compression: config.Compression{Request: config.CompressionInfo{GZIP: false}}},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	hookExecutor := hookexecution.NewHookExecutor(deps.hookExecutionPlanBuilder, hookexecution.EndpointAuction, deps.metricsEngine)
	for _, test := range testCases {
		var req *http.Request
		deps.cfg.MaxRequestSize = test.maxReqSize
		deps.cfg.Compression = test.compressionCfg
		if test.reqContentEnc == "gzip" {
			var compressed bytes.Buffer
			gw := gzip.NewWriter(&compressed)
			_, err := gw.Write(reqBody)
			assert.NoError(t, err, "Error writing gzip compressed request body", test.desc)
			assert.NoError(t, gw.Close(), "Error closing gzip writer", test.desc)

			req = httptest.NewRequest("POST", "/openrtb2/auction", bytes.NewReader(compressed.Bytes()))
			req.Header.Set("Content-Encoding", "gzip")
		} else {
			req = httptest.NewRequest("POST", "/openrtb2/auction", bytes.NewReader(reqBody))
		}
		resReq, impExtInfoMap, _, _, _, _, errL := deps.parseRequest(req, &metrics.Labels{}, hookExecutor)

		if test.expectedErr == "" {
			assert.Nil(t, errL, "Error list should be nil", test.desc)
			assert.NotNil(t, resReq, "Result request should not be nil", test.desc)
			assert.NotNil(t, impExtInfoMap, "Impression info map should not be nil", test.desc)
		} else {
			assert.Nil(t, resReq, "Result request should be nil due to incorrect imp", test.desc)
			assert.Nil(t, impExtInfoMap, "Impression info map should be nil due to incorrect imp", test.desc)
			assert.Len(t, errL, 1, "One error should be returned", test.desc)
			assert.Contains(t, errL[0].Error(), test.expectedErr, "Incorrect error message", test.desc)
		}
	}
}

func TestAuctionResponseHeaders(t *testing.T) {
	testCases := []struct {
		description     string
		httpRequest     *http.Request
		expectedStatus  int
		expectedHeaders func(http.Header)
	}{
		{
			description:    "Success Response",
			httpRequest:    httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json"))),
			expectedStatus: 200,
			expectedHeaders: func(h http.Header) {
				h.Set("X-Prebid", "pbs-go/unknown")
				h.Set("Content-Type", "application/json")
			},
		},
		{
			description:    "Failure Response",
			httpRequest:    httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader("{}")),
			expectedStatus: 400,
			expectedHeaders: func(h http.Header) {
				h.Set("X-Prebid", "pbs-go/unknown")
			},
		},
		{
			description: "Success Response with Chrome BrowsingTopicsHeader",
			httpRequest: func() *http.Request {
				httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
				httpReq.Header.Add(secBrowsingTopics, "sample-value")
				return httpReq
			}(),
			expectedStatus: 200,
			expectedHeaders: func(h http.Header) {
				h.Set("X-Prebid", "pbs-go/unknown")
				h.Set("Content-Type", "application/json")
				h.Set("Observe-Browsing-Topics", "?1")
			},
		},
		{
			description: "Failure Response with Chrome BrowsingTopicsHeader",
			httpRequest: func() *http.Request {
				httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader("{}"))
				httpReq.Header.Add(secBrowsingTopics, "sample-value")
				return httpReq
			}(),
			expectedStatus: 400,
			expectedHeaders: func(h http.Header) {
				h.Set("X-Prebid", "pbs-go/unknown")
				h.Set("Observe-Browsing-Topics", "?1")
			},
		},
	}

	exchange := &nobidExchange{}
	endpoint, _ := NewEndpoint(
		fakeUUIDGenerator{},
		exchange,
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
	)

	for _, test := range testCases {
		recorder := httptest.NewRecorder()

		endpoint(recorder, test.httpRequest, nil)

		expectedHeaders := http.Header{}
		test.expectedHeaders(expectedHeaders)

		assert.Equal(t, test.expectedStatus, recorder.Result().StatusCode, test.description+":statuscode")
		assert.Equal(t, expectedHeaders, recorder.Result().Header, test.description+":statuscode")
	}
}

// StoredRequest testing

// Test stored request data

func TestParseRequestMergeBidderParams(t *testing.T) {
	tests := []struct {
		name               string
		givenRequestBody   string
		expectedImpExt     json.RawMessage
		expectedReqExt     json.RawMessage
		expectedErrorCount int
		expectedErrors     []error
	}{
		{
			name:               "add missing bidder-params from req.ext.prebid.bidderparams to imp[].ext.prebid.bidder",
			givenRequestBody:   validRequest(t, "req-ext-bidder-params.json"),
			expectedImpExt:     getObject(t, "req-ext-bidder-params.json", "expectedImpExt"),
			expectedReqExt:     getObject(t, "req-ext-bidder-params.json", "expectedReqExt"),
			expectedErrorCount: 0,
		},
		{
			name:               "add missing bidder-params from req.ext.prebid.bidderparams to imp[].ext.prebid.bidder with preference for imp[].ext.prebid.bidder params",
			givenRequestBody:   validRequest(t, "req-ext-bidder-params-merge.json"),
			expectedImpExt:     getObject(t, "req-ext-bidder-params-merge.json", "expectedImpExt"),
			expectedReqExt:     getObject(t, "req-ext-bidder-params-merge.json", "expectedReqExt"),
			expectedErrorCount: 0,
		},
		{
			name:               "add missing bidder-params from req.ext.prebid.bidderparams to imp[].ext for backward compatibility",
			givenRequestBody:   validRequest(t, "req-ext-bidder-params-promotion.json"),
			expectedImpExt:     getObject(t, "req-ext-bidder-params-promotion.json", "expectedImpExt"),
			expectedReqExt:     getObject(t, "req-ext-bidder-params-promotion.json", "expectedReqExt"),
			expectedErrorCount: 1,
			expectedErrors: []error{
				&errortypes.Warning{
					WarningCode: 0,
					Message:     "request.imp[0].ext contains unknown bidder: 'arbitraryObject', ignoring",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			deps := &endpointDeps{
				fakeUUIDGenerator{},
				&warningsCheckExchange{},
				ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
				&mockStoredReqFetcher{},
				empty_fetcher.EmptyFetcher{},
				empty_fetcher.EmptyFetcher{},
				&config.Configuration{MaxRequestSize: int64(len(test.givenRequestBody))},
				&metricsConfig.NilMetricsEngine{},
				analyticsBuild.New(&config.Analytics{}),
				map[string]string{},
				false,
				[]byte{},
				openrtb_ext.BuildBidderMap(),
				nil,
				nil,
				hardcodedResponseIPValidator{response: true},
				empty_fetcher.EmptyFetcher{},
				hooks.EmptyPlanBuilder{},
				nil,
				openrtb_ext.NormalizeBidderName,
			}

			hookExecutor := hookexecution.NewHookExecutor(deps.hookExecutionPlanBuilder, hookexecution.EndpointAuction, deps.metricsEngine)

			req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(test.givenRequestBody))

			resReq, _, _, _, _, _, errL := deps.parseRequest(req, &metrics.Labels{}, hookExecutor)

			assert.NoError(t, resReq.RebuildRequest())

			var expIExt, iExt map[string]interface{}
			err := jsonutil.UnmarshalValid(test.expectedImpExt, &expIExt)
			assert.Nil(t, err, "unmarshal() should return nil error")

			assert.NotNil(t, resReq.BidRequest.Imp[0].Ext, "imp[0].Ext should not be nil")
			err = jsonutil.UnmarshalValid(resReq.BidRequest.Imp[0].Ext, &iExt)
			assert.Nil(t, err, "unmarshal() should return nil error")

			assert.Equal(t, expIExt, iExt, "bidderparams in imp[].Ext should match")

			var eReqE, reqE map[string]interface{}
			err = jsonutil.UnmarshalValid(test.expectedReqExt, &eReqE)
			assert.Nil(t, err, "unmarshal() should return nil error")

			err = jsonutil.UnmarshalValid(resReq.BidRequest.Ext, &reqE)
			assert.Nil(t, err, "unmarshal() should return nil error")

			assert.Equal(t, eReqE, reqE, "req.Ext should match")

			assert.Len(t, errL, test.expectedErrorCount, "error length should match")

			assert.Equal(t, errL, test.expectedErrors)
		})
	}
}

func TestParseRequestStoredResponses(t *testing.T) {
	mockStoredResponses := map[string]json.RawMessage{
		"6d718149": json.RawMessage(`[{"bid": [{"id": "bid_id1"],"seat": "appnexus"}]`),
		"6d715835": json.RawMessage(`[{"bid": [{"id": "bid_id2"],"seat": "appnexus"}]`),
	}

	tests := []struct {
		name                    string
		givenRequestBody        string
		expectedStoredResponses stored_responses.ImpsWithBidResponses
		expectedErrorCount      int
		expectedError           string
	}{
		{
			name:             "req imp has valid stored response",
			givenRequestBody: validRequest(t, "req-imp-stored-response.json"),
			expectedStoredResponses: map[string]json.RawMessage{
				"imp-id1": json.RawMessage(`[{"bid": [{"id": "bid_id1"],"seat": "appnexus"}]`),
			},
			expectedErrorCount: 0,
		},
		{
			name:             "req has two imps valid stored responses",
			givenRequestBody: validRequest(t, "req-two-imps-stored-response.json"),
			expectedStoredResponses: map[string]json.RawMessage{
				"imp-id1": json.RawMessage(`[{"bid": [{"id": "bid_id1"],"seat": "appnexus"}]`),
				"imp-id2": json.RawMessage(`[{"bid": [{"id": "bid_id2"],"seat": "appnexus"}]`),
			},
			expectedErrorCount: 0,
		},
		{
			name:                    "req has two imps with missing stored responses",
			givenRequestBody:        validRequest(t, "req-two-imps-missing-stored-response.json"),
			expectedStoredResponses: nil,
			expectedErrorCount:      2,
		},
		{
			name:             "req has two imps: one with stored response and another imp without stored resp",
			givenRequestBody: validRequest(t, "req-two-imps-one-stored-response.json"),
			expectedStoredResponses: map[string]json.RawMessage{
				"imp-id1": json.RawMessage(`[{"bid": [{"id": "bid_id1"],"seat": "appnexus"}]`),
			},
			expectedErrorCount: 1,
			expectedError:      `request validation failed. The StoredAuctionResponse.ID field must be completely present with, or completely absent from, all impressions in request. No StoredAuctionResponse data found for request.imp[1].ext.prebid`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			deps := &endpointDeps{
				fakeUUIDGenerator{},
				&warningsCheckExchange{},
				ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
				&mockStoredReqFetcher{},
				empty_fetcher.EmptyFetcher{},
				empty_fetcher.EmptyFetcher{},
				&config.Configuration{MaxRequestSize: int64(len(test.givenRequestBody))},
				&metricsConfig.NilMetricsEngine{},
				analyticsBuild.New(&config.Analytics{}),
				map[string]string{},
				false,
				[]byte{},
				openrtb_ext.BuildBidderMap(),
				nil,
				nil,
				hardcodedResponseIPValidator{response: true},
				&mockStoredResponseFetcher{mockStoredResponses},
				hooks.EmptyPlanBuilder{},
				nil,
				openrtb_ext.NormalizeBidderName,
			}

			hookExecutor := hookexecution.NewHookExecutor(deps.hookExecutionPlanBuilder, hookexecution.EndpointAuction, deps.metricsEngine)

			req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(test.givenRequestBody))

			_, _, storedResponses, _, _, _, errL := deps.parseRequest(req, &metrics.Labels{}, hookExecutor)

			if test.expectedErrorCount == 0 {
				assert.Equal(t, test.expectedStoredResponses, storedResponses, "stored responses should match")
			} else {
				assert.Contains(t, errL[0].Error(), test.expectedError, "error should match")
			}

		})
	}
}

func TestParseRequestStoredBidResponses(t *testing.T) {
	bidRespId1 := json.RawMessage(`{"id": "resp_id1", "seatbid": [{"bid": [{"id": "bid_id1"}], "seat": "telaria"}], "bidid": "123", "cur": "USD"}`)
	bidRespId2 := json.RawMessage(`{"id": "resp_id2", "seatbid": [{"bid": [{"id": "bid_id2"}], "seat": "amx"}], "bidid": "124", "cur": "USD"}`)
	bidRespId3 := json.RawMessage(`{"id": "resp_id3", "seatbid": [{"bid": [{"id": "bid_id3"}], "seat": "APPNEXUS"}], "bidid": "125", "cur": "USD"}`)
	mockStoredBidResponses := map[string]json.RawMessage{
		"bidResponseId1": bidRespId1,
		"bidResponseId2": bidRespId2,
		"bidResponseId3": bidRespId3,
	}

	tests := []struct {
		name                       string
		givenRequestBody           string
		expectedStoredBidResponses stored_responses.ImpBidderStoredResp
		expectedErrorCount         int
		expectedError              string
	}{
		{
			name:             "req imp has valid stored bid response",
			givenRequestBody: validRequest(t, "imp-with-stored-bid-resp.json"),
			expectedStoredBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"telaria": bidRespId1},
			},
			expectedErrorCount: 0,
		},
		{
			name:             "req imp has valid stored bid response with case not-matching bidder name",
			givenRequestBody: validRequest(t, "imp-with-stored-bid-resp-case-not-matching-bidder-name.json"),
			expectedStoredBidResponses: map[string]map[string]json.RawMessage{
				"imp-id3": {"appnexus": bidRespId3},
			},
			expectedErrorCount: 0,
		},
		{
			name:             "req imp has valid stored bid response with case matching bidder name",
			givenRequestBody: validRequest(t, "imp-with-stored-bid-resp-case-matching-bidder-name.json"),
			expectedStoredBidResponses: map[string]map[string]json.RawMessage{
				"imp-id3": {"appnexus": bidRespId3},
			},
			expectedErrorCount: 0,
		},
		{
			name:             "req has two imps with valid stored bid responses",
			givenRequestBody: validRequest(t, "req-two-imps-stored-bid-responses.json"),
			expectedStoredBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"telaria": bidRespId1},
				"imp-id2": {"amx": bidRespId2},
			},
			expectedErrorCount: 0,
		},
		{
			name:             "req has two imps one with valid stored bid responses and another one without stored bid responses",
			givenRequestBody: validRequest(t, "req-two-imps-with-and-without-stored-bid-responses.json"),
			expectedStoredBidResponses: map[string]map[string]json.RawMessage{
				"imp-id2": {"amx": bidRespId2},
			},
			expectedErrorCount: 0,
		},
		{
			name:                       "req has two imps with missing stored bid responses",
			givenRequestBody:           validRequest(t, "req-two-imps-missing-stored-bid-response.json"),
			expectedStoredBidResponses: nil,
			expectedErrorCount:         1,
		},
		{
			name:                       "req imp has valid stored bid response with non existing bidder name",
			givenRequestBody:           validRequest(t, "imp-with-stored-bid-resp-non-existing-bidder-name.json"),
			expectedStoredBidResponses: nil,
			expectedErrorCount:         1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			deps := &endpointDeps{
				fakeUUIDGenerator{},
				&warningsCheckExchange{},
				ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
				&mockStoredReqFetcher{},
				empty_fetcher.EmptyFetcher{},
				empty_fetcher.EmptyFetcher{},
				&config.Configuration{MaxRequestSize: int64(len(test.givenRequestBody))},
				&metricsConfig.NilMetricsEngine{},
				analyticsBuild.New(&config.Analytics{}),
				map[string]string{},
				false,
				[]byte{},
				map[string]openrtb_ext.BidderName{"telaria": "telaria", "amx": "amx", "appnexus": "appnexus"},
				nil,
				nil,
				hardcodedResponseIPValidator{response: true},
				&mockStoredResponseFetcher{mockStoredBidResponses},
				hooks.EmptyPlanBuilder{},
				nil,
				openrtb_ext.NormalizeBidderName,
			}

			hookExecutor := hookexecution.NewHookExecutor(deps.hookExecutionPlanBuilder, hookexecution.EndpointAuction, deps.metricsEngine)

			req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(test.givenRequestBody))
			_, _, _, storedBidResponses, _, _, errL := deps.parseRequest(req, &metrics.Labels{}, hookExecutor)
			if test.expectedErrorCount == 0 {
				assert.Empty(t, errL)
				assert.Equal(t, test.expectedStoredBidResponses, storedBidResponses, "stored responses should match")
			} else {
				assert.Contains(t, errL[0].Error(), test.expectedError, "error should match")
			}
		})
	}
}

func TestValidateStoredResp(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		&mockStoredResponseFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	testCases := []struct {
		description               string
		givenRequestWrapper       *openrtb_ext.RequestWrapper
		givenHttpRequest          *http.Request
		givenAccount              *config.Account
		expectedErrorList         []error
		hasStoredAuctionResponses bool
		storedBidResponses        stored_responses.ImpBidderStoredResp
	}{
		{
			description: "One imp with stored response, expect validate request to throw no errors",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus":{"placementId": 12345678}, "prebid": {"storedAuctionResponse": {"id": "6d718149-6dfe-25ae-a7d6-305399f77f04"}}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{},
			hasStoredAuctionResponses: true,
			storedBidResponses:        nil,
		},
		{
			description: "Two imps with stored responses, expect validate request to throw no errors",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus":{"placementId": 12345678}, "prebid": {"storedAuctionResponse": {"id": "6d718149-6dfe-25ae-a7d6-305399f77f04"}}}`),
						},
						{
							ID: "Some-Imp-ID2",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus":{"placementId": 12345678}, "prebid": {"storedAuctionResponse": {"id": "6d718149-6dfe-25ae-a7d6-305399f77f04"}}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{},
			hasStoredAuctionResponses: true,
			storedBidResponses:        nil,
		},
		{
			description: "Two imps, one with stored response, expect validate request to throw validation error",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus":{"placementId": 12345678}, "prebid": {"storedAuctionResponse": {"id": "6d718149-6dfe-25ae-a7d6-305399f77f04"}}}`),
						},
						{
							ID: "Some-Imp-ID2",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus":{"placementId": 12345678}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{errors.New("request validation failed. The StoredAuctionResponse.ID field must be completely present with, or completely absent from, all impressions in request. No StoredAuctionResponse data found for request.imp[1].ext.prebid \n")},
			hasStoredAuctionResponses: true,
			storedBidResponses:        nil,
		},
		{
			description: "One imp with stored bid response and corresponding bidder in imp.ext, expect validate request to throw no errors",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus": {"placementId": 12345678}, "prebid": {"storedbidresponse": []}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{},
			hasStoredAuctionResponses: false,
			storedBidResponses:        stored_responses.ImpBidderStoredResp{"Some-Imp-ID": {"appnexus": json.RawMessage(`{"test":true}`)}},
		},
		{
			description: "One imp with 2 stored bid responses and 2 corresponding bidders in imp.ext, expect validate request to throw no errors",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus": {"placementId": 12345678}, "telaria": {"seatCode": "12345678"}, "prebid": {"storedbidresponse": []}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{},
			hasStoredAuctionResponses: false,
			storedBidResponses:        stored_responses.ImpBidderStoredResp{"Some-Imp-ID": {"appnexus": json.RawMessage(`{"test":true}`), "telaria": json.RawMessage(`{"test":true}`)}},
		},
		{
			description: "Two imps, one with 2 stored bid responses and 2 corresponding bidders in imp.ext, expect validate request to throw no errors",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus": {"placementId": 12345678}, "telaria": {"seatCode": "12345678"}, "prebid": {"storedbidresponse": []}}`),
						},
						{
							ID: "Some-Imp-ID2",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus": {"placementId": 12345678}, "telaria": {"seatCode": "12345678"}, "prebid": {"storedbidresponse": []}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{},
			hasStoredAuctionResponses: false,
			storedBidResponses:        stored_responses.ImpBidderStoredResp{"Some-Imp-ID": {"appnexus": json.RawMessage(`{"test":true}`), "telaria": json.RawMessage(`{"test":true}`)}},
		},
		{
			description: "Two imps, both with 2 stored bid responses and 2 corresponding bidders in imp.ext, expect validate request to throw no errors",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus": {"placementId": 12345678}, "telaria": {"seatCode": "12345678"}, "prebid": {"storedbidresponse": []}}`),
						},
						{
							ID: "Some-Imp-ID2",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus": {"placementId": 12345678}, "telaria": {"seatCode": "12345678"}, "prebid": {"storedbidresponse": []}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{},
			hasStoredAuctionResponses: false,
			storedBidResponses: stored_responses.ImpBidderStoredResp{
				"Some-Imp-ID":  {"appnexus": json.RawMessage(`{"test":true}`), "telaria": json.RawMessage(`{"test":true}`)},
				"Some-Imp-ID1": {"appnexus": json.RawMessage(`{"test":true}`), "telaria": json.RawMessage(`{"test":true}`)},
			},
		},
		{
			description: "One imp with 2 stored bid responses and 1 bidder in imp.ext, expect validate request to throw an errors",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus": {"placementId": 12345678}, "prebid": {"storedbidresponse": []}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{errors.New("request validation failed. Stored bid responses are specified for imp Some-Imp-ID. Bidders specified in imp.ext should match with bidders specified in imp.ext.prebid.storedbidresponse")},
			hasStoredAuctionResponses: false,
			storedBidResponses:        stored_responses.ImpBidderStoredResp{"Some-Imp-ID": {"appnexus": json.RawMessage(`{"test":true}`), "telaria": json.RawMessage(`{"test":true}`)}},
		},
		{
			description: "One imp with 1 stored bid responses and 2 bidders in imp.ext, expect validate request to throw an errors",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus": {"placementId": 12345678}, "telaria": {"seatCode": "12345678"}, "prebid": {"storedbidresponse": []}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{errors.New("request validation failed. Stored bid responses are specified for imp Some-Imp-ID. Bidders specified in imp.ext should match with bidders specified in imp.ext.prebid.storedbidresponse")},
			hasStoredAuctionResponses: false,
			storedBidResponses:        stored_responses.ImpBidderStoredResp{"Some-Imp-ID": {"appnexus": json.RawMessage(`{"test":true}`)}},
		},
		{
			description: "One imp with 2 stored bid responses and 2 different bidders in imp.ext, expect validate request to throw an errors",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus": {"placementId": 12345678}, "telaria": {"seatCode": "12345678"}, "prebid": {"storedbidresponse": []}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{errors.New("request validation failed. Stored bid responses are specified for imp Some-Imp-ID. Bidders specified in imp.ext should match with bidders specified in imp.ext.prebid.storedbidresponse")},
			hasStoredAuctionResponses: false,
			storedBidResponses:        stored_responses.ImpBidderStoredResp{"Some-Imp-ID": {"appnexus": json.RawMessage(`{"test":true}`), "rubicon": json.RawMessage(`{"test":true}`)}},
		},
		{
			description: "One imp with 1 stored bid response and 1 ignored bidder in imp.ext and 1 included bidder in imp.ext.prebid.bidder, expect validate request to throw no errors",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus": {"placementId": 12345678}, "prebid": {"bidder":{"telaria": {"seatCode": "12345678"}}, "storedbidresponse": []}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{},
			hasStoredAuctionResponses: false,
			storedBidResponses:        stored_responses.ImpBidderStoredResp{"Some-Imp-ID": {"telaria": json.RawMessage(`{"test":true}`)}},
		},
		{
			description: "One imp with 2 stored bid responses and 1 bidders in imp.ext and 1 in imp.ext.prebid.bidder that is not defined in stored bid responses, expect validate request to throw an error",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"appnexus": {"placementId": 12345678}, "prebid": {"bidder":{"rubicon": {"seatCode": "12345678"}}, "storedbidresponse": []}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{errors.New("request validation failed. Stored bid responses are specified for imp Some-Imp-ID. Bidders specified in imp.ext should match with bidders specified in imp.ext.prebid.storedbidresponse")},
			hasStoredAuctionResponses: false,
			storedBidResponses:        stored_responses.ImpBidderStoredResp{"Some-Imp-ID": {"appnexus": json.RawMessage(`{"test":true}`), "telaria": json.RawMessage(`{"test":true}`)}},
		},
		{
			description: "One imp with 1 stored bid response and 1 in imp.ext.prebid.bidder that is defined in stored bid responses, expect validate request to throw no errors",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"prebid": {"bidder":{"telaria": {"seatCode": "12345678"}}, "storedbidresponse": []}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{},
			hasStoredAuctionResponses: false,
			storedBidResponses:        stored_responses.ImpBidderStoredResp{"Some-Imp-ID": {"telaria": json.RawMessage(`{"test":true}`)}},
		},
		{
			description: "One imp with 1 stored bid response and 1 in imp.ext.prebid.bidder that is not defined in stored bid responses, expect validate request to throw an error",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"prebid": {"bidder":{"telaria": {"seatCode": "12345678"}}, "storedbidresponse": []}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{errors.New("request validation failed. Stored bid responses are specified for imp Some-Imp-ID. Bidders specified in imp.ext should match with bidders specified in imp.ext.prebid.storedbidresponse")},
			hasStoredAuctionResponses: false,
			storedBidResponses:        stored_responses.ImpBidderStoredResp{"Some-Imp-ID": {"appnexus": json.RawMessage(`{"test":true}`)}},
		},
		{
			description: "2 imps, one imp without stored responses, another imp with 1 stored bid response and 1 in imp.ext.prebid.bidder that is not defined in stored bid responses, expect validate request to throw an error",
			givenRequestWrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					ID:  "Some-ID",
					App: &openrtb2.App{},
					Imp: []openrtb2.Imp{
						{
							ID: "Some-Imp-ID",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"prebid": {"bidder":{"telaria": {"seatCode": "12345678"}}}}`),
						},
						{
							ID: "Some-Imp-ID2",
							Banner: &openrtb2.Banner{
								Format: []openrtb2.Format{
									{
										W: 600,
										H: 500,
									},
									{
										W: 300,
										H: 600,
									},
								},
							},
							Ext: []byte(`{"prebid": {"bidder":{"telaria": {"seatCode": "12345678"}}, "storedbidresponse": []}}`),
						},
					},
				},
			},
			expectedErrorList:         []error{errors.New("request validation failed. Stored bid responses are specified for imp Some-Imp-ID2. Bidders specified in imp.ext should match with bidders specified in imp.ext.prebid.storedbidresponse")},
			hasStoredAuctionResponses: false,
			storedBidResponses:        stored_responses.ImpBidderStoredResp{"Some-Imp-ID2": {"appnexus": json.RawMessage(`{"test":true}`)}},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			errorList := deps.validateRequest(test.givenAccount, test.givenHttpRequest, test.givenRequestWrapper, false, test.hasStoredAuctionResponses, test.storedBidResponses, false)
			assert.Equalf(t, test.expectedErrorList, errorList, "Error doesn't match: %s\n", test.description)
		})
	}
}

func TestValidResponseAfterExecutingStages(t *testing.T) {
	const nbr int = 123

	hooksPlanBuilder := mockPlanBuilder{
		entrypointPlan: hooks.Plan[hookstage.Entrypoint]{
			{
				Timeout: 5 * time.Millisecond,
				Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
					entryPointHookUpdateWithErrors,
					entryPointHookUpdateWithErrorsAndWarnings,
				},
			},
			{
				Timeout: 5 * time.Millisecond,
				Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
					entryPointHookUpdate,
				},
			},
		},
		rawAuctionPlan: hooks.Plan[hookstage.RawAuctionRequest]{
			{
				Timeout: 5 * time.Millisecond,
				Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
					rawAuctionHookNone,
				},
			},
		},
	}

	testCases := []struct {
		description string
		file        string
		planBuilder hooks.ExecutionPlanBuilder
	}{
		{
			description: "Assert correct BidResponse when request rejected at entrypoint stage",
			file:        "sample-requests/hooks/auction_entrypoint_reject.json",
			planBuilder: mockPlanBuilder{entrypointPlan: makePlan[hookstage.Entrypoint](mockRejectionHook{nbr, nil})},
		},
		{
			description: "Assert correct BidResponse when request rejected at raw-auction stage",
			file:        "sample-requests/hooks/auction_raw_auction_request_reject.json",
			planBuilder: mockPlanBuilder{rawAuctionPlan: makePlan[hookstage.RawAuctionRequest](mockRejectionHook{nbr, nil})},
		},
		{
			description: "Assert correct BidResponse when request rejected at processed-auction stage",
			file:        "sample-requests/hooks/auction_processed_auction_request_reject.json",
			planBuilder: mockPlanBuilder{processedAuctionPlan: makePlan[hookstage.ProcessedAuctionRequest](mockRejectionHook{nbr, nil})},
		},
		{
			// bidder-request stage doesn't reject whole request, so we do not expect NBR code in response
			description: "Assert correct BidResponse when request rejected at bidder-request stage",
			file:        "sample-requests/hooks/auction_bidder_reject.json",
			planBuilder: mockPlanBuilder{bidderRequestPlan: makePlan[hookstage.BidderRequest](mockRejectionHook{nbr, nil})},
		},
		{
			description: "Assert correct BidResponse when request rejected at raw-bidder-response stage",
			file:        "sample-requests/hooks/auction_bidder_response_reject.json",
			planBuilder: mockPlanBuilder{rawBidderResponsePlan: makePlan[hookstage.RawBidderResponse](mockRejectionHook{nbr, nil})},
		},
		{
			description: "Assert correct BidResponse when request rejected with error from hook",
			file:        "sample-requests/hooks/auction_reject_with_error.json",
			planBuilder: mockPlanBuilder{entrypointPlan: makePlan[hookstage.Entrypoint](mockRejectionHook{nbr, errors.New("dummy")})},
		},
		{
			description: "Assert correct BidResponse with debug information from modules added to ext.prebid.modules",
			file:        "sample-requests/hooks/auction.json",
			planBuilder: hooksPlanBuilder,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fileData, err := os.ReadFile(tc.file)
			assert.NoError(t, err, "Failed to read test file.")

			test, err := parseTestData(fileData, tc.file)
			assert.NoError(t, err, "Failed to parse test file.")
			test.planBuilder = tc.planBuilder
			test.endpointType = OPENRTB_ENDPOINT

			cfg := &config.Configuration{MaxRequestSize: maxSize, AccountDefaults: config.Account{DebugAllow: true}}
			auctionEndpointHandler, _, mockBidServers, mockCurrencyRatesServer, err := buildTestEndpoint(test, cfg)
			assert.NoError(t, err, "Failed to build test endpoint.")

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/openrtb2/auction", bytes.NewReader(test.BidRequest))
			auctionEndpointHandler(recorder, req, nil)
			assert.Equal(t, recorder.Code, http.StatusOK, "Endpoint should return 200 OK.")

			var actualResp openrtb2.BidResponse
			var expectedResp openrtb2.BidResponse
			var actualExt openrtb_ext.ExtBidResponse
			var expectedExt openrtb_ext.ExtBidResponse

			assert.NoError(t, jsonutil.UnmarshalValid(test.ExpectedBidResponse, &expectedResp), "Unable to unmarshal expected BidResponse.")
			assert.NoError(t, jsonutil.UnmarshalValid(recorder.Body.Bytes(), &actualResp), "Unable to unmarshal actual BidResponse.")
			if expectedResp.Ext != nil {
				assert.NoError(t, jsonutil.UnmarshalValid(expectedResp.Ext, &expectedExt), "Unable to unmarshal expected ExtBidResponse.")
				assert.NoError(t, jsonutil.UnmarshalValid(actualResp.Ext, &actualExt), "Unable to unmarshal actual ExtBidResponse.")
			}

			assertBidResponseEqual(t, tc.file, expectedResp, actualResp)
			assert.Equal(t, expectedResp.NBR, actualResp.NBR, "Invalid NBR.")
			assert.Equal(t, expectedExt.Warnings, actualExt.Warnings, "Wrong bidResponse.ext.warnings.")

			if expectedExt.Prebid != nil {
				hookexecution.AssertEqualModulesData(t, expectedExt.Prebid.Modules, actualExt.Prebid.Modules)
			} else {
				assert.Nil(t, actualExt.Prebid, "Invalid BidResponse.ext.prebid")
			}

			// Close servers regardless if the test case was run or not
			for _, mockBidServer := range mockBidServers {
				mockBidServer.Close()
			}
			mockCurrencyRatesServer.Close()
		})
	}
}

func TestSendAuctionResponse_LogsErrors(t *testing.T) {
	hookExecutor := &mockStageExecutor{
		outcomes: []hookexecution.StageOutcome{
			{
				Entity: "bid-request",
				Stage:  hooks.StageBidderRequest.String(),
				Groups: []hookexecution.GroupOutcome{
					{
						InvocationResults: []hookexecution.HookOutcome{
							{
								HookID: hookexecution.HookID{
									ModuleCode:   "foobar",
									HookImplCode: "foo",
								},
								Status:   hookexecution.StatusSuccess,
								Action:   hookexecution.ActionNone,
								Warnings: []string{"warning message"},
							},
						},
					},
				},
			},
		},
	}

	testCases := []struct {
		description    string
		expectedErrors []error
		expectedStatus int
		request        *openrtb2.BidRequest
		response       *openrtb2.BidResponse
		hookExecutor   hookexecution.HookStageExecutor
	}{
		{
			description: "Error logged if hook enrichment fails",
			expectedErrors: []error{
				errors.New("Failed to enrich Bid Response with hook debug information: Invalid JSON Document"),
				errors.New("/openrtb2/auction Failed to send response: json: error calling MarshalJSON for type json.RawMessage: invalid character '.' looking for beginning of value"),
			},
			expectedStatus: 0,
			request:        &openrtb2.BidRequest{ID: "some-id", Test: 1},
			response:       &openrtb2.BidResponse{ID: "some-id", Ext: json.RawMessage("...")},
			hookExecutor:   hookExecutor,
		},
		{
			description: "Error logged if hook enrichment returns warnings",
			expectedErrors: []error{
				errors.New("Value is not a string: 1"),
				errors.New("Value is not a boolean: active"),
			},
			expectedStatus: 0,
			request:        &openrtb2.BidRequest{ID: "some-id", Test: 1, Ext: json.RawMessage(`{"prebid": {"debug": "active", "trace": 1}}`)},
			response:       &openrtb2.BidResponse{ID: "some-id", Ext: json.RawMessage("{}")},
			hookExecutor:   hookExecutor,
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			writer := httptest.NewRecorder()
			labels := metrics.Labels{}
			ao := analytics.AuctionObject{}
			account := &config.Account{DebugAllow: true}

			_, ao = sendAuctionResponse(writer, test.hookExecutor, test.response, test.request, account, labels, ao)

			assert.Equal(t, ao.Errors, test.expectedErrors, "Invalid errors.")
			assert.Equal(t, test.expectedStatus, ao.Status, "Invalid HTTP response status.")
		})
	}
}

func TestParseRequestMultiBid(t *testing.T) {
	tests := []struct {
		name             string
		givenRequestBody string
		expectedReqExt   json.RawMessage
		expectedErrors   []error
	}{
		{
			name:             "validate and build multi-bid extension",
			givenRequestBody: validRequest(t, "multi-bid-error.json"),
			expectedReqExt:   getObject(t, "multi-bid-error.json", "expectedReqExt"),
			expectedErrors: []error{
				&errortypes.Warning{
					WarningCode: errortypes.MultiBidWarningCode,
					Message:     "maxBids not defined for {Bidder:appnexus, Bidders:[], MaxBids:<nil>, TargetBidderCodePrefix:}",
				},
				&errortypes.Warning{
					WarningCode: errortypes.MultiBidWarningCode,
					Message:     "invalid maxBids value, using minimum 1 limit for {Bidder:rubicon, Bidders:[], MaxBids:-1, TargetBidderCodePrefix:rubN}",
				},
				&errortypes.Warning{
					WarningCode: errortypes.MultiBidWarningCode,
					Message:     "invalid maxBids value, using maximum 9 limit for {Bidder:pubmatic, Bidders:[], MaxBids:10, TargetBidderCodePrefix:pm}",
				},
				&errortypes.Warning{
					WarningCode: errortypes.MultiBidWarningCode,
					Message:     "multiBid already defined for pubmatic, ignoring this instance {Bidder:pubmatic, Bidders:[], MaxBids:4, TargetBidderCodePrefix:pubM}",
				},
				&errortypes.Warning{
					WarningCode: errortypes.MultiBidWarningCode,
					Message:     "ignoring bidders from {Bidder:groupm, Bidders:[someBidder], MaxBids:5, TargetBidderCodePrefix:gm}",
				},
				&errortypes.Warning{
					WarningCode: errortypes.MultiBidWarningCode,
					Message:     "multiBid already defined for groupm, ignoring this instance {Bidder:, Bidders:[groupm], MaxBids:6, TargetBidderCodePrefix:}",
				},
				&errortypes.Warning{
					WarningCode: errortypes.MultiBidWarningCode,
					Message:     "ignoring targetbiddercodeprefix for {Bidder:, Bidders:[33across], MaxBids:7, TargetBidderCodePrefix:abc}",
				},
				&errortypes.Warning{
					WarningCode: errortypes.MultiBidWarningCode,
					Message:     "bidder(s) not specified for {Bidder:, Bidders:[], MaxBids:8, TargetBidderCodePrefix:xyz}",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deps := &endpointDeps{
				fakeUUIDGenerator{},
				&warningsCheckExchange{},
				ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
				&mockStoredReqFetcher{},
				empty_fetcher.EmptyFetcher{},
				empty_fetcher.EmptyFetcher{},
				&config.Configuration{MaxRequestSize: int64(len(test.givenRequestBody))},
				&metricsConfig.NilMetricsEngine{},
				analyticsBuild.New(&config.Analytics{}),
				map[string]string{},
				false,
				[]byte{},
				openrtb_ext.BuildBidderMap(),
				nil,
				nil,
				hardcodedResponseIPValidator{response: true},
				empty_fetcher.EmptyFetcher{},
				hooks.EmptyPlanBuilder{},
				nil,
				openrtb_ext.NormalizeBidderName,
			}

			hookExecutor := hookexecution.NewHookExecutor(deps.hookExecutionPlanBuilder, hookexecution.EndpointAuction, deps.metricsEngine)

			req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(test.givenRequestBody))

			resReq, _, _, _, _, _, errL := deps.parseRequest(req, &metrics.Labels{}, hookExecutor)

			assert.NoError(t, resReq.RebuildRequest())

			assert.JSONEq(t, string(test.expectedReqExt), string(resReq.Ext))

			assert.Equal(t, errL, test.expectedErrors, "error length should match")
		})
	}
}

type mockStoredResponseFetcher struct {
	data map[string]json.RawMessage
}

func (cf *mockStoredResponseFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	return nil, nil, nil
}

func (cf *mockStoredResponseFetcher) FetchResponses(ctx context.Context, ids []string) (data map[string]json.RawMessage, errs []error) {
	return cf.data, nil
}

func getObject(t *testing.T, filename, key string) json.RawMessage {
	requestData, err := os.ReadFile("sample-requests/valid-whole/supplementary/" + filename)
	if err != nil {
		t.Fatalf("Failed to fetch a valid request: %v", err)
	}
	testBidRequest, _, _, err := jsonparser.Get(requestData, key)
	assert.NoError(t, err, "Error jsonparsing root.mockBidRequest from file %s. Desc: %v.", filename, err)

	var obj json.RawMessage
	err = jsonutil.UnmarshalValid(testBidRequest, &obj)
	if err != nil {
		t.Fatalf("Failed to fetch object with key '%s' ... got error: %v", key, err)
	}
	return obj
}

func getIntegrationFromRequest(req *openrtb_ext.RequestWrapper) (string, error) {
	reqExt, err := req.GetRequestExt()
	if err != nil {
		return "", err
	}
	reqPrebid := reqExt.GetPrebid()
	return reqPrebid.Integration, nil
}

type mockStageExecutor struct {
	hookexecution.EmptyHookExecutor

	outcomes []hookexecution.StageOutcome
}

func (e mockStageExecutor) GetOutcomes() []hookexecution.StageOutcome {
	return e.outcomes
}

func TestSetSeatNonBidRaw(t *testing.T) {
	type args struct {
		request         *openrtb_ext.RequestWrapper
		auctionResponse *exchange.AuctionResponse
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "nil-auctionResponse",
			args:    args{auctionResponse: nil},
			wantErr: false,
		},
		{
			name:    "nil-bidResponse",
			args:    args{auctionResponse: &exchange.AuctionResponse{BidResponse: nil}},
			wantErr: false,
		},
		{
			name:    "invalid-response.Ext",
			args:    args{auctionResponse: &exchange.AuctionResponse{BidResponse: &openrtb2.BidResponse{Ext: []byte(`invalid_json`)}}},
			wantErr: true,
		},
		{
			name: "update-seatnonbid-in-ext",
			args: args{
				request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{Ext: []byte(`{"prebid": { "returnallbidstatus" : true }}`)}},
				auctionResponse: &exchange.AuctionResponse{
					ExtBidResponse: &openrtb_ext.ExtBidResponse{Prebid: &openrtb_ext.ExtResponsePrebid{SeatNonBid: []openrtb_ext.SeatNonBid{}}},
					BidResponse:    &openrtb2.BidResponse{Ext: []byte(`{}`)},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := setSeatNonBidRaw(tt.args.request, tt.args.auctionResponse); (err != nil) != tt.wantErr {
				t.Errorf("setSeatNonBidRaw() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAliases(t *testing.T) {
	deps := &endpointDeps{
		disabledBidders: map[string]string{"rubicon": "rubicon"},
		bidderMap:       map[string]openrtb_ext.BidderName{"appnexus": openrtb_ext.BidderName("appnexus")},
	}

	testCases := []struct {
		description     string
		aliases         map[string]string
		expectedAliases map[string]string
		expectedError   error
	}{
		{
			description:     "valid case",
			aliases:         map[string]string{"test": "appnexus"},
			expectedAliases: map[string]string{"test": "appnexus"},
			expectedError:   nil,
		},
		{
			description:     "valid case - case insensitive",
			aliases:         map[string]string{"test": "Appnexus"},
			expectedAliases: map[string]string{"test": "appnexus"},
			expectedError:   nil,
		},
		{
			description:     "disabled bidder",
			aliases:         map[string]string{"test": "rubicon"},
			expectedAliases: nil,
			expectedError:   errors.New("request.ext.prebid.aliases.test refers to disabled bidder: rubicon"),
		},
		{
			description:     "coreBidderName not found",
			aliases:         map[string]string{"test": "anyBidder"},
			expectedAliases: nil,
			expectedError:   errors.New("request.ext.prebid.aliases.test refers to unknown bidder: anyBidder"),
		},
		{
			description:     "alias name is coreBidder name",
			aliases:         map[string]string{"appnexus": "appnexus"},
			expectedAliases: nil,
			expectedError:   errors.New("request.ext.prebid.aliases.appnexus defines a no-op alias. Choose a different alias, or remove this entry."),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			err := deps.validateAliases(testCase.aliases)
			if err != nil {
				assert.Equal(t, testCase.expectedError, err)
			} else {
				assert.ObjectsAreEqualValues(testCase.expectedAliases, map[string]string{"test": "appnexus"})
			}
		})
	}
}

func fakeNormalizeBidderName(name string) (openrtb_ext.BidderName, bool) {
	return openrtb_ext.BidderName(strings.ToLower(name)), true
}

func TestValidateOrFillCookieDeprecation(t *testing.T) {
	type args struct {
		httpReq *http.Request
		req     *openrtb_ext.RequestWrapper
		account config.Account
	}
	tests := []struct {
		name          string
		args          args
		wantDeviceExt json.RawMessage
		wantErr       error
	}{
		{
			name:          "account-nil",
			wantDeviceExt: nil,
			wantErr:       nil,
		},
		{
			name: "cookie-deprecation-not-enabled",
			args: args{
				httpReq: &http.Request{
					Header: http.Header{secCookieDeprecation: []string{"example_label_1"}},
				},
				req: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				},
			},
			wantDeviceExt: nil,
			wantErr:       nil,
		},
		{
			name: "cookie-deprecation-disabled-explicitly",
			args: args{
				httpReq: &http.Request{
					Header: http.Header{secCookieDeprecation: []string{"example_label_1"}},
				},
				req: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				},
				account: config.Account{
					Privacy: config.AccountPrivacy{
						PrivacySandbox: config.PrivacySandbox{
							CookieDeprecation: config.CookieDeprecation{
								Enabled: false,
							},
						},
					},
				},
			},
			wantDeviceExt: nil,
			wantErr:       nil,
		},
		{
			name: "cookie-deprecation-enabled-header-not-present-in-request",
			args: args{
				httpReq: &http.Request{},
				req: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				},
				account: config.Account{
					Privacy: config.AccountPrivacy{
						PrivacySandbox: config.PrivacySandbox{
							CookieDeprecation: config.CookieDeprecation{
								Enabled: true,
							},
						},
					},
				},
			},
			wantDeviceExt: nil,
			wantErr:       nil,
		},
		{
			name: "header-present-request-device-nil",
			args: args{
				httpReq: &http.Request{
					Header: http.Header{secCookieDeprecation: []string{"example_label_1"}},
				},
				req: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{},
				},
				account: config.Account{
					Privacy: config.AccountPrivacy{
						PrivacySandbox: config.PrivacySandbox{
							CookieDeprecation: config.CookieDeprecation{
								Enabled: true,
							},
						},
					},
				},
			},
			wantDeviceExt: json.RawMessage(`{"cdep":"example_label_1"}`),
			wantErr:       nil,
		},
		{
			name: "header-present-request-device-ext-nil",
			args: args{
				httpReq: &http.Request{
					Header: http.Header{secCookieDeprecation: []string{"example_label_1"}},
				},
				req: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Device: &openrtb2.Device{
							Ext: nil,
						},
					},
				},
				account: config.Account{
					Privacy: config.AccountPrivacy{
						PrivacySandbox: config.PrivacySandbox{
							CookieDeprecation: config.CookieDeprecation{
								Enabled: true,
							},
						},
					},
				},
			},
			wantDeviceExt: json.RawMessage(`{"cdep":"example_label_1"}`),
			wantErr:       nil,
		},
		{
			name: "header-present-request-device-ext-not-nil",
			args: args{
				httpReq: &http.Request{
					Header: http.Header{secCookieDeprecation: []string{"example_label_1"}},
				},
				req: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Device: &openrtb2.Device{
							Ext: json.RawMessage(`{"foo":"bar"}`),
						},
					},
				},
				account: config.Account{
					Privacy: config.AccountPrivacy{
						PrivacySandbox: config.PrivacySandbox{
							CookieDeprecation: config.CookieDeprecation{
								Enabled: true,
							},
						},
					},
				},
			},
			wantDeviceExt: json.RawMessage(`{"cdep":"example_label_1","foo":"bar"}`),
			wantErr:       nil,
		},
		{
			name: "header-present-with-length-more-than-100",
			args: args{
				httpReq: &http.Request{
					Header: http.Header{secCookieDeprecation: []string{"zjfXqGxXFI8yura8AhQl1DK2EMMmryrC8haEpAlwjoerrFfEo2MQTXUq6cSmLohI8gjsnkGU4oAzvXd4TTAESzEKsoYjRJ2zKxmEa"}},
				},
				req: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Device: &openrtb2.Device{
							Ext: json.RawMessage(`{"foo":"bar"}`),
						},
					},
				},
				account: config.Account{
					Privacy: config.AccountPrivacy{
						PrivacySandbox: config.PrivacySandbox{
							CookieDeprecation: config.CookieDeprecation{
								Enabled: true,
							},
						},
					},
				},
			},
			wantDeviceExt: json.RawMessage(`{"foo":"bar"}`),
			wantErr: &errortypes.Warning{
				Message:     "request.device.ext.cdep must not exceed 100 characters",
				WarningCode: errortypes.SecCookieDeprecationLenWarningCode,
			},
		},
		{
			name: "header-present-request-device-ext-cdep-present",
			args: args{
				httpReq: &http.Request{
					Header: http.Header{secCookieDeprecation: []string{"example_label_1"}},
				},
				req: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Device: &openrtb2.Device{
							Ext: json.RawMessage(`{"foo":"bar","cdep":"example_label_2"}`),
						},
					},
				},
				account: config.Account{
					Privacy: config.AccountPrivacy{
						PrivacySandbox: config.PrivacySandbox{
							CookieDeprecation: config.CookieDeprecation{
								Enabled: true,
							},
						},
					},
				},
			},
			wantDeviceExt: json.RawMessage(`{"foo":"bar","cdep":"example_label_2"}`),
			wantErr:       nil,
		},
		{
			name: "header-present-request-device-ext-invalid",
			args: args{
				httpReq: &http.Request{
					Header: http.Header{secCookieDeprecation: []string{"example_label_1"}},
				},
				req: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						Device: &openrtb2.Device{
							Ext: json.RawMessage(`{`),
						},
					},
				},
				account: config.Account{
					Privacy: config.AccountPrivacy{
						PrivacySandbox: config.PrivacySandbox{
							CookieDeprecation: config.CookieDeprecation{
								Enabled: true,
							},
						},
					},
				},
			},
			wantDeviceExt: json.RawMessage(`{`),
			wantErr: &errortypes.FailedToUnmarshal{
				Message: "expects \" or n, but found \x00",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOrFillCookieDeprecation(tt.args.httpReq, tt.args.req, &tt.args.account)
			assert.Equal(t, tt.wantErr, err)
			if tt.args.req != nil {
				err := tt.args.req.RebuildRequest()
				assert.NoError(t, err)
			}
			if tt.wantDeviceExt == nil {
				if tt.args.req != nil && tt.args.req.Device != nil {
					assert.Nil(t, tt.args.req.Device.Ext)
				}
			} else {
				assert.Equal(t, string(tt.wantDeviceExt), string(tt.args.req.Device.Ext))
			}
		})
	}
}

func TestSetGPCImplicitly(t *testing.T) {
	testCases := []struct {
		description  string
		header       string
		regs         *openrtb2.Regs
		expectError  bool
		expectedRegs *openrtb2.Regs
	}{
		{
			description: "regs_ext_gpc_not_set_and_header_is_1",
			header:      "1",
			regs: &openrtb2.Regs{
				Ext: []byte(`{}`),
			},
			expectError: false,
			expectedRegs: &openrtb2.Regs{
				Ext: []byte(`{"gpc":"1"}`),
			},
		},
		{
			description: "sec_gpc_header_not_set_gpc_should_not_be_modified",
			header:      "",
			regs: &openrtb2.Regs{
				Ext: []byte(`{}`),
			},
			expectError: false,
			expectedRegs: &openrtb2.Regs{
				Ext: []byte(`{}`),
			},
		},
		{
			description: "sec_gpc_header_set_to_2_gpc_should_not_be_modified",
			header:      "2",
			regs: &openrtb2.Regs{
				Ext: []byte(`{}`),
			},
			expectError: false,
			expectedRegs: &openrtb2.Regs{
				Ext: []byte(`{}`),
			},
		},
		{
			description: "sec_gpc_header_set_to_1_and_regs_ext_contains_other_data",
			header:      "1",
			regs: &openrtb2.Regs{
				Ext: []byte(`{"some_other_field":"some_value"}`),
			},
			expectError: false,
			expectedRegs: &openrtb2.Regs{
				Ext: []byte(`{"some_other_field":"some_value","gpc":"1"}`),
			},
		},
		{
			description: "regs_ext_gpc_not_set_and_header_not_set",
			header:      "",
			regs: &openrtb2.Regs{
				Ext: []byte(`{}`),
			},
			expectError: false,
			expectedRegs: &openrtb2.Regs{
				Ext: []byte(`{}`),
			},
		},
		{
			description: "regs_ext_gpc_not_set_and_header_not_1",
			header:      "0",
			regs: &openrtb2.Regs{
				Ext: []byte(`{}`),
			},
			expectError: false,
			expectedRegs: &openrtb2.Regs{
				Ext: []byte(`{}`),
			},
		},
		{
			description: "regs_ext_gpc_is_1_and_header_is_1",
			header:      "1",
			regs: &openrtb2.Regs{
				Ext: []byte(`{"gpc":"1"}`),
			},
			expectError: false,
			expectedRegs: &openrtb2.Regs{
				Ext: []byte(`{"gpc":"1"}`),
			},
		},
		{
			description: "regs_ext_gpc_is_1_and_header_not_1",
			header:      "0",
			regs: &openrtb2.Regs{
				Ext: []byte(`{"gpc":"1"}`),
			},
			expectError: false,
			expectedRegs: &openrtb2.Regs{
				Ext: []byte(`{"gpc":"1"}`),
			},
		},
		{
			description: "regs_ext_other_data_and_header_is_1",
			header:      "1",
			regs: &openrtb2.Regs{
				Ext: []byte(`{"other":"value"}`),
			},
			expectError: false,
			expectedRegs: &openrtb2.Regs{
				Ext: []byte(`{"other":"value","gpc":"1"}`),
			},
		},
		{
			description: "regs_nil_and_header_is_1",
			header:      "1",
			regs:        nil,
			expectError: false,
			expectedRegs: &openrtb2.Regs{
				Ext: []byte(`{"gpc":"1"}`),
			},
		},
		{
			description:  "regs_nil_and_header_not_set",
			header:       "",
			regs:         nil,
			expectError:  false,
			expectedRegs: nil,
		},
		{
			description: "regs_ext_is_nil_and_header_not_set",
			header:      "",
			regs: &openrtb2.Regs{
				Ext: nil,
			},
			expectError: false,
			expectedRegs: &openrtb2.Regs{
				Ext: nil,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			httpReq := &http.Request{
				Header: http.Header{
					http.CanonicalHeaderKey("Sec-GPC"): []string{test.header},
				},
			}

			r := &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Regs: test.regs,
				},
			}

			err := setGPCImplicitly(httpReq, r)

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, r.RebuildRequest())
			if test.expectedRegs == nil {
				assert.Nil(t, r.BidRequest.Regs)
			} else if test.expectedRegs.Ext == nil {
				assert.Nil(t, r.BidRequest.Regs.Ext)
			} else {
				assert.JSONEq(t, string(test.expectedRegs.Ext), string(r.BidRequest.Regs.Ext))
			}
		})
	}
}

func TestValidateRequestCookieDeprecation(t *testing.T) {
	testCases :=
		[]struct {
			name         string
			givenAccount *config.Account
			httpReq      *http.Request
			reqWrapper   *openrtb_ext.RequestWrapper
			wantErrs     []error
			wantCDep     string
		}{
			{
				name: "header-with-length-less-than-100",
				httpReq: func() *http.Request {
					req := httptest.NewRequest("POST", "/openrtb2/auction", nil)
					req.Header.Set(secCookieDeprecation, "sample-value")
					return req
				}(),
				givenAccount: &config.Account{
					ID: "1",
					Privacy: config.AccountPrivacy{
						PrivacySandbox: config.PrivacySandbox{
							CookieDeprecation: config.CookieDeprecation{
								Enabled: true,
								TTLSec:  86400,
							},
						},
					},
				},
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID:  "Some-ID",
						App: &openrtb2.App{},
						Imp: []openrtb2.Imp{
							{
								ID: "Some-Imp-ID",
								Banner: &openrtb2.Banner{
									Format: []openrtb2.Format{
										{
											W: 600,
											H: 500,
										},
										{
											W: 300,
											H: 600,
										},
									},
								},
								Ext: []byte(`{"pubmatic":{"publisherId": 12345678}}`),
							},
						},
					},
				},
				wantErrs: []error{},
				wantCDep: "sample-value",
			},
			{
				name: "header-with-length-more-than-100",
				httpReq: func() *http.Request {
					req := httptest.NewRequest("POST", "/openrtb2/auction", nil)
					req.Header.Set(secCookieDeprecation, "zjfXqGxXFI8yura8AhQl1DK2EMMmryrC8haEpAlwjoerrFfEo2MQTXUq6cSmLohI8gjsnkGU4oAzvXd4TTAESzEKsoYjRJ2zKxmEa")
					return req
				}(),
				givenAccount: &config.Account{
					ID: "1",
					Privacy: config.AccountPrivacy{
						PrivacySandbox: config.PrivacySandbox{
							CookieDeprecation: config.CookieDeprecation{
								Enabled: true,
								TTLSec:  86400,
							},
						},
					},
				},
				reqWrapper: &openrtb_ext.RequestWrapper{
					BidRequest: &openrtb2.BidRequest{
						ID:  "Some-ID",
						App: &openrtb2.App{},
						Imp: []openrtb2.Imp{
							{
								ID: "Some-Imp-ID",
								Banner: &openrtb2.Banner{
									Format: []openrtb2.Format{
										{
											W: 600,
											H: 500,
										},
										{
											W: 300,
											H: 600,
										},
									},
								},
								Ext: []byte(`{"pubmatic":{"publisherId": 12345678}}`),
							},
						},
					},
				},
				wantErrs: []error{
					&errortypes.Warning{
						Message:     "request.device.ext.cdep must not exceed 100 characters",
						WarningCode: errortypes.SecCookieDeprecationLenWarningCode,
					},
				},
				wantCDep: "",
			},
		}

	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&warningsCheckExchange{},
		ortb.NewRequestValidator(openrtb_ext.BuildBidderMap(), map[string]string{}, mockBidderParamValidator{}),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&mockAccountFetcher{},
		&config.Configuration{},
		&metricsConfig.NilMetricsEngine{},
		analyticsBuild.New(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
		hooks.EmptyPlanBuilder{},
		nil,
		openrtb_ext.NormalizeBidderName,
	}

	for _, test := range testCases {
		errs := deps.validateRequest(test.givenAccount, test.httpReq, test.reqWrapper, false, false, stored_responses.ImpBidderStoredResp{}, false)
		assert.Equal(t, test.wantErrs, errs)
		test.reqWrapper.RebuildRequest()
		deviceExt, err := test.reqWrapper.GetDeviceExt()
		assert.NoError(t, err)
		assert.Equal(t, test.wantCDep, deviceExt.GetCDep())
	}
}

func TestSetSecBrowsingTopicsImplicitly(t *testing.T) {
	type args struct {
		httpReq *http.Request
		r       *openrtb_ext.RequestWrapper
		account *config.Account
	}
	tests := []struct {
		name     string
		args     args
		wantUser *openrtb2.User
	}{
		{
			name: "empty HTTP request, no change in user data",
			args: args{
				httpReq: &http.Request{},
				r:       &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}},
				account: &config.Account{Privacy: config.AccountPrivacy{PrivacySandbox: config.PrivacySandbox{TopicsDomain: "ads.pubmatic.com"}}},
			},
			wantUser: nil,
		},
		{
			name: "valid topic in request but topicsdomain not configured by host, no change in user data",
			args: args{
				httpReq: &http.Request{
					Header: http.Header{
						secBrowsingTopics: []string{"(1);v=chrome.1:1:2, ();p=P00000000000"},
					},
				},
				r:       &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}},
				account: &config.Account{Privacy: config.AccountPrivacy{PrivacySandbox: config.PrivacySandbox{TopicsDomain: ""}}},
			},
			wantUser: nil,
		},
		{
			name: "valid topic in request and topicsdomain configured by host, topics copied to user data",
			args: args{
				httpReq: &http.Request{
					Header: http.Header{
						secBrowsingTopics: []string{"(1);v=chrome.1:1:2, ();p=P00000000000"},
					},
				},
				r:       &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}},
				account: &config.Account{Privacy: config.AccountPrivacy{PrivacySandbox: config.PrivacySandbox{TopicsDomain: "ads.pubmatic.com"}}},
			},
			wantUser: &openrtb2.User{
				Data: []openrtb2.Data{
					{
						Name: "ads.pubmatic.com",
						Segment: []openrtb2.Segment{
							{
								ID: "1",
							},
						},
						Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
					},
				},
			},
		},
		{
			name: "valid empty topic in request, no change in user data",
			args: args{
				httpReq: &http.Request{
					Header: http.Header{
						secBrowsingTopics: []string{"();p=P0000000000000000000000000000000"},
					},
				},
				r:       &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}},
				account: &config.Account{Privacy: config.AccountPrivacy{PrivacySandbox: config.PrivacySandbox{TopicsDomain: "ads.pubmatic.com"}}},
			},
			wantUser: nil,
		},
		{
			name: "request with a few valid topics (including duplicate topics, segIDs, matching segtax, segclass, etc) and a few invalid topics(different invalid format), only valid and unique topics copied/merged to/with user data",
			args: args{
				httpReq: &http.Request{
					Header: http.Header{
						secBrowsingTopics: []string{"(1);v=chrome.1:1:2, (1 2);v=chrome.1:1:2,(4);v=chrome.1:1:2,();p=P0000000000,(4);v=chrome.1, 5);v=chrome.1, (6;v=chrome.1, ();v=chrome.1, (	);v=chrome.1, (1);v=chrome.1:1:2, (1 2 4		6 7			4567	  ) ; v=chrome.1: 2 : 3,();p=P0000000000"},
					},
				},
				r: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Data: []openrtb2.Data{
							{
								Name: "chrome.com",
								Segment: []openrtb2.Segment{
									{ID: "1"},
								},
								Ext: json.RawMessage(`{"segtax":603,"segclass":"4"}`),
							},
							{
								Name: "ads.pubmatic.com",
								Segment: []openrtb2.Segment{
									{ID: "1"},
									{ID: "3"},
								},
								Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
							},
						},
					},
				}},
				account: &config.Account{Privacy: config.AccountPrivacy{PrivacySandbox: config.PrivacySandbox{TopicsDomain: "ads.pubmatic.com"}}},
			},
			wantUser: &openrtb2.User{
				Data: []openrtb2.Data{
					{
						Name: "chrome.com",
						Segment: []openrtb2.Segment{
							{ID: "1"},
						},
						Ext: json.RawMessage(`{"segtax":603,"segclass":"4"}`),
					},
					{
						Name: "ads.pubmatic.com",
						Segment: []openrtb2.Segment{
							{ID: "1"},
							{ID: "2"},
							{ID: "3"},
							{ID: "4"},
						},
						Ext: json.RawMessage(`{"segtax":600,"segclass":"2"}`),
					},
					{
						Name: "ads.pubmatic.com",
						Segment: []openrtb2.Segment{
							{ID: "1"},
							{ID: "2"},
							{ID: "4"},
							{ID: "6"},
							{ID: "7"},
							{ID: "4567"},
						},
						Ext: json.RawMessage(`{"segtax":601,"segclass":"3"}`),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setSecBrowsingTopicsImplicitly(tt.args.httpReq, tt.args.r, tt.args.account)

			// sequence is not guaranteed we're using a map to filter segids
			sortUserData(tt.wantUser)
			sortUserData(tt.args.r.User)
			assert.Equal(t, tt.wantUser, tt.args.r.User, tt.name)
		})
	}
}

func sortUserData(user *openrtb2.User) {
	if user != nil {
		sort.Slice(user.Data, func(i, j int) bool {
			if user.Data[i].Name == user.Data[j].Name {
				return string(user.Data[i].Ext) < string(user.Data[j].Ext)
			}
			return user.Data[i].Name < user.Data[j].Name
		})
		for g := range user.Data {
			sort.Slice(user.Data[g].Segment, func(i, j int) bool {
				return user.Data[g].Segment[i].ID < user.Data[g].Segment[j].ID
			})
		}
	}
}
func TestValidateEIDs(t *testing.T) {
	testCases := []struct {
		name                  string
		input                 []openrtb2.EID
		expected              []openrtb2.EID
		numErrors             int
		expectedErrorMessages []string
	}{
		{
			name: "one-eid-one-uid-valid",
			input: []openrtb2.EID{
				{Source: "src1", UIDs: []openrtb2.UID{{ID: "id1"}}},
			},
			expected: []openrtb2.EID{
				{Source: "src1", UIDs: []openrtb2.UID{{ID: "id1"}}},
			},
			numErrors:             0,
			expectedErrorMessages: nil,
		},
		{
			name: "one-eid-one-uid-empty",
			input: []openrtb2.EID{
				{Source: "src2", UIDs: []openrtb2.UID{{ID: ""}}},
			},
			expected:  []openrtb2.EID{},
			numErrors: 2,
			expectedErrorMessages: []string{
				"request.user.eids[0].uids[0] removed due to empty ids",
				"request.user.eids[0] (source: src2) removed due to empty uids",
			},
		},
		{
			name: "many-mixed",
			input: []openrtb2.EID{
				{Source: "src3", UIDs: []openrtb2.UID{{ID: "ID1"}, {ID: "ID2"}}},
				{Source: "src4", UIDs: []openrtb2.UID{{ID: ""}, {ID: "ID1"}}},
				{Source: "src5", UIDs: []openrtb2.UID{{ID: ""}, {ID: ""}}},
			},
			expected: []openrtb2.EID{
				{Source: "src3", UIDs: []openrtb2.UID{{ID: "ID1"}, {ID: "ID2"}}},
				{Source: "src4", UIDs: []openrtb2.UID{{ID: "ID1"}}},
			},
			numErrors: 4,
			expectedErrorMessages: []string{
				"request.user.eids[1].uids[0] removed due to empty ids",
				"request.user.eids[2].uids[0] removed due to empty ids",
				"request.user.eids[2].uids[1] removed due to empty ids",
				"request.user.eids[2] (source: src5) removed due to empty uids",
			},
		},
		{
			name: "one-eid-many-uid-empty",
			input: []openrtb2.EID{
				{Source: "src6", UIDs: []openrtb2.UID{{ID: ""}, {ID: ""}}},
			},
			expected:  []openrtb2.EID{},
			numErrors: 3,
			expectedErrorMessages: []string{
				"request.user.eids[0].uids[0] removed due to empty ids",
				"request.user.eids[0].uids[1] removed due to empty ids",
				"request.user.eids[0] (source: src6) removed due to empty uids",
			},
		},
		{
			name:                  "eid_nil",
			input:                 nil,
			expected:              nil,
			numErrors:             0,
			expectedErrorMessages: nil,
		},
		{
			name: "eid_uid_nil",
			input: []openrtb2.EID{
				{Source: "src7", UIDs: nil},
			},
			expected:  []openrtb2.EID{},
			numErrors: 1,
			expectedErrorMessages: []string{
				"request.user.eids[0] (source: src7) removed due to empty uids",
			},
		},
		{
			name: "source_missing",
			input: []openrtb2.EID{
				{UIDs: []openrtb2.UID{{ID: "id1"}}},
			},
			expected:  []openrtb2.EID{},
			numErrors: 1,
			expectedErrorMessages: []string{
				"request.user.eids[0] removed due to missing source",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validEIDs, errorsList := validateEIDs(tc.input)

			assert.ElementsMatch(t, tc.expected, validEIDs)
			assert.Equal(t, tc.numErrors, len(errorsList))

			// Assert error messages
			assert.Equal(t, len(tc.expectedErrorMessages), len(errorsList))
			for _, err := range errorsList {
				assert.Contains(t, tc.expectedErrorMessages, err.Error())
			}

		})
	}
}

func TestValidateUIDs(t *testing.T) {
	testCases := []struct {
		name              string
		input             []openrtb2.UID
		expectedValidUIDs []openrtb2.UID
		expectedErrors    []error
	}{
		{
			name: "many-uid-valid",
			input: []openrtb2.UID{
				{ID: "id1"},
				{ID: "id2"},
			},
			expectedValidUIDs: []openrtb2.UID{
				{ID: "id1"},
				{ID: "id2"},
			},
			expectedErrors: nil,
		},
		{
			name: "many-uid-empty",
			input: []openrtb2.UID{
				{ID: ""},
				{ID: ""},
			},
			expectedValidUIDs: nil,
			expectedErrors: []error{
				&errortypes.Warning{
					Message:     "request.user.eids[0].uids[0] removed due to empty ids",
					WarningCode: errortypes.InvalidUserUIDsWarningCode,
				},
				&errortypes.Warning{
					Message:     "request.user.eids[0].uids[1] removed due to empty ids",
					WarningCode: errortypes.InvalidUserUIDsWarningCode,
				},
			},
		},
		{
			name: "many-mixed",
			input: []openrtb2.UID{
				{ID: "id1"},
				{ID: ""},
				{ID: "id2"},
				{ID: ""},
			},
			expectedValidUIDs: []openrtb2.UID{
				{ID: "id1"},
				{ID: "id2"},
			},
			expectedErrors: []error{
				&errortypes.Warning{
					Message:     "request.user.eids[0].uids[1] removed due to empty ids",
					WarningCode: errortypes.InvalidUserUIDsWarningCode,
				},
				&errortypes.Warning{
					Message:     "request.user.eids[0].uids[3] removed due to empty ids",
					WarningCode: errortypes.InvalidUserUIDsWarningCode,
				},
			},
		},
		{
			name:              "empty-uid",
			input:             []openrtb2.UID{},
			expectedValidUIDs: nil,
			expectedErrors:    nil,
		},
		{
			name:              "nil-uid",
			input:             nil,
			expectedValidUIDs: nil,
			expectedErrors:    nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validUIDs, errorsList := validateUIDs(tc.input, 0)

			assert.ElementsMatch(t, tc.expectedValidUIDs, validUIDs)
			assert.ElementsMatch(t, tc.expectedErrors, errorsList)
		})
	}
}
func TestValidateUser(t *testing.T) {

	testCases := []struct {
		name         string
		req          *openrtb_ext.RequestWrapper
		expectedErr  []error
		expectedEids []openrtb2.EID
	}{
		{
			name: "Valid_user_with_Geo_accuracy",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Geo: &openrtb2.Geo{
							Accuracy: 10,
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "Invalid_user_with_negative_Geo_accuracy",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Geo: &openrtb2.Geo{
							Accuracy: -10,
						},
					},
				},
			},
			expectedErr: []error{errors.New("request.user.geo.accuracy must be a positive number")},
		},
		{
			name: "Invalid_user.ext.prebid_with_empty_buyeruids",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						Ext: json.RawMessage(`{"prebid": {"buyeruids": {}}}`),
					},
				},
			},
			expectedErr: []error{errors.New(`request.user.ext.prebid requires a "buyeruids" property with at least one ID defined. If none exist, then request.user.ext.prebid should not be defined.`)},
		},
		{
			name: "Invalid_user.eids_with_empty_id",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						EIDs: []openrtb2.EID{
							{
								Source: "source",
								UIDs: []openrtb2.UID{
									{ID: ""},
								},
							},
						},
					},
				},
			},
			expectedErr: []error{&errortypes.Warning{
				Message:     "request.user.eids[0].uids[0] removed due to empty ids",
				WarningCode: errortypes.InvalidUserUIDsWarningCode,
			}, &errortypes.Warning{
				Message:     "request.user.eids[0] (source: source) removed due to empty uids",
				WarningCode: errortypes.InvalidUserEIDsWarningCode,
			}},
			expectedEids: nil,
		},
		{
			name: "Valid_user.eids_with_UID1_id",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						EIDs: []openrtb2.EID{
							{
								Source: "source",
								UIDs: []openrtb2.UID{
									{ID: "UID1"},
								},
							},
						},
					},
				},
			},
			expectedErr: nil,
			expectedEids: []openrtb2.EID{
				{
					Source: "source",
					UIDs: []openrtb2.UID{
						{ID: "UID1"},
					},
				},
			},
		},
		{
			name: "user.eids_with_empty_UIDs",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						EIDs: []openrtb2.EID{
							{
								Source: "source",
								UIDs: []openrtb2.UID{
									{},
								},
							},
						},
					},
				},
			},
			expectedErr: []error{
				&errortypes.Warning{
					Message:     "request.user.eids[0].uids[0] removed due to empty ids",
					WarningCode: errortypes.InvalidUserUIDsWarningCode,
				},
				&errortypes.Warning{
					Message:     "request.user.eids[0] (source: source) removed due to empty uids",
					WarningCode: errortypes.InvalidUserEIDsWarningCode,
				},
			},
			expectedEids: nil,
		},
		{
			name: "user.eids_with_empty_UIDs",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						EIDs: []openrtb2.EID{
							{
								Source: "source",
								UIDs:   []openrtb2.UID{},
							},
						},
					},
				},
			},
			expectedErr: []error{
				&errortypes.Warning{
					Message:     "request.user.eids[0] (source: source) removed due to empty uids",
					WarningCode: errortypes.InvalidUserEIDsWarningCode,
				},
			},
			expectedEids: nil,
		},
		{
			name: "user.eids_with_No_UIDs",
			req: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					User: &openrtb2.User{
						EIDs: []openrtb2.EID{
							{
								Source: "source",
							},
						},
					},
				},
			},
			expectedErr: []error{
				&errortypes.Warning{
					Message:     "request.user.eids[0] (source: source) removed due to empty uids",
					WarningCode: errortypes.InvalidUserEIDsWarningCode,
				},
			},
			expectedEids: nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			deps := &endpointDeps{
				bidderMap: map[string]openrtb_ext.BidderName{
					"appnexus": "appnexus",
				},
			}
			errs := deps.validateUser(test.req, nil, gpplib.GppContainer{})
			assert.Equal(t, test.expectedErr, errs)
			if test.req.User != nil {
				assert.ElementsMatch(t, test.expectedEids, test.req.User.EIDs)
			}
		})
	}
}
