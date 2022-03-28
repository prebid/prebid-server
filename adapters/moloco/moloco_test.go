//go:build !integration
// +build !integration

package moloco

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const testsDir = "molocotest"

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, testsDir, NewMolocoBidder(http.DefaultClient, "https://bidfnt-us.adsmoloco.com/tapjoy", "https://bidfnt-us.adsmoloco.com/tapjoy", "https://bidfnt-eu.adsmoloco.com/tapjoy", "https://bidfnt-asia.adsmoloco.com/tapjoy"))
}
