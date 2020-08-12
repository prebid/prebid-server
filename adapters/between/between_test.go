package between

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	// to do: change to a production test endpoint
	//endpoint := "http://ads.betweendigital.com/s2s"
	adapterstest.RunJSONBidderTest(t, "betweentest", NewBetweenBidder("http://{{.Host}}/"))
}
