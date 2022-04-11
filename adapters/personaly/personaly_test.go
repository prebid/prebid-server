//go:build !integration
// +build !integration

package personaly

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const testsDir = "personalytest"

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, testsDir, NewPersonalyBidder(http.DefaultClient, "http://personaly.com/givemeads"))
}
