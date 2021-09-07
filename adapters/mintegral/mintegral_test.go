package mintegral

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const testsDir = "mintegraltest"

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, testsDir, NewMintegralBidder(http.DefaultClient, "http://sg.mintegral.com/givemeads", "http://hk.mintegral.com/givemeads", "http://sg.mintegral.com/givemeads", "http://vg.mintegral.com/givemeads"))
}
