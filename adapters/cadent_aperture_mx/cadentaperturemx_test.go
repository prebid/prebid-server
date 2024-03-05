package cadentaperturemx

import (
	"testing"

	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderCadentApertureMX, config.Adapter{
		Endpoint: "https://hb.emxdgt.com"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	setTesting(bidder)
	adapterstest.RunJSONBidderTest(t, "cadent_aperture_mxtest", bidder)
}

func setTesting(bidder adapters.Bidder) {
	bidderCadentApertureMX, _ := bidder.(*adapter)
	bidderCadentApertureMX.testing = true
}
