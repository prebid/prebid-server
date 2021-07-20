package aarki

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const testsDir = "aarkitest"

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, testsDir, NewAarkiBidder(http.DefaultClient, "https://tapjoy.aarki.net/rtb/bid", "https://tapjoy.aarki.net/rtb/bid"))
}
