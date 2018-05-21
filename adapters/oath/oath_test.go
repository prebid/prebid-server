package oath

import (
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"net/http"
	"testing"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "oathtest", NewOathBidder(new(http.Client), "http://east-bid.ybp.yahoo.com/bid/appnexuspbs"))
}
