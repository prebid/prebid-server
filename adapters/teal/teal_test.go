package teal

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testEndpoint  = "https://test.example.com/bid"
	testAccount   = "test-account"
	testRequestID = "test-request-banner"
)

// End-to-end MakeRequests/MakeBids behavior is exercised by the JSON fixtures
// under tealtest/exemplary and tealtest/supplemental. The Go tests below cover
// only what fixtures cannot: the Go error-TYPE contract (BadInput vs
// BadServerResponse), short-circuit edges, and error branches of unexported
// helpers that are unreachable through MakeRequests.

func TestJsonSamples(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderTeal, config.Adapter{Endpoint: testEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	require.NoError(t, err, "Builder returned unexpected error")
	adapterstest.RunJSONBidderTest(t, "tealtest", bidder)
}

// TestErrorClassification pins the error-type contract, which the JSON fixtures
// cannot verify (they compare error messages only): MakeRequests parse /
// validation / account-divergence errors are *errortypes.BadInput; MakeBids
// transport errors are BadInput for 4xx and *errortypes.BadServerResponse for
// 5xx and unparseable bodies.
func TestErrorClassification(t *testing.T) {
	bidder := newTealBidder(t)

	t.Run("imp.ext parse error is BadInput", func(t *testing.T) {
		imp := openrtb2.Imp{
			ID:     "impId",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
			Ext:    json.RawMessage(`[]`),
		}
		requests, errs := bidder.MakeRequests(givenBidRequest(imp), &adapters.ExtraRequestInfo{})
		assert.Nil(t, requests)
		require.Len(t, errs, 1)
		assert.True(t, isBadInputErr(errs[0]), "got %T", errs[0])
		assert.Contains(t, errs[0].Error(), "Error parsing imp.ext for impression impId")
	})

	t.Run("imp.ext bidder field malformed is BadInput", func(t *testing.T) {
		imp := givenBannerImp("impId", json.RawMessage(`{"bidder":[]}`))
		requests, errs := bidder.MakeRequests(givenBidRequest(imp), &adapters.ExtraRequestInfo{})
		assert.Nil(t, requests)
		require.Len(t, errs, 1)
		assert.True(t, isBadInputErr(errs[0]), "got %T", errs[0])
		assert.Contains(t, errs[0].Error(), "Error parsing imp.ext for impression impId")
	})

	t.Run("blank account is BadInput", func(t *testing.T) {
		imp := givenBannerImp("imp1", givenImpExt(t, "   ", strPtr("p")))
		requests, errs := bidder.MakeRequests(givenBidRequest(imp), &adapters.ExtraRequestInfo{})
		assert.Nil(t, requests)
		require.Len(t, errs, 1)
		assert.True(t, isBadInputErr(errs[0]), "got %T", errs[0])
		assert.Equal(t, msgAccountValidation, errs[0].Error())
	})

	t.Run("blank placement is BadInput", func(t *testing.T) {
		imp := givenBannerImp("imp1", givenImpExt(t, "acct", strPtr("  ")))
		requests, errs := bidder.MakeRequests(givenBidRequest(imp), &adapters.ExtraRequestInfo{})
		assert.Nil(t, requests)
		require.Len(t, errs, 1)
		assert.True(t, isBadInputErr(errs[0]), "got %T", errs[0])
		assert.Equal(t, msgPlacementValidation, errs[0].Error())
	})

	t.Run("divergent account is BadInput", func(t *testing.T) {
		first := givenBannerImp("imp1", givenImpExt(t, "acct-a", strPtr("p1")))
		second := givenBannerImp("imp2", givenImpExt(t, "acct-b", strPtr("p2")))
		requests, errs := bidder.MakeRequests(givenBidRequest(first, second), &adapters.ExtraRequestInfo{})
		require.Len(t, requests, 1, "first imp still ships")
		require.Len(t, errs, 1)
		assert.True(t, isBadInputErr(errs[0]), "got %T", errs[0])
		assert.Contains(t, errs[0].Error(), "mixed-account requests are not supported")
	})

	t.Run("malformed request.ext propagates parse error", func(t *testing.T) {
		imp := givenBannerImp("imp1", givenImpExt(t, testAccount, nil))
		req := givenBidRequest(imp)
		req.Ext = json.RawMessage(`{"prebid":}`) // syntactically invalid
		requests, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})
		assert.Nil(t, requests)
		require.NotEmpty(t, errs)
		assert.Contains(t, errs[len(errs)-1].Error(), "failed parsing request.ext")
	})

	t.Run("MakeBids 4xx is BadInput", func(t *testing.T) {
		resp, errs := makeBidsCall(t, bidder, &openrtb2.BidRequest{}, http.StatusBadRequest, []byte(`{}`))
		assert.Nil(t, resp)
		require.Len(t, errs, 1)
		assert.True(t, isBadInputErr(errs[0]), "got %T", errs[0])
	})

	t.Run("MakeBids 5xx is BadServerResponse", func(t *testing.T) {
		resp, errs := makeBidsCall(t, bidder, &openrtb2.BidRequest{}, http.StatusInternalServerError, []byte(`{}`))
		assert.Nil(t, resp)
		require.Len(t, errs, 1)
		assert.True(t, isBadServerErr(errs[0]), "got %T", errs[0])
	})

	t.Run("MakeBids unparseable body is BadServerResponse", func(t *testing.T) {
		resp, errs := makeBidsCall(t, bidder, &openrtb2.BidRequest{}, http.StatusOK, []byte("not-json"))
		assert.Nil(t, resp)
		require.Len(t, errs, 1)
		assert.True(t, isBadServerErr(errs[0]), "got %T", errs[0])
	})
}

// TestMakeRequests_NoImps — empty Imp slice returns (nil, nil) before any work
// happens.
func TestMakeRequests_NoImps(t *testing.T) {
	bidder := newTealBidder(t)
	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{ID: "req"}, &adapters.ExtraRequestInfo{})
	assert.Nil(t, requests)
	assert.Nil(t, errs)
}

// TestGetBidType_Undeterminable — getBidType returns an explicit error (not a
// silent banner default) when the imp is missing or declares no media type, so
// MakeBids can skip the bid and surface the issue in logs.
func TestGetBidType_Undeterminable(t *testing.T) {
	bid := &openrtb2.Bid{ImpID: "imp1"}

	t.Run("imp not found", func(t *testing.T) {
		_, err := getBidType(bid, map[string]openrtb2.Imp{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `failed to determine bid type for imp "imp1"`)
	})

	t.Run("imp has no media type", func(t *testing.T) {
		imp := openrtb2.Imp{ID: "imp1"}
		_, err := getBidType(bid, map[string]openrtb2.Imp{imp.ID: imp})
		require.Error(t, err)
		assert.Contains(t, err.Error(), `failed to determine bid type for imp "imp1"`)
	})
}

// TestModifyImp_MalformedExt — modifyImp's parse-error branch. Unreachable via
// MakeRequests (parseImpExt rejects non-object imp.ext first), so it cannot be
// a fixture.
func TestModifyImp_MalformedExt(t *testing.T) {
	imp := &openrtb2.Imp{ID: "broken", Ext: json.RawMessage(`not-json`)}
	got, err := modifyImp(imp, strPtr("p"))
	assert.Nil(t, got)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Error parsing imp.ext for impression broken")
}

// TestDecodeOrEmptyObject — the non-object branch is unreachable via
// MakeRequests (parseImpExt rejects those shapes first); the returned map must
// never be nil so callers can assign into it.
func TestDecodeOrEmptyObject(t *testing.T) {
	t.Run("empty raw returns empty map", func(t *testing.T) {
		out := decodeOrEmptyObject(nil)
		assert.NotNil(t, out)
		assert.Empty(t, out)
	})

	t.Run("non-object input returns empty map", func(t *testing.T) {
		out := decodeOrEmptyObject(json.RawMessage(`[1,2,3]`))
		assert.NotNil(t, out)
		assert.Empty(t, out)
	})

	t.Run("valid object input returns populated map", func(t *testing.T) {
		out := decodeOrEmptyObject(json.RawMessage(`{"a":1,"b":"two"}`))
		assert.Len(t, out, 2)
		assert.JSONEq(t, `1`, string(out["a"]))
		assert.JSONEq(t, `"two"`, string(out["b"]))
	})
}

// TestMergeBidsPBSFlag — pins the null-handling contract: a JSON literal `null`
// (or absent) ext must route through decodeJSONObject so the receiver map is
// non-nil before `ext["bids"] = ...`, which would otherwise panic with
// "assignment to entry in nil map".
func TestMergeBidsPBSFlag(t *testing.T) {
	t.Run("null ext treated as empty object", func(t *testing.T) {
		out, err := mergeBidsPBSFlag(json.RawMessage(`null`))
		require.NoError(t, err)
		require.NotNil(t, out)

		var decoded map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(out, &decoded))
		assert.Len(t, decoded, 1, "null input should produce just the bids stamp")
		assert.JSONEq(t, `{"pbs":1}`, string(decoded["bids"]))
	})

	t.Run("malformed ext returns wrapped error", func(t *testing.T) {
		out, err := mergeBidsPBSFlag(json.RawMessage(`{"bids":`))
		assert.Nil(t, out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed parsing request.ext")
	})
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// strPtr returns a pointer to s. Used for the *string Placement field.
func strPtr(s string) *string {
	return &s
}

// newTealBidder builds the adapter via the canonical Builder.
func newTealBidder(t *testing.T) adapters.Bidder {
	t.Helper()
	bidder, err := Builder(openrtb_ext.BidderTeal, config.Adapter{Endpoint: testEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	require.NoError(t, err)
	require.NotNil(t, bidder)
	return bidder
}

// givenImpExt produces the imp.ext shape Teal expects: {"bidder": ExtImpTeal}.
func givenImpExt(t *testing.T, account string, placement *string) json.RawMessage {
	t.Helper()
	inner := openrtb_ext.ExtImpTeal{Account: account, Placement: placement}
	innerJSON, err := json.Marshal(inner)
	require.NoError(t, err)
	wrapper := map[string]json.RawMessage{"bidder": innerJSON}
	out, err := json.Marshal(wrapper)
	require.NoError(t, err)
	return out
}

// givenBannerImp returns a single banner imp with the supplied id and ext.
func givenBannerImp(id string, ext json.RawMessage) openrtb2.Imp {
	return openrtb2.Imp{
		ID:     id,
		Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		Ext:    ext,
	}
}

// givenBidRequest returns a minimal site-rooted BidRequest with the given imps.
func givenBidRequest(imps ...openrtb2.Imp) *openrtb2.BidRequest {
	return &openrtb2.BidRequest{
		ID:   testRequestID,
		Imp:  imps,
		Site: &openrtb2.Site{ID: "demo-site", Publisher: &openrtb2.Publisher{ID: "demo-publisher"}},
	}
}

// makeBidsCall wires a ResponseData and invokes MakeBids.
func makeBidsCall(t *testing.T, bidder adapters.Bidder, request *openrtb2.BidRequest, status int, body []byte) (*adapters.BidderResponse, []error) {
	t.Helper()
	return bidder.MakeBids(request, &adapters.RequestData{Method: http.MethodPost, Uri: testEndpoint},
		&adapters.ResponseData{StatusCode: status, Body: body})
}

// isBadInputErr returns true if err is or wraps an *errortypes.BadInput.
func isBadInputErr(err error) bool {
	var target *errortypes.BadInput
	return errors.As(err, &target)
}

// isBadServerErr returns true if err is or wraps an *errortypes.BadServerResponse.
func isBadServerErr(err error) bool {
	var target *errortypes.BadServerResponse
	return errors.As(err, &target)
}
