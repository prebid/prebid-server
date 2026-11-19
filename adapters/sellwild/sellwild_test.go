package sellwild

import (
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSellwild, config.Adapter{
		Endpoint: "https://test.endpoint.com/exchange/openrtb2/{{.PublisherID}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "sellwildtest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderSellwild, config.Adapter{
		Endpoint: "{{Malformed}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestMakeBidsStoredBidResponse(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSellwild, config.Adapter{
		Endpoint: "https://test.endpoint.com/exchange/openrtb2/{{.PublisherID}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Prebid Server's stored bid response: empty URI, no ImpIDs, and a body that is not a bid request.
	requestData := &adapters.RequestData{Body: []byte("Stored bid response for impression id: imp-1")}
	responseData := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body: []byte(`{"id":"test-request-id","cur":"USD","seatbid":[{"bid":[` +
			`{"id":"bid-1","impid":"imp-1","price":1.5,"mtype":1},` +
			`{"id":"bid-2","impid":"imp-1","price":1.5}]}]}`),
	}

	bidResponse, errs := bidder.MakeBids(&openrtb2.BidRequest{ID: "test-request-id"}, requestData, responseData)

	if assert.Len(t, bidResponse.Bids, 1) {
		assert.Equal(t, "bid-1", bidResponse.Bids[0].Bid.ID)
		assert.Equal(t, openrtb_ext.BidTypeBanner, bidResponse.Bids[0].BidType)
	}
	if assert.Len(t, errs, 1) {
		assert.EqualError(t, errs[0], "stored bid response bid bid-2 in imp imp-1 is missing mtype")
	}
}
