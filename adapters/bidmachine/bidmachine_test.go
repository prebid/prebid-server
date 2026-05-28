package bidmachine

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBidmachine, config.Adapter{
		Endpoint: "https://{{.Host}}.bidmachine.io"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "bidmachinetest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderBidmachine, config.Adapter{
		Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

func TestBuildEndpointURLRejectsUnsafeURLParts(t *testing.T) {
	bidder := &adapter{endpoint: template.Must(template.New("endpointTemplate").Parse("https://{{.Host}}.bidmachine.io"))}

	_, err := bidder.buildEndpointURL(openrtb_ext.ExtImpBidmachine{Host: "127.0.0.1:6060/debug/pprof#", Path: "auction/rtb/v2", SellerID: "seller"})
	assert.Error(t, err)

	_, err = bidder.buildEndpointURL(openrtb_ext.ExtImpBidmachine{Host: "api-us", Path: "auction#fragment", SellerID: "seller"})
	assert.Error(t, err)

	_, err = bidder.buildEndpointURL(openrtb_ext.ExtImpBidmachine{Host: "api-us", Path: "auction/rtb/v2", SellerID: "seller/../other"})
	assert.Error(t, err)

	_, err = bidder.buildEndpointURL(openrtb_ext.ExtImpBidmachine{Host: "api-us", Path: "auction/rtb/v2", SellerID: ".."})
	assert.Error(t, err)

	_, err = bidder.buildEndpointURL(openrtb_ext.ExtImpBidmachine{Host: "api-us", Path: "auction/../admin", SellerID: "seller"})
	assert.Error(t, err)
}
