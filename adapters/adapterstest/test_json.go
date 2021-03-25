package adapterstest

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"testing"

	"github.com/mxmCherry/openrtb"
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

	actualReqs := testMakeRequestsImpl(t, filename, spec, bidder, &reqInfo)

	testMakeBidsImpl(t, filename, spec, bidder, actualReqs)
}

type testSpec struct {
	BidRequest        openrtb.BidRequest      `json:"mockBidRequest"`
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

func assertMakeBidsOutput(t *testing.T, filename string, actualBidderResp *adapters.BidderResponse, expected []expectedBid) {
	t.Helper()

	if (actualBidderResp == nil || len(actualBidderResp.Bids) == 0) != (len(expected) == 0) {
		if len(expected) == 0 {
			t.Fatalf("%s: expectedBidResponses indicated a nil response, but mockResponses supplied a non-nil response", filename)
		}

		t.Fatalf("%s: mockResponses included unexpected nil or empty response", filename)
	}

	// Expected nil response - give diffBids something to work with.
	if actualBidderResp == nil {
		actualBidderResp = new(adapters.BidderResponse)
	}

	actualBids := actualBidderResp.Bids

	if len(actualBids) != len(expected) {
		t.Fatalf("%s: MakeBids returned wrong bid count. Expected %d, got %d", filename, len(expected), len(actualBids))
	}
	for i := 0; i < len(actualBids); i++ {
		diffBids(t, fmt.Sprintf("%s:  typedBid[%d]", filename, i), actualBids[i], &(expected[i]))
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

// testMakeBidsImpl asserts the results of the bidder MakeRequests implementation against the expected JSON-defined results
// and makes sure no data races occur
func testMakeRequestsImpl(t *testing.T, filename string, spec *testSpec, bidder adapters.Bidder, reqInfo *adapters.ExtraRequestInfo) []*adapters.RequestData {
	t.Helper()

	deepBidReqCopy, shallowBidReqCopy, err := prepareDataRaceCopies(&spec.BidRequest)
	if err != nil {
		t.Errorf("Could not create data race test objects. File: %s Error: %v", filename, err)
		return nil
	}

	// Run MakeRequests
	actualReqs, errs := bidder.MakeRequests(&spec.BidRequest, reqInfo)

	// Compare MakeRequests actual output versus expected values found in JSON file
	assertErrorList(t, fmt.Sprintf("%s: MakeRequests", filename), errs, spec.MakeRequestErrors)
	assertMakeRequestsOutput(t, filename, actualReqs, spec.HttpCalls)

	// Assert no data races occur using original bidRequest copies of references and values
	assertNoDataRace(t, deepBidReqCopy, shallowBidReqCopy, filename)

	return actualReqs
}

func prepareDataRaceCopies(original *openrtb.BidRequest) (*openrtb.BidRequest, *openrtb.BidRequest, error) {

	// Save original bidRequest values to assert no data races occur inside MakeRequests latter
	deepReqCopy, err := deepCopyBidRequest(original)
	if err != nil {
		return nil, nil, err
	}

	// Mocks the shallow copy PBS core provides to adapters
	shallowReqCopy := *original

	// PBS core provides adapters a shallow copy of []Imp elements
	shallowReqCopy.Imp = nil
	for i := 0; i < len(original.Imp); i++ {
		shallowImpCopy := original.Imp[i]
		shallowReqCopy.Imp = append(shallowReqCopy.Imp, shallowImpCopy)
	}

	return deepReqCopy, &shallowReqCopy, nil
}

// deepCopyBidRequest is our own implementation of a deep copy function custom made for an openrtb.BidRequest object
func deepCopyBidRequest(original *openrtb.BidRequest) (*openrtb.BidRequest, error) {
	bytes, err := json.Marshal(original)
	if err != nil {
		return nil, fmt.Errorf("Unable to marshal original bid request: %v", err)
	}

	deepCopy := &openrtb.BidRequest{}
	err = json.Unmarshal(bytes, deepCopy)
	if err != nil {
		return nil, fmt.Errorf("Unable to unmarshal original bid request: %v", err)
	}

	// Make sure all Ext fields remain perfectly equal after the Marshal and unmarshal calls
	deepCopy = deepCopySliceAndExtAdjustments(deepCopy, original)

	return deepCopy, nil
}

// deepCopySliceAndExtAdjustments is necessary in order to not get false positives when a json
// entry initializes an empty array (such as "format": []) or the shallow copy `Ext` fields differ
// in line breaks or tabs with the deepCopy ones
func deepCopySliceAndExtAdjustments(deepCopy *openrtb.BidRequest, original *openrtb.BidRequest) *openrtb.BidRequest {
	if len(deepCopy.Ext) > 0 {
		deepCopy.Ext = make([]byte, len(original.Ext))
		copy(deepCopy.Ext, original.Ext)
	}

	if deepCopy.Site != nil {
		if len(deepCopy.Site.Ext) > 0 {
			deepCopy.Site.Ext = make([]byte, len(original.Site.Ext))
			copy(deepCopy.Site.Ext, original.Site.Ext)
		}
		if original.Site.Cat != nil && len(original.Site.Cat) == 0 {
			deepCopy.Site.Cat = []string{}
		}
		if original.Site.SectionCat != nil && len(original.Site.SectionCat) == 0 {
			deepCopy.Site.SectionCat = []string{}
		}
		if original.Site.PageCat != nil && len(original.Site.PageCat) == 0 {
			deepCopy.Site.PageCat = []string{}
		}
	}

	if deepCopy.App != nil {
		if len(deepCopy.App.Ext) > 0 {
			deepCopy.App.Ext = make([]byte, len(original.App.Ext))
			copy(deepCopy.App.Ext, original.App.Ext)
		}
		if original.App.Cat != nil && len(original.App.Cat) == 0 {
			deepCopy.App.Cat = []string{}
		}
		if original.App.SectionCat != nil && len(original.App.SectionCat) == 0 {
			deepCopy.App.SectionCat = []string{}
		}
		if original.App.PageCat != nil && len(original.App.PageCat) == 0 {
			deepCopy.App.PageCat = []string{}
		}
	}

	if deepCopy.Device != nil && len(deepCopy.Device.Ext) > 0 {
		deepCopy.Device.Ext = make([]byte, len(original.Device.Ext))
		copy(deepCopy.Device.Ext, original.Device.Ext)
	}

	if deepCopy.User != nil {
		if len(deepCopy.User.Ext) > 0 {
			deepCopy.User.Ext = make([]byte, len(original.User.Ext))
			copy(deepCopy.User.Ext, original.User.Ext)
		}
		if original.User.Data != nil && len(original.User.Data) == 0 {
			original.User.Data = []openrtb.Data{}
		}
	}

	if deepCopy.Source != nil && len(deepCopy.Source.Ext) > 0 {
		deepCopy.Source.Ext = make([]byte, len(original.Source.Ext))
		copy(deepCopy.Source.Ext, original.Source.Ext)
	}

	if deepCopy.Regs != nil && len(deepCopy.Regs.Ext) > 0 {
		deepCopy.Regs.Ext = make([]byte, len(original.Regs.Ext))
		copy(deepCopy.Regs.Ext, original.Regs.Ext)
	}

	for i, imp := range deepCopy.Imp {
		if len(imp.Ext) > 0 {
			imp.Ext = make([]byte, len(original.Imp[i].Ext))
			copy(imp.Ext, original.Imp[i].Ext)
		}
		if imp.Banner != nil {
			if len(imp.Banner.Ext) > 0 {
				imp.Banner.Ext = make([]byte, len(original.Imp[i].Banner.Ext))
				copy(imp.Banner.Ext, original.Imp[i].Banner.Ext)
			}
			if original.Imp[i].Banner.Format != nil && len(original.Imp[i].Banner.Format) == 0 {
				imp.Banner.Format = []openrtb.Format{}
			}
			if original.Imp[i].Banner.BType != nil && len(original.Imp[i].Banner.BType) == 0 {
				imp.Banner.BType = []openrtb.BannerAdType{}
			}
			if original.Imp[i].Banner.BAttr != nil && len(original.Imp[i].Banner.BAttr) == 0 {
				imp.Banner.BAttr = []openrtb.CreativeAttribute{}
			}
			if original.Imp[i].Banner.ExpDir != nil && len(original.Imp[i].Banner.ExpDir) == 0 {
				imp.Banner.ExpDir = []openrtb.ExpandableDirection{}
			}
			if original.Imp[i].Banner.API != nil && len(original.Imp[i].Banner.API) == 0 {
				imp.Banner.API = []openrtb.APIFramework{}
			}
		}
		if imp.Video != nil {
			if len(imp.Video.Ext) > 0 {
				imp.Video.Ext = make([]byte, len(original.Imp[i].Video.Ext))
				copy(imp.Video.Ext, original.Imp[i].Video.Ext)
			}
			if original.Imp[i].Video.MIMEs != nil && len(original.Imp[i].Video.MIMEs) == 0 {
				imp.Video.MIMEs = []string{}
			}
			if original.Imp[i].Video.Protocols != nil && len(original.Imp[i].Video.Protocols) == 0 {
				imp.Video.Protocols = []openrtb.Protocol{}
			}
			if original.Imp[i].Video.BAttr != nil && len(original.Imp[i].Video.BAttr) == 0 {
				imp.Video.BAttr = []openrtb.CreativeAttribute{}
			}
			if original.Imp[i].Video.PlaybackMethod != nil && len(original.Imp[i].Video.PlaybackMethod) == 0 {
				imp.Video.PlaybackMethod = []openrtb.PlaybackMethod{}
			}
			if original.Imp[i].Video.Delivery != nil && len(original.Imp[i].Video.Delivery) == 0 {
				imp.Video.Delivery = []openrtb.ContentDeliveryMethod{}
			}
			if original.Imp[i].Video.CompanionAd != nil && len(original.Imp[i].Video.CompanionAd) == 0 {
				imp.Video.CompanionAd = []openrtb.Banner{}
			}
			if original.Imp[i].Video.API != nil && len(original.Imp[i].Video.API) == 0 {
				imp.Video.API = []openrtb.APIFramework{}
			}
			if original.Imp[i].Video.CompanionType != nil && len(original.Imp[i].Video.CompanionType) == 0 {
				imp.Video.CompanionType = []openrtb.CompanionType{}
			}
		}
		if imp.Audio != nil {
			if len(imp.Audio.Ext) > 0 {
				imp.Audio.Ext = make([]byte, len(original.Imp[i].Audio.Ext))
				copy(imp.Audio.Ext, original.Imp[i].Audio.Ext)
			}
			if original.Imp[i].Audio.MIMEs != nil && len(original.Imp[i].Audio.MIMEs) == 0 {
				imp.Audio.MIMEs = []string{}
			}
			if original.Imp[i].Audio.Protocols != nil && len(original.Imp[i].Audio.Protocols) == 0 {
				imp.Audio.Protocols = []openrtb.Protocol{}
			}
			if original.Imp[i].Audio.BAttr != nil && len(original.Imp[i].Audio.BAttr) == 0 {
				imp.Audio.BAttr = []openrtb.CreativeAttribute{}
			}
			if original.Imp[i].Audio.Delivery != nil && len(original.Imp[i].Audio.Delivery) == 0 {
				imp.Audio.Delivery = []openrtb.ContentDeliveryMethod{}
			}
			if original.Imp[i].Audio.CompanionAd != nil && len(original.Imp[i].Audio.CompanionAd) == 0 {
				imp.Audio.CompanionAd = []openrtb.Banner{}
			}
			if original.Imp[i].Audio.API != nil && len(original.Imp[i].Audio.API) == 0 {
				imp.Audio.API = []openrtb.APIFramework{}
			}
			if original.Imp[i].Audio.CompanionType != nil && len(original.Imp[i].Audio.CompanionType) == 0 {
				imp.Audio.CompanionType = []openrtb.CompanionType{}
			}
		}
		if imp.Native != nil {
			if len(imp.Native.Ext) > 0 {
				imp.Native.Ext = make([]byte, len(original.Imp[i].Native.Ext))
				copy(imp.Native.Ext, original.Imp[i].Native.Ext)
			}
			if original.Imp[i].Native.API != nil && len(original.Imp[i].Native.API) == 0 {
				imp.Native.API = []openrtb.APIFramework{}
			}
			if original.Imp[i].Native.BAttr != nil && len(original.Imp[i].Native.BAttr) == 0 {
				imp.Native.BAttr = []openrtb.CreativeAttribute{}
			}
		}
		if imp.PMP != nil {
			if len(imp.PMP.Ext) > 0 {
				imp.PMP.Ext = make([]byte, len(original.Imp[i].PMP.Ext))
				copy(imp.PMP.Ext, original.Imp[i].PMP.Ext)
			}
			if original.Imp[i].PMP.Deals != nil && len(original.Imp[i].PMP.Deals) == 0 {
				imp.PMP.Deals = []openrtb.Deal{}
			}
		}

		if len(imp.Metric) > 0 {
			for j, metric := range imp.Metric {
				if len(metric.Ext) > 0 {
					metric.Ext = make([]byte, len(original.Imp[i].Metric[j].Ext))
					copy(metric.Ext, original.Imp[i].Metric[j].Ext)
				}
			}
		}
	}
	return deepCopy
}

// assertNoDataRace compares the contents of the reference fields found in the original openrtb.BidRequest to their
// original values to make sure they were not modified and we are not incurring indata races. In order to assert
// no data races occur in the []Imp array, we call assertNoImpsDataRace()
func assertNoDataRace(t *testing.T, bidRequestBefore *openrtb.BidRequest, bidRequestAfter *openrtb.BidRequest, filename string) {
	t.Helper()

	// Assert reference fields were not modified by bidder adapter MakeRequests implementation
	assert.Equal(t, bidRequestBefore.Site, bidRequestAfter.Site, "Data race in BidRequest.Site field in file %s", filename)
	assert.Equal(t, bidRequestBefore.App, bidRequestAfter.App, "Data race in BidRequest.App field in file %s", filename)
	assert.Equal(t, bidRequestBefore.Device, bidRequestAfter.Device, "Data race in BidRequest.Device field in file %s", filename)
	assert.Equal(t, bidRequestBefore.User, bidRequestAfter.User, "Data race in BidRequest.User field in file %s", filename)
	assert.Equal(t, bidRequestBefore.Source, bidRequestAfter.Source, "Data race in BidRequest.Source field in file %s", filename)
	assert.Equal(t, bidRequestBefore.Regs, bidRequestAfter.Regs, "Data race in BidRequest.Regs field in file %s", filename)

	// Assert Imps separately
	assertNoImpsDataRace(t, bidRequestBefore.Imp, bidRequestAfter.Imp, filename)
}

// assertNoImpsDataRace compares the contents of the reference fields found in the original openrtb.Imp objects to
// their original values to make sure they were not modified and we are not incurring in data races.
func assertNoImpsDataRace(t *testing.T, impsBefore []openrtb.Imp, impsAfter []openrtb.Imp, filename string) {
	t.Helper()

	if assert.Len(t, impsAfter, len(impsBefore), "Original []Imp array was modified and length is not equal to original after MakeRequests was called. File:%s", filename) {
		// Assert no data races occured in individual Imp elements
		for i := 0; i < len(impsBefore); i++ {
			assert.Equal(t, impsBefore[i].Banner, impsAfter[i].Banner, "Data race in bidRequest.Imp[%d].Banner field. File:%s", i, filename)
			assert.Equal(t, impsBefore[i].Video, impsAfter[i].Video, "Data race in bidRequest.Imp[%d].Video field. File:%s", i, filename)
			assert.Equal(t, impsBefore[i].Audio, impsAfter[i].Audio, "Data race in bidRequest.Imp[%d].Audio field. File:%s", i, filename)
			assert.Equal(t, impsBefore[i].Native, impsAfter[i].Native, "Data race in bidRequest.Imp[%d].Native field. File:%s", i, filename)
			assert.Equal(t, impsBefore[i].PMP, impsAfter[i].PMP, "Data race in bidRequest.Imp[%d].PMP field. File:%s", i, filename)
			assert.Equal(t, impsBefore[i].Secure, impsAfter[i].Secure, "Data race in bidRequest.Imp[%d].Secure field. File:%s", i, filename)
			assert.ElementsMatch(t, impsBefore[i].Metric, impsAfter[i].Metric, "Data race in bidRequest.Imp[%d].[]Metric array. File:%s", i)
		}
	}
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
