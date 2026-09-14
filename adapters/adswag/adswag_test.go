package adswag

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const testEndpoint = "https://test.endpoint.example/prebid/bid"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdswag, config.Adapter{
		Endpoint: testEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1417, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "adswagtest", bidder)
}

func newAdapter() *adapter {
	return &adapter{endpoint: testEndpoint}
}

func bannerImp(ext string) openrtb2.Imp {
	return openrtb2.Imp{
		ID:     "imp-1",
		Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		Ext:    json.RawMessage(ext),
	}
}

func TestEmptyBodyIsNoBid(t *testing.T) {
	req := &openrtb2.BidRequest{ID: "req-1", Imp: []openrtb2.Imp{bannerImp(`{"bidder":{"publisherId":"pub-1"}}`)}}
	for _, body := range []string{"", "  \n"} {
		resp, errs := newAdapter().MakeBids(req, &adapters.RequestData{}, &adapters.ResponseData{
			StatusCode: 200,
			Body:       []byte(body),
		})
		assert.Nil(t, resp, "empty-200 body %q must be a no-bid", body)
		assert.Empty(t, errs)
	}
}

func TestNoContentIsNoBid(t *testing.T) {
	req := &openrtb2.BidRequest{ID: "req-1", Imp: []openrtb2.Imp{bannerImp(`{"bidder":{"publisherId":"pub-1"}}`)}}
	resp, errs := newAdapter().MakeBids(req, &adapters.RequestData{}, &adapters.ResponseData{StatusCode: 204})
	assert.Nil(t, resp)
	assert.Empty(t, errs)
}

// TestCallerRequestNotMutated asserts the copy-on-write surfaces in
// MakeRequests: request.Imp[i].Ext (rewritten for the outgoing request) and
// request.Site/Site.Publisher (publisherId promotion).
func TestCallerRequestNotMutated(t *testing.T) {
	originalExt := `{"gpid":"/1111/homepage#div-1","bidder":{"publisherId":"pub-1","placementId":"plc-1"}}`
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{bannerImp(originalExt)},
		Site: &openrtb2.Site{Domain: "example.com", Publisher: &openrtb2.Publisher{Name: "Example"}},
	}

	reqData, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	assert.Len(t, reqData, 1)

	assert.JSONEq(t, originalExt, string(req.Imp[0].Ext), "caller's imp.ext must not be mutated")
	assert.Empty(t, req.Site.Publisher.ID, "caller's site.publisher must not be mutated")

	var out openrtb2.BidRequest
	assert.NoError(t, json.Unmarshal(reqData[0].Body, &out))
	assert.JSONEq(t,
		`{"gpid":"/1111/homepage#div-1","adswag":{"placement_id":"plc-1"}}`,
		string(out.Imp[0].Ext), "outgoing imp.ext must carry gpid + adswag placement override without the bidder envelope")
	assert.Equal(t, "pub-1", out.Site.Publisher.ID)
	assert.Equal(t, "Example", out.Site.Publisher.Name, "existing publisher fields must be preserved")
}

func TestAppPublisherIDPromotion(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID:  "req-1",
		Imp: []openrtb2.Imp{bannerImp(`{"bidder":{"publisherId":"pub-1"}}`)},
		App: &openrtb2.App{Bundle: "com.example.app"},
	}
	reqData, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	var out openrtb2.BidRequest
	assert.NoError(t, json.Unmarshal(reqData[0].Body, &out))
	assert.Equal(t, "pub-1", out.App.Publisher.ID)
	assert.Nil(t, req.App.Publisher, "caller's app must not be mutated")
}

func TestMissingPublisherIDDropsImp(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{
			bannerImp(`{"bidder":{"publisherId":"pub-1"}}`),
			{ID: "imp-2", Banner: &openrtb2.Banner{}, Ext: json.RawMessage(`{"bidder":{"placementId":"plc-2"}}`)},
		},
		Site: &openrtb2.Site{Domain: "example.com"},
	}
	reqData, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "publisherId")
	assert.Len(t, reqData, 1)

	var out openrtb2.BidRequest
	assert.NoError(t, json.Unmarshal(reqData[0].Body, &out))
	assert.Len(t, out.Imp, 1, "imp without publisherId must be dropped from the outgoing request")
}

func TestResolveBidTypeHonorsMType(t *testing.T) {
	imp := &openrtb2.Imp{ID: "imp-1", Banner: &openrtb2.Banner{}}
	cases := []struct {
		mtype openrtb2.MarkupType
		want  openrtb_ext.BidType
	}{
		{openrtb2.MarkupBanner, openrtb_ext.BidTypeBanner},
		{openrtb2.MarkupVideo, openrtb_ext.BidTypeVideo},
		{openrtb2.MarkupAudio, openrtb_ext.BidTypeAudio},
	}
	for _, c := range cases {
		got, err := resolveBidType(imp, &openrtb2.Bid{ImpID: "imp-1", MType: c.mtype})
		assert.NoError(t, err)
		assert.Equal(t, c.want, got)
	}

	_, err := resolveBidType(imp, &openrtb2.Bid{ImpID: "imp-1", MType: openrtb2.MarkupNative})
	assert.ErrorContains(t, err, "unsupported bid.mtype")
}

func TestResolveBidTypeUnresolvableImp(t *testing.T) {
	_, err := resolveBidType(&openrtb2.Imp{ID: "imp-1"}, &openrtb2.Bid{ImpID: "imp-1"})
	assert.ErrorContains(t, err, "unable to resolve media type")
}

func TestBannerSize(t *testing.T) {
	w, h := bannerSize(&openrtb2.Imp{})
	assert.Zero(t, w)
	assert.Zero(t, h)

	w, h = bannerSize(&openrtb2.Imp{Banner: &openrtb2.Banner{W: openrtb2.Int64Ptr(728), H: openrtb2.Int64Ptr(90)}})
	assert.EqualValues(t, 728, w)
	assert.EqualValues(t, 90, h)

	w, h = bannerSize(&openrtb2.Imp{Banner: &openrtb2.Banner{}})
	assert.Zero(t, w)
	assert.Zero(t, h)
}

// TestLiveCapturedResponse runs MakeBids against a real bid.adswag.ai
// response captured 2026-08-11 (evergreen test publisher, banner 300x250,
// EUR 2.50 display bid) to pin the production response shape: adm carries
// the loader script tag (passed through untouched) and ext.adswag.serve_url
// stays populated as the legacy fallback.
func TestLiveCapturedResponse(t *testing.T) {
	body, err := os.ReadFile("testdata/live-captured-response.json")
	if err != nil {
		t.Fatalf("failed to read captured response fixture: %v", err)
	}

	req := &openrtb2.BidRequest{
		ID: "adswag-pbs-smoke-1",
		Imp: []openrtb2.Imp{{
			ID:     "imp-1",
			Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		}},
	}

	resp, errs := newAdapter().MakeBids(req, &adapters.RequestData{}, &adapters.ResponseData{
		StatusCode: 200,
		Body:       body,
	})
	assert.Empty(t, errs)
	assert.Equal(t, "EUR", resp.Currency)
	assert.Len(t, resp.Bids, 1)

	bid := resp.Bids[0]
	assert.Equal(t, openrtb_ext.BidTypeBanner, bid.BidType)
	assert.Equal(t, 2.5, bid.Bid.Price)
	assert.Equal(t, openrtb2.MarkupBanner, bid.Bid.MType)
	assert.EqualValues(t, 300, bid.Bid.W)
	assert.EqualValues(t, 250, bid.Bid.H)
	// adm is the endpoint's loader script tag, passed through untouched —
	// no iframe synthesis when markup is present.
	assert.Contains(t, bid.Bid.AdM, `<script src="https://ads.adswag.ai/v1/adj?`)
	assert.NotContains(t, bid.Bid.AdM, "<iframe")
	// serve_url stays populated as the legacy fallback.
	assert.Contains(t, string(bid.Bid.Ext), `"serve_url":"https://ads.adswag.ai/v1/ad?`)
	assert.Contains(t, bid.Bid.BURL, "https://ads.adswag.ai/v1/win?")
	assert.Equal(t, []string{"adswag.ai"}, bid.Bid.ADomain)
}

func TestIframeMarkupEscapesURL(t *testing.T) {
	markup := iframeMarkup(`https://ads.example/v1/ad?a=1&b="x"`, 300, 250)
	assert.Contains(t, markup, `src="https://ads.example/v1/ad?a=1&amp;b=&#34;x&#34;"`)
	assert.Contains(t, markup, `width="300" height="250"`)
}
