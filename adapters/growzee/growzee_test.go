package growzee

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderGrowzee, config.Adapter{
		Endpoint: "https://bid.example.com/openrtb?id={{.PublisherID}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
	adapterstest.RunJSONBidderTest(t, "growzeetest", bidder)
}

// The JSON harness sends no ImpIDs, so these cases are built by hand.
func makeBids(t *testing.T, reqData *adapters.RequestData, bid string) (*adapters.BidderResponse, []error) {
	bidder, _ := Builder(openrtb_ext.BidderGrowzee, config.Adapter{
		Endpoint: "https://bid.example.com/openrtb?id={{.PublisherID}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	request := &openrtb2.BidRequest{Imp: []openrtb2.Imp{
		{ID: "imp-1", Banner: &openrtb2.Banner{}},
		{ID: "imp-2", Video: &openrtb2.Video{}},
	}}
	return bidder.MakeBids(request, reqData,
		&adapters.ResponseData{StatusCode: 200, Body: []byte(`{"seatbid":[{"bid":[` + bid + `]}]}`)})
}

// Two requests, one per publisher: the answer to imp-1's request names imp-2.
func TestMakeBidsCrossRequestImpID(t *testing.T) {
	reqData := &adapters.RequestData{Uri: "https://bid.example.com/openrtb?id=a", ImpIDs: []string{"imp-1"}}
	for _, bid := range []string{
		`{"id":"b","impid":"imp-2","price":1,"adm":"<VAST/>"}`,
		`{"id":"b","impid":"imp-2","price":1,"adm":"<VAST/>","mtype":2}`,
	} {
		resp, errs := makeBids(t, reqData, bid)
		assert.Empty(t, resp.Bids, bid)
		assert.Len(t, errs, 1, bid)
	}

	resp, errs := makeBids(t, reqData, `{"id":"b","impid":"imp-1","price":1,"adm":"<div/>"}`)
	assert.Empty(t, errs)
	assert.Equal(t, openrtb_ext.BidTypeBanner, resp.Bids[0].BidType)
}

// A stored bid response has no Uri, no ImpIDs and no imp: only mtype types it.
func TestMakeBidsStoredResponse(t *testing.T) {
	resp, errs := makeBids(t, &adapters.RequestData{}, `{"id":"b","impid":"gone","price":1,"mtype":2}`)
	assert.Empty(t, errs)
	assert.Equal(t, openrtb_ext.BidTypeVideo, resp.Bids[0].BidType)

	resp, errs = makeBids(t, &adapters.RequestData{}, `{"id":"b","impid":"gone","price":1}`)
	assert.Empty(t, resp.Bids)
	assert.Len(t, errs, 1)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderGrowzee, config.Adapter{
		Endpoint: "{{Malformed}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	assert.Error(t, buildErr)
}
