package sspBC

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSspBC, config.Adapter{
		Endpoint: "http://ssp.wp.test/bidder/"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "sspbctest", bidder)
}

func TestInvalidEndpointURL(t *testing.T) {
	invalidEndpointURL := "http://ssp.wp.test   /bidder/"

	_, buildErr := Builder(openrtb_ext.BidderSspBC, config.Adapter{
		Endpoint: invalidEndpointURL}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr == nil {
		t.Fatalf("Adapter allowed invalid endpoint URL %v", invalidEndpointURL)
	}
}

func TestMakeRequests(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderSspBC, config.Adapter{
		Endpoint: "http://ssp.wp.test/bidder/"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	w, h := int64(300), int64(250)

	imp1 := openrtb2.Imp{
		ID:  "imp1",
		Ext: json.RawMessage("{\"bidder\":{\"siteId\": \"237503\", \"id\": \"005\"}}"),
		Banner: &openrtb2.Banner{
			W: &w,
			H: &h,
			Format: []openrtb2.Format{
				{W: 300, H: 250},
			},
		}}

	inputRequest := openrtb2.BidRequest{
		User: &openrtb2.User{ID: "Test123"},
		Imp:  []openrtb2.Imp{imp1},
		Site: &openrtb2.Site{
			Publisher: &openrtb2.Publisher{
				ID: "12345",
			},
		},
		ID: "1234",
	}

	inputExtraRequestInfo := adapters.ExtraRequestInfo{
		PbsEntryPoint: metrics.ReqTypeORTB2Web,
	}

	_, err := bidder.MakeRequests(&inputRequest, &inputExtraRequestInfo)

	if err != nil {
		t.Fatalf("Make requests function returned unexpected error %v", err)
	}
}
