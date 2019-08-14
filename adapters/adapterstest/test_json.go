package adapterstest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
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
	actualReqs, errs := bidder.MakeRequests(&spec.BidRequest, &reqInfo)
	diffErrorLists(t, fmt.Sprintf("%s: MakeRequests", filename), errs, spec.MakeRequestErrors)
	diffHttpRequestLists(t, filename, actualReqs, spec.HttpCalls)

	bidResponses := make([]*adapters.BidderResponse, 0)

	var bidsErrs = make([]error, 0, len(spec.MakeBidsErrors))
	for i := 0; i < len(actualReqs); i++ {
		thisBidResponse, theseErrs := bidder.MakeBids(&spec.BidRequest, spec.HttpCalls[i].Request.ToRequestData(t), spec.HttpCalls[i].Response.ToResponseData(t))
		bidsErrs = append(bidsErrs, theseErrs...)
		bidResponses = append(bidResponses, thisBidResponse)
	}

	diffErrorLists(t, fmt.Sprintf("%s: MakeBids", filename), bidsErrs, spec.MakeBidsErrors)

	for i := 0; i < len(spec.BidResponses); i++ {
		diffBidLists(t, filename, bidResponses[i].Bids, spec.BidResponses[i].Bids)
	}
}

type testSpec struct {
	BidRequest        openrtb.BidRequest    `json:"mockBidRequest"`
	HttpCalls         []httpCall            `json:"httpCalls"`
	BidResponses      []expectedBidResponse `json:"expectedBidResponses"`
	MakeRequestErrors []string              `json:"expectedMakeRequestsErrors"`
	MakeBidsErrors    []string              `json:"expectedMakeBidsErrors"`
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
	Status int             `json:"status"`
	Body   json.RawMessage `json:"body"`
}

func (resp *httpResponse) ToResponseData(t *testing.T) *adapters.ResponseData {
	return &adapters.ResponseData{
		StatusCode: resp.Status,
		Body:       resp.Body,
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

// diffHttpRequests compares the actual http requests to the expected ones.
func diffHttpRequestLists(t *testing.T, filename string, actual []*adapters.RequestData, expected []httpCall) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("%s: MakeRequests had wrong request count. Expected %d, got %d", filename, len(expected), len(actual))
	}
	for i := 0; i < len(actual); i++ {
		diffHttpRequests(t, fmt.Sprintf("%s: httpRequest[%d]", filename, i), actual[i], &(expected[i].Request))
	}
}

func diffErrorLists(t *testing.T, description string, actual []error, expected []string) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("%s had wrong error count. Expected %d, got %d", description, len(expected), len(actual))
	}
	for i := 0; i < len(actual); i++ {
		if expected[i] != actual[i].Error() {
			t.Errorf(`%s error[%d] had wrong message. Expected "%s", got "%s"`, description, i, expected[i], actual[i].Error())
		}
	}
}

func diffBidLists(t *testing.T, filename string, actual []*adapters.TypedBid, expected []expectedBid) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf("%s: MakeBids returned wrong bid count. Expected %d, got %d", filename, len(expected), len(actual))
	}
	for i := 0; i < len(actual); i++ {
		diffBids(t, fmt.Sprintf("%s:  typedBid[%d]", filename, i), actual[i], &(expected[i]))
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
func diffOrtbBids(t *testing.T, description string, actual *openrtb.Bid, expected json.RawMessage) {
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
			t.Fatalf("%s json did not match, but unmarhsalling failed. %v", description, err)
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
