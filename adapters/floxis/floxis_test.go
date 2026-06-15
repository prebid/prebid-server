package floxis

import (
	"encoding/json"
	"testing"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderFloxis, config.Adapter{
		Endpoint: "https://{{.Host}}.floxis.tech/pbs"},
		config.Server{ExternalUrl: "http://hosturl.com", DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "floxistest", bidder)
}

func newAdapter() *adapter {
	return &adapter{endpoint: template.Must(template.New("endpointTemplate").Parse("https://{{.Host}}.floxis.tech/pbs"))}
}

func bannerImp(ext string) openrtb2.Imp {
	return openrtb2.Imp{
		ID:     "imp-1",
		Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		Ext:    json.RawMessage(ext),
	}
}

func TestResolveBidHost(t *testing.T) {
	cases := []struct {
		region  string
		partner string
		want    string
		wantErr bool
	}{
		{"us-e", "floxis", "us-e", false},
		{"eu", "floxis", "eu", false},
		{"apac", "floxis", "apac", false},
		{"", "", "us-e", false},           // both default
		{"", "floxis", "us-e", false},     // empty region defaults to us-e
		{"eu", "", "eu", false},           // empty partner defaults to floxis
		{"mars", "floxis", "mars", false}, // any valid label passes through
		{"us-e", "acme", "acme-us-e", false},
		{"eu", "acme", "acme-eu", false},
		{"a.b", "floxis", "", true},     // invalid region label
		{"us-e", "bad_host!", "", true}, // invalid partner label
		{"evil.com/x", "floxis", "", true},
	}
	for _, c := range cases {
		got, err := resolveBidHost(c.region, c.partner)
		if c.wantErr {
			assert.Error(t, err, "region %q partner %q", c.region, c.partner)
			continue
		}
		assert.NoError(t, err, "region %q partner %q", c.region, c.partner)
		assert.Equal(t, c.want, got, "region %q partner %q", c.region, c.partner)
	}
}

func TestNoImpressions(t *testing.T) {
	req := &openrtb2.BidRequest{ID: "req-1"}
	reqData, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Nil(t, reqData)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "no impressions")
}

func TestSeatIsURLEscaped(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{bannerImp(`{"bidder":{"seat":"a b&c","region":"eu"}}`)},
		Site: &openrtb2.Site{ID: "271"},
	}
	reqData, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	assert.Len(t, reqData, 1)
	assert.Equal(t, "https://eu.floxis.tech/pbs?seat=a+b%26c", reqData[0].Uri)
}

func TestPartnerPrefixesHost(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{bannerImp(`{"bidder":{"seat":"abc","region":"us-e","partner":"acme"}}`)},
		Site: &openrtb2.Site{ID: "271"},
	}
	reqData, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	assert.Equal(t, "https://acme-us-e.floxis.tech/pbs?seat=abc", reqData[0].Uri)
}

func TestInvalidPartnerRejected(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{bannerImp(`{"bidder":{"seat":"abc","partner":"bad_host!"}}`)},
		Site: &openrtb2.Site{ID: "271"},
	}
	_, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "valid host labels")
}

func TestValidNonStandardRegionPassesThrough(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{bannerImp(`{"bidder":{"seat":"abc","region":"mars"}}`)},
		Site: &openrtb2.Site{ID: "271"},
	}
	reqData, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	assert.Equal(t, "https://mars.floxis.tech/pbs?seat=abc", reqData[0].Uri)
}

func TestMissingRegionDefaultsToUSE(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{bannerImp(`{"bidder":{"seat":"abc"}}`)},
		Site: &openrtb2.Site{ID: "271"},
	}
	reqData, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	assert.Equal(t, "https://us-e.floxis.tech/pbs?seat=abc", reqData[0].Uri)
}

func TestInvalidImpExt(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{}, Ext: json.RawMessage(`"not-an-object"`)}},
		Site: &openrtb2.Site{ID: "271"},
	}
	_, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "imp.ext")
}

// TestCallerRequestNotMutated asserts the adapter forwards the request body unchanged and
// does not mutate any caller-owned field (copy-on-write is satisfied by construction).
func TestCallerRequestNotMutated(t *testing.T) {
	imp := openrtb2.Imp{
		ID:     "imp-1",
		Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}},
		Ext:    json.RawMessage(`{"bidder":{"seat":"abc","region":"eu"}}`),
	}
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{imp},
		Site: &openrtb2.Site{ID: "271", Ext: json.RawMessage(`{"amp":0}`)},
	}
	before, _ := json.Marshal(req)

	_, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)

	after, _ := json.Marshal(req)
	assert.JSONEq(t, string(before), string(after), "caller's request must not be mutated")
	assert.Nil(t, req.Imp[0].Secure, "caller's imp[0].Secure must not be mutated")
}

func TestGetMediaTypeForBidByMType(t *testing.T) {
	cases := []struct {
		mtype openrtb2.MarkupType
		want  openrtb_ext.BidType
	}{
		{openrtb2.MarkupBanner, openrtb_ext.BidTypeBanner},
		{openrtb2.MarkupVideo, openrtb_ext.BidTypeVideo},
		{openrtb2.MarkupAudio, openrtb_ext.BidTypeAudio},
		{openrtb2.MarkupNative, openrtb_ext.BidTypeNative},
	}
	for _, c := range cases {
		bt, err := getMediaTypeForBid(nil, openrtb2.Bid{ImpID: "x", MType: c.mtype})
		assert.NoError(t, err)
		assert.Equal(t, c.want, bt)
	}
}

func TestGetMediaTypeForBidUnsupportedMType(t *testing.T) {
	_, err := getMediaTypeForBid(nil, openrtb2.Bid{ImpID: "x", MType: openrtb2.MarkupType(99)})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported bid.mtype")
}

func TestGetMediaTypeForBidSingleFormatFallback(t *testing.T) {
	imps := []openrtb2.Imp{
		{ID: "b", Banner: &openrtb2.Banner{}},
		{ID: "v", Video: &openrtb2.Video{}},
		{ID: "a", Audio: &openrtb2.Audio{}},
		{ID: "n", Native: &openrtb2.Native{}},
	}
	expected := map[string]openrtb_ext.BidType{
		"b": openrtb_ext.BidTypeBanner,
		"v": openrtb_ext.BidTypeVideo,
		"a": openrtb_ext.BidTypeAudio,
		"n": openrtb_ext.BidTypeNative,
	}
	for impID, want := range expected {
		bt, err := getMediaTypeForBid(imps, openrtb2.Bid{ImpID: impID})
		assert.NoError(t, err, impID)
		assert.Equal(t, want, bt, impID)
	}
}

func TestGetMediaTypeForBidMultiFormatNeedsMType(t *testing.T) {
	imps := []openrtb2.Imp{{ID: "m", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}}}
	_, err := getMediaTypeForBid(imps, openrtb2.Bid{ImpID: "m"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires bid.mtype to disambiguate")
}

func TestGetMediaTypeForBidImpWithoutFormat(t *testing.T) {
	imps := []openrtb2.Imp{{ID: "x"}}
	_, err := getMediaTypeForBid(imps, openrtb2.Bid{ImpID: "x"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to resolve media type")
}

func TestGetMediaTypeForBidUnknownImp(t *testing.T) {
	imps := []openrtb2.Imp{{ID: "x", Banner: &openrtb2.Banner{}}}
	_, err := getMediaTypeForBid(imps, openrtb2.Bid{ImpID: "missing"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to find impression")
}
