package crossinstall

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const testsDir = "crossinstalltest"

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, testsDir, NewCrossInstallBidder(http.DefaultClient, "https://useast.crossinstall.com/tapjoy", "https://useast.crossinstall.com/tapjoy", "https://uswest.crossinstall.com/tapjoy"))
}
