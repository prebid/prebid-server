//go:build !integration
// +build !integration

package tapjoy

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const testsDir = "tapjoytest"

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, testsDir, NewTapjoyBidder(http.DefaultClient, "http://tapjoy.com/givemeads"))
}
