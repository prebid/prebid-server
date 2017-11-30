package adapterstest

import (
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
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
//    These tests are tedious to write, but very valuable to verify correct auction behavior.
//
// 2. In the future, we plan to auto-generate documentation from the "ideal" JSON requests.
//    By generating sample docs and running tests on the same code, this will teach publishers
//    how to use your Bidder properly, which should encourage adoption.
//
// To use this method, create the following folders:
//
// adapters/{bidder}/{bidder}test/ideal:
//   Fill this with *.json files which show "perfect" bid requests for your adapter. Your adapter should return
//   no errors on these requests, and you should expect them to serve as public documentation in the future.
//
// adapters/{bidder}/{bidder}test/invalid:
//   Fill this with *.json files which describe *unsupported* bid requests for your adapter. Your adapter should
//   return errors on these requests *without* making any external calls. For example, a request with a Video imp
//   would go here if your adapter doesn't support Video bids.
//
//
// TODO: It may be worth adding a "flawed" directory here, with requests which the Bidder can _handle_, but
// not necessarily ideal. For example: testing a Request with one Imp and one Video bid which gets sent to
// a Bidder which does not support Video. I'll implement this if people like the overall strategy in the PR.
func TestOpenRTB(t *testing.T, rootDir string, bidder adapters.Bidder) {
	t.Helper()

	if idealReqs, err := ioutil.ReadDir(fmt.Sprintf("%s/ideal", rootDir)); err == nil {
		for _, idealReq := range idealReqs {
			TestUsefulRequest(t, fmt.Sprintf("%s/ideal/%s", rootDir, idealReq.Name()), bidder)
		}
	}

	if invalidReqs, err := ioutil.ReadDir(fmt.Sprintf("%s/invalid", rootDir)); err == nil {
		for _, invalidReq := range invalidReqs {
			TestUselessRequest(t, fmt.Sprintf("%s/invalid/%s", rootDir, invalidReq.Name()), bidder)
		}
	}
}

// This method implements the adapters/{bidder}/{bidder}test/ideal tests.
// It expects the Bidder to return no errors.
func TestUsefulRequest(t *testing.T, specPath string, bidder adapters.Bidder) {
	t.Helper()

	specData, err := ioutil.ReadFile(specPath)
	if err != nil {
		t.Fatalf("Failed to read spec file contents.")
	}

	var spec ortbSpec
	if err := json.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("Failed to unmarshal spec file: %v", err)
	}

	actualReqs, errs := bidder.MakeRequests(&spec.BidRequest)
	if len(errs) > 0 {
		t.Errorf("The sample bid request should not have produced an error. Got %v", errs)
	}

	if len(actualReqs) != len(spec.HttpCalls) {
		t.Fatalf("Bidder did not make the expected number of HTTP calls. Expected %d, got %d", len(spec.HttpCalls), len(actualReqs))
	}

	var bids = make([]*adapters.TypedBid, 0, len(spec.Bids))
	for i, expectedCall := range spec.HttpCalls {
		actualReq := actualReqs[i]
		if actualReq == nil {
			t.Errorf("The adpater should never send back a nil request.")
		} else {
			if expectedCall.Request.Uri != actualReq.Uri {
				t.Errorf("HTTP request %d had the wrong Uri. Expected %s, got %s", i, expectedCall.Request.Uri, actualReq.Uri)
			}

			var actualReqData openrtb.BidRequest
			if err := json.Unmarshal(actualReq.Body, &actualReqData); err != nil {
				t.Fatalf("json.httpCalls.request.body unmarshalling failed: %v", err)
			}
			bidRequestDifferences := diffBidRequests(&actualReqData, &(expectedCall.Request.Body))
			if len(bidRequestDifferences) > 0 {
				t.Errorf("HTTP request %d did not match expectations: %v", i, bidRequestDifferences)
			}
		}

		actualBids, bidErrs := bidder.MakeBids(&spec.BidRequest, expectedCall.Response.ToResponseData(t))
		if len(bidErrs) > 0 {
			t.Fatalf("The adapter shouldn't return errors when generating bids. Got: %v", bidErrs)
		}
		bids = append(bids, actualBids...)
	}

	if len(bids) != len(spec.Bids) {
		t.Fatalf("Bidder did not make the expected number of bids. Expected %d, got %d", len(spec.Bids), len(bids))
	}

	for i, bid := range bids {
		expectedBid := spec.Bids[i].ToTypedBid()

		bidDifferences := diffTypedBids(bid, expectedBid)
		if len(bidDifferences) > 0 {
			t.Errorf("Bids did not match expectations: %v", bidDifferences)
		}
	}
}

// This method implements the adapters/{bidder}/{bidder}test/invalid tests.
// It expects the Bidder to return at least one error, and no HTTP calls.
func TestUselessRequest(t *testing.T, specPath string, bidder adapters.Bidder) {
	specData, err := ioutil.ReadFile(specPath)
	if err != nil {
		t.Fatalf("Failed to read spec file contents.")
	}

	var spec ortbSpec
	if err := json.Unmarshal(specData, &spec); err != nil {
		t.Fatalf("Failed to unmarshal spec file: %v", err)
	}

	actualReqs, errs := bidder.MakeRequests(&spec.BidRequest)
	if len(actualReqs) > 0 {
		t.Errorf("A useless request should make no HTTP calls. Got %v", len(actualReqs))
	}
	if len(errs) == 0 {
		t.Errorf("A useless request should return at least one error. Got 0.")
	}
}

type ortbSpec struct {
	BidRequest openrtb.BidRequest `json:"mockBidRequest"`
	HttpCalls  []httpCall         `json:"httpCalls"`
	Bids       []expectedBid      `json:"expectedBids"`
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

func (bid *expectedBid) ToTypedBid() *adapters.TypedBid {
	return &adapters.TypedBid{
		Bid:     bid.Bid,
		BidType: openrtb_ext.BidType(bid.Type),
	}
}

// ---------------------------------------
// TODO: Lots of ugly, repetitive, boilerplate code down here. Am requestin general impressions in a PR before being thorough.
//
// We can't use reflect.DeepEquals because the `Ext` on each OpenRTB type is a byte array which we really want
// to compare *as JSON. Unfortunately, recursive equality bloats into lots of manual code.
//
// It's not terrible, though... This lets us produce much more useful error messages on broken tests.

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

func diffTypedBids(actual *adapters.TypedBid, expected *adapters.TypedBid) (differences []string) {
	differences = diffStrings("typedBid.type", string(actual.BidType), string(expected.BidType), differences)
	differences = diffBids("typedBid.bid", actual.Bid, expected.Bid, differences)
	return differences
}

func diffBids(description string, actual *openrtb.Bid, expected *openrtb.Bid, differences []string) []string {
	differences = diffStrings(fmt.Sprintf("%s.id expected %s, but got %s", description, expected.ID, actual.ID), actual.ID, expected.ID, differences)
	differences = diffStrings(fmt.Sprintf("%s.impid expected %s, but got %s", description, expected.ImpID, actual.ImpID), actual.ImpID, expected.ImpID, differences)
	differences = diffFloats(fmt.Sprintf("%s.price expected %s, but got %s", description, expected.Price, actual.Price), actual.Price, expected.Price, differences)
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
		return append(differences, fmt.Sprintf(`%s "%d" does not match expected "%d."`, description, actual, expected))
	}
}