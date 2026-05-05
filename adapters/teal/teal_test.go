package teal

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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
	testPlacement   = "test-placement300x250"
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

// givenAppBidRequest returns a BidRequest with .App (and no .Site) for the
// app-publisher rewrite path.
func givenAppBidRequest(imps ...openrtb2.Imp) *openrtb2.BidRequest {
	return &openrtb2.BidRequest{
		ID:  testRequestID,
		Imp: imps,
		App: &openrtb2.App{ID: "demo-app", Bundle: "com.demo.app", Publisher: &openrtb2.Publisher{ID: "demo-publisher"}},
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
// MakeRequests — error paths
// ---------------------------------------------------------------------------

// TestMakeRequests_ImpExtParseError mirrors Java
// makeHttpRequestsShouldReturnErrorIfImpExtCouldNotBeParsed. When imp.ext
// cannot be deserialized into ExtImpBidder, parseImpExt returns the
// "Error parsing imp.ext for impression {id}" message wrapped in BadInput.
func TestMakeRequests_ImpExtParseError(t *testing.T) {
	bidder := newTealBidder(t)

	// imp.ext is a JSON array — fails the bidderExt unmarshal step.
	imp := openrtb2.Imp{
		ID:     "impId",
		Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		Ext:    json.RawMessage(`[]`),
	}
	requests, errs := bidder.MakeRequests(givenBidRequest(imp), &adapters.ExtraRequestInfo{})

	assert.Nil(t, requests)
	require.Len(t, errs, 1)
	assert.True(t, isBadInputErr(errs[0]), "expected BadInput, got %T", errs[0])
	assert.True(t, strings.HasPrefix(errs[0].Error(), "Error parsing imp.ext for impression impId"),
		"got %q", errs[0].Error())
}

// TestMakeRequests_BidderFieldMalformed exercises the second parseImpExt
// unmarshal branch — the bidder sub-object exists but is malformed.
func TestMakeRequests_BidderFieldMalformed(t *testing.T) {
	bidder := newTealBidder(t)
	imp := openrtb2.Imp{
		ID:     "impId",
		Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		Ext:    json.RawMessage(`{"bidder":[]}`), // valid wrapper, invalid bidder shape
	}

	requests, errs := bidder.MakeRequests(givenBidRequest(imp), &adapters.ExtraRequestInfo{})

	assert.Nil(t, requests)
	require.Len(t, errs, 1)
	assert.True(t, isBadInputErr(errs[0]))
	assert.Contains(t, errs[0].Error(), "Error parsing imp.ext for impression impId")
}

// TestMakeRequests_AccountValidation mirrors Java
// makeHttpRequestsShouldReturnErrorIfAccountParamFailsValidation.
func TestMakeRequests_AccountValidation(t *testing.T) {
	bidder := newTealBidder(t)

	cases := []struct {
		name    string
		account string
	}{
		{"empty", ""},
		{"single space", " "},
		{"tab", "\t"},
		{"newline", "\n"},
		{"mixed whitespace", "  \t\n  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			imp := givenBannerImp("imp1", givenImpExt(t, tc.account, strPtr("placement")))
			requests, errs := bidder.MakeRequests(givenBidRequest(imp), &adapters.ExtraRequestInfo{})

			assert.Nil(t, requests)
			require.Len(t, errs, 1)
			assert.True(t, isBadInputErr(errs[0]))
			assert.Equal(t, msgAccountValidation, errs[0].Error())
		})
	}
}

// TestMakeRequests_PlacementValidation mirrors Java
// makeHttpRequestsShouldReturnErrorIfPlacementParamFailsValidation.
// Note: a nil placement is permitted; only a non-nil-blank one fails.
func TestMakeRequests_PlacementValidation(t *testing.T) {
	bidder := newTealBidder(t)

	cases := []struct {
		name      string
		placement string
	}{
		{"empty present", ""},
		{"single space", " "},
		{"only whitespace", "   \t\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			imp := givenBannerImp("imp1", givenImpExt(t, "account", strPtr(tc.placement)))
			requests, errs := bidder.MakeRequests(givenBidRequest(imp), &adapters.ExtraRequestInfo{})

			assert.Nil(t, requests)
			require.Len(t, errs, 1)
			assert.True(t, isBadInputErr(errs[0]))
			assert.Equal(t, msgPlacementValidation, errs[0].Error())
		})
	}
}

// TestMakeRequests_PlacementAbsent confirms an absent placement (nil pointer)
// is the explicit "skip M1" signal — no error, no storedrequest injection.
func TestMakeRequests_PlacementAbsent(t *testing.T) {
	bidder := newTealBidder(t)

	imp := givenBannerImp(testImpBannerID, givenImpExt(t, testAccount, nil))
	requests, errs := bidder.MakeRequests(givenBidRequest(imp), &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	require.Len(t, requests, 1)

	// Verify imp.ext.prebid was NOT injected (M1 skipped).
	var body openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &body))
	require.Len(t, body.Imp, 1)

	var ext map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body.Imp[0].Ext, &ext))
	_, hasPrebid := ext["prebid"]
	assert.False(t, hasPrebid, "prebid key must NOT be injected when placement is nil")
}

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

// TestMakeRequests_PartialFailure — 2 imps, 1 invalid 1 valid → 1 RequestData
// + 1 error. Tests the "errs accumulate, valid imps still ship" partial path.
func TestMakeRequests_PartialFailure(t *testing.T) {
	bidder := newTealBidder(t)
	bad := givenBannerImp("bad", givenImpExt(t, "", strPtr("p1"))) // blank account
	good := givenBannerImp("good", givenImpExt(t, "acct", strPtr("p2")))

	requests, errs := bidder.MakeRequests(givenBidRequest(bad, good), &adapters.ExtraRequestInfo{})

	require.Len(t, requests, 1)
	require.Len(t, errs, 1)
	assert.Equal(t, msgAccountValidation, errs[0].Error())

	// The surviving imp should be the "good" one and account should be "acct".
	var body openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &body))
	require.Len(t, body.Imp, 1)
	assert.Equal(t, "good", body.Imp[0].ID)
	require.NotNil(t, body.Site)
	require.NotNil(t, body.Site.Publisher)
	assert.Equal(t, "acct", body.Site.Publisher.ID)
}

// ---------------------------------------------------------------------------
// MakeRequests — happy path & mutation verification
// ---------------------------------------------------------------------------

// TestMakeRequests_AppliesAllMutations — full happy path verifying M1, M2, M3
// land in the outbound body. This is the closest mirror of Java's
// makeHttpRequestsShouldMapParametersCorrectly + makeHttpRequestsShouldAddExtBidsPBSFlag.
func TestMakeRequests_AppliesAllMutations(t *testing.T) {
	bidder := newTealBidder(t)
	imp := givenBannerImp(testImpBannerID, givenImpExt(t, testAccount, strPtr(testPlacement)))
	requests, errs := bidder.MakeRequests(givenBidRequest(imp), &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	require.Len(t, requests, 1)

	rd := requests[0]
	assert.Equal(t, http.MethodPost, rd.Method)
	assert.Equal(t, testEndpoint, rd.Uri)
	assert.Equal(t, []string{testImpBannerID}, rd.ImpIDs)
	assert.Equal(t, []string{"application/json;charset=utf-8"}, rd.Headers["Content-Type"])
	assert.Equal(t, []string{"application/json"}, rd.Headers["Accept"])

	var body openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(rd.Body, &body))

	// M2: Site.Publisher.ID rewritten with first-account.
	require.NotNil(t, body.Site)
	require.NotNil(t, body.Site.Publisher)
	assert.Equal(t, testAccount, body.Site.Publisher.ID)

	// M1: imp.ext.prebid.storedrequest.id == placement.
	require.Len(t, body.Imp, 1)
	storedID := readStoredRequestID(t, body.Imp[0].Ext)
	assert.Equal(t, testPlacement, storedID)

	// M3: request.ext.bids == {"pbs":1}.
	pbs := readBidsPBS(t, body.Ext)
	assert.JSONEq(t, `{"pbs":1}`, string(pbs))
}

// TestMakeRequests_FirstAccountWins mirrors Java's "account = account == null
// ? ext.getAccount() : account" — when imp1.account=A and imp2.account=B, the
// site/app publisher.id stays A.
func TestMakeRequests_FirstAccountWins(t *testing.T) {
	bidder := newTealBidder(t)
	first := givenBannerImp("imp1", givenImpExt(t, "first-account", strPtr("p1")))
	second := givenBannerImp("imp2", givenImpExt(t, "second-account", strPtr("p2")))

	requests, errs := bidder.MakeRequests(givenBidRequest(first, second), &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	require.Len(t, requests, 1)

	var body openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &body))

	require.NotNil(t, body.Site)
	require.NotNil(t, body.Site.Publisher)
	assert.Equal(t, "first-account", body.Site.Publisher.ID,
		"publisher.id must reflect the FIRST surviving imp's account")
}

// TestMakeRequests_AppPublisherRewrite — request.app != nil, request.site == nil
// → app.publisher.id rewritten and request.site stays nil.
func TestMakeRequests_AppPublisherRewrite(t *testing.T) {
	bidder := newTealBidder(t)
	imp := givenBannerImp(testImpBannerID, givenImpExt(t, testAccount, strPtr(testPlacement)))
	requests, errs := bidder.MakeRequests(givenAppBidRequest(imp), &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	require.Len(t, requests, 1)

	var body openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &body))
	assert.Nil(t, body.Site)
	require.NotNil(t, body.App)
	require.NotNil(t, body.App.Publisher)
	assert.Equal(t, testAccount, body.App.Publisher.ID)
}

// TestMakeRequests_BothSiteAndApp — both non-nil → both rewritten.
// Java's TealBidder.modifyBidRequest applies both branches independently.
func TestMakeRequests_BothSiteAndApp(t *testing.T) {
	bidder := newTealBidder(t)
	imp := givenBannerImp(testImpBannerID, givenImpExt(t, testAccount, nil))

	req := &openrtb2.BidRequest{
		ID:   testRequestID,
		Imp:  []openrtb2.Imp{imp},
		Site: &openrtb2.Site{ID: "demo-site", Publisher: &openrtb2.Publisher{ID: "site-pub"}},
		App:  &openrtb2.App{ID: "demo-app", Publisher: &openrtb2.Publisher{ID: "app-pub"}},
	}
	requests, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	require.Len(t, requests, 1)

	var body openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &body))
	require.NotNil(t, body.Site)
	require.NotNil(t, body.Site.Publisher)
	assert.Equal(t, testAccount, body.Site.Publisher.ID)
	require.NotNil(t, body.App)
	require.NotNil(t, body.App.Publisher)
	assert.Equal(t, testAccount, body.App.Publisher.ID)
}

// TestMakeRequests_NoSiteNoApp — both nil → no panic, no publisher rewrite,
// M1 + M3 still applied.
func TestMakeRequests_NoSiteNoApp(t *testing.T) {
	bidder := newTealBidder(t)
	imp := givenBannerImp(testImpBannerID, givenImpExt(t, testAccount, strPtr(testPlacement)))
	req := &openrtb2.BidRequest{ID: testRequestID, Imp: []openrtb2.Imp{imp}}

	requests, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	require.Len(t, requests, 1)

	var body openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &body))
	assert.Nil(t, body.Site)
	assert.Nil(t, body.App)

	// M1 + M3 still applied.
	require.Len(t, body.Imp, 1)
	assert.Equal(t, testPlacement, readStoredRequestID(t, body.Imp[0].Ext))
	assert.JSONEq(t, `{"pbs":1}`, string(readBidsPBS(t, body.Ext)))
}

// TestMakeRequests_PublisherNilOnSite — site is set but site.publisher is nil.
// clonePublisherWithID must construct a fresh Publisher rather than NPE.
func TestMakeRequests_PublisherNilOnSite(t *testing.T) {
	bidder := newTealBidder(t)
	imp := givenBannerImp(testImpBannerID, givenImpExt(t, testAccount, nil))
	req := &openrtb2.BidRequest{
		ID:   testRequestID,
		Imp:  []openrtb2.Imp{imp},
		Site: &openrtb2.Site{ID: "demo-site"}, // publisher nil
	}
	requests, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	require.Len(t, requests, 1)
	var body openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &body))
	require.NotNil(t, body.Site)
	require.NotNil(t, body.Site.Publisher)
	assert.Equal(t, testAccount, body.Site.Publisher.ID)
}

// TestMakeRequests_PreservesExistingImpExtPrebid — imp.ext.prebid contains
// unrelated keys → must be preserved alongside the storedrequest injection.
func TestMakeRequests_PreservesExistingImpExtPrebid(t *testing.T) {
	bidder := newTealBidder(t)
	// Hand-craft an ext that already has prebid.bidder + prebid.foo set.
	preExisting := json.RawMessage(`{
		"bidder":{"account":"test-account","placement":"test-placement300x250"},
		"prebid":{"foo":"bar","keyword":"keep-me","storedrequest":{"unrelated":"value"}}
	}`)
	imp := openrtb2.Imp{
		ID:     testImpBannerID,
		Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		Ext:    preExisting,
	}
	requests, errs := bidder.MakeRequests(givenBidRequest(imp), &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	require.Len(t, requests, 1)
	var body openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &body))

	var ext map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body.Imp[0].Ext, &ext))
	var prebid map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(ext["prebid"], &prebid))

	assert.JSONEq(t, `"bar"`, string(prebid["foo"]))
	assert.JSONEq(t, `"keep-me"`, string(prebid["keyword"]))

	var sr map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(prebid["storedrequest"], &sr))
	// "id" got injected; "unrelated" was preserved.
	assert.JSONEq(t, `"`+testPlacement+`"`, string(sr["id"]))
	assert.JSONEq(t, `"value"`, string(sr["unrelated"]))
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

// TestMakeRequests_PreservesExistingRequestExt — request.ext has unrelated
// top-level keys → mergeBidsPBSFlag preserves them and adds "bids".
func TestMakeRequests_PreservesExistingRequestExt(t *testing.T) {
	bidder := newTealBidder(t)
	imp := givenBannerImp(testImpBannerID, givenImpExt(t, testAccount, nil))
	req := givenBidRequest(imp)
	req.Ext = json.RawMessage(`{"prebid":{"server":{"ttl":3600}},"someFlag":true}`)

	requests, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs)
	require.Len(t, requests, 1)
	var body openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(requests[0].Body, &body))

	var ext map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body.Ext, &ext))
	assert.JSONEq(t, `{"pbs":1}`, string(ext["bids"]), "M3 must inject bids:{pbs:1}")
	assert.JSONEq(t, `{"server":{"ttl":3600}}`, string(ext["prebid"]), "existing prebid key must survive")
	assert.JSONEq(t, `true`, string(ext["someFlag"]), "existing custom key must survive")
}

// TestMakeRequests_RequestExtMalformed — request.ext is malformed JSON →
// mergeBidsPBSFlag returns an error which propagates as an extra entry in errs.
func TestMakeRequests_RequestExtMalformed(t *testing.T) {
	bidder := newTealBidder(t)
	imp := givenBannerImp(testImpBannerID, givenImpExt(t, testAccount, nil))
	req := givenBidRequest(imp)
	req.Ext = json.RawMessage(`{"prebid":}`) // syntactically invalid

	requests, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})

	assert.Nil(t, requests)
	require.NotEmpty(t, errs)
	// Last error is the wrapped parse failure from mergeBidsPBSFlag.
	last := errs[len(errs)-1]
	assert.Contains(t, last.Error(), "failed parsing request.ext")
}

// ---------------------------------------------------------------------------
// MakeBids
// ---------------------------------------------------------------------------

// TestMakeBids_NoContent204 — 204 short-circuits with (nil, nil).
func TestMakeBids_NoContent204(t *testing.T) {
	bidder := newTealBidder(t)
	resp, errs := makeBidsCall(t, bidder, &openrtb2.BidRequest{}, http.StatusNoContent, nil)
	assert.Nil(t, resp)
	assert.Nil(t, errs)
}

// TestMakeBids_BadStatus400 — 400 → BadInput-typed error.
func TestMakeBids_BadStatus400(t *testing.T) {
	bidder := newTealBidder(t)
	resp, errs := makeBidsCall(t, bidder, &openrtb2.BidRequest{}, http.StatusBadRequest, []byte(`{}`))
	assert.Nil(t, resp)
	require.Len(t, errs, 1)
	assert.True(t, isBadInputErr(errs[0]), "400 must produce BadInput, got %T", errs[0])
}

// TestMakeBids_BadStatus500 — 500 → BadServerResponse-typed error.
func TestMakeBids_BadStatus500(t *testing.T) {
	bidder := newTealBidder(t)
	resp, errs := makeBidsCall(t, bidder, &openrtb2.BidRequest{}, http.StatusInternalServerError, []byte(`{}`))
	assert.Nil(t, resp)
	require.Len(t, errs, 1)
	assert.True(t, isBadServerErr(errs[0]), "500 must produce BadServerResponse, got %T", errs[0])
}

// TestMakeBids_BadBody mirrors Java
// makeBidsShouldReturnErrorWhenResponseBodyCouldNotBeParsed.
func TestMakeBids_BadBody(t *testing.T) {
	bidder := newTealBidder(t)
	resp, errs := makeBidsCall(t, bidder, &openrtb2.BidRequest{}, http.StatusOK, []byte("invalid_json"))
	assert.Nil(t, resp)
	require.Len(t, errs, 1)
	// We don't need to pin the exact phrasing of the parse error — only that
	// a parse error occurred.
	assert.NotEmpty(t, errs[0].Error())
}

// TestMakeBids_MediaTypeRouting consolidates Java's banner / video / native
// scenarios plus our Go-specific audio + default + missing-imp cases. Each
// case asserts that getBidType resolves the imp's mediatype based on
// banner > video > audio > native, with banner as the fallback default.
func TestMakeBids_MediaTypeRouting(t *testing.T) {
	bidder := newTealBidder(t)

	cases := []struct {
		name    string
		imp     openrtb2.Imp
		bidImp  string
		wantTyp openrtb_ext.BidType
	}{
		{
			name:    "banner imp returns banner",
			imp:     openrtb2.Imp{ID: "imp1", Banner: &openrtb2.Banner{}},
			bidImp:  "imp1",
			wantTyp: openrtb_ext.BidTypeBanner,
		},
		{
			name:    "video imp returns video",
			imp:     openrtb2.Imp{ID: "imp1", Video: &openrtb2.Video{}},
			bidImp:  "imp1",
			wantTyp: openrtb_ext.BidTypeVideo,
		},
		{
			name:    "audio imp returns audio",
			imp:     openrtb2.Imp{ID: "imp1", Audio: &openrtb2.Audio{}},
			bidImp:  "imp1",
			wantTyp: openrtb_ext.BidTypeAudio,
		},
		{
			name:    "native imp returns native",
			imp:     openrtb2.Imp{ID: "imp1", Native: &openrtb2.Native{}},
			bidImp:  "imp1",
			wantTyp: openrtb_ext.BidTypeNative,
		},
		{
			name:    "no mediatype defaults to banner",
			imp:     openrtb2.Imp{ID: "imp1"},
			bidImp:  "imp1",
			wantTyp: openrtb_ext.BidTypeBanner,
		},
		{
			name:    "bid impid not in request defaults to banner",
			imp:     openrtb2.Imp{ID: "imp1", Video: &openrtb2.Video{}},
			bidImp:  "missing",
			wantTyp: openrtb_ext.BidTypeBanner,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := &openrtb2.BidRequest{Imp: []openrtb2.Imp{tc.imp}}
			body := buildBidResponseJSON(t, "USD", []openrtb2.SeatBid{{
				Bid: []openrtb2.Bid{{ID: "b1", ImpID: tc.bidImp, Price: 1.0}},
			}})

			resp, errs := makeBidsCall(t, bidder, req, http.StatusOK, body)
			assert.Empty(t, errs)
			require.NotNil(t, resp)
			require.Len(t, resp.Bids, 1)
			assert.Equal(t, tc.wantTyp, resp.Bids[0].BidType)
		})
	}
}

// TestGetBidType_Priority — when an imp has BOTH banner and video set, the
// switch must return banner first (priority order matters for fidelity).
func TestGetBidType_Priority(t *testing.T) {
	imp := openrtb2.Imp{ID: "imp1", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}}
	bid := &openrtb2.Bid{ImpID: "imp1"}
	assert.Equal(t, openrtb_ext.BidTypeBanner, getBidType(bid, []openrtb2.Imp{imp}))
}

// TestMakeBids_MultipleSeatBids — multiple seatbids each with multiple bids
// → all bids packaged in order.
func TestMakeBids_MultipleSeatBids(t *testing.T) {
	bidder := newTealBidder(t)
	req := &openrtb2.BidRequest{Imp: []openrtb2.Imp{
		{ID: "imp-banner", Banner: &openrtb2.Banner{}},
		{ID: "imp-video", Video: &openrtb2.Video{}},
	}}
	seatbids := []openrtb2.SeatBid{
		{Seat: "s1", Bid: []openrtb2.Bid{
			{ID: "b1", ImpID: "imp-banner", Price: 1.0},
			{ID: "b2", ImpID: "imp-video", Price: 2.0},
		}},
		{Seat: "s2", Bid: []openrtb2.Bid{
			{ID: "b3", ImpID: "imp-banner", Price: 1.5},
		}},
	}
	body := buildBidResponseJSON(t, "USD", seatbids)

	resp, errs := makeBidsCall(t, bidder, req, http.StatusOK, body)
	assert.Empty(t, errs)
	require.NotNil(t, resp)
	require.Len(t, resp.Bids, 3)
	assert.Equal(t, "b1", resp.Bids[0].Bid.ID)
	assert.Equal(t, openrtb_ext.BidTypeBanner, resp.Bids[0].BidType)
	assert.Equal(t, "b2", resp.Bids[1].Bid.ID)
	assert.Equal(t, openrtb_ext.BidTypeVideo, resp.Bids[1].BidType)
	assert.Equal(t, "b3", resp.Bids[2].Bid.ID)
	assert.Equal(t, openrtb_ext.BidTypeBanner, resp.Bids[2].BidType)
}

// TestMakeBids_EmptySeatBid — seatbid array empty → empty BidderResponse
// (no error, just no bids).
func TestMakeBids_EmptySeatBid(t *testing.T) {
	bidder := newTealBidder(t)
	body := buildBidResponseJSON(t, "USD", nil)
	resp, errs := makeBidsCall(t, bidder, &openrtb2.BidRequest{}, http.StatusOK, body)
	assert.Empty(t, errs)
	require.NotNil(t, resp)
	assert.Empty(t, resp.Bids)
	assert.Equal(t, "USD", resp.Currency)
}

// TestMakeBids_CurrencyPropagation — bidResponse.cur="EUR" surfaces on the
// BidderResponse.
func TestMakeBids_CurrencyPropagation(t *testing.T) {
	bidder := newTealBidder(t)
	req := &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "imp1", Banner: &openrtb2.Banner{}}}}
	body := buildBidResponseJSON(t, "EUR", []openrtb2.SeatBid{{
		Bid: []openrtb2.Bid{{ID: "b1", ImpID: "imp1", Price: 1.0}},
	}})

	resp, errs := makeBidsCall(t, bidder, req, http.StatusOK, body)
	assert.Empty(t, errs)
	require.NotNil(t, resp)
	assert.Equal(t, "EUR", resp.Currency)
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
		{" ", true}, // U+00A0 NO-BREAK SPACE — unicode.IsSpace returns true
		{" ", true}, // U+2003 EM SPACE
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
// History: FuzzMergeBidsPBSFlag (Iter 2) discovered that `null` was unmarshaled
// into a nil map, causing the subsequent `ext["bids"] = ...` to panic with
// "assignment to entry in nil map". Iter 3 hardened mergeBidsPBSFlag (and
// modifyImp) by routing through decodeJSONObject, which guarantees a non-nil
// receiver. This test pins the corrected behavior.
func TestMergeBidsPBSFlag_NullInputHandledAsEmpty(t *testing.T) {
	out, err := mergeBidsPBSFlag(json.RawMessage(`null`))
	require.NoError(t, err)
	require.NotNil(t, out)

	var decoded map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(out, &decoded))
	assert.Len(t, decoded, 1, "null input should produce just the bids stamp")
	assert.JSONEq(t, `{"pbs":1}`, string(decoded["bids"]))
}

// TestModifyImp_NullImpExtHandledAsEmpty pins the same null-handling contract
// for modifyImp (the second site of the Iter 2 fuzz-discovered nil-map bug).
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

// readBidsPBS returns request.ext.bids as raw JSON.
func readBidsPBS(t *testing.T, requestExt json.RawMessage) json.RawMessage {
	t.Helper()
	var ext map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(requestExt, &ext))
	return ext["bids"]
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
