//go:build !integration
// +build !integration

package jampp

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const testsDir = "jampptest"

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, testsDir, NewJamppBidder(http.DefaultClient, "http://jampp.com/givemeads", "http://jampp-us-east.com/givemeads"))
}
