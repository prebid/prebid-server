package rtbstack

import (
	"testing"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRTBStack, config.Adapter{
		Endpoint: "https://{{.Region}}-adx-admixer.rtb-stack.com/pbs?ssp={{.SspID}}&endpoint={{.ZoneID}}&client={{.PartnerId}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "rtbstacktest", bidder)
}

func TestBuilderInvalidTemplate(t *testing.T) {
	_, err := Builder(openrtb_ext.BidderRTBStack, config.Adapter{
		Endpoint: "https://{{.Region}-bad-template"}, config.Server{})

	if err == nil {
		t.Fatal("Builder should return an error for an invalid endpoint template")
	}
}
