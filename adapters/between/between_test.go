package between

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	// to do: change to a production test endpoint
	endpoint := "http://127.0.0.1:8000/openrtb2/auction"
	adapterstest.RunJSONBidderTest(t, "between", NewBetweenBidder(endpoint))
}
