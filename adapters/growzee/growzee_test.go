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

func TestMakeBidsOutboundWithoutOneImp(t *testing.T) {
	bidder, _ := Builder(openrtb_ext.BidderGrowzee, config.Adapter{
		Endpoint: "https://bid.example.com/openrtb?id={{.PublisherID}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	for _, body := range []string{`{}`, `{"imp":[{"id":"a"},{"id":"b"}]}`, `not json`} {
		resp, errs := bidder.MakeBids(&openrtb2.BidRequest{},
			&adapters.RequestData{Body: []byte(body)},
			&adapters.ResponseData{StatusCode: 200, Body: []byte(`{"seatbid":[]}`)})
		assert.Nil(t, resp, body)
		assert.Len(t, errs, 1, body)
	}
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderGrowzee, config.Adapter{
		Endpoint: "{{Malformed}}"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	assert.Error(t, buildErr)
}
