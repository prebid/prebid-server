package taurusx

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const testsDir = "taurusxtest"

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, testsDir, NewTaurusXBidder(http.DefaultClient, "https://useast.taurusx.com/tapjoy", "https://useast.taurusx.com/tapjoy", "https://jp.taurusx.com/tapjoy", "https://sg.taurusx.com/tapjoy"))
}
