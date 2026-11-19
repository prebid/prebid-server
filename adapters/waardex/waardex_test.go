package waardex

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
	bidder, buildErr := Builder(openrtb_ext.BidderWaardex, config.Adapter{
		Endpoint: "http://justbidit2.xyz:8800/hb?pubId={{.ZoneID}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

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

func TestMakeBidsStoredBidResponse(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderWaardex, config.Adapter{
		Endpoint: "http://justbidit2.xyz:8800/hb?pubId={{.ZoneID}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// Prebid Server hands a stored bid response to MakeBids with a plain-text placeholder body
	// instead of the outgoing request, so the media type has to come from the incoming request.
	internalRequest := &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "imp-1", Video: &openrtb2.Video{}}}}
	externalRequest := &adapters.RequestData{Body: []byte("Stored bid response for impression id: imp-1")}
	response := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"seatbid":[{"bid":[{"id":"bid-1","impid":"imp-1","price":1,"crid":"crid-1"}]}]}`),
	}

	bidderResponse, errs := bidder.MakeBids(internalRequest, externalRequest, response)

	assert.Empty(t, errs)
	if assert.Len(t, bidderResponse.Bids, 1) {
		assert.Equal(t, openrtb_ext.BidTypeVideo, bidderResponse.Bids[0].BidType)
		assert.Equal(t, openrtb2.MarkupVideo, bidderResponse.Bids[0].Bid.MType)
	}
}
