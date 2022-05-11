package openrtb2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/metrics"
	metricsConfig "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/stored_responses"
	"github.com/prebid/prebid-server/util/iputil"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb/v15/native1"
	nativeRequests "github.com/mxmCherry/openrtb/v15/native1/request"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/stretchr/testify/assert"
)

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
			"Makes sure we handle (default) aliased bidders properly",
			"aliased",
		},
		{
			"Asserts we return 503s on requests with blacklisted accounts and apps.",
			"blacklisted",
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
	}

	for _, tc := range testSuites {
		testCaseFiles, err := getTestFiles(filepath.Join("sample-requests", tc.sampleRequestsSubDir))
		if assert.NoError(t, err, "Test case %s. Error reading files from directory %s \n", tc.description, tc.sampleRequestsSubDir) {
			for _, testFile := range testCaseFiles {
				fileData, err := ioutil.ReadFile(testFile)
				if assert.NoError(t, err, "Test case %s. Error reading file %s \n", tc.description, testFile) {
					// Retrieve test case input and expected output from JSON file
					test, err := parseTestFile(fileData, testFile)
					if !assert.NoError(t, err) {
						continue
					}

					// Build endpoint for testing. If no error, run test case
					cfg := &config.Configuration{MaxRequestSize: maxSize}
					if test.Config != nil {
						cfg.BlacklistedApps = test.Config.BlacklistedApps
						cfg.BlacklistedAppMap = test.Config.getBlacklistedAppMap()
						cfg.BlacklistedAccts = test.Config.BlacklistedAccounts
						cfg.BlacklistedAcctMap = test.Config.getBlackListedAccountMap()
						cfg.AccountRequired = test.Config.AccountRequired
					}
					test.endpointType = OPENRTB_ENDPOINT

					auctionEndpointHandler, mockBidServers, mockCurrencyRatesServer, err := buildTestEndpoint(test, cfg)
					if assert.NoError(t, err) {
						runTestCase(t, auctionEndpointHandler, test, fileData, testFile)
					}

					// Close servers regardless if the test case was run or not
					for _, mockBidServer := range mockBidServers {
						mockBidServer.Close()
					}
					mockCurrencyRatesServer.Close()
				}
			}
		}
	}
}

func runTestCase(t *testing.T, auctionEndpointHandler httprouter.Handle, test testCase, fileData []byte, testFile string) {
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
		assert.True(t, strings.HasPrefix(actualJsonBidResponse, test.ExpectedErrorMessage), "Actual: %s \nExpected: %s. Filename: %s \n", actualJsonBidResponse, test.ExpectedErrorMessage, testFile)
	}

	if len(test.ExpectedBidResponse) > 0 {
		var expectedBidResponse openrtb2.BidResponse
		var actualBidResponse openrtb2.BidResponse
		var err error

		err = json.Unmarshal(test.ExpectedBidResponse, &expectedBidResponse)
		if assert.NoError(t, err, "Could not unmarshal expected bidResponse taken from test file.\n Test file: %s\n Error:%s\n", testFile, err) {
			err = json.Unmarshal([]byte(actualJsonBidResponse), &actualBidResponse)
			if assert.NoError(t, err, "Could not unmarshal actual bidResponse from auction.\n Test file: %s\n Error:%s\n", testFile, err) {
				assertBidResponseEqual(t, testFile, expectedBidResponse, actualBidResponse)
			}
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
		mockBidderParamValidator{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		cfg,
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{})

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
	doBadAliasRequest(t, "sample-requests/invalid-stored/bad_stored_imp.json", "Invalid request: Invalid JSON in Default Request Settings: invalid character '\"' after object key:value pair at offset 51\n")
	doBadAliasRequest(t, "sample-requests/invalid-stored/bad_incoming_imp.json", "Invalid request: Invalid JSON in Incoming Request: invalid character '\"' after object key:value pair at offset 230\n")
}

// doBadAliasRequest() is a customized variation of doRequest(), above
func doBadAliasRequest(t *testing.T, filename string, expectMsg string) {
	t.Helper()
	fileData := readFile(t, filename)
	testBidRequest, _, _, err := jsonparser.Get(fileData, "mockBidRequest")
	assert.NoError(t, err, "Error jsonparsing root.mockBidRequest from file %s. Desc: %v.", filename, err)

	// aliasJSON lacks a comma after the "appnexus" entry so is bad JSON
	aliasJSON := []byte(`{"ext":{"prebid":{"aliases": {"test1": "appnexus" "test2": "rubicon", "test3": "openx"}}}}`)
	adaptersConfigs := make(map[string]config.Adapter)

	bidderInfos := getBidderInfos(adaptersConfigs, openrtb_ext.CoreBidderNames())

	bidderMap := exchange.GetActiveBidders(bidderInfos)
	disabledBidders := exchange.GetDisabledBiddersErrorMessages(bidderInfos)

	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	endpoint, _ := NewEndpoint(
		fakeUUIDGenerator{},
		&nobidExchange{},
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		disabledBidders,
		aliasJSON,
		bidderMap,
		empty_fetcher.EmptyFetcher{})

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
		mockBidderParamValidator{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}), map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{})

	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil Exchange.")
	}
}

// TestNilValidator makes sure we fail when given nil for the BidderParamValidator.
func TestNilValidator(t *testing.T) {
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	_, err := NewEndpoint(
		fakeUUIDGenerator{},
		&nobidExchange{},
		nil,
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{})

	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil BidderParamValidator.")
	}
}

// TestExchangeError makes sure we return a 500 if the exchange auction fails.
func TestExchangeError(t *testing.T) {
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	endpoint, _ := NewEndpoint(
		fakeUUIDGenerator{},
		&brokenExchange{},
		mockBidderParamValidator{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{})

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
	bidReq := &openrtb2.BidRequest{}

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
	bidReq := &openrtb2.BidRequest{
		Device: &openrtb2.Device{
			UA: "bar",
		},
	}

	setUAImplicitly(httpReq, bidReq)

	if bidReq.Device.UA != "bar" {
		t.Errorf("bidrequest.device.ua should have been \"bar\". Got %s", bidReq.Device.UA)
	}
}

func TestAuctionTypeDefault(t *testing.T) {
	bidReq := &openrtb2.BidRequest{}
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
			mockBidderParamValidator{},
			&mockStoredReqFetcher{},
			empty_fetcher.EmptyFetcher{},
			cfg,
			&metricsConfig.NilMetricsEngine{},
			analyticsConf.NewPBSAnalytics(&config.Analytics{}),
			map[string]string{},
			[]byte{},
			openrtb_ext.BuildBidderMap(),
			empty_fetcher.EmptyFetcher{})

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
		setDoNotTrackImplicitly(httpReq, &test.request)
		assert.Equal(t, test.expectedRequest, test.request)
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
			mockBidderParamValidator{},
			&mockStoredReqFetcher{},
			empty_fetcher.EmptyFetcher{},
			&config.Configuration{MaxRequestSize: maxSize},
			&metricsConfig.NilMetricsEngine{},
			analyticsConf.NewPBSAnalytics(&config.Analytics{}),
			map[string]string{},
			[]byte{},
			openrtb_ext.BuildBidderMap(),
			empty_fetcher.EmptyFetcher{})

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

func TestImplicitSecure(t *testing.T) {
	httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	httpReq.Header.Set(http.CanonicalHeaderKey("X-Forwarded-Proto"), "https")

	imps := []openrtb2.Imp{
		{},
		{},
	}
	setImpsImplicitly(httpReq, imps)
	for _, imp := range imps {
		if imp.Secure == nil || *imp.Secure != 1 {
			t.Errorf("imp.Secure should be set to 1 if the request is https. Got %#v", imp.Secure)
		}
	}
}

func TestRefererParsing(t *testing.T) {
	httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	httpReq.Header.Set("Referer", "http://test.mysite.com")
	bidReq := &openrtb2.BidRequest{}

	setSiteImplicitly(httpReq, bidReq)

	if bidReq.Site == nil {
		t.Fatalf("bidrequest.site should not be nil.")
	}

	if bidReq.Site.Domain != "mysite.com" {
		t.Errorf("Bad bidrequest.site.domain. Expected mysite.com, got %s", bidReq.Site.Domain)
	}
	if bidReq.Site.Page != "http://test.mysite.com" {
		t.Errorf("Bad bidrequest.site.page. Expected mysite.com, got %s", bidReq.Site.Page)
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
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
	}

	testStoreVideoAttr := []bool{true, true, false, false}

	for i, requestData := range testStoredRequests {
		impInfo, errs := parseImpInfo([]byte(requestData))
		assert.Len(t, errs, 0, "No errors should be returned")
		newRequest, impExtInfoMap, errList := deps.processStoredRequests(context.Background(), json.RawMessage(requestData), impInfo)
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
		actualErr := mergeBidderParams(&openrtb_ext.RequestWrapper{BidRequest: &test.givenRequest})

		// errors are only possible from the marshal operation, which is not testable
		assert.NoError(t, actualErr, test.description+":err")

		assert.Equal(t, test.givenRequest.Imp, test.expectedRequestImps, test.description+":imps")
	}
}

func TestAddMissingReqExtParamsInImpExtPrebid(t *testing.T) {
	testCases := []struct {
		description            string
		givenImpExtByBidder    map[string]json.RawMessage
		givenReqExtParams      map[string]map[string]json.RawMessage
		expectedModified       bool
		expectedImpExtByBidder map[string]json.RawMessage
	}{
		{
			description:            "No Prebid Section",
			givenImpExtByBidder:    map[string]json.RawMessage{},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:       false,
			expectedImpExtByBidder: map[string]json.RawMessage{},
		},
		{
			description:            "Malformed Prebid Section",
			givenImpExtByBidder:    map[string]json.RawMessage{"prebid": json.RawMessage(`malformed`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:       false,
			expectedImpExtByBidder: map[string]json.RawMessage{"prebid": json.RawMessage(`malformed`)},
		},
		{
			description:            "No Prebid Bidder Section",
			givenImpExtByBidder:    map[string]json.RawMessage{"prebid": json.RawMessage(`{}`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:       false,
			expectedImpExtByBidder: map[string]json.RawMessage{"prebid": json.RawMessage(`{}`)},
		},
		{
			description:            "Malformed Prebid Bidder Section",
			givenImpExtByBidder:    map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder": malformed}`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:       false,
			expectedImpExtByBidder: map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder": malformed}`)},
		},
		{
			description:            "One Bidder - Modified (no collision)",
			givenImpExtByBidder:    map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder":{"bidder1":{"a":1}}}`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:       true,
			expectedImpExtByBidder: map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder":{"bidder1":{"a":1,"b":2}}}`)},
		},
		{
			description:            "One Bidder - Not Modified (imp.ext wins)",
			givenImpExtByBidder:    map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder":{"bidder1":{"a":1,"b":2}}}`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`4`)}},
			expectedModified:       false,
			expectedImpExtByBidder: map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder":{"bidder1":{"a":1,"b":2}}}`)},
		},
		{
			description:            "One Bidder - Not Modified (reserved bidder ignored)",
			givenImpExtByBidder:    map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder":{"gpid":{"a":1}}}`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"gpid": {"b": json.RawMessage(`2`)}},
			expectedModified:       false,
			expectedImpExtByBidder: map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder":{"gpid":{"a":1}}}`)},
		},
		{
			description:            "One Bidder - Not Modified (reserved bidder ignored - not embedded object)",
			givenImpExtByBidder:    map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder":{"gpid":1}}`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"gpid": {"b": json.RawMessage(`2`)}},
			expectedModified:       false,
			expectedImpExtByBidder: map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder":{"gpid":1}}`)},
		},
		{
			description:            "One Bidder - Not Modified (malformed ignored)",
			givenImpExtByBidder:    map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder":{"bidder1":malformed}}`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:       false,
			expectedImpExtByBidder: map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder":{"bidder1":malformed}}`)},
		},
		{
			description:            "Multiple Bidders - Mixed",
			givenImpExtByBidder:    map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder":{"bidder1":{"a":1},"bidder2":{"a":"one","b":"two"}}}`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}, "bidder2": {"b": json.RawMessage(`"three"`)}},
			expectedModified:       true,
			expectedImpExtByBidder: map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder":{"bidder1":{"a":1,"b":2},"bidder2":{"a":"one","b":"two"}}}`)},
		},
		{
			description:            "Multiple Bidders - None Modified",
			givenImpExtByBidder:    map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder":{"bidder1":{"a":1},"bidder2":{"a":"one","b":"two"}}}`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{},
			expectedModified:       false,
			expectedImpExtByBidder: map[string]json.RawMessage{"prebid": json.RawMessage(`{"bidder":{"bidder1":{"a":1},"bidder2":{"a":"one","b":"two"}}}`)},
		},
	}

	for _, test := range testCases {
		actualModified, actualErr := addMissingReqExtParamsInImpExtPrebid(test.givenImpExtByBidder, test.givenReqExtParams)

		// errors are only possible from the marshal operation, which is not testable
		assert.NoError(t, actualErr, test.description+":err")

		assert.Equal(t, test.expectedModified, actualModified, test.description+":modified")
		assert.Equal(t, test.expectedImpExtByBidder, test.givenImpExtByBidder, test.description+":imp.ext")
	}
}

func TestAddMissingReqExtParamsInImpExt(t *testing.T) {
	testCases := []struct {
		description            string
		givenImpExtByBidder    map[string]json.RawMessage
		givenReqExtParams      map[string]map[string]json.RawMessage
		expectedModified       bool
		expectedImpExtByBidder map[string]json.RawMessage
	}{
		{
			description:            "One Bidder - Modified (no collision)",
			givenImpExtByBidder:    map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1}`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:       true,
			expectedImpExtByBidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`)},
		},
		{
			description:            "One Bidder - Not Modified (imp.ext wins)",
			givenImpExtByBidder:    map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`4`)}},
			expectedModified:       false,
			expectedImpExtByBidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`)},
		},
		{
			description:            "One Bidder - Not Modified (reserved bidder ignored)",
			givenImpExtByBidder:    map[string]json.RawMessage{"gpid": json.RawMessage(`{"a":1}`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"gpid": {"b": json.RawMessage(`2`)}},
			expectedModified:       false,
			expectedImpExtByBidder: map[string]json.RawMessage{"gpid": json.RawMessage(`{"a":1}`)},
		},
		{
			description:            "One Bidder - Not Modified (reserved bidder ignored - not embedded object)",
			givenImpExtByBidder:    map[string]json.RawMessage{"gpid": json.RawMessage(`1`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"gpid": {"b": json.RawMessage(`2`)}},
			expectedModified:       false,
			expectedImpExtByBidder: map[string]json.RawMessage{"gpid": json.RawMessage(`1`)},
		},
		{
			description:            "One Bidder - Not Modified (malformed ignored)",
			givenImpExtByBidder:    map[string]json.RawMessage{"bidder1": json.RawMessage(`malformed`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}},
			expectedModified:       false,
			expectedImpExtByBidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`malformed`)},
		},
		{
			description:            "Multiple Bidders - Mixed",
			givenImpExtByBidder:    map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1}`), "bidder2": json.RawMessage(`{"a":"one","b":"two"}`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{"bidder1": {"b": json.RawMessage(`2`)}, "bidder2": {"b": json.RawMessage(`"three"`)}},
			expectedModified:       true,
			expectedImpExtByBidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1,"b":2}`), "bidder2": json.RawMessage(`{"a":"one","b":"two"}`)},
		},
		{
			description:            "Multiple Bidders - None Modified",
			givenImpExtByBidder:    map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1}`), "bidder2": json.RawMessage(`{"a":"one","b":"two"}`)},
			givenReqExtParams:      map[string]map[string]json.RawMessage{},
			expectedModified:       false,
			expectedImpExtByBidder: map[string]json.RawMessage{"bidder1": json.RawMessage(`{"a":1}`), "bidder2": json.RawMessage(`{"a":"one","b":"two"}`)},
		},
	}

	for _, test := range testCases {
		actualModified, actualErr := addMissingReqExtParamsInImpExt(test.givenImpExtByBidder, test.givenReqExtParams)

		// errors are only possible from the marshal operation, which is not testable
		assert.NoError(t, actualErr, test.description+":err")

		assert.Equal(t, test.expectedModified, actualModified, test.description+":modified")
		assert.Equal(t, test.expectedImpExtByBidder, test.givenImpExtByBidder, test.description+":imp.ext")
	}
}

func TestValidateRequest(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
	}

	testCases := []struct {
		description           string
		givenIsAmp            bool
		givenRequestWrapper   *openrtb_ext.RequestWrapper
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
					Ext: []byte(`{"prebid":{"aliases":{"brightroll":"appnexus"}, "aliasgvlids":{"pubmatic1":1}}}`),
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
					Ext: []byte(`{"prebid":{"aliases":{"brightroll":"appnexus"}, "aliasgvlids":{"brightroll":0}}}`),
				},
			},
			givenIsAmp:            false,
			expectedErrorList:     []error{errors.New("request.ext.prebid.aliasgvlids. Invalid vendorId 0 for alias: brightroll. Choose a different vendorId, or remove this entry.")},
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
					Ext: []byte(`{"prebid":{"aliases":{"brightroll":"appnexus"}, "aliasgvlids":{"brightroll":1}}}`),
				},
			},
			givenIsAmp:            false,
			expectedErrorList:     []error{},
			expectedChannelObject: &openrtb_ext.ExtRequestPrebidChannel{Name: appChannel, Version: ""},
		},
	}

	for _, test := range testCases {
		errorList := deps.validateRequest(test.givenRequestWrapper, test.givenIsAmp, false, nil)
		assert.Equalf(t, test.expectedErrorList, errorList, "Error doesn't match: %s\n", test.description)

		if len(errorList) == 0 {
			requestExt, err := test.givenRequestWrapper.GetRequestExt()
			assert.Empty(t, err, test.description)
			requestPrebid := requestExt.GetPrebid()

			assert.Equalf(t, test.expectedChannelObject, requestPrebid.Channel, "Channel information isn't correct: %s\n", test.description)
		}
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
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
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
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
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
		newRequest, _, errList := deps.processStoredRequests(context.Background(), json.RawMessage(test.givenRawData), impInfo)
		assert.Empty(t, errList, test.description)

		if err := json.Unmarshal(newRequest, req); err != nil {
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
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(len(reqBody) - 1)},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
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
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(len(reqBody))},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
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
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
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

	bidReq := openrtb2.BidRequest{
		Site: &openrtb2.Site{},
	}
	setSiteImplicitly(httpReq, &bidReq)
	assert.JSONEq(t, `{"amp":0}`, string(bidReq.Site.Ext))
}

func TestImplicitAMPOtherExt(t *testing.T) {
	httpReq, err := http.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	if !assert.NoError(t, err) {
		return
	}

	bidReq := openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Ext: json.RawMessage(`{"other":true}`),
		},
	}
	setSiteImplicitly(httpReq, &bidReq)
	assert.JSONEq(t, `{"amp":0,"other":true}`, string(bidReq.Site.Ext))
}

func TestExplicitAMP(t *testing.T) {
	httpReq, err := http.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site-amp.json")))
	if !assert.NoError(t, err) {
		return
	}

	bidReq := openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Ext: json.RawMessage(`{"amp":1}`),
		},
	}
	setSiteImplicitly(httpReq, &bidReq)
	assert.JSONEq(t, `{"amp":1}`, string(bidReq.Site.Ext))
}

// TestContentType prevents #328
func TestContentType(t *testing.T) {
	endpoint, _ := NewEndpoint(
		fakeUUIDGenerator{},
		&mockExchange{},
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{},
	)
	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type should be application/json. Got %s", recorder.Header().Get("Content-Type"))
	}
}

func TestValidateImpExt(t *testing.T) {
	type testCase struct {
		description    string
		impExt         json.RawMessage
		expectedImpExt string
		expectedErrs   []error
	}
	testGroups := []struct {
		description string
		testCases   []testCase
	}{
		{
			"Empty",
			[]testCase{
				{
					description:    "Empty",
					impExt:         nil,
					expectedImpExt: "",
					expectedErrs:   []error{errors.New("request.imp[0].ext is required")},
				},
			},
		},
		{
			"Unknown bidder tests",
			[]testCase{
				{
					description:    "Unknown Bidder only",
					impExt:         json.RawMessage(`{"unknownbidder":{"placement_id":555}}`),
					expectedImpExt: `{"unknownbidder":{"placement_id":555}}`,
					expectedErrs:   []error{errors.New("request.imp[0].ext contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
				},
				{
					description:    "Unknown Prebid Ext Bidder only",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"unknownbidder":{"placement_id":555}}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"unknownbidder":{"placement_id":555}}}}`,
					expectedErrs:   []error{errors.New("request.imp[0].ext contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
				},
				{
					description:    "Unknown Prebid Ext Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"unknownbidder":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"unknownbidder":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs:   []error{errors.New("request.imp[0].ext contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
				},
				{
					description:    "Unknown Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"unknownbidder":{"placement_id":555} ,"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"unknownbidder":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs:   []error{errors.New("request.imp[0].ext contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
				},
				{
					description:    "Unknown Bidder + Disabled Bidder",
					impExt:         json.RawMessage(`{"unknownbidder":{"placement_id":555},"disabledbidder":{"foo":"bar"}}`),
					expectedImpExt: `{"unknownbidder":{"placement_id":555},"disabledbidder":{"foo":"bar"}}`,
					expectedErrs:   []error{errors.New("request.imp[0].ext contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
				},
				{
					description:    "Unknown Bidder + Disabled Prebid Ext Bidder",
					impExt:         json.RawMessage(`{"unknownbidder":{"placement_id":555},"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}}}`),
					expectedImpExt: `{"unknownbidder":{"placement_id":555},"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}}}`,
					expectedErrs:   []error{errors.New("request.imp[0].ext contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
				},
			},
		},
		{
			"Disabled bidder tests",
			[]testCase{
				{
					description:    "Disabled Bidder",
					impExt:         json.RawMessage(`{"disabledbidder":{"foo":"bar"}}`),
					expectedImpExt: `{}`,
					expectedErrs: []error{
						&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."},
						errors.New("request.imp[0].ext must contain at least one bidder"),
					},
					// if only bidder(s) found in request.imp[x].ext.{biddername} or request.imp[x].ext.prebid.bidder.{biddername} are disabled, return error
				},
				{
					description:    "Disabled Prebid Ext Bidder",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}}}`,
					expectedErrs: []error{
						&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."},
						errors.New("request.imp[0].ext must contain at least one bidder"),
					},
				},
				{
					description:    "Disabled Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"disabledbidder":{"foo":"bar"},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs: []error{
						&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."},
						errors.New("request.imp[0].ext must contain at least one bidder"),
					},
				},
				{
					description:    "Disabled Prebid Ext Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"context":{"data":{"keywords":"prebid server example"}}, "prebid":{"bidder":{"disabledbidder":{"foo":"bar"}}}}`,
					expectedErrs: []error{
						&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."},
						errors.New("request.imp[0].ext must contain at least one bidder"),
					},
				},
			},
		},
		{
			"First Party only",
			[]testCase{
				{
					description:    "First Party Data Context",
					impExt:         json.RawMessage(`{"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs: []error{
						errors.New("request.imp[0].ext must contain at least one bidder"),
					},
				},
			},
		},
		{
			"Valid bidder tests",
			[]testCase{
				{
					description:    "Valid bidder root ext",
					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555}}`),
					expectedImpExt: `{"appnexus":{"placement_id":555}}`,
					expectedErrs:   []error{},
				},
				{
					description:    "Valid bidder in prebid field",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555}}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}}}`,
					expectedErrs:   []error{},
				},
				{
					description:    "Valid Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"appnexus":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs:   []error{},
				},
				{
					description:    "Valid Prebid Ext Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555}}} ,"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs:   []error{},
				},
				{
					description:    "Valid Bidder + Unknown Bidder",
					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"unknownbidder":{"placement_id":555}}`),
					expectedImpExt: `{"appnexus":{"placement_id":555},"unknownbidder":{"placement_id":555}}`,
					expectedErrs:   []error{errors.New("request.imp[0].ext contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
				},
				{
					description:    "Valid Bidder + Disabled Bidder",
					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"}}`),
					expectedImpExt: `{"appnexus":{"placement_id":555}}`,
					expectedErrs:   []error{&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."}},
				},
				{
					description:    "Valid Bidder + Disabled Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"appnexus":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs:   []error{&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."}},
				},
				{
					description:    "Valid Bidder + Disabled Bidder + Unknown Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"},"unknownbidder":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"},"unknownbidder":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs:   []error{errors.New("request.imp[0].ext contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
				},
				{
					description:    "Valid Prebid Ext Bidder + Disabled Bidder Ext",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"}}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id": 555},"disabledbidder":{"foo":"bar"}}},"appnexus":{"placement_id":555}}`,
					expectedErrs:   []error{&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."}},
				},
				{
					description:    "Valid Prebid Ext Bidder + Disabled Ext Bidder + First Party Data Context",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"}}},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"prebid":{"bidder":{"appnexus":{"placement_id": 555},"disabledbidder":{"foo":"bar"}}},"appnexus":{"placement_id":555},"context":{"data":{"keywords":"prebid server example"}}}`,
					expectedErrs:   []error{&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'disabledbidder' has been disabled."}},
				},
				{
					description:    "Valid Prebid Ext Bidder + Disabled Ext Bidder + Unknown Ext + First Party Data Context",
					impExt:         json.RawMessage(`{"prebid":{"bidder":{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"},"unknownbidder":{"placement_id":555}}},"context":{"data":{"keywords":"prebid server example"}}}`),
					expectedImpExt: `{"context":{"data":{"keywords":"prebid server example"}},"prebid":{"bidder":{"appnexus":{"placement_id":555},"disabledbidder":{"foo":"bar"},"unknownbidder":{"placement_id":555}}}}`,
					expectedErrs:   []error{errors.New("request.imp[0].ext contains unknown bidder: unknownbidder. Did you forget an alias in request.ext.prebid.aliases?")},
				},
			},
		},
	}

	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(8096)},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{"disabledbidder": "The bidder 'disabledbidder' has been disabled."},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
	}

	for _, group := range testGroups {
		for _, test := range group.testCases {
			imp := &openrtb2.Imp{Ext: test.impExt}

			errs := deps.validateImpExt(imp, nil, 0, false, nil)

			if len(test.expectedImpExt) > 0 {
				assert.JSONEq(t, test.expectedImpExt, string(imp.Ext), "imp.ext JSON does not match expected. Test: %s. %s\n", group.description, test.description)
			} else {
				assert.Empty(t, imp.Ext, "imp.ext expected to be empty but was: %s. Test: %s. %s\n", string(imp.Ext), group.description, test.description)
			}
			assert.Equal(t, test.expectedErrs, errs, "errs slice does not match expected. Test: %s. %s\n", group.description, test.description)
		}
	}
}

func validRequest(t *testing.T, filename string) string {
	requestData, err := ioutil.ReadFile("sample-requests/valid-whole/supplementary/" + filename)
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
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
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

	errL := deps.validateRequest(&openrtb_ext.RequestWrapper{BidRequest: &req}, false, false, nil)

	expectedError := errortypes.Warning{Message: "A prebid request can only process one currency. Taking the first currency in the list, USD, as the active currency"}
	assert.ElementsMatch(t, errL, []error{&expectedError})
}

func TestCCPAInvalid(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
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
			Ext: json.RawMessage(`{"us_privacy": "invalid by length"}`),
		},
	}

	errL := deps.validateRequest(&openrtb_ext.RequestWrapper{BidRequest: &req}, false, false, nil)

	expectedWarning := errortypes.Warning{
		Message:     "CCPA consent is invalid and will be ignored. (request.regs.ext.us_privacy must contain 4 characters)",
		WarningCode: errortypes.InvalidPrivacyConsentWarningCode}
	assert.ElementsMatch(t, errL, []error{&expectedWarning})
}

func TestNoSaleInvalid(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
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

	errL := deps.validateRequest(&openrtb_ext.RequestWrapper{BidRequest: &req}, false, false, nil)

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
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		cfg,
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
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

	deps.validateRequest(&openrtb_ext.RequestWrapper{BidRequest: &req}, false, false, nil)
	assert.NotEmpty(t, req.Source.TID, "Expected req.Source.TID to be filled with a randomly generated UID")
}

func TestSChainInvalid(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
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

	errL := deps.validateRequest(&openrtb_ext.RequestWrapper{BidRequest: &req}, false, false, nil)

	expectedError := errors.New("request.ext.prebid.schains contains multiple schains for bidder appnexus; it must contain no more than one per bidder.")
	assert.ElementsMatch(t, errL, []error{expectedError})
}

func TestMapSChains(t *testing.T) {
	const seller1SChain string = `"schain":{"complete":1,"nodes":[{"asi":"directseller1.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}`
	const seller2SChain string = `"schain":{"complete":2,"nodes":[{"asi":"directseller2.com","sid":"00002","rid":"BidRequest2","hp":2}],"ver":"2.0"}`

	seller1SChainUnpacked := openrtb_ext.ExtRequestPrebidSChainSChain{
		Complete: 1,
		Nodes: []*openrtb_ext.ExtRequestPrebidSChainSChainNode{{
			ASI: "directseller1.com",
			SID: "00001",
			RID: "BidRequest1",
			HP:  1,
		}},
		Ver: "1.0",
	}

	tests := []struct {
		description         string
		bidRequest          openrtb2.BidRequest
		wantReqExtSChain    *openrtb_ext.ExtRequestPrebidSChainSChain
		wantSourceExtSChain *openrtb_ext.ExtRequestPrebidSChainSChain
		wantError           bool
	}{
		{
			description: "invalid req.ext",
			bidRequest: openrtb2.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"schains":invalid}}`),
				Source: &openrtb2.Source{
					Ext: json.RawMessage(`{}`),
				},
			},
			wantError: true,
		},
		{
			description: "invalid source.ext",
			bidRequest: openrtb2.BidRequest{
				Ext: json.RawMessage(`{}`),
				Source: &openrtb2.Source{
					Ext: json.RawMessage(`{"schain":invalid}}`),
				},
			},
			wantError: true,
		},
		{
			description: "req.ext.prebid.schains, req.source.ext.schain and req.ext.schain are nil",
			bidRequest: openrtb2.BidRequest{
				Ext: json.RawMessage(`{}`),
				Source: &openrtb2.Source{
					Ext: json.RawMessage(`{}`),
				},
			},
			wantReqExtSChain:    nil,
			wantSourceExtSChain: nil,
		},
		{
			description: "req.ext.prebid.schains is not nil",
			bidRequest: openrtb2.BidRequest{
				Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],` + seller1SChain + `}]}}`),
				Source: &openrtb2.Source{
					Ext: json.RawMessage(`{}`),
				},
			},
			wantReqExtSChain:    nil,
			wantSourceExtSChain: nil,
		},
		{
			description: "req.source.ext is not nil",
			bidRequest: openrtb2.BidRequest{
				Ext: json.RawMessage(`{}`),
				Source: &openrtb2.Source{
					Ext: json.RawMessage(`{` + seller1SChain + `}`),
				},
			},
			wantReqExtSChain:    nil,
			wantSourceExtSChain: &seller1SChainUnpacked,
		},
		{
			description: "req.ext.schain is not nil",
			bidRequest: openrtb2.BidRequest{
				Ext: json.RawMessage(`{` + seller1SChain + `}`),
				Source: &openrtb2.Source{
					Ext: json.RawMessage(`{}`),
				},
			},
			wantReqExtSChain:    nil,
			wantSourceExtSChain: &seller1SChainUnpacked,
		},
		{
			description: "req.source.ext.schain and req.ext.schain are not nil",
			bidRequest: openrtb2.BidRequest{
				Ext: json.RawMessage(`{` + seller2SChain + `}`),
				Source: &openrtb2.Source{
					Ext: json.RawMessage(`{` + seller1SChain + `}`),
				},
			},
			wantReqExtSChain:    nil,
			wantSourceExtSChain: &seller1SChainUnpacked,
		},
	}

	for _, test := range tests {
		reqWrapper := openrtb_ext.RequestWrapper{
			BidRequest: &test.bidRequest,
		}

		err := mapSChains(&reqWrapper)

		if test.wantError {
			assert.NotNil(t, err, test.description)
		} else {
			assert.Nil(t, err, test.description)

			reqExt, err := reqWrapper.GetRequestExt()
			if err != nil {
				assert.Fail(t, "Error getting request ext from wrapper", test.description)
			}
			reqExtSChain := reqExt.GetSChain()
			assert.Equal(t, test.wantReqExtSChain, reqExtSChain, test.description)

			sourceExt, err := reqWrapper.GetSourceExt()
			if err != nil {
				assert.Fail(t, "Error getting source ext from wrapper", test.description)
			}
			sourceExtSChain := sourceExt.GetSChain()
			assert.Equal(t, test.wantSourceExtSChain, sourceExtSChain, test.description)
		}
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
	testPubExtJSON, err := json.Marshal(testPubExt)
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
		sanitizeRequest(test.req, test.ipValidator)
		assert.Equal(t, test.expectedIPv4, test.req.Device.IP, test.description+":ipv4")
		assert.Equal(t, test.expectedIPv6, test.req.Device.IPv6, test.description+":ipv6")
	}
}

func TestValidateAndFillSourceTID(t *testing.T) {
	testTID := "some-tid"
	testCases := []struct {
		description   string
		req           *openrtb2.BidRequest
		expectRandTID bool
		expectedTID   string
	}{
		{
			description:   "req.Source not present. Expecting a randomly generated TID value",
			req:           &openrtb2.BidRequest{},
			expectRandTID: true,
		},
		{
			description: "req.Source.TID not present. Expecting a randomly generated TID value",
			req: &openrtb2.BidRequest{
				Source: &openrtb2.Source{},
			},
			expectRandTID: true,
		},
		{
			description: "req.Source.TID present. Expecting no change",
			req: &openrtb2.BidRequest{
				Source: &openrtb2.Source{
					TID: testTID,
				},
			},
			expectRandTID: false,
			expectedTID:   testTID,
		},
	}

	for _, test := range testCases {
		_ = validateAndFillSourceTID(test.req)
		if test.expectRandTID {
			assert.NotEmpty(t, test.req.Source.TID, test.description)
			assert.NotEqual(t, test.expectedTID, test.req.Source.TID, test.description)
		} else {
			assert.Equal(t, test.expectedTID, test.req.Source.TID, test.description)
		}
	}
}

func TestEidPermissionsInvalid(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
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

	errL := deps.validateRequest(&openrtb_ext.RequestWrapper{BidRequest: &req}, false, false, nil)

	expectedError := errors.New(`request.ext.prebid.data.eidpermissions[0] missing or empty required field: "bidders"`)
	assert.ElementsMatch(t, errL, []error{expectedError})
}

func TestValidateEidPermissions(t *testing.T) {
	knownBidders := map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")}
	knownAliases := map[string]string{"b": "b"}

	testCases := []struct {
		description   string
		request       *openrtb_ext.ExtRequest
		expectedError error
	}{
		{
			description:   "Valid - Empty ext",
			request:       &openrtb_ext.ExtRequest{},
			expectedError: nil,
		},
		{
			description:   "Valid - Nil ext.prebid.data",
			request:       &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{}},
			expectedError: nil,
		},
		{
			description:   "Valid - Empty ext.prebid.data",
			request:       &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{}}},
			expectedError: nil,
		},
		{
			description:   "Valid - Nil ext.prebid.data.eidpermissions",
			request:       &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: nil}}},
			expectedError: nil,
		},
		{
			description:   "Valid - None",
			request:       &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{}}}},
			expectedError: nil,
		},
		{
			description: "Valid - One",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"a"}},
			}}}},
			expectedError: nil,
		},
		{
			description: "Valid - Many",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"a"}},
				{Source: "sourceB", Bidders: []string{"a"}},
			}}}},
			expectedError: nil,
		},
		{
			description: "Invalid - Missing Source",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"a"}},
				{Bidders: []string{"a"}},
			}}}},
			expectedError: errors.New(`request.ext.prebid.data.eidpermissions[1] missing required field: "source"`),
		},
		{
			description: "Invalid - Duplicate Source",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"a"}},
				{Source: "sourceA", Bidders: []string{"a"}},
			}}}},
			expectedError: errors.New(`request.ext.prebid.data.eidpermissions[1] duplicate entry with field: "source"`),
		},
		{
			description: "Invalid - Missing Bidders - Nil",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"a"}},
				{Source: "sourceB"},
			}}}},
			expectedError: errors.New(`request.ext.prebid.data.eidpermissions[1] missing or empty required field: "bidders"`),
		},
		{
			description: "Invalid - Missing Bidders - Empty",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"a"}},
				{Source: "sourceB", Bidders: []string{}},
			}}}},
			expectedError: errors.New(`request.ext.prebid.data.eidpermissions[1] missing or empty required field: "bidders"`),
		},
		{
			description: "Invalid - Invalid Bidders",
			request: &openrtb_ext.ExtRequest{Prebid: openrtb_ext.ExtRequestPrebid{Data: &openrtb_ext.ExtRequestPrebidData{EidPermissions: []openrtb_ext.ExtRequestPrebidDataEidPermission{
				{Source: "sourceA", Bidders: []string{"a"}},
				{Source: "sourceB", Bidders: []string{"z"}},
			}}}},
			expectedError: errors.New(`request.ext.prebid.data.eidpermissions[1] contains unrecognized bidder "z"`),
		},
	}

	endpoint := &endpointDeps{bidderMap: knownBidders}
	for _, test := range testCases {
		result := endpoint.validateEidPermissions(test.request.Prebid.Data, knownAliases)
		assert.Equal(t, test.expectedError, result, test.description)
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
			description:   "Invalid - Unknown Bidder Case Sensitive",
			bidders:       []string{"A"},
			knownBidders:  map[string]openrtb_ext.BidderName{"a": openrtb_ext.BidderName("a")},
			knownAliases:  map[string]string{"c": "c"},
			expectedError: errors.New(`unrecognized bidder "A"`),
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

	for _, test := range testCases {
		result := validateBidders(test.bidders, test.knownBidders, test.knownAliases)
		assert.Equal(t, test.expectedError, result, test.description)
	}
}

func TestIOS14EndToEnd(t *testing.T) {
	exchange := &nobidExchange{}

	endpoint, _ := NewEndpoint(
		fakeUUIDGenerator{},
		exchange,
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{})

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
	reqBody := validRequest(t, "us-privacy-invalid.json")
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&warningsCheckExchange{},
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(len(reqBody))},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
	}

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
	expectedMessage := "CCPA consent is invalid and will be ignored. (request.regs.ext.us_privacy must contain 4 characters)"
	assert.Equal(t, expectedMessage, actualWarning.Message, "Warning message is incorrect")

	assert.Equal(t, errortypes.InvalidPrivacyConsentWarningCode, actualWarning.WarningCode, "Warning code is incorrect")
}

func TestParseRequestParseImpInfoError(t *testing.T) {
	reqBody := validRequest(t, "imp-info-invalid.json")
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&warningsCheckExchange{},
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(len(reqBody))},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		empty_fetcher.EmptyFetcher{},
	}

	req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(reqBody))

	resReq, impExtInfoMap, _, _, errL := deps.parseRequest(req)

	assert.Nil(t, resReq, "Result request should be nil due to incorrect imp")
	assert.Nil(t, impExtInfoMap, "Impression info map should be nil due to incorrect imp")
	assert.Len(t, errL, 1, "One error should be returned")
	assert.Contains(t, errL[0].Error(), "echovideoattrs of type bool", "Incorrect error message")
}

func TestValidateNativeContextTypes(t *testing.T) {
	impIndex := 4

	testCases := []struct {
		description      string
		givenContextType native1.ContextType
		givenSubType     native1.ContextSubType
		expectedError    string
	}{
		{
			description:      "No Types Specified",
			givenContextType: 0,
			givenSubType:     0,
			expectedError:    "",
		},
		{
			description:      "All Types Exchange Specific",
			givenContextType: 500,
			givenSubType:     500,
			expectedError:    "",
		},
		{
			description:      "Context Type Known Value - Sub Type Unspecified",
			givenContextType: 1,
			givenSubType:     0,
			expectedError:    "",
		},
		{
			description:      "Context Type Negative",
			givenContextType: -1,
			givenSubType:     0,
			expectedError:    "request.imp[4].native.request.context is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Context Type Just Above Range",
			givenContextType: 4, // Range is currently 1-3
			givenSubType:     0,
			expectedError:    "request.imp[4].native.request.context is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Sub Type Negative",
			givenContextType: 1,
			givenSubType:     -1,
			expectedError:    "request.imp[4].native.request.contextsubtype value can't be less than 0. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Content - Sub Type Just Below Range",
			givenContextType: 1, // Content constant
			givenSubType:     9, // Content range is currently 10-15
			expectedError:    "request.imp[4].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Content - Sub Type In Range",
			givenContextType: 1,  // Content constant
			givenSubType:     10, // Content range is currently 10-15
			expectedError:    "",
		},
		{
			description:      "Content - Sub Type In Range - Context Type Exchange Specific Boundary",
			givenContextType: 500,
			givenSubType:     10, // Content range is currently 10-15
			expectedError:    "",
		},
		{
			description:      "Content - Sub Type In Range - Context Type Exchange Specific Boundary + 1",
			givenContextType: 501,
			givenSubType:     10, // Content range is currently 10-15
			expectedError:    "",
		},
		{
			description:      "Content - Sub Type Just Above Range",
			givenContextType: 1,  // Content constant
			givenSubType:     16, // Content range is currently 10-15
			expectedError:    "request.imp[4].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Content - Sub Type Exchange Specific Boundary",
			givenContextType: 1, // Content constant
			givenSubType:     500,
			expectedError:    "",
		},
		{
			description:      "Content - Sub Type Exchange Specific Boundary + 1",
			givenContextType: 1, // Content constant
			givenSubType:     501,
			expectedError:    "",
		},
		{
			description:      "Content - Invalid Context Type",
			givenContextType: 2,  // Not content constant
			givenSubType:     10, // Content range is currently 10-15
			expectedError:    "request.imp[4].native.request.context is 2, but contextsubtype is 10. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Social - Sub Type Just Below Range",
			givenContextType: 2,  // Social constant
			givenSubType:     19, // Social range is currently 20-22
			expectedError:    "request.imp[4].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Social - Sub Type In Range",
			givenContextType: 2,  // Social constant
			givenSubType:     20, // Social range is currently 20-22
			expectedError:    "",
		},
		{
			description:      "Social - Sub Type In Range - Context Type Exchange Specific Boundary",
			givenContextType: 500,
			givenSubType:     20, // Social range is currently 20-22
			expectedError:    "",
		},
		{
			description:      "Social - Sub Type In Range - Context Type Exchange Specific Boundary + 1",
			givenContextType: 501,
			givenSubType:     20, // Social range is currently 20-22
			expectedError:    "",
		},
		{
			description:      "Social - Sub Type Just Above Range",
			givenContextType: 2,  // Social constant
			givenSubType:     23, // Social range is currently 20-22
			expectedError:    "request.imp[4].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Social - Sub Type Exchange Specific Boundary",
			givenContextType: 2, // Social constant
			givenSubType:     500,
			expectedError:    "",
		},
		{
			description:      "Social - Sub Type Exchange Specific Boundary + 1",
			givenContextType: 2, // Social constant
			givenSubType:     501,
			expectedError:    "",
		},
		{
			description:      "Social - Invalid Context Type",
			givenContextType: 3,  // Not social constant
			givenSubType:     20, // Social range is currently 20-22
			expectedError:    "request.imp[4].native.request.context is 3, but contextsubtype is 20. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Product - Sub Type Just Below Range",
			givenContextType: 3,  // Product constant
			givenSubType:     29, // Product range is currently 30-32
			expectedError:    "request.imp[4].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Product - Sub Type In Range",
			givenContextType: 3,  // Product constant
			givenSubType:     30, // Product range is currently 30-32
			expectedError:    "",
		},
		{
			description:      "Product - Sub Type In Range - Context Type Exchange Specific Boundary",
			givenContextType: 500,
			givenSubType:     30, // Product range is currently 30-32
			expectedError:    "",
		},
		{
			description:      "Product - Sub Type In Range - Context Type Exchange Specific Boundary + 1",
			givenContextType: 501,
			givenSubType:     30, // Product range is currently 30-32
			expectedError:    "",
		},
		{
			description:      "Product - Sub Type Just Above Range",
			givenContextType: 3,  // Product constant
			givenSubType:     33, // Product range is currently 30-32
			expectedError:    "request.imp[4].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
		{
			description:      "Product - Sub Type Exchange Specific Boundary",
			givenContextType: 3, // Product constant
			givenSubType:     500,
			expectedError:    "",
		},
		{
			description:      "Product - Sub Type Exchange Specific Boundary + 1",
			givenContextType: 3, // Product constant
			givenSubType:     501,
			expectedError:    "",
		},
		{
			description:      "Product - Invalid Context Type",
			givenContextType: 1,  // Not product constant
			givenSubType:     30, // Product range is currently 30-32
			expectedError:    "request.imp[4].native.request.context is 1, but contextsubtype is 30. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39",
		},
	}

	for _, test := range testCases {
		err := validateNativeContextTypes(test.givenContextType, test.givenSubType, impIndex)
		if test.expectedError == "" {
			assert.NoError(t, err, test.description)
		} else {
			assert.EqualError(t, err, test.expectedError, test.description)
		}
	}
}

func TestValidateNativePlacementType(t *testing.T) {
	impIndex := 4

	testCases := []struct {
		description        string
		givenPlacementType native1.PlacementType
		expectedError      string
	}{
		{
			description:        "Not Specified",
			givenPlacementType: 0,
			expectedError:      "",
		},
		{
			description:        "Known Value",
			givenPlacementType: 1, // Range is currently 1-4
			expectedError:      "",
		},
		{
			description:        "Exchange Specific - Boundary",
			givenPlacementType: 500,
			expectedError:      "",
		},
		{
			description:        "Exchange Specific - Boundary + 1",
			givenPlacementType: 501,
			expectedError:      "",
		},
		{
			description:        "Negative",
			givenPlacementType: -1,
			expectedError:      "request.imp[4].native.request.plcmttype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40",
		},
		{
			description:        "Just Above Range",
			givenPlacementType: 5, // Range is currently 1-4
			expectedError:      "request.imp[4].native.request.plcmttype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40",
		},
	}

	for _, test := range testCases {
		err := validateNativePlacementType(test.givenPlacementType, impIndex)
		if test.expectedError == "" {
			assert.NoError(t, err, test.description)
		} else {
			assert.EqualError(t, err, test.expectedError, test.description)
		}
	}
}

func TestValidateNativeEventTracker(t *testing.T) {
	impIndex := 4
	eventIndex := 8

	testCases := []struct {
		description   string
		givenEvent    nativeRequests.EventTracker
		expectedError string
	}{
		{
			description: "Valid",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{1},
			},
			expectedError: "",
		},
		{
			description: "Event - Exchange Specific - Boundary",
			givenEvent: nativeRequests.EventTracker{
				Event:   500,
				Methods: []native1.EventTrackingMethod{1},
			},
			expectedError: "",
		},
		{
			description: "Event - Exchange Specific - Boundary + 1",
			givenEvent: nativeRequests.EventTracker{
				Event:   501,
				Methods: []native1.EventTrackingMethod{1},
			},
			expectedError: "",
		},
		{
			description: "Event - Negative",
			givenEvent: nativeRequests.EventTracker{
				Event:   -1,
				Methods: []native1.EventTrackingMethod{1},
			},
			expectedError: "request.imp[4].native.request.eventtrackers[8].event is invalid. See section 7.6: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43",
		},
		{
			description: "Event - Just Above Range",
			givenEvent: nativeRequests.EventTracker{
				Event:   5, // Range is currently 1-4
				Methods: []native1.EventTrackingMethod{1},
			},
			expectedError: "request.imp[4].native.request.eventtrackers[8].event is invalid. See section 7.6: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43",
		},
		{
			description: "Methods - Many Valid",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{1, 2},
			},
			expectedError: "",
		},
		{
			description: "Methods - Empty",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{},
			},
			expectedError: "request.imp[4].native.request.eventtrackers[8].method is required. See section 7.7: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43",
		},
		{
			description: "Methods - Exchange Specific - Boundary",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{500},
			},
			expectedError: "",
		},
		{
			description: "Methods - Exchange Specific - Boundary + 1",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{501},
			},
			expectedError: "",
		},
		{
			description: "Methods - Negative",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{-1},
			},
			expectedError: "request.imp[4].native.request.eventtrackers[8].methods[0] is invalid. See section 7.7: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43",
		},
		{
			description: "Methods - Just Above Range",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{3}, // Known values are currently 1-2
			},
			expectedError: "request.imp[4].native.request.eventtrackers[8].methods[0] is invalid. See section 7.7: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43",
		},
		{
			description: "Methods - Mixed Valid + Invalid",
			givenEvent: nativeRequests.EventTracker{
				Event:   1,
				Methods: []native1.EventTrackingMethod{1, -1},
			},
			expectedError: "request.imp[4].native.request.eventtrackers[8].methods[1] is invalid. See section 7.7: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43",
		},
	}

	for _, test := range testCases {
		err := validateNativeEventTracker(test.givenEvent, impIndex, eventIndex)
		if test.expectedError == "" {
			assert.NoError(t, err, test.description)
		} else {
			assert.EqualError(t, err, test.expectedError, test.description)
		}
	}
}

func TestValidateNativeAssetData(t *testing.T) {
	impIndex := 4
	assetIndex := 8

	testCases := []struct {
		description   string
		givenData     nativeRequests.Data
		expectedError string
	}{
		{
			description:   "Valid",
			givenData:     nativeRequests.Data{Type: 1},
			expectedError: "",
		},
		{
			description:   "Exchange Specific - Boundary",
			givenData:     nativeRequests.Data{Type: 500},
			expectedError: "",
		},
		{
			description:   "Exchange Specific - Boundary + 1",
			givenData:     nativeRequests.Data{Type: 501},
			expectedError: "",
		},
		{
			description:   "Not Specified",
			givenData:     nativeRequests.Data{},
			expectedError: "request.imp[4].native.request.assets[8].data.type is invalid. See section 7.4: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40",
		},
		{
			description:   "Negative",
			givenData:     nativeRequests.Data{Type: -1},
			expectedError: "request.imp[4].native.request.assets[8].data.type is invalid. See section 7.4: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40",
		},
		{
			description:   "Just Above Range",
			givenData:     nativeRequests.Data{Type: 13}, // Range is currently 1-12
			expectedError: "request.imp[4].native.request.assets[8].data.type is invalid. See section 7.4: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40",
		},
	}

	for _, test := range testCases {
		err := validateNativeAssetData(&test.givenData, impIndex, assetIndex)
		if test.expectedError == "" {
			assert.NoError(t, err, test.description)
		} else {
			assert.EqualError(t, err, test.expectedError, test.description)
		}
	}
}

func TestAuctionResponseHeaders(t *testing.T) {
	testCases := []struct {
		description     string
		requestBody     string
		expectedStatus  int
		expectedHeaders func(http.Header)
	}{
		{
			description:    "Success Response",
			requestBody:    validRequest(t, "site.json"),
			expectedStatus: 200,
			expectedHeaders: func(h http.Header) {
				h.Set("X-Prebid", "pbs-go/unknown")
				h.Set("Content-Type", "application/json")
			},
		},
		{
			description:    "Failure Response",
			requestBody:    "{}",
			expectedStatus: 400,
			expectedHeaders: func(h http.Header) {
				h.Set("X-Prebid", "pbs-go/unknown")
			},
		},
	}

	exchange := &nobidExchange{}
	endpoint, _ := NewEndpoint(
		fakeUUIDGenerator{},
		exchange,
		mockBidderParamValidator{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		empty_fetcher.EmptyFetcher{})

	for _, test := range testCases {
		httpReq := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(test.requestBody))
		recorder := httptest.NewRecorder()

		endpoint(recorder, httpReq, nil)

		expectedHeaders := http.Header{}
		test.expectedHeaders(expectedHeaders)

		assert.Equal(t, test.expectedStatus, recorder.Result().StatusCode, test.description+":statuscode")
		assert.Equal(t, expectedHeaders, recorder.Result().Header, test.description+":statuscode")
	}
}

// StoredRequest testing

// Test stored request data

func TestValidateBanner(t *testing.T) {
	impIndex := 0

	testCases := []struct {
		description    string
		banner         *openrtb2.Banner
		impIndex       int
		isInterstitial bool
		expectedError  error
	}{
		{
			description:    "isInterstitial Equals False (not set to 1)",
			banner:         &openrtb2.Banner{W: nil, H: nil, Format: nil},
			impIndex:       impIndex,
			isInterstitial: false,
			expectedError:  errors.New("request.imp[0].banner has no sizes. Define \"w\" and \"h\", or include \"format\" elements."),
		},
		{
			description:    "isInterstitial Equals True (is set to 1)",
			banner:         &openrtb2.Banner{W: nil, H: nil, Format: nil},
			impIndex:       impIndex,
			isInterstitial: true,
			expectedError:  nil,
		},
	}

	for _, test := range testCases {
		result := validateBanner(test.banner, test.impIndex, test.isInterstitial)
		assert.Equal(t, test.expectedError, result, test.description)
	}
}

func TestParseRequestMergeBidderParams(t *testing.T) {
	tests := []struct {
		name               string
		givenRequestBody   string
		expectedImpExt     json.RawMessage
		expectedReqExt     json.RawMessage
		expectedErrorCount int
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
			givenRequestBody:   validRequest(t, "req-ext-bidder-params-backward-compatible-merge.json"),
			expectedImpExt:     getObject(t, "req-ext-bidder-params-backward-compatible-merge.json", "expectedImpExt"),
			expectedReqExt:     getObject(t, "req-ext-bidder-params-backward-compatible-merge.json", "expectedReqExt"),
			expectedErrorCount: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			deps := &endpointDeps{
				fakeUUIDGenerator{},
				&warningsCheckExchange{},
				mockBidderParamValidator{},
				&mockStoredReqFetcher{},
				empty_fetcher.EmptyFetcher{},
				empty_fetcher.EmptyFetcher{},
				&config.Configuration{MaxRequestSize: int64(len(test.givenRequestBody))},
				&metricsConfig.NilMetricsEngine{},
				analyticsConf.NewPBSAnalytics(&config.Analytics{}),
				map[string]string{},
				false,
				[]byte{},
				openrtb_ext.BuildBidderMap(),
				nil,
				nil,
				hardcodedResponseIPValidator{response: true},
				empty_fetcher.EmptyFetcher{},
			}

			req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(test.givenRequestBody))

			resReq, _, _, _, errL := deps.parseRequest(req)

			var expIExt, iExt map[string]interface{}
			err := json.Unmarshal(test.expectedImpExt, &expIExt)
			assert.Nil(t, err, "unmarshal() should return nil error")

			assert.NotNil(t, resReq.BidRequest.Imp[0].Ext, "imp[0].Ext should not be nil")
			err = json.Unmarshal(resReq.BidRequest.Imp[0].Ext, &iExt)
			assert.Nil(t, err, "unmarshal() should return nil error")

			assert.Equal(t, expIExt, iExt, "bidderparams in imp[].Ext should match")

			var eReqE, reqE map[string]interface{}
			err = json.Unmarshal(test.expectedReqExt, &eReqE)
			assert.Nil(t, err, "unmarshal() should return nil error")

			err = json.Unmarshal(resReq.BidRequest.Ext, &reqE)
			assert.Nil(t, err, "unmarshal() should return nil error")

			assert.Equal(t, eReqE, reqE, "req.Ext should match")

			assert.Len(t, errL, test.expectedErrorCount, "error length should match")
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
			name:             "req has two imps with missing stored responses",
			givenRequestBody: validRequest(t, "req-two-imps-missing-stored-response.json"),
			expectedStoredResponses: map[string]json.RawMessage{
				"imp-id1": json.RawMessage(`[{"bid": [{"id": "bid_id1"],"seat": "appnexus"}]`),
				"imp-id2": json.RawMessage(nil),
			},
			expectedErrorCount: 0,
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
				mockBidderParamValidator{},
				&mockStoredReqFetcher{},
				empty_fetcher.EmptyFetcher{},
				empty_fetcher.EmptyFetcher{},
				&config.Configuration{MaxRequestSize: int64(len(test.givenRequestBody))},
				&metricsConfig.NilMetricsEngine{},
				analyticsConf.NewPBSAnalytics(&config.Analytics{}),
				map[string]string{},
				false,
				[]byte{},
				openrtb_ext.BuildBidderMap(),
				nil,
				nil,
				hardcodedResponseIPValidator{response: true},
				&mockStoredResponseFetcher{mockStoredResponses},
			}

			req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(test.givenRequestBody))

			_, _, storedResponses, _, errL := deps.parseRequest(req)

			if test.expectedErrorCount == 0 {
				assert.Equal(t, test.expectedStoredResponses, storedResponses, "stored responses should match")
			} else {
				assert.Contains(t, errL[0].Error(), test.expectedError, "error should match")
			}

		})
	}
}

func TestParseRequestStoredBidResponses(t *testing.T) {
	bidRespId1 := json.RawMessage(`{"id": "resp_id1", "seatbid": [{"bid": [{"id": "bid_id1"}], "seat": "testBidder1"}], "bidid": "123", "cur": "USD"}`)
	bidRespId2 := json.RawMessage(`{"id": "resp_id2", "seatbid": [{"bid": [{"id": "bid_id2"}], "seat": "testBidder2"}], "bidid": "124", "cur": "USD"}`)
	mockStoredBidResponses := map[string]json.RawMessage{
		"bidResponseId1": bidRespId1,
		"bidResponseId2": bidRespId2,
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
				"imp-id1": {"testBidder1": bidRespId1},
			},
			expectedErrorCount: 0,
		},
		{
			name:             "req has two imps with valid stored bid responses",
			givenRequestBody: validRequest(t, "req-two-imps-stored-bid-responses.json"),
			expectedStoredBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"testBidder1": bidRespId1},
				"imp-id2": {"testBidder2": bidRespId2},
			},
			expectedErrorCount: 0,
		},
		{
			name:             "req has two imps one with valid stored bid responses and another one without stored bid responses",
			givenRequestBody: validRequest(t, "req-two-imps-with-and-without-stored-bid-responses.json"),
			expectedStoredBidResponses: map[string]map[string]json.RawMessage{
				"imp-id2": {"testBidder2": bidRespId2},
			},
			expectedErrorCount: 0,
		},
		{
			name:             "req has two imps with missing stored bid responses",
			givenRequestBody: validRequest(t, "req-two-imps-missing-stored-bid-response.json"),
			expectedStoredBidResponses: map[string]map[string]json.RawMessage{
				"imp-id1": {"testBidder1": nil},
				"imp-id2": {"testBidder2": nil},
			},
			expectedErrorCount: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			deps := &endpointDeps{
				fakeUUIDGenerator{},
				&warningsCheckExchange{},
				mockBidderParamValidator{},
				&mockStoredReqFetcher{},
				empty_fetcher.EmptyFetcher{},
				empty_fetcher.EmptyFetcher{},
				&config.Configuration{MaxRequestSize: int64(len(test.givenRequestBody))},
				&metricsConfig.NilMetricsEngine{},
				analyticsConf.NewPBSAnalytics(&config.Analytics{}),
				map[string]string{},
				false,
				[]byte{},
				map[string]openrtb_ext.BidderName{"testBidder1": "testBidder1", "testBidder2": "testBidder2"},
				nil,
				nil,
				hardcodedResponseIPValidator{response: true},
				&mockStoredResponseFetcher{mockStoredBidResponses},
			}

			req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(test.givenRequestBody))
			_, _, _, storedBidResponses, errL := deps.parseRequest(req)

			if test.expectedErrorCount == 0 {
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
		mockBidderParamValidator{},
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BuildBidderMap(),
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
		&mockStoredResponseFetcher{},
	}

	testCases := []struct {
		description               string
		givenRequestWrapper       *openrtb_ext.RequestWrapper
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
			description: "One imp with 2 stored bid responses and 1 bidders in imp.ext and 1 in imp.ext.prebid.bidder, expect validate request to throw no errors",
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
			storedBidResponses:        stored_responses.ImpBidderStoredResp{"Some-Imp-ID": {"appnexus": json.RawMessage(`{"test":true}`), "telaria": json.RawMessage(`{"test":true}`)}},
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
		errorList := deps.validateRequest(test.givenRequestWrapper, false, test.hasStoredAuctionResponses, test.storedBidResponses)
		assert.Equalf(t, test.expectedErrorList, errorList, "Error doesn't match: %s\n", test.description)
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
	requestData, err := ioutil.ReadFile("sample-requests/valid-whole/supplementary/" + filename)
	if err != nil {
		t.Fatalf("Failed to fetch a valid request: %v", err)
	}
	testBidRequest, _, _, err := jsonparser.Get(requestData, key)
	assert.NoError(t, err, "Error jsonparsing root.mockBidRequest from file %s. Desc: %v.", filename, err)

	var obj json.RawMessage
	err = json.Unmarshal(testBidRequest, &obj)
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
