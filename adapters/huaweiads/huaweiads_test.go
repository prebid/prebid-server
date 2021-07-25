package huaweiads

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderHuaweiAds, config.Adapter{
		Endpoint:         "https://acd.op.hicloud.com/ppsadx/getResult",
		ExtraAdapterInfo: "{\"trackingUrl\":\"https://events-dra.op.hicloud.com/contserver/tracker/action\"}"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
	adapterstest.RunJSONBidderTest(t, "huaweiadstest", bidder)
}
