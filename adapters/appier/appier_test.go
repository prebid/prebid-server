//go:build !integration
// +build !integration

package appier

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const testsDir = "appiertest"

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, testsDir, NewAppierBidder(http.DefaultClient, "http://useast.appier.com/givemeads", "http://useast.appier.com/givemeads", "http://emea.appier.com/givemeads", "http://jp.appier.com/givemeads", "http://sg.appier.com/givemeads"))
}
