package tappx

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"net/http"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "tappxtest", NewTappxBidder(new(http.Client), "https://{{.Host}}"))
}
