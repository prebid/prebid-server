package sonobi

import (
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
	"net/http"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	sonobiAdapter := NewSonobiBidder(new(http.Client), "https://apex.go.sonobi.com/prebid?partnerid=71d9d3d8af")
	adapterstest.RunJSONBidderTest(t, "sonobitest", sonobiAdapter)
}
