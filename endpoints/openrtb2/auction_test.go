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

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/stored_requests"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"

	"github.com/buger/jsonparser"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb"
	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/util/iputil"
	"github.com/stretchr/testify/assert"
)

const maxSize = 1024 * 256

type testCase struct {
	BidRequest           json.RawMessage   `json:"mockBidRequest"`
	Config               *testConfigValues `json:"mockConfig"`
	ExpectedReturnCode   int               `json:"expectedReturnCode,omitempty"`
	ExpectedErrorMessage string            `json:"expectedErrorMessage"`
	ExpectedBidResponse  json.RawMessage   `json:"expectedBidResponse"`
}

type testConfigValues struct {
	AccountReq            bool     `json:"accountRequired"`
	AliasJSON             string   `json:"aliases"`
	BlacklistedAccountArr []string `json:"blacklistedAccts"`
	BlacklistedAppArr     []string `json:"blacklistedApps"`
	AdapterList           []string `json:"adapterList"`

	blacklistedAccountMap map[string]bool
	blacklistedAppMap     map[string]bool
	adaptersConfig        map[string]config.Adapter
}

func TestJsonSampleRequests(t *testing.T) {
	testSuites := []struct {
		description string
		directory   string
	}{
		{
			"Assert 200s on all bidRequests from exemplary folder",
			"sample-requests/valid-whole/exemplary",
		},
		{
			"Asserts we return 200s on well-formed Native requests.",
			"sample-requests/valid-native",
		},
		{
			"Asserts we return 400s on requests that are not supposed to pass validation",
			"sample-requests/invalid-whole",
		},
		{
			"Asserts we return 400s on requests with Native requests that don't pass validation",
			"sample-requests/invalid-native",
		},
		{
			"Makes sure we handle (default) aliased bidders properly",
			"sample-requests/aliased",
		},
		{
			"Asserts we return 503s on requests with blacklisted accounts and apps.",
			"sample-requests/blacklisted",
		},
		{
			"Assert that requests that come with no user id nor app id return error if the `AccountRequired` field in the `config.Configuration` structure is set to true",
			"sample-requests/account-required/no-account",
		},
		{
			"Assert requests that come with a valid user id nor app id when account is not required",
			"sample-requests/account-required/with-account",
		},
		{
			"Tests diagnostic messages for invalid stored requests",
			"sample-requests/invalid-stored",
		},
		{
			"Make sure requests with disabled bidders will fail",
			"sample-requests/disabled/bad",
		},
		{
			"There are both disabled and non-disabled bidders, we expect a 200",
			"sample-requests/disabled/good",
		},
	}
	for _, test := range testSuites {
		testCasefiles, err := getTestFiles(test.directory)
		assert.NoError(t, err, "Test case %s. Error reading files from directory %s \n", test.description, test.directory)

		for _, file := range testCasefiles {
			data, err := ioutil.ReadFile(file)
			assert.NoError(t, err, "Test case %s. Error reading file %s \n", test.description, file)

			assertTestCaseData(t, data, file)
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

func assertTestCaseData(t *testing.T, fileData []byte, testFile string) {
	t.Helper()

	// Retrieve values from JSON file
	test := jsonParseTestFile(t, fileData, testFile)

	test.Config = parseMaps(test.Config)

	// Run test
	actualCode, actualBidResponse := doRequest(t, test)

	// Assertions
	assert.Equal(t, test.ExpectedReturnCode, actualCode, "Test failed. Filename: %s \n", testFile)

	// Either assert bid response or expected error
	if test.ExpectedReturnCode != 200 {
		// Assert expected error
		assert.True(t, strings.HasPrefix(actualBidResponse, test.ExpectedErrorMessage), "Test failed. %s. Filename: %s \n", actualBidResponse, testFile)
	} else {
		// Assert expected response
		diffJson(t, testFile, []byte(actualBidResponse), test.ExpectedBidResponse)
	}
}

func jsonParseTestFile(t *testing.T, fileData []byte, testFile string) testCase {
	t.Helper()

	parsedTestData := testCase{}
	var err error

	// Get testCase values
	parsedTestData.BidRequest, _, _, err = jsonparser.Get(fileData, "mockBidRequest")
	assert.NoError(t, err, "Error jsonparsing root.mockBidRequest from file %s. Desc: %v.", testFile, err)

	parsedReturnCode, err := jsonparser.GetInt(fileData, "expectedReturnCode")
	assert.NoError(t, err, "Error jsonparsing root.code from file %s. Desc: %v.", testFile, err)

	parsedTestData.ExpectedReturnCode = int(parsedReturnCode)

	if parsedTestData.ExpectedReturnCode != 200 {
		// Get expected error, fail if parsing error
		parsedTestData.ExpectedErrorMessage, err = jsonparser.GetString(fileData, "expectedErrorMessage")
		assert.NoError(t, err, "Error jsonparsing root.expectedErrorMessage from file %s. Desc: %v.", testFile, err)
	} else {
		// Get expected response, fail if parsing error
		parsedTestData.ExpectedBidResponse, _, _, err = jsonparser.Get(fileData, "expectedBidResponse")
		assert.NoError(t, err, "Error jsonparsing root.expectedBidResponse from file %s. Desc: %v.", testFile, err)
	}

	// Get testCaseConfig values
	parsedTestData.Config = &testConfigValues{}
	accReq, err := jsonparser.GetBoolean(fileData, "mockConfig", "accountRequired")
	if err == nil {
		parsedTestData.Config.AccountReq = accReq
	}

	aliases, err := jsonparser.GetString(fileData, "mockConfig", "aliases")
	if err == nil {
		parsedTestData.Config.AliasJSON = aliases
	}

	parsedTestData.Config.BlacklistedAccountArr = parseStringArray(fileData, "blacklistedAccts")
	parsedTestData.Config.BlacklistedAppArr = parseStringArray(fileData, "blacklistedApps")
	parsedTestData.Config.AdapterList = parseStringArray(fileData, "disabledAdapters")

	return parsedTestData
}

func parseStringArray(fileData []byte, jsonField string) []string {
	rarr := []string{}
	jsonparser.ArrayEach(fileData, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		rarr = append(rarr, string(value))
	}, "mockConfig", jsonField)
	return rarr
}

func parseMaps(tc *testConfigValues) *testConfigValues {
	if len(tc.BlacklistedAccountArr) > 0 {
		tc.blacklistedAccountMap = make(map[string]bool, len(tc.BlacklistedAccountArr))
		for _, account := range tc.BlacklistedAccountArr {
			tc.blacklistedAccountMap[account] = true
		}
	}
	if len(tc.BlacklistedAppArr) > 0 {
		tc.blacklistedAppMap = make(map[string]bool, len(tc.BlacklistedAppArr))
		for _, app := range tc.BlacklistedAppArr {
			tc.blacklistedAppMap[app] = true
		}
	}
	if len(tc.AdapterList) > 0 {
		tc.adaptersConfig = make(map[string]config.Adapter, len(tc.AdapterList))
		for _, adapterName := range tc.AdapterList {
			tc.adaptersConfig[adapterName] = config.Adapter{Disabled: true}
		}
	}
	return tc
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
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	metrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{})
	endpoint, _ := NewEndpoint(ex, newParamsValidator(t), empty_fetcher.EmptyFetcher{}, empty_fetcher.EmptyFetcher{}, cfg, metrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), map[string]string{}, []byte{}, openrtb_ext.BidderMap)

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
	disabledBidders := map[string]string{}
	bidderMap := exchange.DisableBidders(getBidderInfos(test.Config.adaptersConfig, openrtb_ext.BidderList()), disabledBidders)

	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	metrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{})

	endpoint, _ := NewEndpoint(
		&nobidExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{
			MaxRequestSize:     maxSize,
			BlacklistedApps:    test.Config.BlacklistedAppArr,
			BlacklistedAppMap:  test.Config.blacklistedAppMap,
			BlacklistedAccts:   test.Config.BlacklistedAccountArr,
			BlacklistedAcctMap: test.Config.blacklistedAccountMap,
			AccountRequired:    test.Config.AccountReq,
		},
		metrics,
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		disabledBidders,
		[]byte(test.Config.AliasJSON),
		bidderMap,
	)

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
	disabledBidders := map[string]string{
		"indexExchange": "Bidder \"indexExchange\" has been deprecated and is no longer available. Please use bidder \"ix\" and note that the bidder params have changed.",
	}
	adaptersConfigs := make(map[string]config.Adapter)
	bidderMap := exchange.DisableBidders(getBidderInfos(adaptersConfigs, openrtb_ext.BidderList()), disabledBidders)

	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	metrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{})
	endpoint, _ := NewEndpoint(&nobidExchange{}, newParamsValidator(t), &mockStoredReqFetcher{}, empty_fetcher.EmptyFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, metrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), disabledBidders, aliasJSON, bidderMap)

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
	metrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{})
	_, err := NewEndpoint(nil, newParamsValidator(t), empty_fetcher.EmptyFetcher{}, empty_fetcher.EmptyFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, metrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), map[string]string{}, []byte{}, openrtb_ext.BidderMap)
	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil Exchange.")
	}
}

// TestNilValidator makes sure we fail when given nil for the BidderParamValidator.
func TestNilValidator(t *testing.T) {
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	metrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{})
	_, err := NewEndpoint(&nobidExchange{}, nil, empty_fetcher.EmptyFetcher{}, empty_fetcher.EmptyFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, metrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), map[string]string{}, []byte{}, openrtb_ext.BidderMap)
	if err == nil {
		t.Errorf("NewEndpoint should return an error when given a nil BidderParamValidator.")
	}
}

// TestExchangeError makes sure we return a 500 if the exchange auction fails.
func TestExchangeError(t *testing.T) {
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	metrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{})
	endpoint, _ := NewEndpoint(&brokenExchange{}, newParamsValidator(t), empty_fetcher.EmptyFetcher{}, empty_fetcher.EmptyFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, metrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), map[string]string{}, []byte{}, openrtb_ext.BidderMap)
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
	bidReq := &openrtb.BidRequest{}

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
	bidReq := &openrtb.BidRequest{
		Device: &openrtb.Device{
			UA: "bar",
		},
	}

	setUAImplicitly(httpReq, bidReq)

	if bidReq.Device.UA != "bar" {
		t.Errorf("bidrequest.device.ua should have been \"bar\". Got %s", bidReq.Device.UA)
	}
}

func TestAuctionTypeDefault(t *testing.T) {
	bidReq := &openrtb.BidRequest{}
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

	metrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{})
	for _, test := range testCases {
		exchange := &nobidExchange{}
		cfg := &config.Configuration{
			MaxRequestSize: maxSize,
			RequestValidation: config.RequestValidation{
				IPv4PrivateNetworksParsed: test.privateNetworksIPv4,
				IPv6PrivateNetworksParsed: test.privateNetworksIPv6,
			},
		}
		endpoint, _ := NewEndpoint(exchange, newParamsValidator(t), &mockStoredReqFetcher{}, empty_fetcher.EmptyFetcher{}, cfg, metrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), map[string]string{}, []byte{}, openrtb_ext.BidderMap)

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
		request         openrtb.BidRequest
		expectedRequest openrtb.BidRequest
	}{
		{
			description:     "Device Missing - Not Set In Header",
			dntHeader:       "",
			request:         openrtb.BidRequest{},
			expectedRequest: openrtb.BidRequest{},
		},
		{
			description: "Device Missing - Set To 0 In Header",
			dntHeader:   "0",
			request:     openrtb.BidRequest{},
			expectedRequest: openrtb.BidRequest{
				Device: &openrtb.Device{
					DNT: &disabled,
				},
			},
		},
		{
			description: "Device Missing - Set To 1 In Header",
			dntHeader:   "1",
			request:     openrtb.BidRequest{},
			expectedRequest: openrtb.BidRequest{
				Device: &openrtb.Device{
					DNT: &enabled,
				},
			},
		},
		{
			description: "Not Set In Request - Not Set In Header",
			dntHeader:   "",
			request: openrtb.BidRequest{
				Device: &openrtb.Device{},
			},
			expectedRequest: openrtb.BidRequest{
				Device: &openrtb.Device{},
			},
		},
		{
			description: "Not Set In Request - Set To 0 In Header",
			dntHeader:   "0",
			request: openrtb.BidRequest{
				Device: &openrtb.Device{},
			},
			expectedRequest: openrtb.BidRequest{
				Device: &openrtb.Device{
					DNT: &disabled,
				},
			},
		},
		{
			description: "Not Set In Request - Set To 1 In Header",
			dntHeader:   "1",
			request: openrtb.BidRequest{
				Device: &openrtb.Device{},
			},
			expectedRequest: openrtb.BidRequest{
				Device: &openrtb.Device{
					DNT: &enabled,
				},
			},
		},
		{
			description: "Set In Request - Not Set In Header",
			dntHeader:   "",
			request: openrtb.BidRequest{
				Device: &openrtb.Device{
					DNT: &enabled,
				},
			},
			expectedRequest: openrtb.BidRequest{
				Device: &openrtb.Device{
					DNT: &enabled,
				},
			},
		},
		{
			description: "Set In Request - Set To 0 In Header",
			dntHeader:   "0",
			request: openrtb.BidRequest{
				Device: &openrtb.Device{
					DNT: &enabled,
				},
			},
			expectedRequest: openrtb.BidRequest{
				Device: &openrtb.Device{
					DNT: &enabled,
				},
			},
		},
		{
			description: "Set In Request - Set To 1 In Header",
			dntHeader:   "1",
			request: openrtb.BidRequest{
				Device: &openrtb.Device{
					DNT: &enabled,
				},
			},
			expectedRequest: openrtb.BidRequest{
				Device: &openrtb.Device{
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

	metrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{})
	for _, test := range testCases {
		exchange := &nobidExchange{}
		endpoint, _ := NewEndpoint(exchange, newParamsValidator(t), &mockStoredReqFetcher{}, empty_fetcher.EmptyFetcher{}, &config.Configuration{MaxRequestSize: maxSize}, metrics, analyticsConf.NewPBSAnalytics(&config.Analytics{}), map[string]string{}, []byte{}, openrtb_ext.BidderMap)

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

	imps := []openrtb.Imp{
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
	bidReq := &openrtb.BidRequest{}

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

// Test the stored request functionality
func TestStoredRequests(t *testing.T) {
	// NewMetrics() will create a new go_metrics MetricsEngine, bypassing the need for a crafted configuration set to support it.
	// As a side effect this gives us some coverage of the go_metrics piece of the metrics engine.
	metrics := pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{})
	deps := &endpointDeps{
		&nobidExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		metrics,
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
	}

	for i, requestData := range testStoredRequests {
		newRequest, errList := deps.processStoredRequests(context.Background(), json.RawMessage(requestData))
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
		if !jsonpatch.Equal(newRequest, expectJson) {
			t.Errorf("Error in processStoredRequests, test %d failed on compare\nFound:\n%s\nExpected:\n%s", i, string(newRequest), string(expectJson))
		}
	}
}

// TestOversizedRequest makes sure we behave properly when the request size exceeds the configured max.
func TestOversizedRequest(t *testing.T) {
	reqBody := validRequest(t, "site.json")
	deps := &endpointDeps{
		&nobidExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(len(reqBody) - 1)},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{}),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
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
		&nobidExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(len(reqBody))},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{}),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
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
		&mockExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{}),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BidderMap,
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

	bidReq := openrtb.BidRequest{
		Site: &openrtb.Site{},
	}
	setSiteImplicitly(httpReq, &bidReq)
	assert.JSONEq(t, `{"amp":0}`, string(bidReq.Site.Ext))
}

func TestImplicitAMPOtherExt(t *testing.T) {
	httpReq, err := http.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	if !assert.NoError(t, err) {
		return
	}

	bidReq := openrtb.BidRequest{
		Site: &openrtb.Site{
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

	bidReq := openrtb.BidRequest{
		Site: &openrtb.Site{
			Ext: json.RawMessage(`{"amp":1}`),
		},
	}
	setSiteImplicitly(httpReq, &bidReq)
	assert.JSONEq(t, `{"amp":1}`, string(bidReq.Site.Ext))
}

// TestContentType prevents #328
func TestContentType(t *testing.T) {
	endpoint, _ := NewEndpoint(
		&mockExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: maxSize},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{}),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		[]byte{},
		openrtb_ext.BidderMap,
	)
	request := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(validRequest(t, "site.json")))
	recorder := httptest.NewRecorder()
	endpoint(recorder, request, nil)

	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type should be application/json. Got %s", recorder.Header().Get("Content-Type"))
	}
}

// TestDisabledBidder makes sure we pass when encountering a disabled bidder in the configuration.
func TestDisabledBidder(t *testing.T) {
	filename := "sample-requests/invalid-whole/unknown-bidder.json"
	fileData, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to fetch a valid request: %v", err)
	}
	testBidRequest, _, _, err := jsonparser.Get(fileData, "mockBidRequest")
	assert.NoError(t, err, "Error jsonparsing root.mockBidRequest from file %s. Desc: %v.", filename, err)

	deps := &endpointDeps{
		&nobidExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{
			MaxRequestSize: int64(len(testBidRequest)),
		},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{}),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{"unknownbidder": "The bidder 'unknownbidder' has been disabled."},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
	}

	req := httptest.NewRequest("POST", "/openrtb2/auction", strings.NewReader(string(testBidRequest)))
	recorder := httptest.NewRecorder()

	deps.Auction(recorder, req, nil)

	if recorder.Code != http.StatusOK {
		t.Errorf("Endpoint should return a 200 if the unknown bidder was disabled.")
	}

	if bytesRead, err := req.Body.Read(make([]byte, 1)); bytesRead != 0 || err != io.EOF {
		t.Errorf("The request body should have been read to completion.")
	}
}

func TestValidateImpExtDisabledBidder(t *testing.T) {
	imp := &openrtb.Imp{
		Ext: json.RawMessage(`{"appnexus":{"placement_id":555},"unknownbidder":{"foo":"bar"}}`),
	}
	deps := &endpointDeps{
		&nobidExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{MaxRequestSize: int64(8096)},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{}),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{"unknownbidder": "The bidder 'unknownbidder' has been disabled."},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
	}
	errs := deps.validateImpExt(imp, nil, 0)
	assert.JSONEq(t, `{"appnexus":{"placement_id":555}}`, string(imp.Ext))
	assert.Equal(t, []error{&errortypes.BidderTemporarilyDisabled{Message: "The bidder 'unknownbidder' has been disabled."}}, errs)
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
		&nobidExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{}),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
	}

	ui := uint64(1)
	req := openrtb.BidRequest{
		ID: "someID",
		Imp: []openrtb.Imp{
			{
				ID: "imp-ID",
				Banner: &openrtb.Banner{
					W: &ui,
					H: &ui,
				},
				Ext: json.RawMessage("{\"appnexus\": {\"placementId\": 5667}}"),
			},
		},
		Site: &openrtb.Site{
			ID: "myID",
		},
		Cur: []string{"USD", "EUR"},
	}

	errL := deps.validateRequest(&req)

	expectedError := errortypes.Warning{Message: "A prebid request can only process one currency. Taking the first currency in the list, USD, as the active currency"}
	assert.ElementsMatch(t, errL, []error{&expectedError})
}

func TestCCPAInvalid(t *testing.T) {
	deps := &endpointDeps{
		&nobidExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{}),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
	}

	ui := uint64(1)
	req := openrtb.BidRequest{
		ID: "someID",
		Imp: []openrtb.Imp{
			{
				ID: "imp-ID",
				Banner: &openrtb.Banner{
					W: &ui,
					H: &ui,
				},
				Ext: json.RawMessage(`{"appnexus": {"placementId": 5667}}`),
			},
		},
		Site: &openrtb.Site{
			ID: "myID",
		},
		Regs: &openrtb.Regs{
			Ext: json.RawMessage(`{"us_privacy":"invalid by length"}`),
		},
	}

	errL := deps.validateRequest(&req)

	expectedWarning := errortypes.InvalidPrivacyConsent{Message: "CCPA consent is invalid and will be ignored. (request.regs.ext.us_privacy must contain 4 characters)"}
	assert.ElementsMatch(t, errL, []error{&expectedWarning})

	assert.Empty(t, req.Regs.Ext, "Invalid Consent Removed From Request")
}

func TestValidateSourceTID(t *testing.T) {
	cfg := &config.Configuration{
		AutoGenSourceTID: true,
	}

	deps := &endpointDeps{
		&nobidExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		cfg,
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{}),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
	}

	ui := uint64(1)
	req := openrtb.BidRequest{
		ID: "someID",
		Imp: []openrtb.Imp{
			{
				ID: "imp-ID",
				Banner: &openrtb.Banner{
					W: &ui,
					H: &ui,
				},
				Ext: json.RawMessage(`{"appnexus": {"placementId": 5667}}`),
			},
		},
		Site: &openrtb.Site{
			ID: "myID",
		},
		Regs: &openrtb.Regs{
			Ext: json.RawMessage(`{"us_privacy":"invalid by length"}`),
		},
	}

	deps.validateRequest(&req)
	assert.NotEmpty(t, req.Source.TID, "Expected req.Source.TID to be filled with a randomly generated UID")
}

func TestSChainInvalid(t *testing.T) {
	deps := &endpointDeps{
		&nobidExchange{},
		newParamsValidator(t),
		&mockStoredReqFetcher{},
		empty_fetcher.EmptyFetcher{},
		empty_fetcher.EmptyFetcher{},
		&config.Configuration{},
		pbsmetrics.NewMetrics(metrics.NewRegistry(), openrtb_ext.BidderList(), config.DisabledMetrics{}),
		analyticsConf.NewPBSAnalytics(&config.Analytics{}),
		map[string]string{},
		false,
		[]byte{},
		openrtb_ext.BidderMap,
		nil,
		nil,
		hardcodedResponseIPValidator{response: true},
	}

	ui := uint64(1)
	req := openrtb.BidRequest{
		ID: "someID",
		Imp: []openrtb.Imp{
			{
				ID: "imp-ID",
				Banner: &openrtb.Banner{
					W: &ui,
					H: &ui,
				},
				Ext: json.RawMessage(`{"appnexus": {"placementId": 5667}}`),
			},
		},
		Site: &openrtb.Site{
			ID: "myID",
		},
		Regs: &openrtb.Regs{
			Ext: json.RawMessage(`{"us_privacy":"abcd"}`),
		},
		Ext: json.RawMessage(`{"prebid":{"schains":[{"bidders":["appnexus"],"schain":{"complete":1,"nodes":[{"asi":"directseller1.com","sid":"00001","rid":"BidRequest1","hp":1}],"ver":"1.0"}}, {"bidders":["appnexus"],"schain":{"complete":1,"nodes":[{"asi":"directseller2.com","sid":"00002","rid":"BidRequest2","hp":1}],"ver":"1.0"}}]}}`),
	}

	errL := deps.validateRequest(&req)

	expectedError := fmt.Errorf("request.ext.prebid.schains contains multiple schains for bidder appnexus; it must contain no more than one per bidder.")
	assert.ElementsMatch(t, errL, []error{expectedError})
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
		pub           *openrtb.Publisher
		expectedAccID string
	}{
		{
			description: "Publisher.ID and Publisher.Ext.Prebid.ParentAccount both present",
			pub: &openrtb.Publisher{
				ID:  testPubID,
				Ext: testPubExtJSON,
			},
			expectedAccID: testParentAccount,
		},
		{
			description: "Only Publisher.Ext.Prebid.ParentAccount present",
			pub: &openrtb.Publisher{
				ID:  "",
				Ext: testPubExtJSON,
			},
			expectedAccID: testParentAccount,
		},
		{
			description: "Only Publisher.ID present",
			pub: &openrtb.Publisher{
				ID: testPubID,
			},
			expectedAccID: testPubID,
		},
		{
			description:   "Neither Publisher.ID or Publisher.Ext.Prebid.ParentAccount present",
			pub:           &openrtb.Publisher{},
			expectedAccID: pbsmetrics.PublisherUnknown,
		},
		{
			description:   "Publisher is nil",
			pub:           nil,
			expectedAccID: pbsmetrics.PublisherUnknown,
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
		req          *openrtb.BidRequest
		ipValidator  iputil.IPValidator
		expectedIPv4 string
		expectedIPv6 string
	}{
		{
			description: "Empty",
			req: &openrtb.BidRequest{
				Device: &openrtb.Device{
					IP:   "",
					IPv6: "",
				},
			},
			expectedIPv4: "",
			expectedIPv6: "",
		},
		{
			description: "Valid",
			req: &openrtb.BidRequest{
				Device: &openrtb.Device{
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
			req: &openrtb.BidRequest{
				Device: &openrtb.Device{
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
			req: &openrtb.BidRequest{
				Device: &openrtb.Device{
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
			req: &openrtb.BidRequest{
				Device: &openrtb.Device{
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
		req           *openrtb.BidRequest
		expectRandTID bool
		expectedTID   string
	}{
		{
			description:   "req.Source not present. Expecting a randomly generated TID value",
			req:           &openrtb.BidRequest{},
			expectRandTID: true,
		},
		{
			description: "req.Source.TID not present. Expecting a randomly generated TID value",
			req: &openrtb.BidRequest{
				Source: &openrtb.Source{},
			},
			expectRandTID: true,
		},
		{
			description: "req.Source.TID present. Expecting no change",
			req: &openrtb.BidRequest{
				Source: &openrtb.Source{
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

// nobidExchange is a well-behaved exchange which always bids "no bid".
type nobidExchange struct {
	gotRequest *openrtb.BidRequest
}

func (e *nobidExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, ids exchange.IdFetcher, labels pbsmetrics.Labels, categoriesFetcher *stored_requests.CategoryFetcher, debugLog *exchange.DebugLog) (*openrtb.BidResponse, error) {
	e.gotRequest = bidRequest
	return &openrtb.BidResponse{
		ID:    bidRequest.ID,
		BidID: "test bid id",
		NBR:   openrtb.NoBidReasonCodeUnknownError.Ptr(),
	}, nil
}

type brokenExchange struct{}

func (e *brokenExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, ids exchange.IdFetcher, labels pbsmetrics.Labels, categoriesFetcher *stored_requests.CategoryFetcher, debugLog *exchange.DebugLog) (*openrtb.BidResponse, error) {
	return nil, errors.New("Critical, unrecoverable error.")
}

// StoredRequest testing

// Test stored request data

// Stored Requests
// first below is valid JSON
// second below is identical to first but with extra '}' for invalid JSON
var testStoredRequestData = map[string]json.RawMessage{
	"2": json.RawMessage(`{
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
}

// Incoming requests with stored request IDs
var testStoredRequests = []string{
	`{
		"id": "ThisID",
		"imp": [
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
	`{
		"id": "ThisID",
		"imp": [
			{
				"id": "adUnit2",
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
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
							"id": "1"
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
				"id": "adUnit1",
				"ext": {
					"appnexus": {
						"placementId": "abc",
						"position": "above",
						"reserve": 0.35
					},
					"rubicon": {
						"accountId": "abc"
					},
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
	`{
		"id": "ThisID",
		"imp": [
			{
				"id": "adUnit2",
				"ext": {
					"prebid": {
						"storedrequest": {
							"id": "1"
						}
					},
					"appnexus": {
						"placementId": "def",
						"position": "above",
						"trafficSourceCode": "mysite.com"
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
				"id": "adUnit1",
				"ext": {
					"appnexus": {
						"placementId": "abc",
						"position": "above",
						"reserve": 0.35
					},
					"rubicon": {
						"accountId": "abc"
					},
					"prebid": {
						"storedrequest": {
							"id": "1"
						}
					}
				}
			}
		],
		"tmax": 500,
		"ext": {
			"prebid": {
				"targeting": {
					"pricegranularity": "low"
				},
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
			"id": "adUnit1",
			"ext": {
				"appnexus": {
					"placementId": "abc",
					"position": "above",
					"reserve": 0.35
				},
				"rubicon": {
					"accountId": "abc"
				},
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

type mockStoredReqFetcher struct {
}

func (cf mockStoredReqFetcher) FetchRequests(ctx context.Context, requestIDs []string, impIDs []string) (requestData map[string]json.RawMessage, impData map[string]json.RawMessage, errs []error) {
	return testStoredRequestData, testStoredImpData, nil
}

type mockExchange struct {
	lastRequest *openrtb.BidRequest
}

func (m *mockExchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, ids exchange.IdFetcher, labels pbsmetrics.Labels, categoriesFetcher *stored_requests.CategoryFetcher, debugLog *exchange.DebugLog) (*openrtb.BidResponse, error) {
	m.lastRequest = bidRequest
	return &openrtb.BidResponse{
		SeatBid: []openrtb.SeatBid{{
			Bid: []openrtb.Bid{{
				AdM: "<script></script>",
			}},
		}},
	}, nil
}

func getBidderInfos(cfg map[string]config.Adapter, biddersNames []openrtb_ext.BidderName) adapters.BidderInfos {
	biddersInfos := make(adapters.BidderInfos)
	for _, name := range biddersNames {
		adapterConfig, ok := cfg[string(name)]
		if !ok {
			adapterConfig = config.Adapter{}
		}
		biddersInfos[string(name)] = newBidderInfo(adapterConfig)
	}
	return biddersInfos
}

func newBidderInfo(cfg config.Adapter) adapters.BidderInfo {
	status := adapters.StatusActive
	if cfg.Disabled == true {
		status = adapters.StatusDisabled
	}

	return adapters.BidderInfo{
		Status: status,
	}
}

type hardcodedResponseIPValidator struct {
	response bool
}

func (v hardcodedResponseIPValidator) IsValid(net.IP, iputil.IPVersion) bool {
	return v.response
}
