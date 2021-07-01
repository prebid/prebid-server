package adapterstest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"testing"

	"github.com/mitchellh/copystructure"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/stretchr/testify/assert"
	"github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"

	"net/http"
)

// RunJSONBidderTest is a helper method intended to unit test Bidders' adapters.
// It requires that:
//
//   1. Bidders communicate with external servers over HTTP.
//   2. The HTTP request bodies are legal JSON.
//
// Although the project does not require it, we _strongly_ recommend that all Bidders write tests using this.
// Doing so has the following benefits:
//
// 1. This includes some basic tests which confirm that your Bidder is "well-behaved" for all the input samples.
//    For example, "no nil bids are allowed in the returned array".
//    These tests are tedious to write, but help prevent bugs during auctions.
//
// 2. In the future, we plan to auto-generate documentation from the "exemplary" test files.
//    Those docs will teach publishers how to use your Bidder, which should encourage adoption.
//
// To use this method, create *.json files in the following directories:
//
// adapters/{bidder}/{bidder}test/exemplary:
//
//   These show "ideal" BidRequests for your Bidder. If possible, configure your servers to return the same
//   expected responses forever. If your server responds appropriately, our future auto-generated documentation
//   can guarantee Publishers that your adapter works as documented.
//
// adapters/{bidder}/{bidder}test/supplemental:
//
//   Fill this with *.json files which are useful test cases, but are not appropriate for public example docs.
//   For example, a file in this directory might make sure that a mobile-only Bidder returns errors on non-mobile requests.
//
// Then create a test in your adapters/{bidder}/{bidder}_test.go file like so:
//
//   func TestJsonSamples(t *testing.T) {
//     adapterstest.RunJSONBidderTest(t, "{bidder}test", instanceOfYourBidder)
//   }
//
func RunJSONBidderTest(t *testing.T, rootDir string, bidder adapters.Bidder) {
	runTests(t, fmt.Sprintf("%s/exemplary", rootDir), bidder, false, false, false)
	runTests(t, fmt.Sprintf("%s/supplemental", rootDir), bidder, true, false, false)
	runTests(t, fmt.Sprintf("%s/amp", rootDir), bidder, true, true, false)
	runTests(t, fmt.Sprintf("%s/video", rootDir), bidder, false, false, true)
	runTests(t, fmt.Sprintf("%s/videosupplemental", rootDir), bidder, true, false, true)
}

// runTests runs all the *.json files in a directory. If allowErrors is false, and one of the test files
// expects errors from the bidder, then the test will fail.
func runTests(t *testing.T, directory string, bidder adapters.Bidder, allowErrors, isAmpTest, isVideoTest bool) {
	if specFiles, err := ioutil.ReadDir(directory); err == nil {
		for _, specFile := range specFiles {
			fileName := fmt.Sprintf("%s/%s", directory, specFile.Name())
			specData, err := loadFile(fileName)
			if err != nil {
				t.Fatalf("Failed to load contents of file %s: %v", fileName, err)
			}

			if !allowErrors && specData.expectsErrors() {
				t.Fatalf("Exemplary spec %s must not expect errors.", fileName)
			}
			runSpec(t, fileName, specData, bidder, isAmpTest, isVideoTest)
		}
	}
}

// LoadFile reads and parses a file as a test case. If something goes wrong, it returns an error.
func loadFile(filename string) (*testSpec, error) {
	specData, err := ioutil.ReadFile(filename)
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
	reqInfo := adapters.ExtraRequestInfo{}
	if isAmpTest {
		// simulates AMP entry point
		reqInfo.PbsEntryPoint = "amp"
	} else if isVideoTest {
		reqInfo.PbsEntryPoint = "video"
	}

	requests := testMakeRequestsImpl(t, filename, spec, bidder, &reqInfo)

	testMakeBidsImpl(t, filename, spec, bidder, requests)
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
	Bids     []expectedBid `json:"bids"`
	Currency string        `json:"currency"`
}

type expectedBid struct {
	Bid  json.RawMessage `json:"bid"`
	Type string          `json:"type"`
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
	for i := 0; i < len(actual); i++ {
		diffHttpRequests(t, fmt.Sprintf("%s: httpRequest[%d]", filename, i), actual[i], &(expected[i].Request))
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
		} else {
			t.Fatalf(`invalid comparison type "%s"`, expected[i].Comparison)
		}
	}
}

func assertMakeBidsOutput(t *testing.T, filename string, bidderResponse *adapters.BidderResponse, expected []expectedBid) {
	t.Helper()

	if (bidderResponse == nil || len(bidderResponse.Bids) == 0) != (len(expected) == 0) {
		if len(expected) == 0 {
			t.Fatalf("%s: expectedBidResponses indicated a nil response, but mockResponses supplied a non-nil response", filename)
		}

		t.Fatalf("%s: mockResponses included unexpected nil or empty response", filename)
	}

	// Expected nil response - give diffBids something to work with.
	if bidderResponse == nil {
		bidderResponse = new(adapters.BidderResponse)
	}

	if len(bidderResponse.Bids) != len(expected) {
		t.Fatalf("%s: MakeBids returned wrong bid count. Expected %d, got %d", filename, len(expected), len(bidderResponse.Bids))
	}
	for i := 0; i < len(bidderResponse.Bids); i++ {
		diffBids(t, fmt.Sprintf("%s:  typedBid[%d]", filename, i), bidderResponse.Bids[i], &(expected[i]))
	}
}

// diffHttpRequests compares the actual HTTP request data to the expected one.
// It assumes that the request bodies are JSON
func diffHttpRequests(t *testing.T, description string, actual *adapters.RequestData, expected *httpRequest) {
	if actual == nil {
		t.Errorf("Bidders cannot return nil HTTP calls. %s was nil.", description)
		return
	}

	diffStrings(t, fmt.Sprintf("%s.uri", description), actual.Uri, expected.Uri)
	if expected.Headers != nil {
		actualHeader, _ := json.Marshal(actual.Headers)
		expectedHeader, _ := json.Marshal(expected.Headers)
		diffJson(t, description, actualHeader, expectedHeader)
	}
	diffJson(t, description, actual.Body, expected.Body)
}

func diffBids(t *testing.T, description string, actual *adapters.TypedBid, expected *expectedBid) {
	if actual == nil {
		t.Errorf("Bidders cannot return nil TypedBids. %s was nil.", description)
		return
	}

	diffStrings(t, fmt.Sprintf("%s.type", description), string(actual.BidType), string(expected.Type))
	diffOrtbBids(t, fmt.Sprintf("%s.bid", description), actual.Bid, expected.Bid)
}

// diffOrtbBids compares the actual Bid made by the adapter to the expectation from the JSON file.
func diffOrtbBids(t *testing.T, description string, actual *openrtb2.Bid, expected json.RawMessage) {
	if actual == nil {
		t.Errorf("Bidders cannot return nil Bids. %s was nil.", description)
		return
	}

	actualJson, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("%s failed to marshal actual Bid into JSON. %v", description, err)
	}

	diffJson(t, description, actualJson, expected)
}

func diffStrings(t *testing.T, description string, actual string, expected string) {
	if actual != expected {
		t.Errorf(`%s "%s" does not match expected "%s."`, description, actual, expected)
	}
}

// diffJson compares two JSON byte arrays for structural equality. It will produce an error if either
// byte array is not actually JSON.
func diffJson(t *testing.T, description string, actual []byte, expected []byte) {
	if len(actual) == 0 && len(expected) == 0 {
		return
	}
	if len(actual) == 0 || len(expected) == 0 {
		t.Fatalf("%s json diff failed. Expected %d bytes in body, but got %d.", description, len(expected), len(actual))
	}
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

// testMakeRequestsImpl asserts the resulting values of the bidder's `MakeRequests()` implementation
// against the expected JSON-defined results and ensures we do not encounter data races in the process.
// To assert no data races happen we make use of:
//  1) A shallow copy of the unmarshalled openrtb2.BidRequest that will provide reference values to
//     shared memory that we don't want the adapters' implementation of `MakeRequests()` to modify.
//  2) A deep copy that will preserve the original values of all the fields. This copy remains untouched
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

		bidsErrs = append(bidsErrs, theseErrs...)
		bidResponses = append(bidResponses, thisBidResponse)
	}

	// Assert actual errors thrown by MakeBids implementation versus expected JSON-defined spec.MakeBidsErrors
	assertErrorList(t, fmt.Sprintf("%s: MakeBids", filename), bidsErrs, spec.MakeBidsErrors)

	// Assert MakeBids implementation BidResponses with expected JSON-defined spec.BidResponses[i].Bids
	for i := 0; i < len(spec.BidResponses); i++ {
		assertMakeBidsOutput(t, filename, bidResponses[i], spec.BidResponses[i].Bids)
	}
}
