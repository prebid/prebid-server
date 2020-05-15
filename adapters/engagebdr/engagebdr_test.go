package engagebdr

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"net/http"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "engagebdrtest", NewEngageBDRBidder(new(http.Client), "http://dsp.bnmla.com/hb"))
}
