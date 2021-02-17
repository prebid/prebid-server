package adocean

import (
	"net/http"
	"testing"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "adoceantest", NewAdOceanBidder(new(http.Client), "https://{{.Host}}"))
}
