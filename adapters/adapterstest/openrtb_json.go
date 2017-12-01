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
// adapters/{bidder}/{bidder}test/supplementary:
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
	runTests(t, fmt.Sprintf("%s/supplementary", rootDir), bidder, true)
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
	bidRequestDifferences := diffBidRequests(&actualReqData, &(expected.Body))
	if len(bidRequestDifferences) > 0 {
		t.Errorf("%s did not match expectations: %v", description, bidRequestDifferences)
	}
}

func diffBidRequests(actual *openrtb.BidRequest, expected *openrtb.BidRequest) (differences []string) {
	differences = diffStrings("request.id", actual.ID, expected.ID, differences)
	differences = diffImpLists("request.imp", actual.Imp, expected.Imp, differences)
	differences = diffSites("request.site", actual.Site, expected.Site, differences)
	differences = diffApps("request.app", actual.App, expected.App, differences)
	differences = diffDevices("request.device", actual.Device, expected.Device, differences)
	differences = diffUsers("request.user", actual.User, expected.User, differences)
	differences = diffInts("request.test", int(actual.Test), int(expected.Test), differences)
	differences = diffInts("request.at", int(actual.AT), int(expected.AT), differences)

	return
}

func diffImpLists(description string, actual []openrtb.Imp, expected []openrtb.Imp, differences []string) []string {
	if len(actual) != len(expected) {
		return append(differences, fmt.Sprintf(`%s expected %d elements, but got %d.`, description, len(expected), len(actual)))
	}
	for i := 0; i < len(actual); i++ {
		differences = diffImps(fmt.Sprintf("%s[%d]", description, i), actual[i], expected[i], differences)
	}
	return differences
}

func diffImps(description string, actual openrtb.Imp, expected openrtb.Imp, differences []string) []string {
	differences = diffStrings(fmt.Sprintf("%s.id", description), actual.ID, expected.ID, differences)
	differences = diffBanners(fmt.Sprintf("%s.banner", description), actual.Banner, expected.Banner, differences)
	differences = diffStrings(fmt.Sprintf("%s.tagid", description), actual.TagID, expected.TagID, differences)
	differences = diffFloats(fmt.Sprintf("%s.bidfloor", description), actual.BidFloor, expected.BidFloor, differences)
	differences = diffJson(fmt.Sprintf("%s.ext", description), actual.Ext, expected.Ext, differences)
	return differences
}

func diffSites(description string, actual *openrtb.Site, expected *openrtb.Site, differences []string) []string {
	return differences
}

func diffDevices(description string, actual *openrtb.Device, expected *openrtb.Device, differences []string) []string {
	return differences
}

func diffUsers(description string, actual *openrtb.User, expected *openrtb.User, differences []string) []string {
	return differences
}

func diffApps(description string, actual *openrtb.App, expected *openrtb.App, differences []string) []string {
	return differences
}

func diffBanners(description string, actual *openrtb.Banner, expected *openrtb.Banner, differences []string) []string {
	differences = diffFormatLists(fmt.Sprintf("%s.format", description), actual.Format, expected.Format, differences)
	differences = diffAdPos(fmt.Sprintf("%s.pos", description), actual.Pos, expected.Pos, differences)
	return differences
}

func diffFormatLists(description string, actual []openrtb.Format, expected []openrtb.Format, differences []string) []string {
	if len(actual) != len(expected) {
		return append(differences, fmt.Sprintf(`%s expected %d elements, but got %d.`, description, len(expected), len(actual)))
	}
	for i := 0; i < len(actual); i++ {
		differences = diffFormats(fmt.Sprintf("%s[%d]", description, i), actual[i], expected[i], differences)
	}
	return differences
}

func diffFormats(description string, actual openrtb.Format, expected openrtb.Format, differences []string) []string {
	differences = diffInts(fmt.Sprintf("%s.w", description), int(actual.W), int(actual.W), differences)
	differences = diffInts(fmt.Sprintf("%s.h", description), int(actual.H), int(actual.H), differences)
	return differences
}

func diffAdPos(description string, actual *openrtb.AdPosition, expected *openrtb.AdPosition, differences []string) []string {
	if (actual == nil) != (expected == nil) {
		return append(differences, fmt.Sprintf("%s expects nil: %t, got nil: %t.", description, expected == nil, actual == nil))
	}
	if actual == nil || expected == nil {
		return differences
	}
	differences = diffInts(description, int(*actual), int(*expected), differences)
	return differences
}

func diffJson(description string, actual openrtb.RawJSON, expected openrtb.RawJSON, differences []string) []string {
	var parsedActual interface{}
	var parsedExpected interface{}
	json.Unmarshal(actual, &parsedActual)
	json.Unmarshal(expected, &parsedExpected)

	if !reflect.DeepEqual(parsedActual, parsedExpected) {
		return append(differences, fmt.Sprintf(`%s JSON does not match. Expected: %v, Actual: %v.`, description, parsedExpected, parsedActual))
	}
	return differences
}

func diffBids(t *testing.T, index int, actual *adapters.TypedBid, expected *expectedBid) (differences []string) {
	differences = diffStrings("typedBid.type", string(actual.BidType), string(expected.Type), differences)
	differences = diffOrtbBids("typedBid.bid", actual.Bid, expected.Bid, differences)
	return differences
}

func diffTypedBids(actual *adapters.TypedBid, expected *adapters.TypedBid) (differences []string) {
	differences = diffStrings("typedBid.type", string(actual.BidType), string(expected.BidType), differences)
	differences = diffOrtbBids("typedBid.bid", actual.Bid, expected.Bid, differences)
	return differences
}

func diffOrtbBids(description string, actual *openrtb.Bid, expected *openrtb.Bid, differences []string) []string {
	differences = diffStrings(fmt.Sprintf("%s.id expected %s, but got %s", description, expected.ID, actual.ID), actual.ID, expected.ID, differences)
	differences = diffStrings(fmt.Sprintf("%s.impid expected %s, but got %s", description, expected.ImpID, actual.ImpID), actual.ImpID, expected.ImpID, differences)
	differences = diffFloats(fmt.Sprintf("%s.price expected %f, but got %f", description, expected.Price, actual.Price), actual.Price, expected.Price, differences)
	differences = diffStrings(fmt.Sprintf("%s.adm expected %s, but got %s", description, expected.AdM, actual.AdM), actual.AdM, expected.AdM, differences)
	differences = diffStrings(fmt.Sprintf("%s.adid expected %s, but got %s", description, expected.AdID, actual.AdID), actual.AdID, expected.AdID, differences)
	differences = diffStringLists(fmt.Sprintf("%s.adomain", description), actual.ADomain, expected.ADomain, differences)
	differences = diffStrings(fmt.Sprintf("%s.iurl", description), actual.IURL, expected.IURL, differences)
	differences = diffStrings(fmt.Sprintf("%s.cid", description), actual.CID, expected.CID, differences)
	differences = diffStrings(fmt.Sprintf("%s.crid", description), actual.CrID, expected.CrID, differences)
	differences = diffInts(fmt.Sprintf("%s.w", description), int(actual.W), int(expected.W), differences)
	differences = diffInts(fmt.Sprintf("%s.h", description), int(actual.H), int(expected.H), differences)
	differences = diffJson(fmt.Sprintf("%s.ext", description), actual.Ext, expected.Ext, differences)
	return differences
}

func diffStringLists(description string, actual []string, expected []string, differences []string) []string {
	if len(actual) != len(expected) {
		return append(differences, fmt.Sprintf(`%s expected %d elements, but got %d.`, description, len(expected), len(actual)))
	}
	for i := 0; i < len(actual); i++ {
		differences = diffStrings(fmt.Sprintf("%s[%d]", description, i), actual[i], expected[i], differences)
	}
	return differences
}

func diffStrings(description string, actual string, expected string, differences []string) []string {
	if actual == expected {
		return differences
	} else {
		return append(differences, fmt.Sprintf(`%s "%s" does not match expected "%s."`, description, actual, expected))
	}
}

func diffInts(description string, actual int, expected int, differences []string) []string {
	if actual == expected {
		return differences
	} else {
		return append(differences, fmt.Sprintf(`%s "%d" does not match expected "%d."`, description, actual, expected))
	}
}

func diffFloats(description string, actual float64, expected float64, differences []string) []string {
	if actual == expected {
		return differences
	} else {
		return append(differences, fmt.Sprintf(`%s "%f" does not match expected "%f."`, description, actual, expected))
	}
}
