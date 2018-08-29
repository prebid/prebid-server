package adkernelAdn

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "adkerneladntest", NewAdkernelAdnAdapter("http://{{.Host}}/rtbpub?account={{.PublisherID}}"))
}
