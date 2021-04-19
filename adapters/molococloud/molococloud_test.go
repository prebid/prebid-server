package molococloud

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const testsDir = "molococloudtest"

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, testsDir, NewMolocoCloudBidder(http.DefaultClient, "https://bidfnt-us.adsmoloco.com/private_tapjoy", "https://bidfnt-us.adsmoloco.com/private_tapjoy", "https://bidfnt-us.adsmoloco.com/private_tapjoy", "https://bidfnt-us.adsmoloco.com/private_tapjoy"))
}
