package waardex

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderWaardex, config.Adapter{
		Endpoint: "http://justbidit2.xyz:8800/hb?zone={{.ZoneID}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "waardextest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderWaardex, config.Adapter{
		Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

// --- MakeRequests ---

func TestMakeRequests_NoImpressions(t *testing.T) {
	adapter := &waardexAdapter{
		EndpointTemplate: template.Must(template.New("endpointTemplate").Parse("http://example.com?zone={{.ZoneID}}")),
	}
	request := &openrtb2.BidRequest{}

	requests, errs := adapter.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Nil(t, requests)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "No impression in the bid request")
}

func TestMakeRequests_AllImpressionsInvalid(t *testing.T) {
	adapter := &waardexAdapter{
		EndpointTemplate: template.Must(template.New("endpointTemplate").Parse("http://example.com?zone={{.ZoneID}}")),
	}
	request := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{
			{ID: "1", Ext: json.RawMessage("malformed")},
		},
	}

	requests, errs := adapter.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Nil(t, requests)
	require.Len(t, errs, 1)
}

func TestMakeRequests_DispatchEmptyReturnsNil(t *testing.T) {
	adapter := &waardexAdapter{
		EndpointTemplate: template.Must(template.New("endpointTemplate").Parse("http://example.com?zone={{.ZoneID}}")),
	}
	// audio+native is multi-format but splitMultiFormatImp returns no imps
	request := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{makeImp(t, "1", false, false, true, true, 7)},
	}

	requests, errs := adapter.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Nil(t, requests)
	assert.Len(t, errs, 0)
}

func TestMakeRequests_BuildAdapterRequestError(t *testing.T) {
	failTemplate, err := template.New("endpointTemplate").Funcs(template.FuncMap{
		"fail": func() (string, error) { return "", errors.New("boom") },
	}).Parse("{{fail}}")
	require.NoError(t, err)

	adapter := &waardexAdapter{EndpointTemplate: failTemplate}
	request := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{makeImp(t, "1", true, false, false, false, 7)},
	}

	requests, errs := adapter.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Len(t, requests, 0)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "boom")
}

// --- Helpers ---

func makeImp(t *testing.T, id string, withBanner, withVideo, withNative, withAudio bool, zoneID int) openrtb2.Imp {
	t.Helper()
	imp := openrtb2.Imp{ID: id}
	if withBanner {
		imp.Banner = &openrtb2.Banner{}
	}
	if withVideo {
		imp.Video = &openrtb2.Video{}
	}
	if withNative {
		imp.Native = &openrtb2.Native{Request: `{"ver":"1.2"}`}
	}
	if withAudio {
		imp.Audio = &openrtb2.Audio{MinDuration: 1}
	}
	// ext: {"bidder":{"zoneId":<zoneID>}}
	extObj := map[string]interface{}{"bidder": map[string]interface{}{"zoneId": zoneID}}
	raw, err := json.Marshal(extObj)
	require.NoError(t, err)
	imp.Ext = raw
	return imp
}

// --- getImpressionsInfo ---

func TestGetImpressionsInfo_FiltersInvalidAndKeepsValid(t *testing.T) {
	validImp := makeImp(t, "1", true, false, false, false, 10)
	invalidImp := openrtb2.Imp{ID: "2", Ext: json.RawMessage("malformed")}

	imps, exts, errs := getImpressionsInfo([]openrtb2.Imp{validImp, invalidImp})

	assert.Len(t, errs, 1, "expected one error for invalid imp ext")
	require.Len(t, imps, 1)
	require.Len(t, exts, 1)
	assert.Equal(t, "1", imps[0].ID)
	assert.Equal(t, 10, exts[0].ZoneId)
}

// --- dispatchImpressions ---

func TestDispatchImpressions_GroupsByZoneAndSplitsMultiFormat(t *testing.T) {
	imp1 := makeImp(t, "1", true, false, false, false, 100) // banner only
	imp2 := makeImp(t, "2", true, true, false, false, 100)  // banner + video (multi)
	imp3 := makeImp(t, "3", false, true, false, false, 200) // video only different zone

	imps := []openrtb2.Imp{imp1, imp2, imp3}
	exts := []openrtb_ext.ExtImpWaardex{{ZoneId: 100}, {ZoneId: 100}, {ZoneId: 200}}

	grouped := dispatchImpressions(imps, exts)

	// zone 100 should have imp1 and two split imps from imp2
	g100 := grouped[openrtb_ext.ExtImpWaardex{ZoneId: 100}]
	require.Len(t, g100, 3)
	// Expect exactly one banner-only and one video-only from the split, plus original banner-only
	var banners, videos int
	for _, imp := range g100 {
		if imp.Banner != nil && imp.Video == nil {
			banners++
		}
		if imp.Video != nil && imp.Banner == nil {
			videos++
		}
		// ext is nil in dispatched imps
		assert.Nil(t, imp.Ext)
	}
	assert.Equal(t, 2, banners)
	assert.Equal(t, 1, videos)

	// zone 200 should have one video-only
	g200 := grouped[openrtb_ext.ExtImpWaardex{ZoneId: 200}]
	require.Len(t, g200, 1)
	assert.NotNil(t, g200[0].Video)
	assert.Nil(t, g200[0].Banner)
}

// --- isMultiFormatImp ---

func TestIsMultiFormatImp(t *testing.T) {
	single := makeImp(t, "1", true, false, false, false, 1)
	multi := makeImp(t, "2", true, true, false, false, 1)
	none := openrtb2.Imp{ID: "3"}

	assert.False(t, isMultiFormatImp(&single))
	assert.True(t, isMultiFormatImp(&multi))
	assert.False(t, isMultiFormatImp(&none))
}

// --- splitMultiFormatImp ---

func TestSplitMultiFormatImp_BannerVideoOnly(t *testing.T) {
	imp := makeImp(t, "m", true, true, true, true, 5)
	// ensure it is multi-format
	require.True(t, isMultiFormatImp(&imp))

	split := splitMultiFormatImp(&imp)
	// Only banner and video are produced
	require.Len(t, split, 2)
	// first banner-only, second video-only (order not guaranteed; verify counts)
	var banners, videos, natives, audios int
	for _, s := range split {
		if s.Banner != nil {
			banners++
		}
		if s.Video != nil {
			videos++
		}
		if s.Native != nil {
			natives++
		}
		if s.Audio != nil {
			audios++
		}
	}
	assert.Equal(t, 1, banners)
	assert.Equal(t, 1, videos)
	assert.Equal(t, 0, natives)
	assert.Equal(t, 0, audios)
}

// --- createBidRequest ---

func TestCreateBidRequest_ShallowCopyAndStripPublishers(t *testing.T) {
	sitePublisher := &openrtb2.Publisher{ID: "pub-site"}
	appPublisher := &openrtb2.Publisher{ID: "pub-app"}
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Site: &openrtb2.Site{ID: "site-1", Publisher: sitePublisher},
		App:  &openrtb2.App{ID: "app-1", Publisher: appPublisher},
		Imp:  []openrtb2.Imp{makeImp(t, "i1", true, false, false, false, 9)},
	}

	newImps := []openrtb2.Imp{makeImp(t, "i2", false, true, false, false, 9)}
	out := createBidRequest(req, newImps)

	// Ensure original not mutated
	require.NotNil(t, req.Site.Publisher)
	require.NotNil(t, req.App.Publisher)
	assert.Equal(t, 1, len(req.Imp))

	// Ensure new request contains provided imps and stripped publishers
	require.Equal(t, newImps, out.Imp)
	if out.Site != nil {
		assert.Nil(t, out.Site.Publisher)
	}
	if out.App != nil {
		assert.Nil(t, out.App.Publisher)
	}
}

// --- buildEndpointURL ---

func TestBuildEndpointURL(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderWaardex, config.Adapter{
		Endpoint: "http://example.test/hb?zone={{.ZoneID}}"}, config.Server{})
	require.NoError(t, err)

	url, err := bidder.(*waardexAdapter).buildEndpointURL(&openrtb_ext.ExtImpWaardex{ZoneId: 321})
	require.NoError(t, err)
	assert.Equal(t, "http://example.test/hb?zone=321", url)
}

// --- MakeBids ---

func TestMakeBids_NoContent(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderWaardex, config.Adapter{Endpoint: "http://e"}, config.Server{})
	require.NoError(t, err)

	br, errs := bidder.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{}, &adapters.ResponseData{StatusCode: http.StatusNoContent})
	assert.Nil(t, br)
	assert.Nil(t, errs)
}

func TestMakeBids_NonOKStatus(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderWaardex, config.Adapter{Endpoint: "http://e"}, config.Server{})
	require.NoError(t, err)

	br, errs := bidder.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{}, &adapters.ResponseData{StatusCode: http.StatusBadRequest})
	assert.Nil(t, br)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "Unexpected http status code")
}

func TestMakeBids_BadJSON(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderWaardex, config.Adapter{Endpoint: "http://e"}, config.Server{})
	require.NoError(t, err)

	br, errs := bidder.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{}, &adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte("not-json")})
	assert.Nil(t, br)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "Bad server response")
}

func TestMakeBids_InvalidSeatBidsCount(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderWaardex, config.Adapter{Endpoint: "http://e"}, config.Server{})
	require.NoError(t, err)

	payload := openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{}} // zero seatbids
	raw, _ := json.Marshal(payload)
	br, errs := bidder.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{}, &adapters.ResponseData{StatusCode: http.StatusOK, Body: raw})
	assert.Nil(t, br)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "Invalid SeatBids count")
}

func TestMakeBids_SuccessWithTypesAndCurrency(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderWaardex, config.Adapter{Endpoint: "http://e"}, config.Server{})
	require.NoError(t, err)

	bids := []openrtb2.Bid{
		{ID: "b1", ImpID: "1", Price: 1.1, MType: openrtb2.MarkupBanner},
		{ID: "b2", ImpID: "2", Price: 2.2, MType: openrtb2.MarkupVideo},
	}
	payload := openrtb2.BidResponse{Cur: "EUR", SeatBid: []openrtb2.SeatBid{{Bid: bids}}}
	raw, _ := json.Marshal(payload)
	br, errs := bidder.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{}, &adapters.ResponseData{StatusCode: http.StatusOK, Body: raw})

	require.Nil(t, errs)
	require.NotNil(t, br)
	assert.Equal(t, "EUR", br.Currency)
	require.Len(t, br.Bids, 2)
	assert.Equal(t, openrtb_ext.BidTypeBanner, br.Bids[0].BidType)
	assert.Equal(t, openrtb_ext.BidTypeVideo, br.Bids[1].BidType)
}

// --- getMediaTypeForBid ---

func TestGetMediaTypeForBid(t *testing.T) {
	type tc struct {
		mtype openrtb2.MarkupType
		want  openrtb_ext.BidType
		ok    bool
	}
	tests := []tc{
		{openrtb2.MarkupBanner, openrtb_ext.BidTypeBanner, true},
		{openrtb2.MarkupAudio, openrtb_ext.BidTypeAudio, true},
		{openrtb2.MarkupNative, openrtb_ext.BidTypeNative, true},
		{openrtb2.MarkupVideo, openrtb_ext.BidTypeVideo, true},
		{openrtb2.MarkupType(99), "", false},
	}
	for _, tt := range tests {
		bid := &openrtb2.Bid{MType: tt.mtype}
		got, err := getMediaTypeForBid(bid)
		if tt.ok {
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		} else {
			require.Error(t, err)
		}
	}
}
