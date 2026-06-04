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

// ---------------------------------------------------------------------------
// Constants & test helpers
// ---------------------------------------------------------------------------

const (
	testEndpoint    = "https://test.example.com/bid"
	testAccount     = "test-account"
	testImpBannerID = "test-imp-banner"
	testRequestID   = "test-request-banner"
)

// strPtr returns a pointer to s. Used for the *string Placement field.
func strPtr(s string) *string {
	return &s
}

// newTealBidder builds the adapter via the canonical Builder. Test helper to
// keep individual tests focused on behavior rather than wiring.
func newTealBidder(t *testing.T) adapters.Bidder {
	t.Helper()
	bidder, err := Builder(openrtb_ext.BidderTeal, config.Adapter{Endpoint: testEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	require.NoError(t, err)
	require.NotNil(t, bidder)
	return bidder
}

// givenImpExt produces the imp.ext shape Teal expects: {"bidder": ExtImpTeal}.
// When placement is nil it is omitted (mirrors omitempty on ExtImpTeal.Placement).
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

// makeBidsCall is a tiny helper that wires a ResponseData and invokes MakeBids.
func makeBidsCall(t *testing.T, bidder adapters.Bidder, request *openrtb2.BidRequest, status int, body []byte) (*adapters.BidderResponse, []error) {
	t.Helper()
	return bidder.MakeBids(request, &adapters.RequestData{Method: http.MethodPost, Uri: testEndpoint},
		&adapters.ResponseData{StatusCode: status, Body: body})
}

// ---------------------------------------------------------------------------
// JSON sample driver (kept first per convention)
//
// End-to-end MakeRequests/MakeBids behavior is exercised by the JSON fixtures
// under tealtest/exemplary and tealtest/supplemental. The Go tests below cover
// only what fixtures cannot: unexported-helper units, the Go error-TYPE contract
// (BadInput vs BadServerResponse), and the (nil request) short-circuit edges.
// ---------------------------------------------------------------------------

func TestJsonSamples(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderTeal, config.Adapter{Endpoint: testEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	require.NoError(t, err, "Builder returned unexpected error")
	adapterstest.RunJSONBidderTest(t, "tealtest", bidder)
}

// ---------------------------------------------------------------------------
// Builder
// ---------------------------------------------------------------------------

// TestBuilder mirrors Java TealBidderTest.creationShouldFailOnInvalidEndpointUrl.
// The Go port matches Java's HttpUtil.validateUrl semantics: empty, malformed,
// and non-absolute URLs are all rejected; only well-formed absolute URLs
// (scheme + host) succeed.
func TestBuilder(t *testing.T) {
	t.Run("empty endpoint rejected", func(t *testing.T) {
		bidder, err := Builder(openrtb_ext.BidderTeal, config.Adapter{Endpoint: ""}, config.Server{})
		assert.Error(t, err)
		assert.Nil(t, bidder)
		assert.Contains(t, err.Error(), "endpoint is required")
	})

	t.Run("valid endpoint succeeds", func(t *testing.T) {
		bidder, err := Builder(openrtb_ext.BidderTeal,
			config.Adapter{Endpoint: testEndpoint},
			config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
		require.NoError(t, err)
		require.NotNil(t, bidder)
	})

	t.Run("malformed endpoint rejected", func(t *testing.T) {
		// Mirrors Java's HttpUtil.validateUrl rejecting "invalid_url".
		bidder, err := Builder(openrtb_ext.BidderTeal, config.Adapter{Endpoint: "invalid_url"}, config.Server{})
		assert.Error(t, err)
		assert.Nil(t, bidder)
		assert.Contains(t, err.Error(), "invalid endpoint")
	})

	t.Run("relative URL rejected", func(t *testing.T) {
		// "/some/path" is parseable but lacks scheme + host, so it fails the
		// absolute-URL check that mirrors Java's HttpUtil.validateUrl.
		bidder, err := Builder(openrtb_ext.BidderTeal, config.Adapter{Endpoint: "/relative/path"}, config.Server{})
		assert.Error(t, err)
		assert.Nil(t, bidder)
		assert.Contains(t, err.Error(), "must be an absolute URL")
	})
}

// ---------------------------------------------------------------------------
// Error-type contract
//
// The JSON fixtures verify error MESSAGES but not Go error TYPES. This test
// pins the type contract in one place: MakeRequests parse/validation and
// account-divergence errors are *errortypes.BadInput; MakeBids transport errors
// are BadInput for 4xx and *errortypes.BadServerResponse for 5xx / unparseable
// bodies. Per-scenario coverage lives in the tealtest fixtures.
// ---------------------------------------------------------------------------

func TestErrorClassification(t *testing.T) {
	bidder := newTealBidder(t)

	t.Run("imp.ext parse error is BadInput", func(t *testing.T) {
		// imp.ext is a JSON array — fails the ExtImpBidder unmarshal.
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
		// Valid wrapper, invalid bidder shape — fails the second parseImpExt unmarshal.
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
		imp := givenBannerImp(testImpBannerID, givenImpExt(t, testAccount, nil))
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

// ---------------------------------------------------------------------------
// MakeRequests — (nil request) short-circuit edges (cannot be JSON fixtures)
// ---------------------------------------------------------------------------

// TestMakeRequests_NoImps — defensive: empty Imp slice returns (nil, nil)
// before any work happens.
func TestMakeRequests_NoImps(t *testing.T) {
	bidder := newTealBidder(t)
	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{ID: "req"}, &adapters.ExtraRequestInfo{})
	assert.Nil(t, requests)
	assert.Nil(t, errs)
}

// TestMakeRequests_AllImpsFailValidation — every imp fails: returns (nil, errs)
// without dispatching any HTTP request. Mirrors the empty-survivor short-circuit.
func TestMakeRequests_AllImpsFailValidation(t *testing.T) {
	bidder := newTealBidder(t)
	imp1 := givenBannerImp("imp1", givenImpExt(t, "", strPtr("placement"))) // blank account
	imp2 := givenBannerImp("imp2", givenImpExt(t, "acct", strPtr("")))      // blank placement

	requests, errs := bidder.MakeRequests(givenBidRequest(imp1, imp2), &adapters.ExtraRequestInfo{})

	assert.Nil(t, requests)
	require.Len(t, errs, 2)
	assert.Equal(t, msgAccountValidation, errs[0].Error())
	assert.Equal(t, msgPlacementValidation, errs[1].Error())
}

// ---------------------------------------------------------------------------
// getBidType — unexported helper units
// ---------------------------------------------------------------------------

// TestGetBidType_Priority — when an imp has BOTH banner and video set, the
// switch must return banner first (priority order matters for fidelity).
func TestGetBidType_Priority(t *testing.T) {
	imp := openrtb2.Imp{ID: "imp1", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}}
	bid := &openrtb2.Bid{ImpID: "imp1"}
	typ, err := getBidType(bid, map[string]openrtb2.Imp{imp.ID: imp})
	require.NoError(t, err)
	assert.Equal(t, openrtb_ext.BidTypeBanner, typ)
}

// TestGetBidType_Undeterminable — getBidType returns the reviewer-prescribed
// error (not a silent banner default) when the imp is missing or declares no
// media type, so MakeBids can skip the bid and surface the issue in logs.
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

// ---------------------------------------------------------------------------
// MakeBids — empty-response edge (empty-but-non-nil BidderResponse)
// ---------------------------------------------------------------------------

// TestMakeBids_EmptySeatBid — seatbid array empty → empty BidderResponse
// (no error, just no bids) with currency still propagated.
func TestMakeBids_EmptySeatBid(t *testing.T) {
	bidder := newTealBidder(t)
	body := buildBidResponseJSON(t, "USD", nil)
	resp, errs := makeBidsCall(t, bidder, &openrtb2.BidRequest{}, http.StatusOK, body)
	assert.Empty(t, errs)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Bids)
	assert.Equal(t, "USD", resp.Currency)
}

// ---------------------------------------------------------------------------
// Internal helper unit tests
// ---------------------------------------------------------------------------

// TestIsBlank covers the unicode whitespace branch table-driven.
func TestIsBlank(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", true},
		{" ", true},
		{"\t", true},
		{"\n", true},
		{"\n  \t", true},
		{"   \r\n\t", true},
		{" ", true}, // U+00A0 NO-BREAK SPACE — unicode.IsSpace returns true
		{" ", true}, // U+2003 EM SPACE
		{"abc", false},
		{" a ", false},
		{".", false},
		{"0", false},
	}
	for _, tc := range cases {
		t.Run("input="+tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, isBlank(tc.in))
		})
	}
}

// TestModifyImp_HandlesNonObjectPrebid — direct test of modifyImp's tolerance
// for non-object `prebid` and `prebid.storedrequest` values. Mirrors Java
// getOrCreate (which calls .createObjectNode() when the existing node is
// non-object). Note: the public MakeRequests path can never deliver these
// shapes because parseImpExt's strict ExtImpBidder unmarshal rejects them
// first — but defended-in-depth tolerance is preserved in modifyImp for
// fidelity with Java's TealBidder.modifyImp.
func TestModifyImp_HandlesNonObjectPrebid(t *testing.T) {
	cases := []struct {
		name string
		ext  json.RawMessage
	}{
		{"prebid is string",
			json.RawMessage(`{"bidder":{"account":"a","placement":"p"},"prebid":"foo"}`)},
		{"prebid is array",
			json.RawMessage(`{"bidder":{"account":"a","placement":"p"},"prebid":[1,2,3]}`)},
		{"prebid is number",
			json.RawMessage(`{"bidder":{"account":"a","placement":"p"},"prebid":42}`)},
		{"prebid.storedrequest is array",
			json.RawMessage(`{"bidder":{"account":"a","placement":"p"},"prebid":{"storedrequest":[1,2]}}`)},
		{"prebid.storedrequest is string",
			json.RawMessage(`{"bidder":{"account":"a","placement":"p"},"prebid":{"storedrequest":"oops"}}`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			imp := &openrtb2.Imp{ID: "imp1", Ext: tc.ext}
			got, err := modifyImp(imp, strPtr("p"))
			require.NoError(t, err, "modifyImp must tolerate non-object prebid/storedrequest")
			require.NotNil(t, got)
			assert.Equal(t, "p", readStoredRequestID(t, got.Ext),
				"storedrequest.id must be set even when existing prebid was non-object")
		})
	}
}

// TestModifyImp_PlacementNil_PassthroughByValue — when placement is nil,
// modifyImp must short-circuit and return the SAME pointer it received.
// This mirrors Java's `if (placement == null) return imp;` early-return.
func TestModifyImp_PlacementNil_PassthroughByValue(t *testing.T) {
	imp := &openrtb2.Imp{ID: "x", Ext: json.RawMessage(`{"bidder":{"account":"a"}}`)}
	got, err := modifyImp(imp, nil)
	assert.NoError(t, err)
	assert.Same(t, imp, got, "passthrough must return the same pointer when placement is nil")
}

// TestModifyImp_MalformedExt — modifyImp encounters non-object ext → returns
// the same wrapped parse error MakeRequests prefixes BadInput on.
func TestModifyImp_MalformedExt(t *testing.T) {
	imp := &openrtb2.Imp{ID: "broken", Ext: json.RawMessage(`not-json`)}
	got, err := modifyImp(imp, strPtr("p"))
	assert.Nil(t, got)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Error parsing imp.ext for impression broken")
}

// TestModifyImp_EmptyExt — no existing imp.ext at all → modifyImp seeds a
// fresh {"prebid":{"storedrequest":{"id":<placement>}}}.
func TestModifyImp_EmptyExt(t *testing.T) {
	imp := &openrtb2.Imp{ID: "fresh", Ext: nil}
	got, err := modifyImp(imp, strPtr("p"))
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "p", readStoredRequestID(t, got.Ext))
}

// TestModifyImp_NullImpExtHandledAsEmpty pins the null-handling contract for
// modifyImp (a site of the historical nil-map bug): JSON literal `null` imp.ext
// is treated as an empty object, so storedrequest injection still succeeds.
func TestModifyImp_NullImpExtHandledAsEmpty(t *testing.T) {
	placement := "test-placement"
	imp := &openrtb2.Imp{ID: "imp-1", Ext: json.RawMessage(`null`)}
	out, err := modifyImp(imp, &placement)
	require.NoError(t, err)
	require.NotNil(t, out)

	var ext map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(out.Ext, &ext))
	require.Contains(t, ext, "prebid")

	var prebid map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(ext["prebid"], &prebid))
	require.Contains(t, prebid, "storedrequest")

	var storedRequest map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(prebid["storedrequest"], &storedRequest))
	assert.JSONEq(t, `"test-placement"`, string(storedRequest["id"]))
}

// TestDecodeOrEmptyObject covers the three decode branches: empty input,
// non-object input, valid object input.
func TestDecodeOrEmptyObject(t *testing.T) {
	t.Run("empty raw returns empty map", func(t *testing.T) {
		out := decodeOrEmptyObject(nil)
		assert.NotNil(t, out)
		assert.Empty(t, out)
	})

	t.Run("array input returns empty map", func(t *testing.T) {
		out := decodeOrEmptyObject(json.RawMessage(`[1,2,3]`))
		assert.NotNil(t, out)
		assert.Empty(t, out)
	})

	t.Run("string input returns empty map", func(t *testing.T) {
		out := decodeOrEmptyObject(json.RawMessage(`"hello"`))
		assert.NotNil(t, out)
		assert.Empty(t, out)
	})

	t.Run("number input returns empty map", func(t *testing.T) {
		out := decodeOrEmptyObject(json.RawMessage(`42`))
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

// TestMergeBidsPBSFlag covers the no-existing / has-existing / overwrite
// branches plus the parse-error branch.
func TestMergeBidsPBSFlag(t *testing.T) {
	t.Run("no existing ext", func(t *testing.T) {
		out, err := mergeBidsPBSFlag(nil)
		require.NoError(t, err)

		var ext map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(out, &ext))
		assert.JSONEq(t, `{"pbs":1}`, string(ext["bids"]))
		assert.Len(t, ext, 1)
	})

	t.Run("preserves existing ext", func(t *testing.T) {
		out, err := mergeBidsPBSFlag(json.RawMessage(`{"foo":42}`))
		require.NoError(t, err)

		var ext map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(out, &ext))
		assert.JSONEq(t, `{"pbs":1}`, string(ext["bids"]))
		assert.JSONEq(t, `42`, string(ext["foo"]))
	})

	t.Run("overwrites existing bids key", func(t *testing.T) {
		out, err := mergeBidsPBSFlag(json.RawMessage(`{"bids":{"old":true}}`))
		require.NoError(t, err)

		var ext map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(out, &ext))
		assert.JSONEq(t, `{"pbs":1}`, string(ext["bids"]))
	})

	t.Run("malformed ext returns wrapped error", func(t *testing.T) {
		out, err := mergeBidsPBSFlag(json.RawMessage(`{"bids":`))
		assert.Nil(t, out)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed parsing request.ext")
	})
}

// TestMergeBidsPBSFlag_NullInputHandledAsEmpty verifies that JSON literal
// `null` is treated as an empty object (mirrors Java's
// ObjectUtils.defaultIfNull(request.getExt(), ExtRequest.empty()) pattern).
//
// Contract: mergeBidsPBSFlag must route a `null` (or absent) ext through
// decodeJSONObject so the receiver map is non-nil before `ext["bids"] = ...`,
// which would otherwise panic with "assignment to entry in nil map".
func TestMergeBidsPBSFlag_NullInputHandledAsEmpty(t *testing.T) {
	out, err := mergeBidsPBSFlag(json.RawMessage(`null`))
	require.NoError(t, err)
	require.NotNil(t, out)

	var decoded map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(out, &decoded))
	assert.Len(t, decoded, 1, "null input should produce just the bids stamp")
	assert.JSONEq(t, `{"pbs":1}`, string(decoded["bids"]))
}

// TestClonePublisherWithID — both branches: nil publisher → fresh; non-nil →
// copy with overwritten ID, original untouched.
func TestClonePublisherWithID(t *testing.T) {
	t.Run("nil publisher creates fresh", func(t *testing.T) {
		got := clonePublisherWithID(nil, "acct")
		require.NotNil(t, got)
		assert.Equal(t, "acct", got.ID)
	})

	t.Run("non-nil publisher copy with overwrite", func(t *testing.T) {
		orig := &openrtb2.Publisher{ID: "orig-id", Name: "orig-name"}
		got := clonePublisherWithID(orig, "acct")
		require.NotNil(t, got)
		assert.NotSame(t, orig, got, "must return a fresh struct, not the same pointer")
		assert.Equal(t, "acct", got.ID)
		assert.Equal(t, "orig-name", got.Name, "non-ID fields must be preserved")
		// Original untouched.
		assert.Equal(t, "orig-id", orig.ID)
	})
}

// TestStandardHeaders pins the exact header set Teal expects per Java.
func TestStandardHeaders(t *testing.T) {
	h := standardHeaders()
	assert.Equal(t, []string{"application/json;charset=utf-8"}, h["Content-Type"])
	assert.Equal(t, []string{"application/json"}, h["Accept"])
}

// ---------------------------------------------------------------------------
// Test helpers — local utilities for assertions
// ---------------------------------------------------------------------------

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

// readStoredRequestID extracts imp.ext.prebid.storedrequest.id as a string.
// Fails the test if the path is missing or not a string.
func readStoredRequestID(t *testing.T, impExt json.RawMessage) string {
	t.Helper()
	var ext map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(impExt, &ext))
	var prebid map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(ext["prebid"], &prebid))
	var sr map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(prebid["storedrequest"], &sr))
	var id string
	require.NoError(t, json.Unmarshal(sr["id"], &id))
	return id
}

// buildBidResponseJSON marshals an openrtb2.BidResponse with the supplied
// currency and seatbids. Used to simulate Teal's response body.
func buildBidResponseJSON(t *testing.T, currency string, seatbids []openrtb2.SeatBid) []byte {
	t.Helper()
	resp := openrtb2.BidResponse{ID: testRequestID, Cur: currency, SeatBid: seatbids}
	out, err := json.Marshal(resp)
	require.NoError(t, err)
	return out
}
