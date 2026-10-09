package floxis

import (
	"encoding/json"
	"fmt"
	"testing"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	}{
		{"us-e", "floxis", "us-e"},
		{"eu", "floxis", "eu"},
		{"apac", "floxis", "apac"},
		{"", "", "us-e"},
		{"", "floxis", "us-e"},
		{"eu", "", "eu"},
		{"mars", "floxis", "mars"},
		{"us-e", "acme", "acme-us-e"},
		{"eu", "acme", "acme-eu"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, resolveBidHost(c.region, c.partner), "region %q partner %q", c.region, c.partner)
	}
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

func TestAliasEndpointWithoutHostMacroIgnoresRegionAndPartner(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderName("adapex"), config.Adapter{Endpoint: "https://hb.adapex.io/pbs"}, config.Server{})
	assert.NoError(t, buildErr)

	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{bannerImp(`{"bidder":{"seat":"abc","region":"eu","partner":"acme"}}`)},
		Site: &openrtb2.Site{ID: "271"},
	}
	reqData, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	assert.Len(t, reqData, 1)
	assert.Equal(t, "https://hb.adapex.io/pbs?seat=abc", reqData[0].Uri)
}

func TestAliasEndpointWithoutHostMacroRoutesOncePerSeat(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderName("adapex"), config.Adapter{Endpoint: "https://hb.adapex.io/pbs"}, config.Server{})
	assert.NoError(t, buildErr)

	imp1 := bannerImp(`{"bidder":{"seat":"seat-a","region":"eu"}}`)
	imp2 := bannerImp(`{"bidder":{"seat":"seat-a","region":"apac","partner":"acme"}}`)
	imp2.ID = "imp-2"
	imp3 := bannerImp(`{"bidder":{"seat":"seat-b","region":"eu"}}`)
	imp3.ID = "imp-3"
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{imp1, imp2, imp3},
		Site: &openrtb2.Site{ID: "271"},
	}
	reqData, errs := bidder.MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	assert.Len(t, reqData, 2)
	assert.Equal(t, "https://hb.adapex.io/pbs?seat=seat-a", reqData[0].Uri)
	assert.Equal(t, []string{"imp-1", "imp-2"}, reqData[0].ImpIDs)
	assert.Equal(t, "https://hb.adapex.io/pbs?seat=seat-b", reqData[1].Uri)
	assert.Equal(t, []string{"imp-3"}, reqData[1].ImpIDs)
}

func TestMakeBidsRejectsImpressionsFromAnotherRequest(t *testing.T) {
	for _, mtype := range []openrtb2.MarkupType{0, openrtb2.MarkupBanner} {
		t.Run(fmt.Sprintf("mtype-%d", mtype), func(t *testing.T) {
			imp1 := bannerImp(`{"bidder":{"seat":"seat-a"}}`)
			imp2 := bannerImp(`{"bidder":{"seat":"seat-b"}}`)
			imp2.ID = "imp-2"
			request := &openrtb2.BidRequest{ID: "request", Imp: []openrtb2.Imp{imp1, imp2}}
			bidder, err := Builder(openrtb_ext.BidderName("adapex"), config.Adapter{Endpoint: "https://hb.adapex.io/pbs"}, config.Server{})
			require.NoError(t, err)
			requests, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
			require.Empty(t, errs)
			require.Len(t, requests, 2)

			body, err := json.Marshal(openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{{Bid: []openrtb2.Bid{
				{ID: "valid", ImpID: "imp-1", Price: 1, MType: mtype},
				{ID: "wrong-seat", ImpID: "imp-2", Price: 2, MType: mtype},
				{ID: "unknown", ImpID: "unknown", Price: 3, MType: mtype},
			}}}})
			require.NoError(t, err)
			response, errs := bidder.MakeBids(request, requests[0], &adapters.ResponseData{StatusCode: 200, Body: body})
			require.Len(t, errs, 2)
			require.Len(t, response.Bids, 1)
			assert.Equal(t, "valid", response.Bids[0].Bid.ID)
			assert.Contains(t, errs[0].Error(), "not included in the outgoing request")
		})
	}
}

func TestMakeBidsWithoutImpIDsSupportsStoredResponses(t *testing.T) {
	request := &openrtb2.BidRequest{Imp: []openrtb2.Imp{bannerImp(`{"bidder":{"seat":"seat-a"}}`)}}
	response, errs := newAdapter().MakeBids(request, &adapters.RequestData{}, &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(`{"seatbid":[{"bid":[{"id":"stored","impid":"stored-imp","price":1,"mtype":1}]}]}`),
	})
	require.Empty(t, errs)
	require.Len(t, response.Bids, 1)
	assert.Equal(t, "stored", response.Bids[0].Bid.ID)
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
