package adapterstest

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"io/ioutil"
	"reflect"
	"testing"
)

// TestOpenRTB is a helper method intended for Bidders which use OpenRTB to communicate with their servers.
//
// We _strongly_ recommend that all Bidders write tests using this, for the following reasons:
//
// 1. This includes some basic tests which confirm that your Bidder is "well-behaved" for all the input samples.
//    For example, "no nil bids are allowed in the returned array".
//    These tests are tedious to write, but help prevent bugs during auctions.
//
// 2. In the future, we plan to auto-generate documentation from the "exemplary" test files.
//    By using this structure, those docs will teach publishers how to use your Bidder, which should encourage adoption.
//
// To use this method, create *.json files in the following directories:
//
// adapters/{bidder}/{bidder}test/exemplary:
//
//   These show "ideal" BidRequests for your Bidder. If possible, configure your servers to return the same
//   expected responses forever. In the future, we plan to auto-generate Publisher-facing docs from these examples.
//   If your server responds appropriately, we can guarantee Publishers that your adapter works as documented.
//
// adapters/{bidder}/{bidder}test/supplemental:
//
//   Fill this with *.json files which are useful test cases, but are not appropriate for public example docs.
//   For example, a file in this directory might make sure that a mobile-only Bidderreturns errors on non-mobile requests.
//
// Then create a test in your adapters/{bidder}/{bidder}_test.go file like so:
//
//   func TestJsonSamples(t *testing.T) {
//     adapterstest.TestOpenRTB(t, "{bidder}test", someBidderInstance)
//   }
//
func TestOpenRTB(t *testing.T, rootDir string, bidder adapters.Bidder) {
	runTests(t, fmt.Sprintf("%s/exemplary", rootDir), bidder, false)
	runTests(t, fmt.Sprintf("%s/supplemental", rootDir), bidder, true)
}

// runTests runs all the *.json files in a directory. If allowErrors is false, and one of the test files
// expects errors from the bidder, then the test will fail.
func runTests(t *testing.T, directory string, bidder adapters.Bidder, allowErrors bool) {
	if specFiles, err := ioutil.ReadDir(directory); err == nil {
		for _, specFile := range specFiles {
			fileName := fmt.Sprintf("%s/%s", directory, specFile.Name())
			specData, err := loadFile(fileName)
			if err != nil {
				t.Fatalf("Failed to load contents of file %s: %v", fileName, err)
			}

			if allowErrors != specData.expectsErrors() {
				t.Fatalf("Exemplary spec %s must not expect errors.", fileName)
			}
			runSpec(t, fileName, specData, bidder)
		}
	}
}

// LoadFile reads and parses a file as a test case. If something goes wrong, it returns an error.
func loadFile(filename string) (*ortbSpec, error) {
	specData, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file %s: %v", filename, err)
	}

	var spec ortbSpec
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
func runSpec(t *testing.T, filename string, spec *ortbSpec, bidder adapters.Bidder) {
	actualReqs, errs := bidder.MakeRequests(&spec.BidRequest)
	compareErrorLists(t, fmt.Sprintf("%s: MakeRequests", filename), errs, spec.MakeRequestErrors)
	compareHttpRequestLists(t, filename, actualReqs, spec.HttpCalls)

	var bids = make([]*adapters.TypedBid, 0, len(spec.Bids))
	var bidsErrs = make([]error, 0, len(spec.MakeBidsErrors))
	for i := 0; i < len(actualReqs); i++ {
		theseBids, theseErrs := bidder.MakeBids(&spec.BidRequest, spec.HttpCalls[i].Response.ToResponseData(t))
		bids = append(bids, theseBids...)
		bidsErrs = append(bidsErrs, theseErrs...)
	}

	compareErrorLists(t, fmt.Sprintf("%s: MakeBids", filename), bidsErrs, spec.MakeBidsErrors)
	compareBidLists(t, filename, bids, spec.Bids)
}

func compareErrorLists(t *testing.T, description string, actual []error, expected []string) {
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

func compareHttpRequestLists(t *testing.T, filename string, actual []*adapters.RequestData, expected []httpCall) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("%s: MakeRequests had wrong request count. Expected %d, got %d", filename, len(expected), len(actual))
	}
	for i := 0; i < len(actual); i++ {
		diffHttpRequests(t, fmt.Sprintf("%s: httpRequest[%d]", filename, i), actual[i], &(expected[i].Request))
	}
}

func compareBidLists(t *testing.T, filename string, actual []*adapters.TypedBid, expected []expectedBid) {
	t.Helper()

	if len(actual) != len(expected) {
		t.Fatalf("%s: MakeBids returned wrong bid count. Expected %d, got %d", filename, len(expected), len(actual))
	}
	for i := 0; i < len(actual); i++ {
		diffBids(t, i, actual[i], &(expected[i]))
	}
}

type ortbSpec struct {
	BidRequest        openrtb.BidRequest `json:"mockBidRequest"`
	HttpCalls         []httpCall         `json:"httpCalls"`
	Bids              []expectedBid      `json:"expectedBids"`
	MakeRequestErrors []string           `json:"expectedMakeRequestsErrors"`
	MakeBidsErrors    []string           `json:"expectedMakeBidsErrors"`
}

func (spec *ortbSpec) expectsErrors() bool {
	return len(spec.MakeRequestErrors) > 0 || len(spec.MakeBidsErrors) > 0
}

type httpCall struct {
	Request  httpRequest  `json:"expectedRequest"`
	Response httpResponse `json:"mockResponse"`
}

type httpRequest struct {
	Body openrtb.BidRequest `json:"body"`
	Uri  string             `json:"uri"`
}

type httpResponse struct {
	Status int                 `json:"status"`
	Body   openrtb.BidResponse `json:"body"`
}

func (resp *httpResponse) ToResponseData(t *testing.T) *adapters.ResponseData {
	t.Helper()

	bodyBytes, err := json.Marshal(resp.Body)
	if err != nil {
		t.Fatalf("Failed to marshal httpResponse.Body")
	}
	return &adapters.ResponseData{
		StatusCode: resp.Status,
		Body:       bodyBytes,
	}
}

type expectedBid struct {
	Bid  *openrtb.Bid `json:"bid"`
	Type string       `json:"type"`
}

// ---------------------------------------
// TODO: Lots of ugly, repetitive, boilerplate code down here. Am requesting general impressions in a PR before being thorough.
//
// We can't use reflect.DeepEquals because the `Ext` on each OpenRTB type is a byte array which we really want
// to compare *as JSON. Unfortunately, recursive equality bloats into lots of manual code.
//
// It's not terrible, though... This lets us produce much more useful error messages on broken tests.

func diffHttpRequests(t *testing.T, description string, actual *adapters.RequestData, expected *httpRequest) {
	if actual == nil {
		t.Errorf("%s should not be nil.", description)
		return
	}

	if expected.Uri != actual.Uri {
		t.Errorf("%s had wrong Uri. Expected %s, got %s", description, expected.Uri, actual.Uri)
	}

	var actualReqData openrtb.BidRequest
	if err := json.Unmarshal(actual.Body, &actualReqData); err != nil {
		t.Fatalf("%s unmarshalling failed. Does your Bidder send an OpenRTB BidRequest? %v", description, err)
	}

	diffBidRequests(t, &actualReqData, &(expected.Body))
}

func diffBidRequests(t *testing.T, actual *openrtb.BidRequest, expected *openrtb.BidRequest) {
	diffStrings(t, "request.id", actual.ID, expected.ID)
	diffImpLists(t, "request.imp", actual.Imp, expected.Imp)
	diffSites(t, "request.site", actual.Site, expected.Site)
	diffApps(t, "request.app", actual.App, expected.App)
	diffDevices(t, "request.device", actual.Device, expected.Device)
	diffUsers(t, "request.user", actual.User, expected.User)
	diffInts(t, "request.test", int(actual.Test), int(expected.Test))
	diffInts(t, "request.at", int(actual.AT), int(expected.AT))

	return
}

func diffImpLists(t *testing.T, description string, actual []openrtb.Imp, expected []openrtb.Imp) {
	if len(actual) != len(expected) {
		t.Errorf(`%s expected %d elements, but got %d.`, description, len(expected), len(actual))
	}
	for i := 0; i < len(actual); i++ {
		diffImps(t, fmt.Sprintf("%s[%d]", description, i), actual[i], expected[i])
	}
}

func diffImps(t *testing.T, description string, actual openrtb.Imp, expected openrtb.Imp) {
	diffStrings(t, fmt.Sprintf("%s.id", description), actual.ID, expected.ID)
	diffBanners(t, fmt.Sprintf("%s.banner", description), actual.Banner, expected.Banner)
	diffStrings(t, fmt.Sprintf("%s.tagid", description), actual.TagID, expected.TagID)
	diffFloats(t, fmt.Sprintf("%s.bidfloor", description), actual.BidFloor, expected.BidFloor)
	diffJson(t, fmt.Sprintf("%s.ext", description), actual.Ext, expected.Ext)
}

func diffSites(t *testing.T, description string, actual *openrtb.Site, expected *openrtb.Site) {
}

func diffDevices(t *testing.T, description string, actual *openrtb.Device, expected *openrtb.Device) {
}

func diffUsers(t *testing.T, description string, actual *openrtb.User, expected *openrtb.User) {
}

func diffApps(t *testing.T, description string, actual *openrtb.App, expected *openrtb.App) {
}

func diffBanners(t *testing.T, description string, actual *openrtb.Banner, expected *openrtb.Banner) {
	diffFormatLists(t, fmt.Sprintf("%s.format", description), actual.Format, expected.Format)
	diffAdPos(t, fmt.Sprintf("%s.pos", description), actual.Pos, expected.Pos)
}

func diffFormatLists(t *testing.T, description string, actual []openrtb.Format, expected []openrtb.Format) {
	if len(actual) != len(expected) {
		t.Errorf(`%s expected %d elements, but got %d.`, description, len(expected), len(actual))
	}
	for i := 0; i < len(actual); i++ {
		diffFormats(t, fmt.Sprintf("%s[%d]", description, i), actual[i], expected[i])
	}
}

func diffFormats(t *testing.T, description string, actual openrtb.Format, expected openrtb.Format) {
	diffInts(t, fmt.Sprintf("%s.w", description), int(actual.W), int(actual.W))
	diffInts(t, fmt.Sprintf("%s.h", description), int(actual.H), int(actual.H))
}

func diffAdPos(t *testing.T, description string, actual *openrtb.AdPosition, expected *openrtb.AdPosition) {
	if (actual == nil) != (expected == nil) {
		t.Errorf("%s expects nil: %t, got nil: %t.", description, expected == nil, actual == nil)
	}
	if actual == nil || expected == nil {
		return
	}
	diffInts(t, description, int(*actual), int(*expected))
}

func diffJson(t *testing.T, description string, actual openrtb.RawJSON, expected openrtb.RawJSON) {
	var parsedActual interface{}
	var parsedExpected interface{}
	json.Unmarshal(actual, &parsedActual)
	json.Unmarshal(expected, &parsedExpected)

	if !reflect.DeepEqual(parsedActual, parsedExpected) {
		t.Errorf(`%s JSON does not match. Expected: %v, Actual: %v.`, description, parsedExpected, parsedActual)
	}
}

func diffBids(t *testing.T, index int, actual *adapters.TypedBid, expected *expectedBid) {
	diffStrings(t, "typedBid.type", string(actual.BidType), string(expected.Type))
	diffOrtbBids(t, "typedBid.bid", actual.Bid, expected.Bid)
}

func diffTypedBids(t *testing.T, actual *adapters.TypedBid, expected *adapters.TypedBid) {
	diffStrings(t, "typedBid.type", string(actual.BidType), string(expected.BidType))
	diffOrtbBids(t, "typedBid.bid", actual.Bid, expected.Bid)
}

func diffOrtbBids(t *testing.T, description string, actual *openrtb.Bid, expected *openrtb.Bid) {
	diffStrings(t, fmt.Sprintf("%s.id expected %s, but got %s", description, expected.ID, actual.ID), actual.ID, expected.ID)
	diffStrings(t, fmt.Sprintf("%s.impid expected %s, but got %s", description, expected.ImpID, actual.ImpID), actual.ImpID, expected.ImpID)
	diffFloats(t, fmt.Sprintf("%s.price expected %f, but got %f", description, expected.Price, actual.Price), actual.Price, expected.Price)
	diffStrings(t, fmt.Sprintf("%s.adm expected %s, but got %s", description, expected.AdM, actual.AdM), actual.AdM, expected.AdM)
	diffStrings(t, fmt.Sprintf("%s.adid expected %s, but got %s", description, expected.AdID, actual.AdID), actual.AdID, expected.AdID)
	diffStringLists(t, fmt.Sprintf("%s.adomain", description), actual.ADomain, expected.ADomain)
	diffStrings(t, fmt.Sprintf("%s.iurl", description), actual.IURL, expected.IURL)
	diffStrings(t, fmt.Sprintf("%s.cid", description), actual.CID, expected.CID)
	diffStrings(t, fmt.Sprintf("%s.crid", description), actual.CrID, expected.CrID)
	diffInts(t, fmt.Sprintf("%s.w", description), int(actual.W), int(expected.W))
	diffInts(t, fmt.Sprintf("%s.h", description), int(actual.H), int(expected.H))
	diffJson(t, fmt.Sprintf("%s.ext", description), actual.Ext, expected.Ext)
}

func diffStringLists(t *testing.T, description string, actual []string, expected []string) {
	if len(actual) != len(expected) {
		t.Errorf(`%s expected %d elements, but got %d.`, description, len(expected), len(actual))
		return
	}
	for i := 0; i < len(actual); i++ {
		diffStrings(t, fmt.Sprintf("%s[%d]", description, i), actual[i], expected[i])
	}
}

func diffStrings(t *testing.T, description string, actual string, expected string) {
	if actual != expected {
		t.Errorf(`%s "%s" does not match expected "%s."`, description, actual, expected)
	}
}

func diffInts(t *testing.T, description string, actual int, expected int) {
	if actual != expected {
		t.Errorf(`%s "%d" does not match expected "%d."`, description, actual, expected)
	}
}

func diffFloats(t *testing.T, description string, actual float64, expected float64) {
	if actual != expected {
		t.Errorf(`%s "%f" does not match expected "%f."`, description, actual, expected)
	}
}
