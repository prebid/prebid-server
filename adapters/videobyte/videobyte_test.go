package videobyte

import (
	"testing"

	"github.com/prebid/prebid-server/config"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder("videobyte", config.Adapter{Endpoint: "https://mock.videobyte.com"}, config.Server{ExternalUrl: "http://hosturl.com", GdprID: 1, Datacenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
	adapterstest.RunJSONBidderTest(t, "videobytetest", bidder)
}
