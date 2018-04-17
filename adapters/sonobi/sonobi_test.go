package sonobi

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"net/http"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "sonobitest", NewSonobiBidder(http.DefaultClient, "endpoint"))
}
