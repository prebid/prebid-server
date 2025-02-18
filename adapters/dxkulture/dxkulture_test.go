package dxkulture

import (
	"testing"

	"github.com/prebid/prebid-server/v2/config"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder("dxkulture", config.Adapter{Endpoint: "https://ads.dxkulture.com/pbs"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
	adapterstest.RunJSONBidderTest(t, "dxkulturetest", bidder)
}
