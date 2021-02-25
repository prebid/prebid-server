package adyoulike

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
)

const testsBidderEndpoint = "https://localhost/bid/4"

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdyoulike, config.Adapter{
		Endpoint: testsBidderEndpoint})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "adyouliketest", bidder)
}

func TestMakeRequestInvalidParams(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdyoulike, config.Adapter{
		Endpoint: testsBidderEndpoint})

	assert.Nil(t, buildErr, "buildErr must be nil")

	var reqInfo adapters.ExtraRequestInfo
	var req openrtb.BidRequest
	req.ID = "test_id"

	reqExt := `{"prebid":{}}`
	impExt := `{"bidder":{"placementId":123}}`
	req.Ext = []byte(reqExt)

	req.Imp = append(req.Imp, openrtb.Imp{ID: "1_0", Ext: []byte(impExt)})

	bids, errs := bidder.MakeRequests(&req, &reqInfo)

	assert.EqualError(t, errs[0], "Key path not found")
	assert.Len(t, bids, 0)
}

func TestOpenRTBSurpriseResponse(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdyoulike, config.Adapter{
		Endpoint: testsBidderEndpoint})

	assert.Nil(t, buildErr, "buildErr must be nil")

	bidResponse, errs := bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusNoContent, Body: []byte("")})
	if bidResponse != nil && errs != nil {
		t.Fatalf("Expected no bids and no errors. Got %d bids and %d", len(bidResponse.Bids), len(errs))
	}

	bidResponse, errs = bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusServiceUnavailable, Body: []byte("")})
	if bidResponse != nil || len(errs) != 1 {
		t.Fatalf("Expected one error and no bids. Got %d bids and %d", len(bidResponse.Bids), len(errs))
	}

	bidResponse, errs = bidder.MakeBids(nil, nil,
		&adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte("{:'not-valid-json'}")})
	if bidResponse != nil || len(errs) != 1 {
		t.Fatalf("Expected one error and no bids. Got %d bids and %d", len(bidResponse.Bids), len(errs))
	}
}
