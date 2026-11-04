package floxis

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const testEndpoint = "https://us-e.floxis.tech/pbs"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderFloxis, config.Adapter{
		Endpoint: testEndpoint},
		config.Server{ExternalUrl: "http://hosturl.com", DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "floxistest", bidder)
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

func TestSeatIsURLEscaped(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{bannerImp(`{"bidder":{"seat":"a b&c"}}`)},
		Site: &openrtb2.Site{ID: "271"},
	}
	reqData, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	assert.Len(t, reqData, 1)
	assert.Equal(t, "https://us-e.floxis.tech/pbs?seat=a+b%26c", reqData[0].Uri)
}

func TestHostIsFixedRegardlessOfImpParams(t *testing.T) {
	req := &openrtb2.BidRequest{
		ID:   "req-1",
		Imp:  []openrtb2.Imp{bannerImp(`{"bidder":{"seat":"abc","region":"eu","partner":"acme"}}`)},
		Site: &openrtb2.Site{ID: "271"},
	}
	reqData, errs := newAdapter().MakeRequests(req, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	assert.Len(t, reqData, 1)
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
		Ext:    json.RawMessage(`{"bidder":{"seat":"abc"}}`),
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
		bt, err := getMediaTypeForBid([]openrtb2.Imp{{ID: "x"}}, openrtb2.Bid{ImpID: "x", MType: c.mtype})
		assert.NoError(t, err)
		assert.Equal(t, c.want, bt)
	}
}

func TestGetMediaTypeForBidUnsupportedMType(t *testing.T) {
	_, err := getMediaTypeForBid([]openrtb2.Imp{{ID: "x"}}, openrtb2.Bid{ImpID: "x", MType: openrtb2.MarkupType(99)})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported bid.mtype")
}

func TestGetMediaTypeForBidUnknownImpWithMType(t *testing.T) {
	_, err := getMediaTypeForBid([]openrtb2.Imp{{ID: "x", Banner: &openrtb2.Banner{}}}, openrtb2.Bid{ImpID: "missing", MType: openrtb2.MarkupBanner})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to find impression missing for bid")
}

func TestMakeBidsRejectsBidForOtherOutboundRequest(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Banner: &openrtb2.Banner{}, Ext: json.RawMessage(`{"bidder":{"seat":"seat-1"}}`)},
			{ID: "imp-2", Video: &openrtb2.Video{}, Ext: json.RawMessage(`{"bidder":{"seat":"seat-2"}}`)},
		},
	}
	adapter := newAdapter()
	requests, errs := adapter.MakeRequests(request, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	if !assert.Len(t, requests, 2) {
		return
	}
	response := &adapters.ResponseData{
		StatusCode: 200,
		Body:       []byte(`{"seatbid":[{"bid":[{"id":"valid","impid":"imp-1","price":1,"mtype":1},{"id":"cross-seat","impid":"imp-2","price":1,"mtype":2}]}]}`),
	}
	bids, errs := adapter.MakeBids(request, requests[0], response)
	if assert.Len(t, errs, 1) {
		assert.Contains(t, errs[0].Error(), "unable to find impression imp-2 for bid")
	}
	if assert.NotNil(t, bids) && assert.Len(t, bids.Bids, 1) {
		assert.Equal(t, "valid", bids.Bids[0].Bid.ID)
	}
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
