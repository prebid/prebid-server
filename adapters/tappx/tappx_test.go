package tappx

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"net/http"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "tappx", NewTappxBidder(new(http.Client), "http://tappx.com"))
}
