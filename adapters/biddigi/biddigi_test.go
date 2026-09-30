package biddigi

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

const testsBidderEndpoint = "https://biddigi-auction-service.biddigi25.workers.dev/inbound/openrtb2/auction"

func buildTestBidder(t *testing.T) adapters.Bidder {
	t.Helper()
	bidder, buildErr := Builder(openrtb_ext.BidderBiddigi, config.Adapter{
		Endpoint: testsBidderEndpoint}, config.Server{})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
	return bidder
}

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "biddigitest", buildTestBidder(t))
}

func TestNoContentResponse(t *testing.T) {
	bidResponse, errs := buildTestBidder(t).MakeBids(nil, nil, &adapters.ResponseData{StatusCode: 204})
	assert.Nil(t, bidResponse)
	assert.Empty(t, errs)
}

// The credential must reach BidDigi as a header and must not survive into the marshalled body.
// Asserted directly rather than only through the JSON suite, because a regression here leaks a
// secret into every debug dump and stored auction record downstream.
func TestSeatKeyTravelsAsHeaderAndNotInBody(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID:  "req-1",
		Imp: []openrtb2.Imp{{ID: "imp-1", Ext: jsonRaw(`{"bidder":{"seatKey":"bds_secret","placementId":"p1"}}`)}},
	}

	reqs, errs := buildTestBidder(t).MakeRequests(request, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	assert.Len(t, reqs, 1)

	assert.Equal(t, "Bearer bds_secret", reqs[0].Headers.Get("Authorization"))
	assert.NotContains(t, string(reqs[0].Body), "bds_secret", "the seat key must never appear in the request body")
	assert.NotContains(t, string(reqs[0].Body), "seatKey")
	assert.Contains(t, string(reqs[0].Body), `"placementId":"p1"`)
	assert.Equal(t, []string{"imp-1"}, reqs[0].ImpIDs)
}

// One HTTP request carries one Authorization header, so imps authenticating as different seats
// must be split rather than silently sent under whichever key happened to be first.
func TestImpsAreGroupedBySeatKey(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{
			{ID: "imp-b", Ext: jsonRaw(`{"bidder":{"seatKey":"key-b"}}`)},
			{ID: "imp-a", Ext: jsonRaw(`{"bidder":{"seatKey":"key-a"}}`)},
			{ID: "imp-b2", Ext: jsonRaw(`{"bidder":{"seatKey":"key-b"}}`)},
		},
	}

	reqs, errs := buildTestBidder(t).MakeRequests(request, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	assert.Len(t, reqs, 2)

	// Deterministic order: sorted by key, so "key-a" precedes "key-b".
	assert.Equal(t, "Bearer key-a", reqs[0].Headers.Get("Authorization"))
	assert.Equal(t, []string{"imp-a"}, reqs[0].ImpIDs)
	assert.Equal(t, "Bearer key-b", reqs[1].Headers.Get("Authorization"))
	assert.Equal(t, []string{"imp-b", "imp-b2"}, reqs[1].ImpIDs)
}

// A bad imp must not take the rest of the request down with it.
func TestBadImpIsReportedAndOthersSurvive(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{
			{ID: "imp-bad", Ext: jsonRaw(`{"bidder":{}}`)},
			{ID: "imp-good", Ext: jsonRaw(`{"bidder":{"seatKey":"key-a"}}`)},
		},
	}

	reqs, errs := buildTestBidder(t).MakeRequests(request, &adapters.ExtraRequestInfo{})
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "seatKey")
	assert.Len(t, reqs, 1)
	assert.Equal(t, []string{"imp-good"}, reqs[0].ImpIDs)
}

// ext.prebid and friends belong to PBS and other modules; an adapter that flattens imp.ext breaks
// features it has nothing to do with.
func TestOtherImpExtFieldsArePreserved(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID:  "req-1",
		Imp: []openrtb2.Imp{{ID: "imp-1", Ext: jsonRaw(`{"bidder":{"seatKey":"k"},"gpid":"/1/home","data":{"x":1}}`)}},
	}

	reqs, errs := buildTestBidder(t).MakeRequests(request, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	assert.Len(t, reqs, 1)

	var sent openrtb2.BidRequest
	assert.NoError(t, jsonutil.Unmarshal(reqs[0].Body, &sent))
	assert.JSONEq(t, `{"gpid":"/1/home","data":{"x":1}}`, string(sent.Imp[0].Ext))
}

// With no placementId and nothing else in imp.ext, an empty object would be noise on the wire.
func TestEmptyImpExtIsDropped(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID:  "req-1",
		Imp: []openrtb2.Imp{{ID: "imp-1", Ext: jsonRaw(`{"bidder":{"seatKey":"k"}}`)}},
	}

	reqs, errs := buildTestBidder(t).MakeRequests(request, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)

	var sent openrtb2.BidRequest
	assert.NoError(t, jsonutil.Unmarshal(reqs[0].Body, &sent))
	assert.Nil(t, sent.Imp[0].Ext)
}

// PBS hands every adapter the same *BidRequest. Writing Imp on it would corrupt what other
// bidders see.
func TestCallerRequestIsNotMutated(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{
			{ID: "imp-a", Ext: jsonRaw(`{"bidder":{"seatKey":"key-a"}}`)},
			{ID: "imp-b", Ext: jsonRaw(`{"bidder":{"seatKey":"key-b"}}`)},
		},
	}

	_, errs := buildTestBidder(t).MakeRequests(request, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)

	assert.Len(t, request.Imp, 2, "the caller's imp slice must be untouched")
	assert.JSONEq(t, `{"bidder":{"seatKey":"key-a"}}`, string(request.Imp[0].Ext))
	assert.JSONEq(t, `{"bidder":{"seatKey":"key-b"}}`, string(request.Imp[1].Ext))
}

func TestGetMediaTypeForBid(t *testing.T) {
	bidType, err := getMediaTypeForBid(openrtb2.Bid{MType: openrtb2.MarkupBanner})
	assert.NoError(t, err)
	assert.Equal(t, openrtb_ext.BidTypeBanner, bidType)

	bidType, err = getMediaTypeForBid(openrtb2.Bid{MType: openrtb2.MarkupVideo})
	assert.NoError(t, err)
	assert.Equal(t, openrtb_ext.BidTypeVideo, bidType)

	bidType, err = getMediaTypeForBid(openrtb2.Bid{MType: openrtb2.MarkupNative})
	assert.NoError(t, err)
	assert.Equal(t, openrtb_ext.BidTypeNative, bidType)

	_, err = getMediaTypeForBid(openrtb2.Bid{ID: "no-mtype"})
	assert.Error(t, err)

	_, err = getMediaTypeForBid(openrtb2.Bid{ID: "audio", MType: openrtb2.MarkupAudio})
	assert.Error(t, err)
}

func jsonRaw(s string) []byte { return []byte(s) }
