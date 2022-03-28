//go:build !integration
// +build !integration

package kadenai

import (
	"net/http"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

const testsDir = "kadenaitest"

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, testsDir, NewKadenAIBidder(http.DefaultClient, "http://kadenai.com/givemeads"))
}
