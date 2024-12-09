package videobyte

import (
	"testing"

	"github.com/prebid/prebid-server/v3/config"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder("videobyte", config.Adapter{Endpoint: "https://mock.videobyte.com"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
	adapterstest.RunJSONBidderTest(t, "videobytetest", bidder)
}
