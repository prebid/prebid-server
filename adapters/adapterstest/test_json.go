package adapterstest

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/mitchellh/copystructure"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
)

const jsonFileExtension string = ".json"

var supportedDirs = map[string]struct{}{
	"exemplary":         {},
	"supplemental":      {},
	"amp":               {},
	"video":             {},
	"videosupplemental": {},
}

// RunJSONBidderTest is a helper method intended to unit test Bidders' adapters.
// It requires that:
//
//  1. Bidders communicate with external servers over HTTP.
//  2. The HTTP request bodies are legal JSON.
//
// Although the project does not require it, we _strongly_ recommend that all Bidders write tests using this.
// Doing so has the following benefits:
//
//  1. This includes some basic tests which confirm that your Bidder is "well-behaved" for all the input samples.
//     For example, "no nil bids are allowed in the returned array".
//     These tests are tedious to write, but help prevent bugs during auctions.
//
//  2. In the future, we plan to auto-generate documentation from the "exemplary" test files.
//     Those docs will teach publishers how to use your Bidder, which should encourage adoption.
//
// To use this method, create *.json files in the following directories:
//
// adapters/{bidder}/{bidder}test/exemplary:
//
//	These show "ideal" BidRequests for your Bidder. If possible, configure your servers to return the same
//	expected responses forever. If your server responds appropriately, our future auto-generated documentation
//	can guarantee Publishers that your adapter works as documented.
//
// adapters/{bidder}/{bidder}test/supplemental:
//
//	Fill this with *.json files which are useful test cases, but are not appropriate for public example docs.
//	For example, a file in this directory might make sure that a mobile-only Bidder returns errors on non-mobile requests.
//
// Then create a test in your adapters/{bidder}/{bidder}_test.go file like so:
//
//	func TestJsonSamples(t *testing.T) {
//	  adapterstest.RunJSONBidderTest(t, "{bidder}test", instanceOfYourBidder)
//	}
func RunJSONBidderTest(t *testing.T, rootDir string, bidder adapters.Bidder) {
	err := filepath.WalkDir(rootDir, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		isJsonFile := !info.IsDir() && filepath.Ext(info.Name()) == jsonFileExtension
		RunSingleJSONBidderTest(t, bidder, path, isJsonFile)
		return nil
	})
	assert.NoError(t, err, "Error reading files from directory %s \n", rootDir)
}

func RunSingleJSONBidderTest(t *testing.T, bidder adapters.Bidder, path string, isJsonFile bool) {
	base := filepath.Base(filepath.Dir(path))
	if _, ok := supportedDirs[base]; !ok {
		return
	}

	allowErrors := base != "exemplary" && base != "video"
	if isJsonFile {
		specData, err := loadFile(path)
		if err != nil {
			t.Fatalf("Failed to load contents of file %s: %v", path, err)
		}

		if !allowErrors && specData.expectsErrors() {
			t.Fatalf("Exemplary spec %s must not expect errors.", path)
		}

		runSpec(t, path, specData, bidder, base == "amp", base == "videosupplemental" || base == "video")
	}
}

// LoadFile reads and parses a file as a test case. If something goes wrong, it returns an error.
func loadFile(filename string) (*testSpec, error) {
	specData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file %s: %v", filename, err)
	}

	var spec testSpec
	if err := json.Unmarshal(specData, &spec); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal JSON from file: %v", err)
	}

	return &spec, nil
}

// runSpec runs a single test case. It will make sure:
//
//   - That the Bidder does not return nil HTTP requests, bids, or errors inside their lists
//   - That the Bidder's HTTP calls match the spec's expectations.
//   - That the Bidder's Bids match the spec's expectations
//   - That the Bidder's errors match the spec's expectations
//
// More assertions will almost certainly be added in the future, as bugs come up.
func runSpec(t *testing.T, filename string, spec *testSpec, bidder adapters.Bidder, isAmpTest, isVideoTest bool) {
	reqInfo := getTestExtraRequestInfo(t, filename, spec, isAmpTest, isVideoTest)
	requests := testMakeRequestsImpl(t, filename, spec, bidder, reqInfo)

	testMakeBidsImpl(t, filename, spec, bidder, requests)
}

// getTestExtraRequestInfo builds the ExtraRequestInfo object that will be passed to testMakeRequestsImpl
func getTestExtraRequestInfo(t *testing.T, filename string, spec *testSpec, isAmpTest, isVideoTest bool) *adapters.ExtraRequestInfo {
	t.Helper()

	var reqInfo adapters.ExtraRequestInfo

	// If test request.ext defines its own currency rates, add currency conversion to reqInfo
	reqWrapper := &openrtb_ext.RequestWrapper{}
	reqWrapper.BidRequest = &spec.BidRequest

	reqExt, err := reqWrapper.GetRequestExt()
	assert.NoError(t, err, "Could not unmarshall test request ext. %s", filename)

	reqPrebid := reqExt.GetPrebid()
	if reqPrebid != nil && reqPrebid.CurrencyConversions != nil && len(reqPrebid.CurrencyConversions.ConversionRates) > 0 {
		err = currency.ValidateCustomRates(reqPrebid.CurrencyConversions)
		assert.NoError(t, err, "Error validating currency rates in the test request: %s", filename)

		// Get currency rates conversions from the test request.ext
		conversions := currency.NewRates(reqPrebid.CurrencyConversions.ConversionRates)

		// Create return adapters.ExtraRequestInfo object
		reqInfo = adapters.NewExtraRequestInfo(conversions)
	} else {
		reqInfo = adapters.ExtraRequestInfo{}
	}

	// Set PbsEntryPoint if either isAmpTest or isVideoTest is true
	if isAmpTest {
		// simulates AMP entry point
		reqInfo.PbsEntryPoint = "amp"
	} else if isVideoTest {
		reqInfo.PbsEntryPoint = "video"
	}

	return &reqInfo
}

type testSpec struct {
	BidRequest        openrtb2.BidRequest     `json:"mockBidRequest"`
	HttpCalls         []httpCall              `json:"httpCalls"`
	BidResponses      []expectedBidResponse   `json:"expectedBidResponses"`
	MakeRequestErrors []testSpecExpectedError `json:"expectedMakeRequestsErrors"`
	MakeBidsErrors    []testSpecExpectedError `json:"expectedMakeBidsErrors"`
}

type testSpecExpectedError struct {
	Value      string `json:"value"`
	Comparison string `json:"comparison"`
}

func (spec *testSpec) expectsErrors() bool {
	return len(spec.MakeRequestErrors) > 0 || len(spec.MakeBidsErrors) > 0
}

type httpCall struct {
	Request  httpRequest  `json:"expectedRequest"`
	Response httpResponse `json:"mockResponse"`
}

func (req *httpRequest) ToRequestData(t *testing.T) *adapters.RequestData {
	return &adapters.RequestData{
		Method: "POST",
		Uri:    req.Uri,
		Body:   req.Body,
	}
}

type httpRequest struct {
	Body    json.RawMessage `json:"body"`
	Uri     string          `json:"uri"`
	Headers http.Header     `json:"headers"`
	ImpIDs  []string        `json:"impIDs"`
}

type httpResponse struct {
	Status  int             `json:"status"`
	Body    json.RawMessage `json:"body"`
	Headers http.Header     `json:"headers"`
}

func (resp *httpResponse) ToResponseData(t *testing.T) *adapters.ResponseData {
	return &adapters.ResponseData{
		StatusCode: resp.Status,
		Body:       resp.Body,
		Headers:    resp.Headers,
	}
}

type expectedBidResponse struct {
	Bids                 []expectedBid   `json:"bids"`
	Currency             string          `json:"currency"`
	FledgeAuctionConfigs json.RawMessage `json:"fledgeauctionconfigs,omitempty"`
}

type expectedBid struct {
	Bid   json.RawMessage `json:"bid"`
	Type  string          `json:"type"`
	Seat  string          `json:"seat"`
	Video json.RawMessage `json:"video,omitempty"`
}

// ---------------------------------------
// Lots of ugly, repetitive code below here.
//
// reflect.DeepEquals doesn't work because each OpenRTB field has an `ext []byte`, but we really care if those are JSON-equal
//
// Marshalling the structs and then using a JSON-diff library isn't great either, since

// assertMakeRequestsOutput compares the actual http requests to the expected ones.
func assertMakeRequestsOutput(t *testing.T, filename string, actual []*adapters.RequestData, expected []httpCall) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("%s: MakeRequests had wrong request count. Expected %d, got %d", filename, len(expected), len(actual))
	}

	for i := 0; i < len(expected); i++ {
		var err error
		for j := 0; j < len(actual); j++ {
			if err = diffHttpRequests(fmt.Sprintf("%s: httpRequest[%d]", filename, i), actual[j], &(expected[i].Request)); err == nil {
				break
			}
		}
		assert.NoError(t, err, fmt.Sprintf("%s Expected RequestData was not returned by adapters' MakeRequests() implementation: httpRequest[%d]", filename, i))
	}
}

func assertErrorList(t *testing.T, description string, actual []error, expected []testSpecExpectedError) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("%s had wrong error count. Expected %d, got %d (%v)", description, len(expected), len(actual), actual)
	}
	for i := 0; i < len(actual); i++ {
		if expected[i].Comparison == "literal" {
			if expected[i].Value != actual[i].Error() {
				t.Errorf(`%s error[%d] had wrong message. Expected "%s", got "%s"`, description, i, expected[i].Value, actual[i].Error())
			}
		} else if expected[i].Comparison == "regex" {
			if matched, _ := regexp.MatchString(expected[i].Value, actual[i].Error()); !matched {
				t.Errorf(`%s error[%d] had wrong message. Expected match with regex "%s", got "%s"`, description, i, expected[i].Value, actual[i].Error())
			}
		} else if expected[i].Comparison == "startswith" {
			if !strings.HasPrefix(actual[i].Error(), expected[i].Value) {
				t.Errorf(`%s error[%d] had wrong message. Expected to start with "%s", got "%s"`, description, i, expected[i].Value, actual[i].Error())
			}
		} else {
			t.Fatalf(`invalid comparison type "%s"`, expected[i].Comparison)
		}
	}
}

func assertMakeBidsOutput(t *testing.T, filename string, bidderResponse *adapters.BidderResponse, expected expectedBidResponse) {
	t.Helper()
	if !assert.Len(t, bidderResponse.Bids, len(expected.Bids), "%s: Wrong MakeBids bidderResponse.Bids count. len(bidderResponse.Bids) = %d vs len(spec.BidResponses.Bids) = %d", filename, len(bidderResponse.Bids), len(expected.Bids)) {
		return
	}
	for i := 0; i < len(bidderResponse.Bids); i++ {
		diffBids(t, fmt.Sprintf("%s:  typedBid[%d]", filename, i), bidderResponse.Bids[i], &(expected.Bids[i]))
	}
	if expected.FledgeAuctionConfigs != nil {
		assert.NotNilf(t, bidderResponse.FledgeAuctionConfigs, "%s: expected fledgeauctionconfigs in bidderResponse", filename)
		fledgeAuctionConfigsJson, err := json.Marshal(bidderResponse.FledgeAuctionConfigs)
		assert.NoErrorf(t, err, "%s: failed to marshal actual FledgeAuctionConfig response into JSON.", filename)
		assert.JSONEqf(t, string(expected.FledgeAuctionConfigs), string(fledgeAuctionConfigsJson), "%s: incorrect fledgeauctionconfig", filename)
	} else {
		assert.Nilf(t, bidderResponse.FledgeAuctionConfigs, "%s: unexpected fledgeauctionconfigs in bidderResponse", filename)
	}
}

// diffHttpRequests compares the actual HTTP request data to the expected one.
// It assumes that the request bodies are JSON
func diffHttpRequests(description string, actual *adapters.RequestData, expected *httpRequest) error {

	if actual == nil {
		return fmt.Errorf("Bidders cannot return nil HTTP calls. %s was nil.", description)
	}

	if expected.Uri != actual.Uri {
		return fmt.Errorf(`%s.uri "%s" does not match expected "%s."`, description, actual.Uri, expected.Uri)
	}

	if expected.Headers != nil {
		actualHeader, err := json.Marshal(actual.Headers)
		if err != nil {
			return fmt.Errorf(`%s actual.Headers could not be marshalled. Error: %s"`, description, err.Error())
		}
		expectedHeader, err := json.Marshal(expected.Headers)
		if err != nil {
			return fmt.Errorf(`%s expected.Headers could not be marshalled. Error: %s"`, description, err.Error())
		}
		if err := diffJson(description, actualHeader, expectedHeader); err != nil {
			return err
		}
	}

	if len(expected.ImpIDs) < 1 {
		return fmt.Errorf(`expected.ImpIDs must contain at least one imp ID`)
	}

	opt := cmpopts.SortSlices(func(a, b string) bool { return a < b })
	if !cmp.Equal(expected.ImpIDs, actual.ImpIDs, opt) {
		return fmt.Errorf(`%s actual.ImpIDs "%q" do not match expected "%q"`, description, actual.ImpIDs, expected.ImpIDs)
	}
	return diffJson(description, actual.Body, expected.Body)
}

func diffBids(t *testing.T, description string, actual *adapters.TypedBid, expected *expectedBid) {
	if actual == nil {
		t.Errorf("Bidders cannot return nil TypedBids. %s was nil.", description)
		return
	}

	assert.Equal(t, string(expected.Seat), string(actual.Seat), fmt.Sprintf(`%s.seat "%s" does not match expected "%s."`, description, string(actual.Seat), string(expected.Seat)))
	assert.Equal(t, string(expected.Type), string(actual.BidType), fmt.Sprintf(`%s.type "%s" does not match expected "%s."`, description, string(actual.BidType), string(expected.Type)))
	assert.NoError(t, diffOrtbBids(fmt.Sprintf("%s.bid", description), actual.Bid, expected.Bid))
	if expected.Video != nil {
		assert.NoError(t, diffBidVideo(fmt.Sprintf("%s.video", description), actual.BidVideo, expected.Video))
	}
}

// diffOrtbBids compares the actual Bid made by the adapter to the expectation from the JSON file.
func diffOrtbBids(description string, actual *openrtb2.Bid, expected json.RawMessage) error {
	if actual == nil {
		return fmt.Errorf("Bidders cannot return nil Bids. %s was nil.", description)
	}

	actualJson, err := json.Marshal(actual)
	if err != nil {
		return fmt.Errorf("%s failed to marshal actual Bid into JSON. %v", description, err)
	}

	return diffJson(description, actualJson, expected)
}

func diffBidVideo(description string, actual *openrtb_ext.ExtBidPrebidVideo, expected json.RawMessage) error {
	actualJson, err := json.Marshal(actual)
	if err != nil {
		return fmt.Errorf("%s failed to marshal actual Bid Video into JSON. %v", description, err)
	}

	return diffJson(description, actualJson, []byte(expected))
}

// diffJson compares two JSON byte arrays for structural equality. It will produce an error if either
// byte array is not actually JSON.
func diffJson(description string, actual []byte, expected []byte) error {
	if len(actual) == 0 && len(expected) == 0 {
		return nil
	}
	if len(actual) == 0 || len(expected) == 0 {
		return fmt.Errorf("%s json diff failed. Expected %d bytes in body, but got %d.", description, len(expected), len(actual))
	}
	diff, err := gojsondiff.New().Compare(actual, expected)
	if err != nil {
		return fmt.Errorf("%s json diff failed. %v", description, err)
	}

	if diff.Modified() {
		var left interface{}
		if err := json.Unmarshal(actual, &left); err != nil {
			return fmt.Errorf("%s json did not match, but unmarshalling failed. %v", description, err)
		}
		printer := formatter.NewAsciiFormatter(left, formatter.AsciiFormatterConfig{
			ShowArrayIndex: true,
		})
		output, err := printer.Format(diff)
		if err != nil {
			return fmt.Errorf("%s did not match, but diff formatting failed. %v", description, err)
		} else {
			return fmt.Errorf("%s json did not match expected.\n\n%s", description, output)
		}
	}
	return nil
}

// testMakeRequestsImpl asserts the resulting values of the bidder's `MakeRequests()` implementation
// against the expected JSON-defined results and ensures we do not encounter data races in the process.
// To assert no data races happen we make use of:
//  1. A shallow copy of the unmarshalled openrtb2.BidRequest that will provide reference values to
//     shared memory that we don't want the adapters' implementation of `MakeRequests()` to modify.
//  2. A deep copy that will preserve the original values of all the fields. This copy remains untouched
//     by the adapters' processes and serves as reference of what the shared memory values should still
//     be after the `MakeRequests()` call.
func testMakeRequestsImpl(t *testing.T, filename string, spec *testSpec, bidder adapters.Bidder, reqInfo *adapters.ExtraRequestInfo) []*adapters.RequestData {
	t.Helper()

	deepBidReqCopy, shallowBidReqCopy, err := getDataRaceTestCopies(&spec.BidRequest)
	assert.NoError(t, err, "Could not create request copies. %s", filename)

	// Run MakeRequests
	requests, errs := bidder.MakeRequests(&spec.BidRequest, reqInfo)

	// Compare MakeRequests actual output versus expected values found in JSON file
	assertErrorList(t, fmt.Sprintf("%s: MakeRequests", filename), errs, spec.MakeRequestErrors)
	assertMakeRequestsOutput(t, filename, requests, spec.HttpCalls)

	// Assert no data races occur using original bidRequest copies of references and values
	assert.Equal(t, deepBidReqCopy, shallowBidReqCopy, "Data race found. Test: %s", filename)

	return requests
}

// getDataRaceTestCopies returns a deep copy and a shallow copy of the original bidRequest that will get
// compared to verify no data races occur.
func getDataRaceTestCopies(original *openrtb2.BidRequest) (*openrtb2.BidRequest, *openrtb2.BidRequest, error) {
	cpy, err := copystructure.Copy(original)
	if err != nil {
		return nil, nil, err
	}
	deepReqCopy := cpy.(*openrtb2.BidRequest)

	shallowReqCopy := *original

	// Prebid Server core makes shallow copies of imp elements and adapters are allowed to make changes
	// to them. Therefore, we need shallow copies of Imp elements here so our test replicates that
	// functionality and only fail when actual shared momory gets modified.
	if original.Imp != nil {
		shallowReqCopy.Imp = make([]openrtb2.Imp, len(original.Imp))
		copy(shallowReqCopy.Imp, original.Imp)
	}

	return deepReqCopy, &shallowReqCopy, nil
}

// testMakeBidsImpl asserts the results of the bidder MakeBids implementation against the expected JSON-defined results
func testMakeBidsImpl(t *testing.T, filename string, spec *testSpec, bidder adapters.Bidder, makeRequestsOut []*adapters.RequestData) {
	t.Helper()

	bidResponses := make([]*adapters.BidderResponse, 0)
	var bidsErrs = make([]error, 0, len(spec.MakeBidsErrors))

	// We should have as many bids as number of adapters.RequestData found in MakeRequests output
	for i := 0; i < len(makeRequestsOut); i++ {
		// Run MakeBids with JSON refined spec.HttpCalls info that was asserted to match MakeRequests
		// output inside testMakeRequestsImpl
		thisBidResponse, theseErrs := bidder.MakeBids(&spec.BidRequest, spec.HttpCalls[i].Request.ToRequestData(t), spec.HttpCalls[i].Response.ToResponseData(t))

		if theseErrs != nil {
			bidsErrs = append(bidsErrs, theseErrs...)
		}
		if thisBidResponse != nil {
			bidResponses = append(bidResponses, thisBidResponse)
		}
	}

	// Assert actual errors thrown by MakeBids implementation versus expected JSON-defined spec.MakeBidsErrors
	assertErrorList(t, fmt.Sprintf("%s: MakeBids", filename), bidsErrs, spec.MakeBidsErrors)

	// Assert MakeBids implementation BidResponses with expected JSON-defined spec.BidResponses[i].Bids
	if assert.Len(t, bidResponses, len(spec.BidResponses), "%s: MakeBids len(bidResponses) = %d vs len(spec.BidResponses) = %d", filename, len(bidResponses), len(spec.BidResponses)) {
		for i := 0; i < len(spec.BidResponses); i++ {
			assertMakeBidsOutput(t, filename, bidResponses[i], spec.BidResponses[i])
		}
	}
}
