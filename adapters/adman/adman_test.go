package adman

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	admanAdapter := NewAdmanBidder(new(http.Client), "http://eu-ams-1.admanmedia.com/?c=o&m=ortb")
	adapterstest.RunJSONBidderTest(t, "admantest", admanAdapter)
}
