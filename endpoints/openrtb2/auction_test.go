package openrtb2

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-server/firstpartydata"

	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/metrics"
	metricsConfig "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/util/iputil"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb/v15/native1"
	nativeRequests "github.com/mxmCherry/openrtb/v15/native1/request"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/stretchr/testify/assert"
)

const maxSize = 1024 * 256

type testCase struct {
	BidRequest           json.RawMessage   `json:"mockBidRequest"`
	Config               *testConfigValues `json:"config"`
	ExpectedReturnCode   int               `json:"expectedReturnCode,omitempty"`
	ExpectedErrorMessage string            `json:"expectedErrorMessage"`
	ExpectedBidResponse  json.RawMessage   `json:"expectedBidResponse"`
}

type testConfigValues struct {
	AccountRequired     bool                          `json:"accountRequired"`
	AliasJSON           string                        `json:"aliases"`
	BlacklistedAccounts []string                      `json:"blacklistedAccts"`
	BlacklistedApps     []string                      `json:"blacklistedApps"`
	DisabledAdapters    []string                      `json:"disabledAdapters"`
	CurrencyRates       map[string]map[string]float64 `json:"currencyRates"`
	MockBidder          mockBidExchangeBidder         `json:"mockBidder"`
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
	for _, test := range testSuites {
		testCaseFiles, err := getTestFiles(filepath.Join("sample-requests", test.sampleRequestsSubDir))
		if assert.NoError(t, err, "Test case %s. Error reading files from directory %s \n", test.description, test.sampleRequestsSubDir) {
			for _, file := range testCaseFiles {
				data, err := ioutil.ReadFile(file)
				if assert.NoError(t, err, "Test case %s. Error reading file %s \n", test.description, file) {
					runTestCase(t, data, file)
				}
			}
		}
	}
}

func getTestFiles(dir string) ([]string, error) {
	var filesToAssert []string

	fileList, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	// Append the path of every file found in `dir` to the `filesToAssert` array
	for _, fileInfo := range fileList {
		filesToAssert = append(filesToAssert, filepath.Join(dir, fileInfo.Name()))
	}

	return filesToAssert, nil
}

func runTestCase(t *testing.T, fileData []byte, testFile string) {
	t.Helper()

	// Retrieve values from JSON file
	test := parseTestFile(t, fileData, testFile)

	// Run test
	actualCode, actualJsonBidResponse := doRequest(t, test)

	// Assertions
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

func parseTestFile(t *testing.T, fileData []byte, testFile string) testCase {
	t.Helper()

	parsedTestData := testCase{}
	var err, errEm error

	// Get testCase values
	parsedTestData.BidRequest, _, _, err = jsonparser.Get(fileData, "mockBidRequest")
	assert.NoError(t, err, "Error jsonparsing root.mockBidRequest from file %s. Desc: %v.", testFile, err)

	// Get testCaseConfig values
	parsedTestData.Config = &testConfigValues{}
	var jsonTestConfig json.RawMessage

	jsonTestConfig, _, _, err = jsonparser.Get(fileData, "config")
	if err == nil {
		err = json.Unmarshal(jsonTestConfig, parsedTestData.Config)
		assert.NoError(t, err, "Error unmarshaling root.config from file %s. Desc: %v.", testFile, err)
	}

	// Get the return code we expect PBS to throw back given test's bidRequest and config
	parsedReturnCode, err := jsonparser.GetInt(fileData, "expectedReturnCode")
	assert.NoError(t, err, "Error jsonparsing root.code from file %s. Desc: %v.", testFile, err)

	// Get both bid response and error message, if any
	parsedTestData.ExpectedBidResponse, _, _, err = jsonparser.Get(fileData, "expectedBidResponse")
	parsedTestData.ExpectedErrorMessage, errEm = jsonparser.GetString(fileData, "expectedErrorMessage")

	assert.Falsef(t, (err == nil && errEm == nil), "Test case file can't have both a valid expectedBidResponse and a valid expectedErrorMessage, fields are mutually exclusive")
	assert.Falsef(t, (err != nil && errEm != nil), "Test case file should come with either a valid expectedBidResponse or a valid expectedErrorMessage, not both.")

	parsedTestData.ExpectedReturnCode = int(parsedReturnCode)

	return parsedTestData
}

func (tc *testConfigValues) getBlacklistedAppMap() map[string]bool {
	var blacklistedAppMap map[string]bool

	if len(tc.BlacklistedApps) > 0 {
		blacklistedAppMap = make(map[string]bool, len(tc.BlacklistedApps))
		for _, app := range tc.BlacklistedApps {
			blacklistedAppMap[app] = true
		}
	}
	return blacklistedAppMap
}

func (tc *testConfigValues) getBlackListedAccountMap() map[string]bool {
	var blacklistedAccountMap map[string]bool

	if len(tc.BlacklistedAccounts) > 0 {
		blacklistedAccountMap = make(map[string]bool, len(tc.BlacklistedAccounts))
		for _, account := range tc.BlacklistedAccounts {
			blacklistedAccountMap[account] = true
		}
	}
	return blacklistedAccountMap
}

func (tc *testConfigValues) getAdaptersConfigMap() map[string]config.Adapter {
	var adaptersConfig map[string]config.Adapter

	if len(tc.DisabledAdapters) > 0 {
		adaptersConfig = make(map[string]config.Adapter, len(tc.DisabledAdapters))
		for _, adapterName := range tc.DisabledAdapters {
			adaptersConfig[adapterName] = config.Adapter{Disabled: true}
		}
	}
	return adaptersConfig
}

// Once unmarshalled, bidResponse objects can't simply be compared with an `assert.Equalf()` call
// because tests fail if the elements inside the `bidResponse.SeatBid` and `bidResponse.SeatBid.Bid`
// arrays, if any, are not listed in the exact same order in the actual version and in the expected version.
func assertBidResponseEqual(t *testing.T, testFile string, expectedBidResponse openrtb2.BidResponse, actualBidResponse openrtb2.BidResponse) {

	//Assert non-array BidResponse fields
	assert.Equalf(t, expectedBidResponse.ID, actualBidResponse.ID, "BidResponse.ID doesn't match expected. Test: %s\n", testFile)
	assert.Equalf(t, expectedBidResponse.BidID, actualBidResponse.BidID, "BidResponse.BidID doesn't match expected. Test: %s\n", testFile)
	assert.Equalf(t, expectedBidResponse.NBR, actualBidResponse.NBR, "BidResponse.NBR doesn't match expected. Test: %s\n", testFile)
	assert.Equalf(t, expectedBidResponse.Cur, actualBidResponse.Cur, "BidResponse.Cur doesn't match expected. Test: %s\n", testFile)

	//Assert []SeatBid and their Bid elements independently of their order
	assert.Len(t, actualBidResponse.SeatBid, len(expectedBidResponse.SeatBid), "BidResponse.SeatBid array doesn't match expected. Test: %s\n", testFile)

	//Given that bidResponses have the same length, compare them in an order-independent way using maps
	var actualSeatBidsMap map[string]openrtb2.SeatBid = make(map[string]openrtb2.SeatBid, 0)
	for _, seatBid := range actualBidResponse.SeatBid {
		actualSeatBidsMap[seatBid.Seat] = seatBid
	}

	var expectedSeatBidsMap map[string]openrtb2.SeatBid = make(map[string]openrtb2.SeatBid, 0)
	for _, seatBid := range expectedBidResponse.SeatBid {
		expectedSeatBidsMap[seatBid.Seat] = seatBid
	}

	for k, expectedSeatBid := range expectedSeatBidsMap {
		//Assert non-array SeatBid fields
		assert.Equalf(t, expectedSeatBid.Seat, actualSeatBidsMap[k].Seat, "actualSeatBidsMap[%s].Seat doesn't match expected. Test: %s\n", k, testFile)
		assert.Equalf(t, expectedSeatBid.Group, actualSeatBidsMap[k].Group, "actualSeatBidsMap[%s].Group doesn't match expected. Test: %s\n", k, testFile)
		assert.Equalf(t, expectedSeatBid.Ext, actualSeatBidsMap[k].Ext, "actualSeatBidsMap[%s].Ext doesn't match expected. Test: %s\n", k, testFile)
		assert.Len(t, actualSeatBidsMap[k].Bid, len(expectedSeatBid.Bid), "BidResponse.SeatBid[].Bid array doesn't match expected. Test: %s\n", testFile)

		//Assert Bid arrays
		assert.ElementsMatch(t, expectedSeatBid.Bid, actualSeatBidsMap[k].Bid, "BidResponse.SeatBid array doesn't match expected. Test: %s\n", testFile)
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
		newParamsValidator(t),
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

func doRequest(t *testing.T, test testCase) (int, string) {
	bidderInfos := getBidderInfos(test.Config.getAdaptersConfigMap(), openrtb_ext.CoreBidderNames())
	bidderMap := exchange.GetActiveBidders(bidderInfos)
	disabledBidders := exchange.GetDisabledBiddersErrorMessages(bidderInfos)

	mockExchange := newMockBidExchange(test.Config.MockBidder, test.Config.CurrencyRates)

	endpoint, _ := NewEndpoint(
		fakeUUIDGenerator{},
		mockExchange,
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{
			MaxRequestSize:     maxSize,
			BlacklistedApps:    test.Config.BlacklistedApps,
			BlacklistedAppMap:  test.Config.getBlacklistedAppMap(),
			BlacklistedAccts:   test.Config.BlacklistedAccounts,
			BlacklistedAcctMap: test.Config.getBlackListedAccountMap(),
			AccountRequired:    test.Config.AccountRequired,
		},
		&metricsConfig.NilMetricsEngine{},
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		disabledBidders,
		[]byte(test.Config.AliasJSON),
		bidderMap,
		empty_fetcher.EmptyFetcher{})

	request := httptest.NewRequest("POST", "/openrtb2/auction", bytes.NewReader(test.BidRequest))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil) //Request comes from the unmarshalled mockBidRequest
	return recorder.Code, recorder.Body.String()
}

// fetchFiles returns a list of the files from dir, or fails the test if an error occurs.
func fetchFiles(t *testing.T, dir string) []os.FileInfo {
	t.Helper()
	requestFiles, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("Failed to read folder: %s", dir)
	}
	return requestFiles
}

func readFile(t *testing.T, filename string) []byte {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filename, err)
	}
	return data
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
		newParamsValidator(t),
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
		newParamsValidator(t),
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
		newParamsValidator(t),
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
			newParamsValidator(t),
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
			newParamsValidator(t),
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
		newParamsValidator(t),
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

//impExtByBidder map[string]json.RawMessage, reqExtParams map[string]map[string]json.RawMessage) (bool, error) {

func TestValidateRequest(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		newParamsValidator(t),
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
		errorList := deps.validateRequest(test.givenRequestWrapper, test.givenIsAmp, false)
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
		newParamsValidator(t),
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
		newParamsValidator(t),
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
			description:            "GenerateRequestID is true, rawData is an app request, has stored bid, and stored bidrequestID is not the macro {{UUID}}, we should generate uuid",
			givenRawData:           testBidRequests[3],
			givenGenerateRequestID: true,
			expectedID:             uuid,
		},
		{
			description:            "GenerateRequestID is false, rawData is an app request and has stored bid, but stored bidrequestID is the macro {{UUID}}, so we should generate uuid",
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
			description:            "GenerateRequestID is true, but rawData is a site request, we should not generate uuid",
			givenRawData:           testBidRequests[1],
			givenGenerateRequestID: true,
			expectedID:             "ThisID",
		},
		{
			description:            "Macro ID {{UUID}} case sensitivity check meaning a macro that is lowercase {{uuid}} shouldn't generate a uuid",
			givenRawData:           testBidRequests[2],
			givenGenerateRequestID: false,
			expectedID:             "ThisID",
		},
		{
			description:            "Test to check that stored requests are being merged when Macro ID is present with a site rquest",
			givenRawData:           testBidRequests[5],
			givenGenerateRequestID: false,
			expectedID:             "ThisID",
			expectedCur:            "USD",
		},
		{
			description:            "Test to check that stored requests are being merged when Generate Request ID flag with a site rquest",
			givenRawData:           testBidRequests[5],
			givenGenerateRequestID: true,
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
		newParamsValidator(t),
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
		newParamsValidator(t),
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
		newParamsValidator(t),
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
		newParamsValidator(t),
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
		newParamsValidator(t),
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

			errs := deps.validateImpExt(imp, nil, 0, false)

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
		newParamsValidator(t),
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

	errL := deps.validateRequest(&openrtb_ext.RequestWrapper{BidRequest: &req}, false, false)

	expectedError := errortypes.Warning{Message: "A prebid request can only process one currency. Taking the first currency in the list, USD, as the active currency"}
	assert.ElementsMatch(t, errL, []error{&expectedError})
}

func TestCCPAInvalid(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		newParamsValidator(t),
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

	errL := deps.validateRequest(&openrtb_ext.RequestWrapper{BidRequest: &req}, false, false)

	expectedWarning := errortypes.Warning{
		Message:     "CCPA consent is invalid and will be ignored. (request.regs.ext.us_privacy must contain 4 characters)",
		WarningCode: errortypes.InvalidPrivacyConsentWarningCode}
	assert.ElementsMatch(t, errL, []error{&expectedWarning})
}

func TestNoSaleInvalid(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		newParamsValidator(t),
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

	errL := deps.validateRequest(&openrtb_ext.RequestWrapper{BidRequest: &req}, false, false)

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
		newParamsValidator(t),
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

	deps.validateRequest(&openrtb_ext.RequestWrapper{BidRequest: &req}, false, false)
	assert.NotEmpty(t, req.Source.TID, "Expected req.Source.TID to be filled with a randomly generated UID")
}

func TestSChainInvalid(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		newParamsValidator(t),
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

	errL := deps.validateRequest(&openrtb_ext.RequestWrapper{BidRequest: &req}, false, false)

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

	for _, tt := range tests {
		reqWrapper := openrtb_ext.RequestWrapper{
			BidRequest: &tt.bidRequest,
		}

		err := mapSChains(&reqWrapper)

		if tt.wantError {
			assert.NotNil(t, err, tt.description)
		} else {
			assert.Nil(t, err, tt.description)

			reqExt, err := reqWrapper.GetRequestExt()
			if err != nil {
				assert.Fail(t, "Error getting request ext from wrapper", tt.description)
			}
			reqExtSChain := reqExt.GetSChain()
			assert.Equal(t, tt.wantReqExtSChain, reqExtSChain, tt.description)

			sourceExt, err := reqWrapper.GetSourceExt()
			if err != nil {
				assert.Fail(t, "Error getting source ext from wrapper", tt.description)
			}
			sourceExtSChain := sourceExt.GetSChain()
			assert.Equal(t, tt.wantSourceExtSChain, sourceExtSChain, tt.description)
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
		newParamsValidator(t),
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

	errL := deps.validateRequest(&openrtb_ext.RequestWrapper{BidRequest: &req}, false, false)

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
		newParamsValidator(t),
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
		newParamsValidator(t),
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
		newParamsValidator(t),
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

	resReq, impExtInfoMap, _, errL := deps.parseRequest(req)

	assert.Nil(t, resReq, "Result request should be nil due to incorrect imp")
	assert.Nil(t, impExtInfoMap, "Impression info map should be nil due to incorrect imp")
	assert.Len(t, errL, 1, "One error should be returned")
	assert.Contains(t, errL[0].Error(), "echovideoattrs of type bool", "Incorrect error message")
}

func TestAuctionFirstPartyData(t *testing.T) {
	reqBody := validRequest(t, "first-party-data.json")
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&mockExchangeFPD{},
		newParamsValidator(t),
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
	resultRequest := deps.ex.(*mockExchangeFPD).lastRequest
	resultFPD := deps.ex.(*mockExchangeFPD).firstPartyData

	assert.Len(t, resultFPD, 2, "Result FPD length is incorrect")

	assert.NotNil(t, resultFPD[openrtb_ext.BidderName("bidder1")], "Result FPD for bidder1 is incorrect")
	assert.NotNil(t, resultFPD[openrtb_ext.BidderName("bidder1")].Site, "Result FPD for bidder1.Site is incorrect")
	assert.Nil(t, resultFPD[openrtb_ext.BidderName("bidder1")].App, "Result FPD for bidder1.App is incorrect")
	assert.NotNil(t, resultFPD[openrtb_ext.BidderName("bidder1")].User, "Result FPD for bidder1.User is incorrect")

	assert.NotNil(t, resultFPD[openrtb_ext.BidderName("bidder2")], "Result FPD for bidder2 is incorrect")
	assert.NotNil(t, resultFPD[openrtb_ext.BidderName("bidder2")].Site, "Result FPD for bidder2.Site is incorrect")
	assert.Nil(t, resultFPD[openrtb_ext.BidderName("bidder2")].App, "Result FPD for bidder2.App is incorrect")
	assert.NotNil(t, resultFPD[openrtb_ext.BidderName("bidder2")].User, "Result FPD for bidder2.User is incorrect")

	assert.Nil(t, resultRequest.App, "Result request App should be nil")
	assert.Nil(t, resultRequest.Site.Content.Data, "Result request Site.Content.Data is incorrect")
	assert.JSONEq(t, string(resultRequest.Site.Ext), `{"amp": 1}`, "Result request Site.Ext is incorrect")
	assert.Nil(t, resultRequest.User.Ext, "Result request User.Ext is incorrect")
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
		newParamsValidator(t),
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

// warningsCheckExchange is a well-behaved exchange which stores all incoming warnings.
type warningsCheckExchange struct {
	auctionRequest exchange.AuctionRequest
}

func (e *warningsCheckExchange) HoldAuction(ctx context.Context, r exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	e.auctionRequest = r
	return nil, nil
}

// nobidExchange is a well-behaved exchange which always bids "no bid".
type nobidExchange struct {
	gotRequest *openrtb2.BidRequest
}

func (e *nobidExchange) HoldAuction(ctx context.Context, r exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	e.gotRequest = r.BidRequest
	return &openrtb2.BidResponse{
		ID:    r.BidRequest.ID,
		BidID: "test bid id",
		NBR:   openrtb2.NoBidReasonCodeUnknownError.Ptr(),
	}, nil
}

type mockBidExchange struct {
	mockBidder mockBidExchangeBidder
	pbsRates   map[string]map[string]float64
}

func newMockBidExchange(bidder mockBidExchangeBidder, mockCurrencyConversionRates map[string]map[string]float64) *mockBidExchange {
	if bidder.BidCurrency == "" {
		bidder.BidCurrency = "USD"
	}

	return &mockBidExchange{
		mockBidder: bidder,
		pbsRates:   mockCurrencyConversionRates,
	}
}

// getAuctionCurrencyRates copies the logic of the exchange package for testing purposes
func (e *mockBidExchange) getAuctionCurrencyRates(customRates *openrtb_ext.ExtRequestCurrency) currency.Conversions {
	if customRates == nil {
		// The timestamp is required for the function signature, but is not used and its
		// value has no significance in the tests
		return currency.NewRates(e.pbsRates)
	}

	usePbsRates := true
	if customRates.UsePBSRates != nil {
		usePbsRates = *customRates.UsePBSRates
	}

	if !usePbsRates {
		// The timestamp is required for the function signature, but is not used and its
		// value has no significance in the tests
		return currency.NewRates(customRates.ConversionRates)
	}

	// Both PBS and custom rates can be used, check if ConversionRates is not empty
	if len(customRates.ConversionRates) == 0 {
		// Custom rates map is empty, use PBS rates only
		return currency.NewRates(e.pbsRates)
	}

	// Return an AggregateConversions object that includes both custom and PBS currency rates but will
	// prioritize custom rates over PBS rates whenever a currency rate is found in both
	return currency.NewAggregateConversions(currency.NewRates(customRates.ConversionRates), currency.NewRates(e.pbsRates))
}

// mockBidExchange is a well-behaved exchange that lists the bidders found in every bidRequest.Imp[i].Ext
// into the bidResponse.Ext to assert the bidder adapters that were not filtered out in the validation process
func (e *mockBidExchange) HoldAuction(ctx context.Context, r exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	bidResponse := &openrtb2.BidResponse{
		ID:    r.BidRequest.ID,
		BidID: "test bid id",
		NBR:   openrtb2.NoBidReasonCodeUnknownError.Ptr(),
	}

	// Use currencies inside r.BidRequest.Cur, if any, and convert currencies if needed
	if len(r.BidRequest.Cur) == 0 {
		r.BidRequest.Cur = []string{"USD"}
	}

	var currencyFrom string = e.mockBidder.getBidCurrency()
	var conversionRate float64 = 0.00
	var err error

	var requestExt openrtb_ext.ExtRequest
	if len(r.BidRequest.Ext) > 0 {
		if err := json.Unmarshal(r.BidRequest.Ext, &requestExt); err != nil {
			return nil, fmt.Errorf("request.ext is invalid: %v", err)
		}
	}

	conversions := e.getAuctionCurrencyRates(requestExt.Prebid.CurrencyConversions)
	for _, bidReqCur := range r.BidRequest.Cur {
		if conversionRate, err = conversions.GetRate(currencyFrom, bidReqCur); err == nil {
			bidResponse.Cur = bidReqCur
			break
		}
	}

	if conversionRate == 0 {
		// Can't have bids if there's not even a 1 USD to 1 USD conversion rate
		return nil, errors.New("Can't produce bid with no valid currency to use or currency conversion to convert to.")
	}

	if len(r.BidRequest.Imp) > 0 {
		var SeatBidMap = make(map[string]openrtb2.SeatBid, 0)
		for _, imp := range r.BidRequest.Imp {
			var bidderExts map[string]json.RawMessage
			if err := json.Unmarshal(imp.Ext, &bidderExts); err != nil {
				return nil, err
			}

			if rawPrebidExt, ok := bidderExts[openrtb_ext.PrebidExtKey]; ok {
				var prebidExt openrtb_ext.ExtImpPrebid
				if err := json.Unmarshal(rawPrebidExt, &prebidExt); err == nil && prebidExt.Bidder != nil {
					for bidder, ext := range prebidExt.Bidder {
						if ext == nil {
							continue
						}

						bidderExts[bidder] = ext
					}
				}
			}

			for bidderNameOrAlias := range bidderExts {
				if isBidderToValidate(bidderNameOrAlias) {
					if val, ok := SeatBidMap[bidderNameOrAlias]; ok {
						val.Bid = append(val.Bid, openrtb2.Bid{ID: e.mockBidder.getBidId(bidderNameOrAlias)})
					} else {
						SeatBidMap[bidderNameOrAlias] = openrtb2.SeatBid{
							Seat: e.mockBidder.getSeatName(bidderNameOrAlias),
							Bid: []openrtb2.Bid{
								{
									ID:    e.mockBidder.getBidId(bidderNameOrAlias),
									Price: e.mockBidder.getBidPrice() * conversionRate,
								},
							},
						}
					}
				}
			}
		}
		for _, seatBid := range SeatBidMap {
			bidResponse.SeatBid = append(bidResponse.SeatBid, seatBid)
		}
	}

	return bidResponse, nil
}

type mockBidExchangeBidder struct {
	BidCurrency string  `json:"currency"`
	BidPrice    float64 `json:"price"`
}

func (bidder mockBidExchangeBidder) getBidCurrency() string {
	return bidder.BidCurrency
}
func (bidder mockBidExchangeBidder) getBidPrice() float64 {
	return bidder.BidPrice
}
func (bidder mockBidExchangeBidder) getSeatName(bidderNameOrAlias string) string {
	return fmt.Sprintf("%s-bids", bidderNameOrAlias)
}
func (bidder mockBidExchangeBidder) getBidId(bidderNameOrAlias string) string {
	return fmt.Sprintf("%s-bid", bidderNameOrAlias)
}

type brokenExchange struct{}

func (e *brokenExchange) HoldAuction(ctx context.Context, r exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	return nil, errors.New("Critical, unrecoverable error.")
}

// StoredRequest testing

// Test stored request data

// Stored Requests
// first below is valid JSON
// second below is identical to first but with extra '}' for invalid JSON
var testStoredRequestData = map[string]json.RawMessage{
	"1": json.RawMessage(`{"id": "{{UUID}}"}`),
	"2": json.RawMessage(`{
		"id": "{{uuid}}",
		"tmax": 500,
		"ext": {
			"prebid": {
				"targeting": {
					"pricegranularity": "low"
				}
			}
		}
	}`),
	"3": json.RawMessage(`{
		"tmax": 500,
				"ext": {
						"prebid": {
								"targeting": {
										"pricegranularity": "low"
								}
						}
				}}
		}`),
	"4": json.RawMessage(`{"id": "{{UUID}}", "cur": ["USD"]}`),
}

// Stored Imp Requests
// first below has valid JSON but doesn't match schema
// second below has invalid JSON (missing comma after rubicon accountId entry) but otherwise matches schema
// third below has valid JSON and matches schema
var testStoredImpData = map[string]json.RawMessage{
	"1": json.RawMessage(`{
			"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": "abc",
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": "abc"
				}
			},
			"video":{
				"w":200,
				"h":300
			}
		}`),
	"2": json.RawMessage(`{
			"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": "abc",
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": "abc"
				}
			}
		}`),
	"7": json.RawMessage(`{
			"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": 12345678,
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": 23456789
					"siteId": 113932,
					"zoneId": 535510
				}
			}
		}`),
	"9": json.RawMessage(`{
"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": 12345678,
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": 23456789,
					"siteId": 113932,
					"zoneId": 535510
				}
			}
		}`),
	"10": json.RawMessage(`{
			"ext": {
				"appnexus": {
					"placementId": 12345678,
					"position": "above",
					"reserve": 0.35
				}
			}
		}`),
}

// Incoming requests with stored request IDs
var testStoredRequests = []string{
	`{
		"id": "ThisID",
		"imp": [
			{
				"video":{
					"h":300,
					"w":200
				},
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
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"imp": [
			{
				"id": "adUnit2",
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						},
						"options": {
							"echovideoattrs": true
						}
					},
					"appnexus": {
						"placementId": "def",
						"trafficSourceCode": "mysite.com",
						"reserve": null
					},
					"rubicon": null
				}
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"imp": [
			{
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
			}
		],
		"ext": {
			"prebid": {
				"storedrequest": {
					"id": "2"
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"imp": [
			{
				"id": "some-static-imp",
				"video":{
					"mimes":["video/mp4"]
				},
				"ext": {
					"appnexus": {
						"placementId": "abc",
						"position": "below"
					}
				}
			},
			{
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						}
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
}

// The expected requests after stored request processing
var testFinalRequests = []string{
	`{
		"id": "ThisID",
		"imp": [
			{
				"video":{
					"h":300,
					"w":200
				},
				"ext":{
					"appnexus":{
						"placementId":"abc",
						"position":"above",
						"reserve":0.35
					},
					"prebid":{
						"storedrequest":{
							"id":"1"
						},
					"options":{
						"echovideoattrs":true
					}
				},
				"rubicon":{
					"accountId":"abc"
				}
			},
			"id":"adUnit1"
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
			}
		}
}
	}`,
	`{
		"id": "ThisID",
		"imp": [
			{
				"video":{
					"w":200,
					"h":300
				},
				"ext":{
					"appnexus":{
						"placementId":"def",
						"position":"above",
						"trafficSourceCode":"mysite.com"
					},
					"prebid":{
						"storedrequest":{
							"id":"1"
						},
						"options":{
							"echovideoattrs":true
						}
					}
				},
				"id":"adUnit2"
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
	`{
  		"ext": {
  		  "prebid": {
  		    "storedrequest": {
  		      "id": "2"
  		    },
  		    "targeting": {
  		      "pricegranularity": "low"
  		    }
  		  }
  		},
  		"id": "ThisID",
  		"imp": [
  		  {
  		    "ext": {
  		      "appnexus": {
  		        "placementId": "abc",
  		        "position": "above",
  		        "reserve": 0.35
  		      },
  		      "prebid": {
  		        "storedrequest": {
  		          "id": "2"
  		        },
  		        "options":{
					"echovideoattrs":false
				}
  		      },
  		      "rubicon": {
  		        "accountId": "abc"
  		      }
  		    },
  		    "id": "adUnit1"
  		  }
  		],
  		"tmax": 500
	}
`,
	`{
	"id": "ThisID",
	"imp": [
		{
    		"id": "some-static-imp",
    		"video": {
    		  "mimes": [
    		    "video/mp4"
    		  ]
    		},
    		"ext": {
    		  "appnexus": {
    		    "placementId": "abc",
    		    "position": "below"
    		  }
    		}
  		},
  		{
  		  "ext": {
  		    "appnexus": {
  		      "placementId": "abc",
  		      "position": "above",
  		      "reserve": 0.35
  		    },
  		    "prebid": {
  		      "storedrequest": {
  		        "id": "1"
  		      }
  		    },
  		    "rubicon": {
  		      "accountId": "abc"
  		    }
  		  },
  		  "id": "adUnit1",
		  "video":{
				"w":200,
				"h":300
          }
  		}
	],
	"ext": {
		"prebid": {
			"cache": {
				"markup": 1
			},
			"targeting": {
			}
		}
	}
}`,
}

var testStoredImpIds = []string{
	"adUnit1", "adUnit2", "adUnit1", "some-static-imp",
}

var testStoredImps = []string{
	`{
		"id": "adUnit1",
        "ext": {
        	"appnexus": {
        		"placementId": "abc",
        		"position": "above",
        		"reserve": 0.35
        	},
        	"rubicon": {
        		"accountId": "abc"
        	}
        },
		"video":{
        	"w":200,
        	"h":300
		}
	}`,
	`{
		"id": "adUnit1",
        "ext": {
        	"appnexus": {
        		"placementId": "abc",
        		"position": "above",
        		"reserve": 0.35
        	},
        	"rubicon": {
        		"accountId": "abc"
        	}
        },
		"video":{
        	"w":200,
        	"h":300
		}
	}`,
	`{
			"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": "abc",
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": "abc"
				}
			}
		}`,
	``,
}

var testBidRequests = []string{
	`{
		"id": "ThisID",
		"app": {
			"id": "123"
		},
		"imp": [
			{
				"video":{
					"h":300,
					"w":200
				},
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
			}
		],
		"ext": {
			"prebid": {
				"cache": {
					"markup": 1
				},
				"targeting": {
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"site": {
			"page": "prebid.org"
		},
		"imp": [
			{
				"id": "adUnit2",
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						},
						"options": {
							"echovideoattrs": true
						}
					},
					"appnexus": {
						"placementId": "def",
						"trafficSourceCode": "mysite.com",
						"reserve": null
					},
					"rubicon": null
				}
			}
		],
		"ext": {
			"prebid": {
				"targeting": {
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"app": {
			"id": "123"
		},
		"imp": [
			{
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
			}
		],
		"ext": {
			"prebid": {
				"storedrequest": {
					"id": "2"
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"app": {
			"id": "123"
		},
		"imp": [
			{
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
			}
		],
		"ext": {
			"prebid": {
				"storedrequest": {
					"id": "2"
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"app": {
			"id": "123"
		},
		"imp": [
			{
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						},
						"options": {
							"echovideoattrs": false
						}
					}
				}
			}
		],
		"ext": {
			"prebid": {
				"storedrequest": {
					"id": "1"
				}
			}
		}
	}`,
	`{
		"id": "ThisID",
		"imp": [{
			"id": "some-impression-id",
			"banner": {
				"format": [{
						"w": 600,
						"h": 500
					},
					{
						"w": 300,
						"h": 600
					}
				]
			},
			"ext": {
				"appnexus": {
					"placementId": 12883451
				}
			}
		}],
		"ext": {
			"prebid": {
				"debug": true,
				"storedrequest": {
					"id": "4"
				}
			}
		},
	  "site": {
		"page": "https://example.com"
	  }
	}`,
}

type mockStoredReqFetcher struct {
}

func (cf mockStoredReqFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	return testStoredRequestData, testStoredImpData, nil
}

func (cf mockStoredReqFetcher) FetchResponses(ctx context.Context, ids []string) (data map[string]json.RawMessage, errs []error) {
	return nil, nil
}

type mockExchange struct {
	lastRequest *openrtb2.BidRequest
}

func (m *mockExchange) HoldAuction(ctx context.Context, r exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	m.lastRequest = r.BidRequest
	return &openrtb2.BidResponse{
		SeatBid: []openrtb2.SeatBid{{
			Bid: []openrtb2.Bid{{
				AdM: "<script></script>",
			}},
		}},
	}, nil
}

type mockExchangeFPD struct {
	lastRequest    *openrtb2.BidRequest
	firstPartyData map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData
}

func (m *mockExchangeFPD) HoldAuction(ctx context.Context, r exchange.AuctionRequest, debugLog *exchange.DebugLog) (*openrtb2.BidResponse, error) {
	m.lastRequest = r.BidRequest
	m.firstPartyData = r.FirstPartyData
	return &openrtb2.BidResponse{}, nil
}

func getBidderInfos(cfg map[string]config.Adapter, biddersNames []openrtb_ext.BidderName) config.BidderInfos {
	biddersInfos := make(config.BidderInfos)
	for _, name := range biddersNames {
		adapterConfig, ok := cfg[string(name)]
		if !ok {
			adapterConfig = config.Adapter{}
		}
		biddersInfos[string(name)] = newBidderInfo(adapterConfig)
	}
	return biddersInfos
}

func newBidderInfo(cfg config.Adapter) config.BidderInfo {
	return config.BidderInfo{
		Enabled: !cfg.Disabled,
	}
}

type hardcodedResponseIPValidator struct {
	response bool
}

func (v hardcodedResponseIPValidator) IsValid(net.IP, iputil.IPVersion) bool {
	return v.response
}

type fakeUUIDGenerator struct {
	id  string
	err error
}

func (f fakeUUIDGenerator) Generate() (string, error) {
	return f.id, f.err
}

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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			deps := &endpointDeps{
				fakeUUIDGenerator{},
				&warningsCheckExchange{},
				newParamsValidator(t),
				&mockStoredReqFetcher{},
				empty_fetcher.EmptyFetcher{},
				empty_fetcher.EmptyFetcher{},
				&config.Configuration{MaxRequestSize: int64(len(tt.givenRequestBody))},
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

			req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(tt.givenRequestBody))

			resReq, _, _, errL := deps.parseRequest(req)

			var expIExt, iExt map[string]interface{}
			err := json.Unmarshal(tt.expectedImpExt, &expIExt)
			assert.Nil(t, err, "unmarshal() should return nil error")

			assert.NotNil(t, resReq.BidRequest.Imp[0].Ext, "imp[0].Ext should not be nil")
			err = json.Unmarshal(resReq.BidRequest.Imp[0].Ext, &iExt)
			assert.Nil(t, err, "unmarshal() should return nil error")

			assert.Equal(t, expIExt, iExt, "bidderparams in imp[].Ext should match")

			var eReqE, reqE map[string]interface{}
			err = json.Unmarshal(tt.expectedReqExt, &eReqE)
			assert.Nil(t, err, "unmarshal() should return nil error")

			err = json.Unmarshal(resReq.BidRequest.Ext, &reqE)
			assert.Nil(t, err, "unmarshal() should return nil error")

			assert.Equal(t, eReqE, reqE, "req.Ext should match")

			assert.Len(t, errL, tt.expectedErrorCount, "error length should match")
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
		expectedStoredResponses map[string]json.RawMessage
		expectedErrorCount      int
		expectedError           string
	}{
		{
			name:             "reqimp has valid stored response",
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			deps := &endpointDeps{
				fakeUUIDGenerator{},
				&warningsCheckExchange{},
				newParamsValidator(t),
				&mockStoredReqFetcher{},
				empty_fetcher.EmptyFetcher{},
				empty_fetcher.EmptyFetcher{},
				&config.Configuration{MaxRequestSize: int64(len(tt.givenRequestBody))},
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

			req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(tt.givenRequestBody))

			_, _, storedResponses, errL := deps.parseRequest(req)

			if tt.expectedErrorCount == 0 {
				assert.Equal(t, tt.expectedStoredResponses, storedResponses, "stored responses should match")
			} else {
				assert.Contains(t, errL[0].Error(), tt.expectedError, "error should match")
			}

		})
	}
}

func TestValidateStoredResp(t *testing.T) {
	deps := &endpointDeps{
		fakeUUIDGenerator{},
		&nobidExchange{},
		newParamsValidator(t),
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
		description         string
		givenRequestWrapper *openrtb_ext.RequestWrapper
		expectedErrorList   []error
		hasStoredResponses  bool
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
			expectedErrorList:  []error{},
			hasStoredResponses: true,
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
			expectedErrorList:  []error{},
			hasStoredResponses: true,
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
			expectedErrorList:  []error{errors.New("request validation failed. The StoredAuctionResponse.ID field must be completely present with, or completely absent from, all impressions in request. No StoredAuctionResponse data found for request.imp[1].ext.prebid \n")},
			hasStoredResponses: true,
		},
	}

	for _, test := range testCases {
		errorList := deps.validateRequest(test.givenRequestWrapper, false, test.hasStoredResponses)
		assert.Equalf(t, test.expectedErrorList, errorList, "Error doesn't match: %s\n", test.description)
	}
}

func TestProcessStoredAuctionResponsesErrors(t *testing.T) {
	deps := &endpointDeps{}

	testCases := []struct {
		description       string
		requestJson       []byte
		expectedErrorList []error
	}{
		{
			description: "Invalid stored auction response format: empty stored Auction Response Id",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
    			      "prebid": {
    			        "storedauctionresponse": {
    			        }
    			      }
    			    }
    			  }
    			]}`),
			expectedErrorList: []error{errors.New("request.imp[0] has ext.prebid.storedauctionresponse specified, but \"id\" field is missing ")},
		},
		{
			description: "Invalid stored auction response format: empty stored Auction Response Id in second imp",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": "imp-id1",
    			    "ext": {
    			      "prebid": {
    			        "storedauctionresponse": {
							"id":"123"
    			        }
    			      }
    			    }
    			  },
			      {
    			    "id": "imp-id2",
    			    "ext": {
    			      "prebid": {
    			        "storedauctionresponse": {
							"id":""
    			        }
    			      }
    			    }
    			  }
    			]}`),
			expectedErrorList: []error{errors.New("request.imp[1] has ext.prebid.storedauctionresponse specified, but \"id\" field is missing ")},
		},
		{
			description: "Invalid stored auction response format: integer imp Id",
			requestJson: []byte(`{"imp": [
    			  {
    			    "id": 123,
    			    "ext": {
    			      "prebid": {
    			        "storedauctionresponse": {
							"id":"123"
    			        }
    			      }
    			    }
    			  }
    			]}`),
			expectedErrorList: []error{errors.New("Value is not a string: 123")},
		},
	}

	for _, test := range testCases {
		_, errorList := deps.processStoredAuctionResponses(nil, test.requestJson)
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
