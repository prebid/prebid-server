package unicorn

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const testsDir = "unicorntest"

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, testsDir, NewUnicornBidder(http.DefaultClient, "https://jp.unicorn.com/tapjoy", "https://jp.unicorn.com/tapjoy"))
}
